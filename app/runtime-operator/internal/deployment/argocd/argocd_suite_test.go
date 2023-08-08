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
	"testing"

	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	deployapp "github.com/nautes-labs/nautes/app/runtime-operator/internal/deployment/argocd"
	secmock "github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/mock"
	secprovider "github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/provider"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	convert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var mgr deployapp.Syncer
var accessInfo interfaces.AccessInfo
var mockcli *mockClient
var nautesCFG *nautescfg.Config

const argoNamespace = "argocd"
const nautesNamespace = "nautes"
const argocdRBACConfigMapName = "argocd-rbac-cm"
const argocdRBACConfigMapKey = "policy.csv"

func TestArgocd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Argocd Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	fmt.Printf("start test env: %s\n", time.Now())
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	err := errors.New("")
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = argocrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = nautescrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	fmt.Printf("init test env: %s\n", time.Now())

	initEnv()
	fmt.Printf("init env finish: %s\n", time.Now())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func initEnv() {
	// set tenant cluster client, fake client
	mockcli = &mockClient{}
	mgr = deployapp.Syncer{
		K8sClient: mockcli,
	}

	// put nautes config into ctx, it contain a fake secret client.
	kubeconfigCR := convert.ConvertRestConfigToApiConfig(*cfg)
	kubeconfig, err := clientcmd.Write(kubeconfigCR)
	Expect(err).Should(BeNil())
	err = os.Setenv("TEST_KUBECONFIG", string(kubeconfig))
	Expect(err).Should(BeNil())
	secprovider.SecretProviders = map[string]secprovider.NewClient{"mock": secmock.NewMock}
	nautesCFG, err = nautescfg.NewConfig(`
secret:
  repoType: mock
`)
	Expect(err).Should(BeNil())
	ctx = context.Background()
	ctx = runtimecontext.NewNautesConfigContext(ctx, *nautesCFG)

	secClient, err := secprovider.GetSecretClient(ctx)
	Expect(err).Should(BeNil())
	ctx = runtimecontext.NewSecretClientContext(ctx, secClient)

	// create access info of envtest
	accessInfo = interfaces.AccessInfo{
		Name:       "test-cluster",
		Type:       "k8s",
		Kubernetes: cfg,
	}

	err = k8sClient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: argoNamespace,
		},
	})
	Expect(err).Should(BeNil())

	err = k8sClient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nautesNamespace,
		},
	})
	Expect(err).Should(BeNil())

	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      argocdRBACConfigMapName,
			Namespace: argoNamespace,
		},
		Data: map[string]string{
			"policy.csv": ``,
		},
	}
	err = k8sClient.Create(context.Background(), rbaccm)
	Expect(err).Should(BeNil())
}

func randNum() string {
	return fmt.Sprintf("%04d", rand.Intn(9999))
}

func isNotTerminatingAndBelongsToProduct(res client.Object, productName string) bool {
	if !res.GetDeletionTimestamp().IsZero() {
		return false
	}
	labels := res.GetLabels()
	name, ok := labels[nautescrd.LABEL_BELONG_TO_PRODUCT]
	if !ok || name != productName {
		return false
	}
	return true
}

type mockClient struct {
	provider    *nautescrd.CodeRepoProvider
	product     *nautescrd.Product
	codeRepo    *nautescrd.CodeRepo
	environment []nautescrd.Environment
	cluster     []nautescrd.Cluster
	deployments []nautescrd.DeploymentRuntime
}

func (c *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	discovered := false
	switch obj.(type) {
	case *nautescrd.CodeRepo:
		obj.(*nautescrd.CodeRepo).ObjectMeta = c.codeRepo.ObjectMeta
		obj.(*nautescrd.CodeRepo).Spec = c.codeRepo.Spec
		discovered = true
	case *nautescrd.CodeRepoProvider:
		obj.(*nautescrd.CodeRepoProvider).ObjectMeta = c.provider.ObjectMeta
		obj.(*nautescrd.CodeRepoProvider).Spec = c.provider.Spec
		discovered = true
	case *nautescrd.Product:
		obj.(*nautescrd.Product).ObjectMeta = c.product.ObjectMeta
		obj.(*nautescrd.Product).Spec = c.product.Spec
		discovered = true
	case *nautescrd.Environment:
		for _, env := range c.environment {
			if env.Name == key.Name && env.Namespace == key.Namespace {
				obj.(*nautescrd.Environment).ObjectMeta = env.ObjectMeta
				obj.(*nautescrd.Environment).Spec = env.Spec
				discovered = true
			}
		}
	case *nautescrd.Cluster:
		for _, cluster := range c.cluster {
			if cluster.Name == key.Name && cluster.Namespace == key.Namespace {
				obj.(*nautescrd.Cluster).ObjectMeta = cluster.ObjectMeta
				obj.(*nautescrd.Cluster).Spec = cluster.Spec
				obj.(*nautescrd.Cluster).Status = cluster.Status
				discovered = true
			}
		}
	default:
		return fmt.Errorf("unknow obj type")
	}

	if discovered {
		return nil
	}
	return fmt.Errorf("resource %s/%s not found", key.Namespace, key.Name)
}

func (c *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	switch list.(type) {
	case *nautescrd.DeploymentRuntimeList:
		list.(*nautescrd.DeploymentRuntimeList).Items = c.deployments
		return nil
	case *nautescrd.ProjectPipelineRuntimeList:
		return nil
	}
	return fmt.Errorf("unknow list type")

}

func (c *mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (c *mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (c *mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (c *mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (c *mockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (c *mockClient) Status() client.StatusWriter {
	return nil
}

func (c *mockClient) Scheme() *runtime.Scheme {
	return nil
}

func (c *mockClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (c *mockClient) SubResource(subResource string) client.SubResourceClient {
	panic("not implemented") // TODO: Implement
}
