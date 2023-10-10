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

package productprovider

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	syncer "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var syncInstance syncer.ProductProviderSyncer

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Cluster operator")
}

var _ = BeforeSuite(func() {
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

	fmt.Printf("init test env: %s\n", time.Now())
	ProductProviders = map[string]ProviderFacotry{
		"mock": &MockProductProviderFactory{},
	}

	tmp := &ProductProviderSyncer{
		NautesConfig: configs.NautesConfigs{Namespace: "default", Name: "nautes"},
		Rest:         cfg,
	}
	err = tmp.Setup()
	Expect(err).NotTo(HaveOccurred())
	syncInstance = tmp

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nautes",
			Namespace: "default",
		},
		Data: map[string]string{"config": `
deploy:
  argocd:
    namespace: default
`},
	}

	err = k8sClient.Create(context.TODO(), cm)
	Expect(err).NotTo(HaveOccurred())

	fmt.Printf("init env finish: %s\n", time.Now())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func randNum() string {
	return fmt.Sprintf("%04d", rand.Intn(9999))
}
