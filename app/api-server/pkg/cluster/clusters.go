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

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

const (
	_PhysicalDeployment RuntimeType   = "physical_deployment"
	_PhysicalPipeline   RuntimeType   = "physical_pipeline"
	_VirtualDeployment  RuntimeType   = "virtual_deployment"
	_VirtualPipeline    RuntimeType   = "virtual_pipeline"
	_Save               ClusterAction = "Save Cluster"
	_Remove             ClusterAction = "Remove Clustr"
)

type RuntimeType string
type ClusterAction string

var (
	clusterCommonConfig     *ClusterOperatorCommonConfig
	componentsOfClusterType *ComponentsOfClusterType
	thirdPartComponents     []*ThridPartComponent
)

func init() {
	clusterCommonConfig, _ = GetClusterCommonConfig()
	componentsOfClusterType, _ = GetComponentsOfClusterType()
	thirdPartComponents, _ = GetThirdPartComponentsList()
}

type ClusterManager struct {
	params   *ClusterManagerParams
	dex      DexOperator
	resource ResourceOperator
}

type ClusterManagerParams struct {
	Cluster *resourcev1alpha1.Cluster
	Repo    *RepositoriesInfo
	Cert    *Cert
	Configs *nautesconfigs.Config
}

type RepositoriesInfo struct {
	ClusterTemplateDir string
	TenantRepoDir      string
	TenantRepoURL      string
}

type Cert struct {
	Default string
	Gitlab  string
}

type ResourceOperator interface {
}

type DexOperator interface {
	Add()
	Remove()
}

func NewClusterManger(params *ClusterManagerParams, dex DexOperator, resource ResourceOperator) (*ClusterManager, error) {

	return &ClusterManager{
		params:   params,
		dex:      dex,
		resource: resource,
	}, nil
}

func (c *ClusterManager) SaveCluster() {

}

func (c *ClusterManager) RemoveCluster() {

}

func GetCustomComponents(cluster *resourcev1alpha1.Cluster) ([]string, error) {
	if componentsOfClusterType == nil {
		return nil, fmt.Errorf("failed to get custom component, missing cluster category configuration")
	}

	var customComponentsInstallPath []string
	var err error

	if IsHostCluser(cluster) {
		customComponentsInstallPath, err = getCustomComponentInstallPath(cluster, "host")
		if err != nil {
			return nil, err
		}
	}

	if IsPhysicalDeploymentRuntime(cluster) {
		customComponentsInstallPath, err = getCustomComponentInstallPath(cluster, _PhysicalDeployment)
		if err != nil {
			return nil, err
		}
	}

	if IsPhysicalProjectPipelineRuntime(cluster) {
		customComponentsInstallPath, err = getCustomComponentInstallPath(cluster, _PhysicalPipeline)
		if err != nil {
			return nil, err
		}
	}

	if IsVirtualDeploymentRuntime(cluster) {
		customComponentsInstallPath, err = getCustomComponentInstallPath(cluster, _VirtualDeployment)
		if err != nil {
			return nil, err
		}
	}

	if IsVirtualProjectPipelineRuntime(cluster) {
		customComponentsInstallPath, err = getCustomComponentInstallPath(cluster, _VirtualPipeline)
		if err != nil {
			return nil, err
		}
	}

	return customComponentsInstallPath, nil
}

func getCustomComponentInstallPath(cluster *resourcev1alpha1.Cluster, componentType RuntimeType) ([]string, error) {
	var customComponentsInstallPath []string

	componentsType := componentsOfClusterType.Host
	if componentType == _PhysicalDeployment {
		componentsType = componentsOfClusterType.Worker.Physical.Deployment
	} else if componentType == _PhysicalPipeline {
		componentsType = componentsOfClusterType.Worker.Physical.Pipeline
	} else if componentType == _VirtualDeployment {
		componentsType = componentsOfClusterType.Worker.Virtual.Deployment
	} else if componentType == _VirtualPipeline {
		componentsType = componentsOfClusterType.Worker.Virtual.Pipeline
	}

	for _, componentType := range componentsType {
		thirdPartComponent := GetThirdPartComponentByType(thirdPartComponents, componentType)
		if thirdPartComponent == nil {
			return nil, fmt.Errorf("failed to get third part component %s", componentType)
		}

		key := fmt.Sprintf("%s&%s", cluster.Spec.Usage, cluster.Spec.ClusterType)
		val, ok := thirdPartComponent.InstallPath[key]
		if !ok {
			val, ok = thirdPartComponent.InstallPath["*"]
			if !ok {
				return nil, fmt.Errorf("failed to get component %s install path", thirdPartComponent.Name)
			}
		}

		if val == nil {
			return nil, fmt.Errorf("failed to get custom component install path for cluster %s", cluster.Name)
		}

		customComponentsInstallPath = append(customComponentsInstallPath, val...)
	}

	return customComponentsInstallPath, nil
}

func (c *ClusterManager) getClusterCommonConfig(action ClusterAction) (string, error) {
	return "", nil
}

func (c *ClusterManager) loadTemplateNodesTree() (string, error) {
	return "", nil
}

func (c *ClusterManager) reanderTemplate() (string, error) {
	return "", nil
}

func (c *ClusterManager) createClusterToNautes() (string, error) {
	return "", nil
}

func (c *ClusterManager) createDexCallback() (string, error) {
	return "", nil
}

func (c *ClusterManager) removeClusterToNautes() (string, error) {
	return "", nil
}

func (c *ClusterManager) removeDexCallback() (string, error) {
	return "", nil
}
