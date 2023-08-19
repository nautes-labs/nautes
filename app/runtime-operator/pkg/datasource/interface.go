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

package datasource

import (
	"reflect"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

// NamespaceUsage store namespaces in each cluster. Format: map[cluster]namespaces
type NamespaceUsage map[string][]string

func (nu NamespaceUsage) GetNamespacesInCluster(name string) []string {
	return nu[name]
}

// DataSource store all product resouces in kubernetes.
type DataSource interface {
	GetProduct() (*nautescrd.Product, error)
	GetCodeRepo(name string) (*nautescrd.CodeRepo, error)
	GetClusterByRuntime(runtime nautescrd.Runtime) (*nautescrd.Cluster, error)
	ListPipelineRuntimes() ([]nautescrd.ProjectPipelineRuntime, error)
	// ListUsedNamespces should return all namespaces used by product
	ListUsedNamespaces(opts ...ListOption) (NamespaceUsage, error)
	ListUsedCodeRepos(opts ...ListOption) ([]nautescrd.CodeRepo, error)
	ListUsedURLs(opts ...ListOption) ([]string, error)
}

type ListOption func(*ListOptions)

func InCluster(cluster string) ListOption {
	return func(lo *ListOptions) { lo.inCluster = cluster }
}

func ExcludeRuntimes(runtimes []nautescrd.Runtime) ListOption {
	return func(lo *ListOptions) { lo.excludeRuntimes = runtimes }
}

func WithOutDeletedRuntimes() ListOption {
	return func(lo *ListOptions) { lo.withOutDeletedRuntimes = true }
}

func WithOutProductInfo() ListOption {
	return func(lo *ListOptions) { lo.withOutProductInfo = true }
}

type ListOptions struct {
	inCluster              string
	excludeRuntimes        []nautescrd.Runtime
	withOutDeletedRuntimes bool
	withOutProductInfo     bool
}

func (o *ListOptions) IsRequestCluster(clusterName string) bool {
	if o.inCluster != "" && o.inCluster != clusterName {
		return false
	}
	return true
}

func (o *ListOptions) IsExcludeRuntime(runtime nautescrd.Runtime) bool {
	for _, excludeRuntime := range o.excludeRuntimes {
		if reflect.TypeOf(runtime) == reflect.TypeOf(excludeRuntime) &&
			runtime.GetProduct() == excludeRuntime.GetProduct() &&
			runtime.GetName() == excludeRuntime.GetName() {
			return true
		}
	}
	return false
}

func (o *ListOptions) MatchRequest(runtime nautescrd.Runtime, clusterName string) bool {
	if o.withOutDeletedRuntimes && !runtime.GetDeletionTimestamp().IsZero() {
		return false
	}

	if !o.IsRequestCluster(clusterName) {
		return false
	}

	if o.IsExcludeRuntime(runtime) {
		return false
	}

	return true
}
