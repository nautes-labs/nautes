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
	"os"

	"gopkg.in/yaml.v2"
)

const (
	ThirdPartComponentsPath     = "/opt/nautes/config/thirdPartComponents.yaml"
	ComponentsOfClusterTypePath = "/opt/nautes/config/componentCategoryDefinition.yaml"
	ClusterCommonConfigPath     = "/opt/nautes/config/clusterCommonConfig.yaml"
	EnvthirdPartComponents      = "THIRD_PART_COMPONENTS"
	EnvComponentsOfClusterType  = "COMPONENTS_OF_CLUSTER_TYPE"
	EnvClusterCommonConfig      = "CLUSTER_COMMON_CONFIG"
)

// Configuration information of third-party components required for cluster registration.
type ThridPartComponent struct {
	Name        string              `yaml:"name"`
	Namespace   string              `yaml:"namespace"`
	Type        string              `yaml:"type"`
	Default     bool                `yaml:"default"`
	General     bool                `yaml:"general"`
	Properties  []Propertie         `yaml:"properties"`
	InstallPath map[string][]string `yaml:"installPath"`
}

type Propertie struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	RegexPattern string `yaml:"regexPattern"`
	Required     bool   `yaml:"required"`
	Default      string `yaml:"default"`
}

// Components required for cluster registration.
type ComponentsOfClusterType struct {
	Host   []string    `yaml:"host"`
	Worker ClusterType `yaml:"worker"`
}

type ClusterType struct {
	Physical ClusterWorkerType `yaml:"physical"`
	Virtual  ClusterWorkerType `yaml:"virtual"`
}

type ClusterWorkerType struct {
	Deployment []string `yaml:"deployment"`
	Pipeline   []string `yaml:"pipeline"`
}

// General configuration required for cluster registration.
type ClusterOperatorCommonConfig struct {
	Save   ClusterConfig `yaml:"save"`
	Remove ClusterConfig `yaml:"remove"`
}

type ClusterConfig struct {
	Host   []string            `yaml:"host"`
	Worker ClusterCommonConfig `yaml:"worker"`
}

type ClusterCommonConfig struct {
	Physical []string `yaml:"physical"`
	Virtual  []string `yaml:"virtual"`
}

func GetThirdPartComponentsList() ([]*ThridPartComponent, error) {
	content, err := loadThirdPartComponentsList()
	if err != nil {
		return nil, err
	}

	components := []*ThridPartComponent{}
	err = yaml.Unmarshal([]byte(content), &components)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal third part component configuration, err: %s", err)
	}

	return components, nil
}

func loadThirdPartComponentsList() (string, error) {
	content, err := loadConfigFile(ThirdPartComponentsPath, EnvthirdPartComponents)
	if err != nil {
		return "", err
	}

	return content, nil
}

func GetThirdPartComponentByName(list []*ThridPartComponent, name string) *ThridPartComponent {
	for _, item := range list {
		if item.Name == name {
			return item
		}
	}

	return nil
}

func GetThirdPartComponentByType(list []*ThridPartComponent, componentType string) *ThridPartComponent {
	for _, item := range list {
		if item.Type == componentType {
			return item
		}
	}

	return nil
}

func GetComponentsOfClusterType() (*ComponentsOfClusterType, error) {
	content, err := loadComponentsOfClusterType()
	if err != nil {
		return nil, err
	}

	componentTypes := &ComponentsOfClusterType{}
	err = yaml.Unmarshal([]byte(content), &componentTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %s", err)
	}

	return componentTypes, nil
}

func GetComponentsByClusterType() {

}

func loadComponentsOfClusterType() (string, error) {
	content, err := loadConfigFile(ComponentsOfClusterTypePath, EnvComponentsOfClusterType)
	if err != nil {
		return "", err
	}

	return content, nil
}

func GetClusterCommonConfig() (*ClusterOperatorCommonConfig, error) {
	content, err := loadClusterCommonConfig()
	if err != nil {
		return nil, err
	}

	config := &ClusterOperatorCommonConfig{}
	err = yaml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %s", err)
	}

	return config, nil
}

func loadClusterCommonConfig() (string, error) {
	content, err := loadConfigFile(ClusterCommonConfigPath, EnvClusterCommonConfig)
	if err != nil {
		return "", err
	}

	return content, nil
}

func loadConfigFile(path, env string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil && env != "" {
		if alternatePath := os.Getenv(env); alternatePath != "" {
			bytes, err = os.ReadFile(alternatePath)
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to read configuration file: %v", err)
	}

	return string(bytes), nil
}
