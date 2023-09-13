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
	ThirdPartComponentsFile        = "thirdPartComponents.yaml"
	ComponentCategoryFile          = "componentCategoryDefinition.yaml"
	ClusterCommonConfigFile        = "clusterCommonConfig.yaml"
	EnvthirdPartComponents         = "THIRD_PART_COMPONENTS"
	EnvComponentCategoryDefinition = "COMPONENT_CATEGORY_DEFINITION"
	EnvClusterCommonConfig         = "CLUSTER_COMMON_CONFIG"
	DefaultConfigPath              = "/opt/nautes/config"
)

var (
	// componentTypeMap is a map of component type names. To add a new component type configuration,
	// you should follow the format below and provide a brief description for clarity.
	//
	// Example:
	// "newComponentType": true,  // Description of what the new component type does.
	//
	// Please ensure that the component type name is in double quotes, followed by a colon and "true".
	// Add a comment after the colon to describe the purpose or functionality of the component type.
	componentTypeMap = map[string]bool{
		"certManagement":      true,
		"deployment":          true,
		"eventListener":       true,
		"gateway":             true,
		"multiTenant":         true,
		"pipeline":            true,
		"progressiveDelivery": true,
		"secretManagement":    true,
		"secretSync":          true,
		"oauthProxy":          true,
	}
)

var (
	ThirdPartComponentsPath         = fmt.Sprintf("%s/%s", DefaultConfigPath, ThirdPartComponentsFile)
	ComponentCategoryDefinitionPath = fmt.Sprintf("%s/%s", DefaultConfigPath, ComponentCategoryFile)
	ClusterCommonConfigPath         = fmt.Sprintf("%s/%s", DefaultConfigPath, ClusterCommonConfigFile)
)

const (
	CLUSTER_TYPE_PHYSICAL = "physical"
	CLUSTER_TYPE_VIRTUAL  = "virtual"
)

const (
	CLUSTER_USAGE_HOST   = "host"
	CLUSTER_USAGE_WORKER = "worker"
)

const (
	ClusterWorkTypeDeployment = "deployment"
	ClusterWorkTypePipeline   = "pipeline"
)

type ClusterComponentConfig struct {
	thirdPartComponentlist []*ThridPartComponent
	clusterCommonConfig    *ClusterCommonConfig
	componentsDefinition   *UsageComponentDefinition
}

// Configuration information of third-party components required for cluster registration.
type ThridPartComponent struct {
	Name        string
	Namespace   string
	Type        string
	Default     bool
	General     bool
	Properties  []Propertie
	InstallPath map[string][]string
}

type Propertie struct {
	Name         string
	Type         string
	RegexPattern string
	Required     bool
	Default      string
}

// Components required for cluster registration.
type UsageComponentDefinition struct {
	Host   ComponentList
	Worker ClusterTypeComponentDefinition
}

type ClusterTypeComponentDefinition struct {
	Physical WorkerTypeComponentDefinition
	Virtual  WorkerTypeComponentDefinition
}

type WorkerTypeComponentDefinition struct {
	Deployment ComponentList
	Pipeline   ComponentList
}

type ComponentList []string

// General configuration required for cluster registration.
type ClusterCommonConfig struct {
	Save   EnvironmentCommonConfig
	Remove EnvironmentCommonConfig
}

type EnvironmentCommonConfig struct {
	Host   []string
	Worker CommonConfig
}

type CommonConfig struct {
	Physical []string
	Virtual  []string
}

type ClusterInfo struct {
	Name        string
	Usage       string
	ClusterType string
	WorkType    string
}

func NewClusterComponentConfig() (*ClusterComponentConfig, error) {
	list, err := GetThirdPartComponentsList(ThirdPartComponentsPath)
	if err != nil {
		return nil, err
	}

	componentsDefinition, err := GetComponentCategoryDefinition(ComponentCategoryDefinitionPath)
	if err != nil {
		return nil, err
	}

	config, err := GetClusterCommonConfig(ClusterCommonConfigPath)
	if err != nil {
		return nil, err
	}

	return &ClusterComponentConfig{
		thirdPartComponentlist: list,
		componentsDefinition:   componentsDefinition,
		clusterCommonConfig:    config,
	}, nil
}

