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

func IsVirtual(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_VIRTUAL
}

func IsPhysical(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_PHYSICAL
}

func IsHostCluser(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_PHYSICAL && cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_HOST
}

func IsVirtualDeploymentRuntime(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_VIRTUAL && cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_WORKER && cluster.Spec.WorkerType == resourcev1alpha1.ClusterWorkTypeDeployment
}

func IsPhysicalDeploymentRuntime(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_PHYSICAL && cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_WORKER && cluster.Spec.WorkerType == resourcev1alpha1.ClusterWorkTypeDeployment
}

func IsVirtualProjectPipelineRuntime(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_VIRTUAL && cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_WORKER && cluster.Spec.WorkerType == resourcev1alpha1.ClusterWorkTypePipeline
}

func IsPhysicalProjectPipelineRuntime(cluster *resourcev1alpha1.Cluster) bool {
	return cluster.Spec.ClusterType == resourcev1alpha1.CLUSTER_TYPE_PHYSICAL && cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_WORKER && cluster.Spec.WorkerType == resourcev1alpha1.ClusterWorkTypePipeline
}
