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

package cluster

import (
	"encoding/json"
	"fmt"
	"os"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type DexConfig struct {
	Issuer        string          `yaml:"issuer"`
	Storage       Storage         `yaml:"storage"`
	Web           Web             `yaml:"web"`
	Connectors    []Connector     `yaml:"connectors"`
	OAuth2        OAuth           `yaml:"oauth2"`
	StaticClients []StaticClients `yaml:"staticClients"`
}

type OAuth struct {
	SkipApprovalScreen bool `yaml:"skipApprovalScreen"`
}

type Connector struct {
	Type   string           `yaml:"type"`
	ID     string           `yaml:"id"`
	Name   string           `yaml:"name"`
	Config ConnectorsConfig `yaml:"config"`
}

type ConnectorsConfig struct {
	BaseURL      string `yaml:"baseURL"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	RedirectURI  string `yaml:"redirectURI"`
}

type Web struct {
	HTTP string `yaml:"http"`
}

type Storage struct {
	Type string `yaml:"type"`
}

type StaticClients struct {
	ID           string   `yaml:"id"`
	RedirectURIs []string `yaml:"redirectURIs"`
	Name         string   `yaml:"name"`
	Secret       string   `yaml:"secret"`
}

func (c *ClusterManagement) getExistDeploymentRedirectURI(params *ClusterRegistrationParams, tenantRepoDir, clusterName string) (string, error) {
	cluster, err := c.getExistCluster(tenantRepoDir, clusterName)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	copyParams := *params
	copyParams.Cluster = cluster

	return c.getDeploymentRedirectURI(&copyParams)
}

func (c *ClusterManagement) getDeploymentRedirectURI(params *ClusterRegistrationParams) (string, error) {
	deploy, err := NewDeploymentServer(params.Cluster)
	if err != nil {
		return "", err
	}

	url, err := deploy.GetOauthURL(params)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (c *ClusterManagement) getExistOAuthProxyRedirectURI(params *ClusterRegistrationParams, tenantRepoDir, clusterName string) (string, error) {
	cluster, err := c.getExistCluster(tenantRepoDir, clusterName)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	copyParams := *params
	copyParams.Cluster = cluster

	return c.getOAuthProxyRedirectURI(&copyParams)
}

func (c *ClusterManagement) getOAuthProxyRedirectURI(params *ClusterRegistrationParams) (string, error) {
	oauth, err := NewOAuthProxyServer(params.Cluster)
	if err != nil {
		return "", err
	}

	url := oauth.GenerateOAuthProxyRedirect(params.Cluster)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (c *ClusterManagement) appendDexRedirectURIs(cm *v1.ConfigMap, url string) error {
	err := c.updateDexConfig(cm, url, appendDexRedirectURI)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	return nil
}

func (c *ClusterManagement) removeDexRedirectURIs(cm *v1.ConfigMap, url string) error {
	err := c.updateDexConfig(cm, url, removeDexRedirectURI)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	return nil
}

func (c *ClusterManagement) updateDexConfig(cm *v1.ConfigMap, redirectURIs string, fn DexCallback) error {
	config := &DexConfig{}
	if err := yaml.Unmarshal([]byte(cm.Data["config.yaml"]), config); err != nil {
		return err
	}

	if len(config.StaticClients) == 0 {
		config.StaticClients = []StaticClients{
			{
				RedirectURIs: make([]string, 0),
			},
		}
	}

	if err := fn(config, redirectURIs); err != nil {
		return err
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	cm.Data["config.yaml"] = string(bytes)

	return nil
}

type DexCallback func(dex *DexConfig, url string) error

func appendDexRedirectURI(dex *DexConfig, url string) error {
	if len(dex.StaticClients) == 0 {
		return fmt.Errorf("failed to get static clients for dex config")
	}

	dex.StaticClients[0].RedirectURIs = utilstring.AddIfNotExists(dex.StaticClients[0].RedirectURIs, url)

	return nil
}

func removeDexRedirectURI(dex *DexConfig, url string) error {
	if len(dex.StaticClients) == 0 {
		return fmt.Errorf("failed to get static clients for dex config")
	}

	dex.StaticClients[0].RedirectURIs = utilstring.RemoveStringFromSlice(dex.StaticClients[0].RedirectURIs, url)

	return nil
}

func (c *ClusterManagement) getExistCluster(tenantRepoDir string, clusterName string) (*resourcev1alpha1.Cluster, error) {
	clusterResourcePath := fmt.Sprintf("%s/%s.yaml", concatClustersDir(tenantRepoDir), clusterName)
	yamlData, err := c.file.ReadFile(clusterResourcePath)
	if err != nil {
		return nil, err
	}

	var data interface{}
	var oldCluster = resourcev1alpha1.Cluster{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &oldCluster)
	if err != nil {
		return nil, err
	}

	return &oldCluster, nil
}

func (c *ClusterManagement) getSubClusterByHostCluster(tenantRepoDir string, hostClusterName string) ([]*resourcev1alpha1.Cluster, error) {
	clusterDir := concatClustersDir(tenantRepoDir)
	files, err := c.file.ListFilesInDirectory(clusterDir)
	if err != nil {
		return nil, err
	}

	clusters := make([]*resourcev1alpha1.Cluster, 0)

	for _, filePath := range files {
		data, err := c.file.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		c := resourcev1alpha1.Cluster{}
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}
		if c.Name != "" &&
			c.Namespace != "" &&
			c.Spec.HostCluster == hostClusterName {
			clusters = append(clusters, &c)
		}
	}

	return clusters, nil
}
