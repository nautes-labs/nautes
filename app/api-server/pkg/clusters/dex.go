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
	"fmt"

	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

const (
	DexConfigName = "config.yaml"
)

func (c *ClusterManagement) GetDeploymentRedirectURI(params *ClusterRegistrationParams) (string, error) {
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

func (c *ClusterManagement) GetOAuthProxyRedirectURI(params *ClusterRegistrationParams) (string, error) {
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

func (c *ClusterManagement) AppendDexRedirectURIs(url string) error {
	cm, err := c.k8sClient.GetConfigMap(client.ObjectKey{Namespace: "dex", Name: "dex"})
	if err != nil {
		return err
	}

	err = c.UpdateDexConfig(cm, url, appendDexRedirectURI)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	err = c.k8sClient.UpdateConfigMap(cm)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClusterManagement) RemoveDexRedirectURIs(url string) error {
	cm, err := c.k8sClient.GetConfigMap(client.ObjectKey{Namespace: "dex", Name: "dex"})
	if err != nil {
		return err
	}

	err = c.UpdateDexConfig(cm, url, removeDexRedirectURI)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	err = c.k8sClient.UpdateConfigMap(cm)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClusterManagement) UpdateDexConfig(cm *v1.ConfigMap, redirectURIs string, fn DexCallback) error {
	config := &DexConfig{}
	if err := yaml.Unmarshal([]byte(cm.Data[DexConfigName]), config); err != nil {
		return err
	}

	if len(config.StaticClients) > 0 {
		if err := fn(config, redirectURIs); err != nil {
			return err
		}
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	cm.Data[DexConfigName] = string(bytes)

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
