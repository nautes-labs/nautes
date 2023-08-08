// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocd

import (
	"encoding/json"
	"fmt"
	"net/url"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type ClusterOperation interface {
	GetClusterInfo(clusterURL string) (*ClusterResponse, error)
	GetCluster(clusterURL string) (*ClusterData, error)
	CreateCluster(clusterInstance *ClusterInstance) error
	DeleteCluster(clusterURL string) (string, error)
	UpdateCluster(clusterInstance *ClusterInstance) error
}

type cluster struct {
	argocd *ArgocdClient
}

func NewArgocdCluster(argocd *ArgocdClient) ClusterOperation {
	return &cluster{argocd: argocd}
}

type tlsClientConfig struct {
	Insecure bool   `json:"insecure"`
	CaData   string `json:"caData"`
	CertData string `json:"certData"`
	KeyData  string `json:"keyData"`
}

type ClusterDataConfig struct {
	TlsClientConfig tlsClientConfig `json:"tlsClientConfig"`
}

type Info struct {
	serverVersion string
}

type ConnectionState struct {
	AttemptedAt string `json:"attemptedAt"`
	Message     string `json:"message"`
	Status      string `json:"status"`
}

type ClusterResponse struct {
	Name            string          `json:"name"`
	ConnectionState ConnectionState `json:"connectionState"`
	Server          string          `json:"server"`
}

type ClusterData struct {
	Server string            `json:"server"`
	Name   string            `json:"name"`
	Config ClusterDataConfig `json:"config"`
	Info   Info              `json:"info"`
	Labels map[string]string `json:"labels"`
}

type ClusterInstance struct {
	Name            string                        `json:"name"`
	ApiServer       string                        `json:"apiServer"`
	ClusterType     string                        `json:"clusterType"`
	HostCluster     string                        `json:"hostType"`
	Provider        string                        `json:"provider"`
	CertificateData string                        `json:"certficateData"`
	CaData          string                        `json:"caData"`
	KeyData         string                        `json:"keyData"`
	Usage           resourcev1alpha1.ClusterUsage `json:"usage"`
}

// sync cluster to argocd
func (c *cluster) CreateCluster(clusterInstance *ClusterInstance) error {
	var serverVersion = "1.21+"

	tlsConfig := tlsClientConfig{
		Insecure: false,
		CaData:   clusterInstance.CaData,
		CertData: clusterInstance.CertificateData,
		KeyData:  clusterInstance.KeyData,
	}

	data := &ClusterData{
		Server: clusterInstance.ApiServer,
		Name:   clusterInstance.Name,
		Config: ClusterDataConfig{
			TlsClientConfig: tlsConfig,
		},
		Info: Info{
			serverVersion: serverVersion,
		},
	}

	if clusterInstance.Usage == resourcev1alpha1.CLUSTER_USAGE_HOST {
		data.Labels = map[string]string{
			"cluster.nautes.io/usage": "host",
		}
	} else {
		data.Labels = map[string]string{
			"cluster.nautes.io/usage": "worker",
		}
	}

	url := fmt.Sprintf("%s/api/v1/clusters?upsert=true", c.argocd.client.url)
	bearerToken := spliceBearerToken(c.argocd.client.token)

	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		SetBody(data).
		Post(url)
	if err != nil {
		return err
	}
	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("Failed to sync cluster, the status code is %d,  err: %s", result.StatusCode(), result)
}

// remove cluster from argo
func (c *cluster) DeleteCluster(clusterURL string) (string, error) {
	_, err := c.GetCluster(clusterURL)
	if err != nil {
		return "", err
	}

	clusterName := url.QueryEscape(clusterURL)
	url := fmt.Sprintf("%s/api/v1/clusters/%s", c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		Delete(url)
	if err != nil {
		return "", err
	}

	if result.StatusCode() == 200 {
		return string(result.Body()), nil
	}

	return "", fmt.Errorf("DeleteCluster result: %v", result)
}

func (c *cluster) GetClusterInfo(clusterURL string) (*ClusterResponse, error) {
	clusterName := url.QueryEscape(clusterURL)
	url := splitRequestClusterUrl(c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		Get(url)
	if err != nil {
		return nil, err
	}

	if result.StatusCode() == 403 {
		return nil, &IsNoAuth{
			Message: string(result.Body()),
		}
	}

	if result.StatusCode() == 200 {
		response := &ClusterResponse{}
		err := json.Unmarshal(result.Body(), response)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, fmt.Errorf("Get cluster info result: %v", result)
}

func (c *cluster) GetCluster(clusterURL string) (*ClusterData, error) {
	clusterName := url.QueryEscape(clusterURL)
	url := splitRequestClusterUrl(c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		Get(url)
	if err != nil {
		return nil, err
	}

	if result.StatusCode() == 200 {
		clusterData := &ClusterData{}
		err := json.Unmarshal(result.Body(), clusterData)
		if err != nil {
			return nil, err
		}

		return clusterData, nil
	}

	return nil, fmt.Errorf("Get cluster result: %v", result)
}

func (c *cluster) UpdateCluster(clusterInstance *ClusterInstance) error {
	var serverVersion = "1.21+"

	clusterName := url.QueryEscape(clusterInstance.ApiServer)
	url := splitRequestClusterUrl(c.argocd.client.url, clusterName)

	tlsConfig := tlsClientConfig{
		Insecure: false,
		CaData:   clusterInstance.CaData,
		CertData: clusterInstance.CertificateData,
		KeyData:  clusterInstance.KeyData,
	}

	data := &ClusterData{
		Server: clusterInstance.ApiServer,
		Name:   clusterInstance.Name,
		Config: ClusterDataConfig{
			TlsClientConfig: tlsConfig,
		},
		Info: Info{
			serverVersion: serverVersion,
		},
	}

	if clusterInstance.Usage == resourcev1alpha1.CLUSTER_USAGE_HOST {
		data.Labels = map[string]string{
			"cluster.nautes.io/usage": "host",
		}
	} else {
		data.Labels = map[string]string{
			"cluster.nautes.io/usage": "worker",
		}
	}

	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		SetBody(data).
		Put(url)
	if err != nil {
		return err
	}

	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("UpdateCluster result: %v", result)
}

func splitRequestClusterUrl(host, name string) string {
	return fmt.Sprintf("%s/api/v1/clusters/%s?id.type=url", host, name)
}
