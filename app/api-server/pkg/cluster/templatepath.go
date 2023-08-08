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
	_ClustersDir                = "nautes/overlays/production/clusters"
	_TenantProductionDir        = "tenant/production"
	_HostClusterAppSetFile      = "tenant/production/host-cluster-appset.yaml"
	_HostClusterAppSetFileName  = "host-cluster-appset.yaml"
	_IngressTektonDashboardFile = "oauth2-proxy/overlays/production/ingress-tekton-dashborard.yaml"
)

func concatHostClustesrDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _HostClustersDir)
}

func concatRuntimesDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _RuntimesDir)
}

func concatClustersDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _ClustersDir)
}

func concatTenantProductionDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _TenantProductionDir)
}

func concatSpecifiedVclustersDir(root, hostClusterName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", root, _HostClustersDir, hostClusterName, _VclustersDir)
}

func concatTektonDashborardFilePath(root, hostCluster string) string {
	return fmt.Sprintf("%s/%s/%s/%s", root, _HostClustersDir, hostCluster, _IngressTektonDashboardFile)
}

func concatHostClusterAppsetFilePath(root string) string {
	return fmt.Sprintf("%s/%s", root, _HostClusterAppSetFile)
}
