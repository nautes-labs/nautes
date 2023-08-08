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

package data

import (
	"context"
	"fmt"

	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMap struct {
	Data map[string]string `yaml:"data"`
}

type DexConfig struct {
	Issuer  string `yaml:"issuer"`
	Storage struct {
		Type string `yaml:"type"`
	} `yaml:"storage"`
	Web struct {
		HTTP string `yaml:"http"`
	} `yaml:"web"`
	Connectors []struct {
		Type   string `yaml:"type"`
		ID     string `yaml:"id"`
		Name   string `yaml:"name"`
		Config struct {
			BaseURL      string `yaml:"baseURL"`
			ClientID     string `yaml:"clientID"`
			ClientSecret string `yaml:"clientSecret"`
			RedirectURI  string `yaml:"redirectURI"`
		} `yaml:"config"`
	} `yaml:"connectors"`
	Oauth2 struct {
		SkipApprovalScreen bool `yaml:"skipApprovalScreen"`
	} `yaml:"oauth2"`
	StaticClients []struct {
		ID           string   `yaml:"id"`
		RedirectURIs []string `yaml:"redirectURIs"`
		Name         string   `yaml:"name"`
		Secret       string   `yaml:"secret"`
	} `yaml:"staticClients"`
}

type Dex struct {
	k8sClient client.Client
}

func (d *Dex) UpdateRedirectURIs(url string) error {
	cm, err := d.GetDexConfig()
	if err != nil {
		return err
	}

	cm.Data["config.yaml"], err = UpdateConfigURIs(cm.Data["config.yaml"], url)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	err = d.k8sClient.Update(context.Background(), cm)
	if err != nil {
		return err
	}

	return nil
}

func (d *Dex) RemoveRedirectURIs(url string) error {
	cm, err := d.GetDexConfig()
	if err != nil {
		return err
	}

	cm.Data["config.yaml"], err = RemoveConfigURIs(cm.Data["config.yaml"], url)
	if err != nil {
		return fmt.Errorf("failed to update configmap dex in namespace dex, err: %w", err)
	}

	err = d.k8sClient.Update(context.Background(), cm)
	if err != nil {
		return err
	}

	return nil
}

func (d *Dex) GetDexConfig() (*corev1.ConfigMap, error) {
	var namespace = "dex"
	var configMapName = "dex"
	var ns corev1.Namespace
	err := d.k8sClient.Get(context.Background(), client.ObjectKey{Name: namespace}, &ns)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s, err: %w", namespace, err)
	}

	cm := &corev1.ConfigMap{}
	err = d.k8sClient.Get(context.Background(), client.ObjectKey{Name: configMapName, Namespace: namespace}, cm)
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap dex in namespace %s, err: %w", namespace, err)
	}

	return cm, nil
}

func UpdateConfigURIs(configYAML string, redirectURIs string) (string, error) {
	config := DexConfig{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return "", err
	}

	if len(config.StaticClients) > 0 {
		config.StaticClients[0].RedirectURIs = utilstring.AddIfNotExists(config.StaticClients[0].RedirectURIs, redirectURIs)
	}

	configYAMLBytes, err := yaml.Marshal(&config)
	if err != nil {
		return "", err
	}

	return string(configYAMLBytes), nil
}

func RemoveConfigURIs(configYAML string, redirectURIs string) (string, error) {
	config := DexConfig{}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
		return "", err
	}

	if len(config.StaticClients) > 0 {
		config.StaticClients[0].RedirectURIs = utilstring.RemoveStringFromSlice(config.StaticClients[0].RedirectURIs, redirectURIs)
	}

	configYAMLBytes, err := yaml.Marshal(&config)
	if err != nil {
		return "", err
	}

	return string(configYAMLBytes), nil
}
