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
)

const (
	_ClusterKustomizationFile = "nautes/overlays/production/clusters/kustomization.yaml"
	_NutesClusters            = "nautes/overlays/production/clusters"
)

func addCluster(clusters []resourcev1alpha1.Cluster, currentCluster *resourcev1alpha1.Cluster) []resourcev1alpha1.Cluster {
	for idx, cluster := range clusters {
		if cluster.Name == currentCluster.Name {
			clusters[idx] = cluster
			return clusters
		}
	}

	clusters = append(clusters, *currentCluster)

	return clusters
}

func concatClustersDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _ClustersDir)
}

func concatHostClustesrDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _HostClustersDir)
}

func concatRuntimesDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _RuntimesDir)
}

func concatSpecifiedVclustersDir(root, hostClusterName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", root, _HostClustersDir, hostClusterName, _VclustersDir)
}

func concatTenantProductionDir(root string) string {
	return fmt.Sprintf("%s/%s", root, _TenantProductionDir)
}
