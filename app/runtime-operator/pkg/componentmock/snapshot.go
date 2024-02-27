// Copyright 2024 Nautes Authors
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

package componentmock

import (
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
)

// s *Snapshot database.Snapshot
type Snapshot struct {
	Product            v1alpha1.Product
	CodeRepos          map[string]v1alpha1.CodeRepo
	Clusters           map[string]v1alpha1.Cluster
	Environments       map[string]v1alpha1.Environment
	MiddlewareRuntimes map[string]v1alpha1.MiddlewareRuntime
}

func (s *Snapshot) GetProduct(name string) (*v1alpha1.Product, error) {
	if s.Product.Name == name {
		productCopy := s.Product.DeepCopy()
		return productCopy, nil
	}
	return nil, fmt.Errorf("product %s not found", name)
}

func (s *Snapshot) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Snapshot) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Snapshot) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	if codeRepo, ok := s.CodeRepos[name]; ok {
		codeRepoCopy := codeRepo.DeepCopy()
		return codeRepoCopy, nil
	}
	return nil, fmt.Errorf("code repo %s not found", name)
}

func (s *Snapshot) GetCodeRepoByURL(url string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Snapshot) GetCluster(name string) (*v1alpha1.Cluster, error) {
	if cluster, ok := s.Clusters[name]; ok {
		clusterCopy := cluster.DeepCopy()
		return clusterCopy, nil
	}
	return nil, fmt.Errorf("cluster %s not found", name)
}

func (s *Snapshot) GetClusterByRuntime(runtime v1alpha1.Runtime) (*v1alpha1.Cluster, error) {
	envName := runtime.GetDestination()
	env, err := s.GetEnvironment(envName)
	if err != nil {
		return nil, err
	}

	clusterName := env.Spec.Cluster
	cluster, err := s.GetCluster(clusterName)
	if err != nil {
		return nil, err
	}

	clusterCopy := cluster.DeepCopy()
	return clusterCopy, nil
}

func (s *Snapshot) GetEnvironment(name string) (*v1alpha1.Environment, error) {
	if env, ok := s.Environments[name]; ok {
		envCopy := env.DeepCopy()
		return envCopy, nil
	}
	return nil, fmt.Errorf("environment %s not found", name)
}

func (s *Snapshot) ListPipelineRuntimes() ([]v1alpha1.ProjectPipelineRuntime, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Snapshot) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
	switch runtimeType {
	case v1alpha1.RuntimeTypeMiddlewareRuntime:
		if runtime, ok := s.MiddlewareRuntimes[name]; ok {
			runtimeCopy := runtime.DeepCopy()
			return runtimeCopy, nil
		}
		return nil, fmt.Errorf("middleware runtime %s not found", name)
	default:
		return nil, fmt.Errorf("runtime type %s not supported", runtimeType)
	}
}

// ListUsedNamespces should return all namespaces used by product
func (s *Snapshot) ListUsedNamespaces(opts ...database.ListOption) (database.NamespaceUsage, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Snapshot) ListUsedCodeRepos(opts ...database.ListOption) ([]v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}
