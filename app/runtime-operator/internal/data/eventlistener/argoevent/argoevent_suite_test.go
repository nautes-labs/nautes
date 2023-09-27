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

package argoevent_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	argoEventNamespace = "argoevent"
	nautesNamespace    = "nautes"
)

func TestArgoevent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Argoevent Suite")
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
			filepath.Join("..", "..", "..", "..", "..", "..", "test", "crd", "argoevent"),
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
	err = sensorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = eventsourcev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nautesNamespace,
		},
	}
	err = k8sClient.Create(context.TODO(), ns)
	Expect(err).Should(BeNil())

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: argoEventNamespace,
		},
	}
	err = k8sClient.Create(context.TODO(), ns)
	Expect(err).Should(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

type mockDB struct {
	cluster   v1alpha1.Cluster
	codeRepos []v1alpha1.CodeRepo
	provider  v1alpha1.CodeRepoProvider
}

func (m *mockDB) GetCluster(name string) (*v1alpha1.Cluster, error) {
	return &m.cluster, nil
}

func (m *mockDB) GetProduct(name string) (*v1alpha1.Product, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockDB) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	if m.provider.Name == name {
		return m.provider.DeepCopy(), nil
	}
	return nil, fmt.Errorf("code repo provider %s not found", name)
}

func (m *mockDB) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	for i := range m.codeRepos {
		if m.codeRepos[i].Name == name {
			return m.codeRepos[i].DeepCopy(), nil
		}
	}
	return nil, fmt.Errorf("code repo %s not found", name)
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

func (m *mockDB) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
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

type mockSecMgr struct {
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (m *mockSecMgr) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) CreateUser(ctx context.Context, user syncer.User) error {
	return nil
}

func (m *mockSecMgr) GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	return nil
}

func (m *mockSecMgr) RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	return nil
}

// GetAccessInfo should return the information on how to access the cluster
func (m *mockSecMgr) GetAccessInfo(ctx context.Context) (string, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) DeleteUser(ctx context.Context, user syncer.User) error {
	panic("not implemented") // TODO: Implement
}

type mockSecSyncer struct{}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (m *mockSecSyncer) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

// CreateSecret will create a secret object (sercret in kubernetes, file in host) from secret database to dest environment.
func (m *mockSecSyncer) CreateSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	return nil
}

func (m *mockSecSyncer) RemoveSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	return nil
}

type mockGateway struct{}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (m *mockGateway) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

// CreateEntryPoint will create an entrypoint on gateway.
// EntryPoint may forward to a kubernetes service or a remote Service.
// Gateway should support at least one of them.
func (m *mockGateway) CreateEntryPoint(ctx context.Context, entrypoint syncer.EntryPoint) error {
	return nil
}

func (m *mockGateway) RemoveEntryPoint(ctx context.Context, entrypoint syncer.EntryPoint) error {
	return nil
}
