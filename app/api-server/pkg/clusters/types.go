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
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	kubernetes "github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	clusterconfig "github.com/nautes-labs/nautes/pkg/config/cluster"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

type RuntimeType string
type ClusterAction string

type ClusterManagement struct {
	file                   FileOperation
	k8sClient              kubernetes.KubernetesOperation
	clusterComponentConfig *clusterconfig.ClusterComponentConfig
	nodestree              nodestree.NodesTree
	clusterTemplateDir     string
}

type ClusterRegistrationParams struct {
	Cluster       *resourcev1alpha1.Cluster
	Repo          *RepositoriesInfo
	Cert          *Cert
	NautesConfigs nautesconfigs.Nautes
	SecretConfigs nautesconfigs.SecretRepo
	OauthConfigs  nautesconfigs.OAuth
	GitConfigs    nautesconfigs.GitRepo
	Clusters      []resourcev1alpha1.Cluster
	HostCluster   *resourcev1alpha1.Cluster
	Vcluster      *VclusterInfo
}

type VclusterInfo struct {
	HttpsNodePort string
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

type ClusterUsage string

// components options
type Cluster struct {
	Name        string
	ApiServer   string
	Usage       string
	ClusterType string
	WorkerType  string
}

// components
type DefaultValueOptions struct {
	Cluster *Cluster
}

type DeploymentServer struct {
	Argocd *Argocd
}

type GatewayServer struct {
	Traefik *Traefik
}

type OAuthProxyServer struct {
	OAuth2Proxy *OAuth2Proxy
}

type PipelineServer struct {
	Tekton *Tekton
}

type SecretManagementServer struct {
	Vault *Vault
}