func (c *ClusterComponentConfig) GetThirdPartComponentByType(componentType string) *ThridPartComponent {
	for _, item := range c.thirdPartComponentlist {
		if item.Type == componentType {
			return item
		}
	}

	return nil
}

func (c *ClusterComponentConfig) GetThirdPartComponentByName(name string) (*ThridPartComponent, error) {
	for _, item := range c.thirdPartComponentlist {
		if item.Name == name {
			return item, nil
		}
	}

	return nil, fmt.Errorf("failed to get component %s from third-part components list", name)
}

func (c *ClusterComponentConfig) getComponentsDefinitionMap() map[string]ComponentList {
	componentsDefinitionMap := make(map[string]ComponentList, 0)
	componentsDefinitionMap[CLUSTER_USAGE_HOST+"|"+CLUSTER_TYPE_PHYSICAL+"|"] = c.componentsDefinition.Host
	componentsDefinitionMap[CLUSTER_USAGE_WORKER+"|"+CLUSTER_TYPE_PHYSICAL+"|"+ClusterWorkTypeDeployment] = c.componentsDefinition.Worker.Physical.Deployment
	componentsDefinitionMap[CLUSTER_USAGE_WORKER+"|"+CLUSTER_TYPE_VIRTUAL+"|"+ClusterWorkTypeDeployment] = c.componentsDefinition.Worker.Virtual.Deployment
	componentsDefinitionMap[CLUSTER_USAGE_WORKER+"|"+CLUSTER_TYPE_PHYSICAL+"|"+ClusterWorkTypePipeline] = c.componentsDefinition.Worker.Physical.Pipeline
	componentsDefinitionMap[CLUSTER_USAGE_WORKER+"|"+CLUSTER_TYPE_VIRTUAL+"|"+ClusterWorkTypePipeline] = c.componentsDefinition.Worker.Virtual.Pipeline

	return componentsDefinitionMap
}

func (c *ClusterComponentConfig) GetClusterComponentsDefinition(cluster *ClusterInfo) ([]string, error) {
	componentsDefinitionMap := c.getComponentsDefinitionMap()
	key := fmt.Sprintf("%s|%s|%s", cluster.Usage, cluster.ClusterType, cluster.WorkType)
	componentList, exists := componentsDefinitionMap[key]
	if !exists {
		return nil, fmt.Errorf("failed to match component type [%s], please check if the cluster %s is correct", key, cluster.Name)
	}

	return componentList, nil
}

func GetComponentCategoryDefinition(path string) (*UsageComponentDefinition, error) {
	content, err := loadComponentCategoryDefinition(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load component category definition, err: %s", err)
	}

	usageComponentDefinition := &UsageComponentDefinition{}
	err = yaml.Unmarshal([]byte(content), &usageComponentDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %s", err)
	}

	return usageComponentDefinition, nil
}

func GetThirdPartComponentsList(path string) ([]*ThridPartComponent, error) {
	content, err := loadThirdPartComponentsList(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load third-part components list, err: %s", err)
	}

	components := []*ThridPartComponent{}
	err = yaml.Unmarshal([]byte(content), &components)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal third part component configuration, err: %s", err)
	}

	for _, component := range components {
		if !componentTypeMap[component.Type] {
			return nil, fmt.Errorf("there is no component type for %s", component.Type)
		}
	}

	return components, nil
}

func GetClusterCommonConfig(path string) (*ClusterCommonConfig, error) {
	content, err := loadClusterCommonConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster common config, err: %s", err)
	}

	config := &ClusterCommonConfig{}
	err = yaml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %s", err)
	}

	return config, nil
}

func loadThirdPartComponentsList(path string) (string, error) {
	if val := os.Getenv(EnvthirdPartComponents); val != "" {
		path = val
	}

	return loadConfigFile(path)
}

func loadComponentCategoryDefinition(path string) (string, error) {
	if val := os.Getenv(EnvComponentCategoryDefinition); val != "" {
		path = val
	}

	return loadConfigFile(path)
}

func loadClusterCommonConfig(path string) (string, error) {
	if val := os.Getenv(EnvClusterCommonConfig); val != "" {
		path = val
	}

	return loadConfigFile(path)
}

func loadConfigFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read configuration file: %v", err)
	}

	return string(bytes), nil
}
