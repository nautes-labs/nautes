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

package initinfo

import "k8s.io/client-go/rest"

type ComponentInitInfo struct {
	ClusterConnectInfo ClusterConnectInfo
	ProductName        string
	ClusterName        string
	Labels             map[string]string
}

type ClusterType string

const (
	ClusterTypeKubernetes = "kubernetes"
)

type ClusterConnectInfo struct {
	ClusterType ClusterType
	Kubernetes  *ClusterConnectInfoKubernetes
}

type ClusterConnectInfoKubernetes struct {
	Config *rest.Config
}