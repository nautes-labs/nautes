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

package tekton_test

import (
	"context"
	"testing"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/pkg/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTekton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tekton Suite")
}

var fakeHook *component.Hook

type mockSecretSync struct{}

func (mss *mockSecretSync) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) CreateSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

func (mss *mockSecretSync) RemoveSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

type mockPluginManager struct{}

// GetHookFactory will return a 'HookFactory' that can implement the hook based on the pipeline type and hook type.
// If the corresponding plugin cannot be found, an error is returned.
func (mpm *mockPluginManager) GetHookFactory(pipelineType string, hookName string) (component.HookFactory, error) {
	return &mockPlugin{}, nil
}

type mockPlugin struct{}

func (mp *mockPlugin) GetPipelineType() (string, error) {
	return "", nil
}

func (mp *mockPlugin) GetHooksMetadata() ([]resource.HookMetadata, error) {
	return nil, nil
}

func (mp *mockPlugin) BuildHook(hookName string, info component.HookBuildData) (*component.Hook, error) {
	return fakeHook, nil
}

type mockSnapShot struct{}

func (mss *mockSnapShot) GetProduct(name string) (*v1alpha1.Product, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetCodeRepoByURL(url string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetCluster(name string) (*v1alpha1.Cluster, error) {
	return &v1alpha1.Cluster{}, nil
}

func (mss *mockSnapShot) GetClusterByRuntime(runtime v1alpha1.Runtime) (*v1alpha1.Cluster, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) ListPipelineRuntimes() ([]v1alpha1.ProjectPipelineRuntime, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
	panic("not implemented") // TODO: Implement
}

// ListUsedNamespces should return all namespaces used by product
func (mss *mockSnapShot) ListUsedNamespaces(opts ...database.ListOption) (database.NamespaceUsage, error) {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSnapShot) ListUsedCodeRepos(opts ...database.ListOption) ([]v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}
