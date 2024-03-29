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

package hnc_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestHnc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hnc Suite")
}

var testEnv *envtest.Environment
var k8sClient client.Client
var restCFG *rest.Config

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	var NotUseExistingCluster bool
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "..", "..", "test", "crd", "hnc"),
		},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &NotUseExistingCluster,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	restCFG = cfg

	//+kubebuilder:scaffold:scheme
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = rbacv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hncv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

type mockDB struct {
	product v1alpha1.Product
	cluster v1alpha1.Cluster
}

func (m *mockDB) GetProduct(name string) (*v1alpha1.Product, error) {
	return m.product.DeepCopy(), nil
}

func (m *mockDB) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	if name == m.product.Name {
		return nil, fmt.Errorf("product not found")
	}

	return &v1alpha1.CodeRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
		Spec: v1alpha1.CodeRepoSpec{
			Product:  name,
			Project:  "",
			RepoName: "test",
			URL:      "ssh://127.0.0.1/test/test.gi",
		},
	}, nil
}

func (m *mockDB) GetCluster(name string) (*v1alpha1.Cluster, error) {
	return m.cluster.DeepCopy(), nil
}

func (m *mockDB) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetCodeRepoByURL(url string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetClusterByRuntime(runtime v1alpha1.Runtime) (*v1alpha1.Cluster, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) ListPipelineRuntimes() ([]v1alpha1.ProjectPipelineRuntime, error) {
	panic("not implemented") // TODO: Implement
}

// ListUsedNamespces should return all namespaces used by product
func (m *mockDB) ListUsedNamespaces(opts ...database.ListOption) (database.NamespaceUsage, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) ListUsedCodeRepos(opts ...database.ListOption) ([]v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) ListUsedURLs(opts ...database.ListOption) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
	panic("not implemented") // TODO: Implement
}

type mockDeployer struct{}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (md *mockDeployer) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (m *mockDeployer) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployer) CreateProduct(ctx context.Context, name string) error {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployer) DeleteProduct(ctx context.Context, name string) error {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployer) AddProductUser(ctx context.Context, request component.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployer) DeleteProductUser(ctx context.Context, request component.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

// SyncApp should deploy the given apps, and clean up expired apps in cache.
// All apps share one cache.
func (md *mockDeployer) CreateApp(ctx context.Context, app component.Application) error {
	return nil
}

func (md *mockDeployer) DeleteApp(ctx context.Context, app component.Application) error {
	return nil
}
