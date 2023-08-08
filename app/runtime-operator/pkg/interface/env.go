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

package interfaces

import (
	"context"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/client-go/rest"
)

const (
	ACCESS_TYPE_K8S AccessType = "kubernetes"
)

type AccessType string

type AccessInfo struct {
	Name string
	Type AccessType
	// TODO ssh type should replace to  go package structure in the future
	SSH        []*string
	Kubernetes *rest.Config
}

type EnvManager interface {
	GetAccessInfo(ctx context.Context, cluster nautescrd.Cluster) (*AccessInfo, error)
	Sync(ctx context.Context, task RuntimeSyncTask) (*EnvironmentDeploymentResult, error)
	Remove(ctx context.Context, task RuntimeSyncTask) error
}

type EnvironmentDeploymentResult struct {
	AdditionalResources []AdditionalResource
	Error               error
}

type AdditionalResource struct {
	Type  string
	Name  string
	Error error
}

type Runtime interface {
	GetProduct() string
	GetName() string
	GetDestination() string
	GetNamespaces() []string
}
