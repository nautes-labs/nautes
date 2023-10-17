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

package coderepoprovider_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	secretprovider "github.com/nautes-labs/nautes/app/base-operator/internal/secret/provider"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	coderepoprovider "github.com/nautes-labs/nautes/app/base-operator/internal/coderepo/provider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}

var crProvider coderepoprovider.ProductProviderCodeRepo
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var coderepoProviderCR *nautescrd.CodeRepoProvider

var _ = BeforeSuite(func() {
	secretprovider.SecretProviders = map[string]secretprovider.NewClient{
		"mock": newMockSecretProvider,
	}
	coderepoprovider.ProductProviderCodeRepoFactory = map[string]coderepoprovider.NewProductProviderCoderRepo{
		"gitlab": newMockProductProvider,
	}
	crProvider = *coderepoprovider.NewProductProviderCodeRepo()

	initK8S()

	fmt.Printf("init env finish: %s\n", time.Now())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func initK8S() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	fmt.Printf("start test env: %s\n", time.Now())
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../../../../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nautescrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
}

type mockProductProvider struct{}

func newMockProductProvider(token string, coderpeoProvider nautescrd.CodeRepoProvider, cfg nautescfg.Config) (baseinterface.ProductProvider, error) {
	return &mockProductProvider{}, nil
}

func (m *mockProductProvider) GetProducts() ([]nautescrd.Product, error) {
	return []nautescrd.Product{}, nil
}

func (m *mockProductProvider) GetProductMeta(ctx context.Context, ID string) (baseinterface.ProductMeta, error) {
	return baseinterface.ProductMeta{}, nil
}

func (p *mockProductProvider) GetCodeRepoProvider(ctx context.Context) (baseinterface.CodeRepoProvider, error) {
	return baseinterface.CodeRepoProvider{}, nil
}

type mockSecretClient struct{}

func newMockSecretProvider(cfg nautescfg.SecretRepo) (baseinterface.SecretClient, error) {
	return &mockSecretClient{}, nil
}

func (m *mockSecretClient) GetGitRepoRootToken(ctx context.Context, name string) (string, error) {
	return "helpme", nil
}

func (m *mockSecretClient) Logout() {}
