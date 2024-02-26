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
	"strings"

	"github.com/mitchellh/copystructure"
)

var (
	ThirdPartComponentsPath         = fmt.Sprintf("%s/%s", DefaultConfigPath, ThirdPartComponentsFile)
	ComponentCategoryDefinitionPath = fmt.Sprintf("%s/%s", DefaultConfigPath, ComponentCategoryFile)
	ClusterCommonConfigPath         = fmt.Sprintf("%s/%s", DefaultConfigPath, ClusterCommonConfigFile)
)

type ClusterComponentConfig struct {
	thirdPartComponentlist []*ThridPartComponent
	clusterCommonConfig    *ClusterCommonConfig
	componentsDefinition   *UsageComponentDefinition
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
	Middleware ComponentList
}

type ComponentList []string

// General configuration required for cluster registration.
type ClusterCommonConfig struct {
	Save   UsageCommonConfig
	Remove UsageCommonConfig
}

type UsageCommonConfig struct {
	Host   CommonConfig
	Worker WorkerTypeCommonConfig
}

type WorkerTypeCommonConfig struct {
	Physical CommonConfig
	Virtual  CommonConfig
}

type CommonConfig []string

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

	componentsDefinition, err := GetComponentDefinition(ComponentCategoryDefinitionPath)
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
	for idx, item := range c.thirdPartComponentlist {
		if item.Name == name {
			return c.thirdPartComponentlist[idx], nil
		}
	}

	return nil, fmt.Errorf("failed to get component %s from third-part components list", name)
}

func (c *ClusterComponentConfig) GetDefaultThirdPartComponentByType(componentType string) (*ThridPartComponent, error) {
	for _, item := range c.thirdPartComponentlist {
		if item.Type == componentType && item.Default {
			return item, nil
		}
	}

	return nil, fmt.Errorf("failed to get %s default component from third-part components list", componentType)
}

func (c *ClusterComponentConfig) getComponentsDefinitionMap() map[string]ComponentList {
	componentsDefinitionMap := make(map[string]ComponentList, 0)
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_HOST, CLUSTER_TYPE_PHYSICAL, ""}, "|")] = c.componentsDefinition.Host
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_PHYSICAL, ClusterWorkTypeDeployment}, "|")] = c.componentsDefinition.Worker.Physical.Deployment
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_VIRTUAL, ClusterWorkTypeDeployment}, "|")] = c.componentsDefinition.Worker.Virtual.Deployment
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_PHYSICAL, ClusterWorkTypePipeline}, "|")] = c.componentsDefinition.Worker.Physical.Pipeline
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_VIRTUAL, ClusterWorkTypePipeline}, "|")] = c.componentsDefinition.Worker.Virtual.Pipeline
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_PHYSICAL, ClusterWorkTypeMiddleware}, "|")] = c.componentsDefinition.Worker.Physical.Middleware
	componentsDefinitionMap[strings.Join([]string{CLUSTER_USAGE_WORKER, CLUSTER_TYPE_VIRTUAL, ClusterWorkTypeMiddleware}, "|")] = c.componentsDefinition.Worker.Virtual.Middleware
	return componentsDefinitionMap
}

func (c *ClusterComponentConfig) GetClusterComponentsDefinition(cluster *ClusterInfo) (ComponentList, error) {
	componentsDefinitionMap := c.getComponentsDefinitionMap()
	key := fmt.Sprintf("%s|%s|%s", cluster.Usage, cluster.ClusterType, cluster.WorkType)
	componentList, exists := componentsDefinitionMap[key]
	if !exists {
		return nil, fmt.Errorf("failed to match component type [%s], please check if the cluster %s is correct", key, cluster.Name)
	}

	return componentList, nil
}

func (c *ClusterComponentConfig) GetClusterCommonConfig(operation ClusterOperation, cluster *ClusterInfo) (CommonConfig, error) {
	clusterCommonConfigMap := c.getClusterCommonConfigMap()
	key := fmt.Sprintf("%s|%s|%s", operation, cluster.Usage, cluster.ClusterType)
	config, exists := clusterCommonConfigMap[key]
	if !exists {
		return nil, fmt.Errorf("failed to match component type [%s], please check if the cluster %s is correct", key, cluster.Name)
	}

	return config, nil
}

func (c *ClusterComponentConfig) getClusterCommonConfigMap() map[string]CommonConfig {
	copy, _ := copystructure.Copy(c.clusterCommonConfig)
	clusterCommonConfig := copy.(*ClusterCommonConfig)
	clusterCommonConfigMap := make(map[string]CommonConfig, 0)
	clusterCommonConfigMap[strings.Join([]string{string(ToSave), CLUSTER_USAGE_HOST, CLUSTER_TYPE_PHYSICAL}, "|")] = clusterCommonConfig.Save.Host
	clusterCommonConfigMap[strings.Join([]string{string(ToRemove), CLUSTER_USAGE_HOST, CLUSTER_TYPE_PHYSICAL}, "|")] = clusterCommonConfig.Remove.Host

	clusterCommonConfigMap[strings.Join([]string{string(ToSave), CLUSTER_USAGE_WORKER, CLUSTER_TYPE_PHYSICAL}, "|")] = clusterCommonConfig.Save.Worker.Physical
	clusterCommonConfigMap[strings.Join([]string{string(ToRemove), CLUSTER_USAGE_WORKER, CLUSTER_TYPE_PHYSICAL}, "|")] = clusterCommonConfig.Remove.Worker.Physical

	clusterCommonConfigMap[strings.Join([]string{string(ToSave), CLUSTER_USAGE_WORKER, CLUSTER_TYPE_VIRTUAL}, "|")] = clusterCommonConfig.Save.Worker.Virtual
	clusterCommonConfigMap[strings.Join([]string{string(ToRemove), CLUSTER_USAGE_WORKER, CLUSTER_TYPE_VIRTUAL}, "|")] = clusterCommonConfig.Remove.Worker.Virtual

	return clusterCommonConfigMap
}
