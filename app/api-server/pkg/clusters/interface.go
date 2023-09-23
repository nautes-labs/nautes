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

import resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

type ClusterRegistrationOperator interface {
	GetClsuter(tenantLocalPath, clusterName string) (*resourcev1alpha1.Cluster, error)
	ListClusters(tenantLocalPath string) ([]*resourcev1alpha1.Cluster, error)
	SaveCluster(params *ClusterRegistrationParams) error
	RemoveCluster(params *ClusterRegistrationParams) error
}

// components
type DefaultValueProvider interface {
	GetDefaultValue(field string, opt *DefaultValueOptions) (string, error)
}

type Pipeline interface {
	DefaultValueProvider
	GetPipelineServer(param *ClusterRegistrationParams) *PipelineServer
}

type Deployment interface {
	DefaultValueProvider
	GetDeploymentServer(param *ClusterRegistrationParams) *DeploymentServer
	GetOauthURL(param *ClusterRegistrationParams) (string, error)
}

type Gateway interface {
	DefaultValueProvider
	GetGatewayServer(param *ClusterRegistrationParams) *GatewayServer
}

type OAuthProxy interface {
	DefaultValueProvider
	GetOauthProxyServer(param *ClusterRegistrationParams) *OAuthProxyServer
	GenerateOAuthProxyRedirect(cluster *resourcev1alpha1.Cluster) string
}

type SecretManagement interface {
	DefaultValueProvider
}
