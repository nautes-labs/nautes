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
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

type ClusterUsage string

type Traefik struct {
	HttpNodePort  string
	HttpsNodePort string
}

type Vcluster struct {
	Name          string
	Namespace     string
	ApiServer     string
	HttpsNodePort string
	TLSSan        string
	HostCluster   *HostCluster
}

type HostCluster struct {
	Name                 string
	ApiServer            string
	ArgocdProject        string
	PrimaryDomain        string
	Host                 string
	OAuthURL             string
	ProjectPipelineItems []*ProjectPipelineItem
}

type ArgocdConfig struct {
	Host    string
	URL     string
	Project string
}

type TektonConfig struct {
	Host          string
	HttpsNodePort int
	URL           string
}

type Runtime struct {
	Name                string
	ClusterName         string
	Type                string
	PrimaryDomain       string
	MountPath           string
	ApiServer           string
	OAuthURL            string
	ArgocdConfig        *ArgocdConfig
	TektonConfig        *TektonConfig
	ProjectPipelineItem *ProjectPipelineItem
}

type ProjectPipelineItem struct {
	Name            string
	HostClusterName string
	TektonConfig    *TektonConfig
}

type CaBundleList struct {
	Default string
	Gitlab  string
}

type ClusterRegistrationParam struct {
	RepoURL                      string
	ClusterTemplateRepoLocalPath string
	TenantConfigRepoLocalPath    string
	GitRepoHTTPSURL              string
	Vcluster                     *Vcluster
	Cluster                      *resourcev1alpha1.Cluster
	ArgocdHost                   string
	TektonHost                   string
	Traefik                      *Traefik
	Configs                      *nautesconfigs.Config
	CaBundleList                 CaBundleList
}

type ClusterRegistration struct {
	Cluster                      *resourcev1alpha1.Cluster
	ClusterResouceFiles          []string
	ClusterTemplateRepoLocalPath string
	TenantConfigRepoLocalPath    string
	RepoURL                      string
	Usage                        ClusterUsage
	HostCluster                  *HostCluster
	HostClusterNames             []string
	VclusterNames                []string
	Vcluster                     *Vcluster
	Runtime                      *Runtime
	Traefik                      *Traefik
	NautesConfigs                nautesconfigs.Nautes
	GitConfigs                   nautesconfigs.GitRepo
	SecretConfigs                nautesconfigs.SecretRepo
	OauthConfigs                 nautesconfigs.OAuth
	CaBundleList                 CaBundleList
}
