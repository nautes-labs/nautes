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

package product

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/app/base-operator/internal/syncer/productprovider"
	syncer "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var syncInstance syncer.ProductSyncer

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Cluster operator")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	fmt.Printf("start test env: %s\n", time.Now())
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../../../../..", "config", "crd", "bases"), filepath.Join("../../../../..", "test", "crd", "argocd")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nautescrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = argocrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	fmt.Printf("init test env: %s\n", time.Now())
	productprovider.ProductProviders = map[string]productprovider.ProviderFacotry{
		"gitlab": &productprovider.MockProductProviderFactory{},
	}
	tmp := &ProductSyncer{
		NautesConfig: configs.NautesConfigs{
			Namespace: "default",
			Name:      "nautes",
		},
		Rest: cfg,
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

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nautes",
		},
	}
	err = k8sClient.Create(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Create(context.TODO(), &nautescrd.ProductProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: nautescrd.ProductProviderSpec{
			Type: "gitlab",
			Name: "fake",
		},
	})
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
