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

import "fmt"

const (
	_HostClusterPlaceholder = "_HOST_CLUSTER_"
	_RuntimePlaceholder     = "_RUNTIME_"
	_VclusterPlaceholder    = "_VCLUSTER_"
	_ArgocdOAuthSuffix      = "api/dex/callback"
	_TektonOAuthSuffix      = "oauth2/callback"
)

const (
	_HostClustersDir     = "host-clusters"
	_RuntimesDir         = "runtimes"
	_VclustersDir        = "vclusters"
	_VclusterAppSetFile  = "production/vcluster-appset.yaml"
	_ClustersDir         = "nautes/overlays/production/clusters"
	_TenantProductionDir = "tenant/production"
)

const (
	HttpsNodePort = "httpsNodePort"
	HttpNodePort  = "httpNodePort"
	Host          = "host"
)

func returnResourceFilePath(dir, name string) string {
	return fmt.Sprintf("%s/%s.yaml", dir, name)
}
