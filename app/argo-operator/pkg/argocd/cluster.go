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

const Authorization = "Authorization"

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

	address := fmt.Sprintf("%s/api/v1/clusters?upsert=true", c.argocd.client.url)
	bearerToken := spliceBearerToken(c.argocd.client.token)

	result, err := c.argocd.http.R().
		SetHeader(Authorization, bearerToken).
		SetBody(data).
		Post(address)
	if err != nil {
		return err
	}
	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("failed to sync cluster, the status code is %d,  err: %s", result.StatusCode(), result)
}

// remove cluster from argo
func (c *cluster) DeleteCluster(clusterURL string) (string, error) {
	_, err := c.GetCluster(clusterURL)
	if err != nil {
		return "", err
	}

	clusterName := url.QueryEscape(clusterURL)
	address := fmt.Sprintf("%s/api/v1/clusters/%s", c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)

	result, err := c.argocd.http.R().
		SetHeader(Authorization, bearerToken).
		Delete(address)
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
	address := splitRequestClusterUrl(c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)

	result, err := c.argocd.http.R().
		SetHeader(Authorization, bearerToken).
		Get(address)
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

	return nil, fmt.Errorf("failed to get cluster information, err: %s", result)
}

func (c *cluster) GetCluster(clusterURL string) (*ClusterData, error) {
	clusterName := url.QueryEscape(clusterURL)
	address := splitRequestClusterUrl(c.argocd.client.url, clusterName)
	bearerToken := spliceBearerToken(c.argocd.client.token)

	result, err := c.argocd.http.R().
		SetHeader(Authorization, bearerToken).
		Get(address)
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

	return nil, fmt.Errorf("failed to get cluster, err: %s", result)
}

func (c *cluster) UpdateCluster(clusterInstance *ClusterInstance) error {
	var serverVersion = "1.21+"

	clusterName := url.QueryEscape(clusterInstance.ApiServer)
	address := splitRequestClusterUrl(c.argocd.client.url, clusterName)

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

	result, err := c.argocd.http.R().
		SetHeader(Authorization, bearerToken).
		SetBody(data).
		Put(address)
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
