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

const (
	ThirdPartComponentsFile        = "thirdPartComponents.yaml"
	ComponentCategoryFile          = "componentCategoryDefinition.yaml"
	ClusterCommonConfigFile        = "clusterCommonConfig.yaml"
	EnvthirdPartComponents         = "THIRD_PART_COMPONENTS"
	EnvComponentCategoryDefinition = "COMPONENT_CATEGORY_DEFINITION"
	EnvClusterCommonConfig         = "CLUSTER_COMMON_CONFIG"
	DefaultConfigPath              = "/opt/nautes/config"
)

const (
	CertManagement      ComponentType = "certManagement"
	Deployment          ComponentType = "deployment"
	EventListener       ComponentType = "eventListener"
	Gateway             ComponentType = "gateway"
	MultiTenant         ComponentType = "multiTenant"
	Pipeline            ComponentType = "pipeline"
	ProgressiveDelivery ComponentType = "progressiveDelivery"
	SecretSync          ComponentType = "secretSync"
	OauthProxy          ComponentType = "oauthProxy"
	SecretManagement    ComponentType = "secretManagement"
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
	ClusterWorkTypeMiddleware = "middleware"
)

const (
	ToSave   ClusterOperation = "save"
	ToRemove ClusterOperation = "remove"
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
		string(CertManagement):      true,
		string(Deployment):          true,
		string(EventListener):       true,
		string(Gateway):             true,
		string(MultiTenant):         true,
		string(Pipeline):            true,
		string(ProgressiveDelivery): true,
		string(SecretManagement):    true,
		string(SecretSync):          true,
		string(OauthProxy):          true,
	}
)

type ComponentType string
type ClusterOperation string
