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

package argocd_test

import (
	"context"
	"path/filepath"
	"testing"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployment/argocd"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
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
)

const (
	nautesNamespace = "nautes"
	argoCDNamespace = "argocd"
)

func TestArgocd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Argocd Suite")
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
			filepath.Join("..", "..", "..", "..", "..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "..", "..", "..", "..", "test", "crd", "argocd"),
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
	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = argov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	arogcdNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: argoCDNamespace,
		},
	}

	nautesNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nautesNamespace,
		},
	}
	err = k8sClient.Create(context.Background(), arogcdNamespace)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(context.Background(), nautesNamespace)
	Expect(err).NotTo(HaveOccurred())

	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      argocd.ArgocdRBACConfigMapName,
			Namespace: argoCDNamespace,
		},
	}
	err = k8sClient.Create(context.Background(), rbaccm)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Generate by m *mockSecMgr component.SecretManagement
type mockSecMgr struct{}

func (m *mockSecMgr) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	return nil
}

func (m *mockSecMgr) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	return nil
}

func (m *mockSecMgr) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) GetAccessInfo(ctx context.Context) (string, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) CreateAccount(ctx context.Context, account component.MachineAccount) (*component.AuthInfo, error) {
	return nil, nil
}

func (m *mockSecMgr) DeleteAccount(ctx context.Context, account component.MachineAccount) error {
	panic("not implemented") // TODO: Implement
}

// Generate by m *mockMultiTenant component.MultiTenant
type mockMultiTenant struct {
	spaces []string
}

func (m *mockMultiTenant) ListSpaces(ctx context.Context, productName string) ([]component.SpaceStatus, error) {
	spaces := make([]component.SpaceStatus, len(m.spaces))
	for i := 0; i < len(m.spaces); i++ {
		spaces[i] = component.SpaceStatus{
			Space: component.Space{
				ResourceMetaData: component.ResourceMetaData{
					Product: productName,
					Name:    m.spaces[i],
				},
				SpaceType: "",
				Kubernetes: &component.SpaceKubernetes{
					Namespace: m.spaces[i],
				},
			},
			Accounts: []string{},
		}
	}
	return spaces, nil
}

func (m *mockMultiTenant) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) CreateProduct(ctx context.Context, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteProduct(ctx context.Context, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) CreateSpace(ctx context.Context, productName string, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteSpace(ctx context.Context, productName string, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetSpace(ctx context.Context, productName string, name string) (*component.SpaceStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) AddSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) CreateAccount(ctx context.Context, productName string, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteAccount(ctx context.Context, productName string, name string) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetAccount(ctx context.Context, productName string, name string) (*component.MachineAccount, error) {
	panic("not implemented") // TODO: Implement
}
