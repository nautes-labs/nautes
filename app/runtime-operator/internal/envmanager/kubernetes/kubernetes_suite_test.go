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

package kubernetes_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	externalsecretcrd "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/jinzhu/copier"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	secmock "github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/mock"
	secprovider "github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/provider"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	convert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

func TestKubernetes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Suite")
}

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var nautesCFG *nautescfg.Config
var mockcli *mockClient

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

	err = externalsecretcrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = hncv1alpha2.AddToScheme(scheme.Scheme)
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
	secprovider.SecretProviders = map[string]secprovider.NewClient{"vault": secmock.NewMock}
	kubeconfigCR := convert.ConvertRestConfigToApiConfig(*cfg)

	kubeconfig, err := clientcmd.Write(kubeconfigCR)
	Expect(err).Should(BeNil())

	err = os.Setenv("TEST_KUBECONFIG", string(kubeconfig))
	Expect(err).Should(BeNil())

	nautesCFG, err = nautescfg.NewConfig(`
secret:
  repoType: vault
`)
	Expect(err).Should(BeNil())
	ctx = context.Background()
	ctx = runtimecontext.NewNautesConfigContext(ctx, *nautesCFG)

	secClient, err := secprovider.GetSecretClient(ctx)
	Expect(err).Should(BeNil())
	ctx = runtimecontext.NewSecretClientContext(ctx, secClient)

	k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nautes",
		},
	})

	mockcli = &mockClient{}

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
	resourceDBs map[string]*resourceDB
	scheme      *runtime.Scheme
}

type resourceDB struct {
	resource map[string]client.Object
}

func newMockClient() *mockClient {
	return &mockClient{
		resourceDBs: map[string]*resourceDB{},
		scheme:      scheme.Scheme,
	}
}

func (c *mockClient) getResourceType(obj runtime.Object) (string, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind), nil
}

func (c *mockClient) getResourceDB(obj runtime.Object) (*resourceDB, error) {
	index, err := c.getResourceType(obj)
	if err != nil {
		return nil, err
	}
	if c.resourceDBs[index] == nil {
		c.resourceDBs[index] = &resourceDB{
			resource: map[string]client.Object{},
		}
	}

	return c.resourceDBs[index], nil
}

func (c *resourceDB) GetIndex(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())
}

func (r *resourceDB) GetResource(obj client.Object) (client.Object, error) {
	objInDB, ok := r.resource[r.GetIndex(obj)]
	if !ok {
		return nil, fmt.Errorf("resouce %s not found", r.GetIndex(obj))
	}

	return objInDB, nil
}

func (r *resourceDB) AddResource(obj client.Object) error {
	index := r.GetIndex(obj)
	if _, ok := r.resource[index]; ok {
		return fmt.Errorf("resource %s is already exited.", index)
	}

	r.resource[index] = obj
	return nil
}

func (r *resourceDB) UpdateResource(obj client.Object) error {
	index := r.GetIndex(obj)
	if _, ok := r.resource[index]; !ok {
		return fmt.Errorf("resource %s is not exited.", index)
	}

	r.resource[index] = obj
	return nil
}

func (r *resourceDB) DeleteResource(obj client.Object) error {
	index := r.GetIndex(obj)
	_, ok := r.resource[index]
	if !ok {
		return nil
	}

	delete(r.resource, index)
	return nil
}

func (c *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	obj.SetName(key.Name)
	obj.SetNamespace(key.Namespace)
	resourceDB, err := c.getResourceDB(obj)
	if err != nil {
		return err
	}

	objInClient, err := resourceDB.GetResource(obj)
	if err != nil {
		return err
	}

	if err := copier.Copy(obj, objInClient); err != nil {
		return err
	}

	return nil
}

func (c *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	resourceType, err := c.getResourceType(list)
	if err != nil {
		return err
	}

	switch resourceType {
	case "nautes.resource.nautes.io/v1alpha1/CodeRepoList":
		resourceDB, _ := c.getResourceDB(&nautescrd.CodeRepo{})
		objList := &nautescrd.CodeRepoList{
			Items: []nautescrd.CodeRepo{},
		}

		for _, codeRepo := range resourceDB.resource {
			obj := &nautescrd.CodeRepo{}
			copier.Copy(obj, codeRepo)
			objList.Items = append(objList.Items, *obj)
		}

		copier.Copy(list, objList)
	case "nautes.resource.nautes.io/v1alpha1/ArtifactRepoList":
		resourceDB, _ := c.getResourceDB(&nautescrd.ArtifactRepo{})
		objList := &nautescrd.ArtifactRepoList{
			Items: []nautescrd.ArtifactRepo{},
		}

		for _, artiRepo := range resourceDB.resource {
			obj := &nautescrd.ArtifactRepo{}
			copier.Copy(obj, artiRepo)
			objList.Items = append(objList.Items, *obj)
		}

		copier.Copy(list, objList)
	case "nautes.resource.nautes.io/v1alpha1/DeploymentRuntimeList":
		resourceDB, _ := c.getResourceDB(&nautescrd.DeploymentRuntime{})
		objList := &nautescrd.DeploymentRuntimeList{
			Items: []nautescrd.DeploymentRuntime{},
		}

		for _, deployRuntime := range resourceDB.resource {
			obj := &nautescrd.DeploymentRuntime{}
			copier.Copy(obj, deployRuntime)
			objList.Items = append(objList.Items, *obj)
		}

		copier.Copy(list, objList)
	case "nautes.resource.nautes.io/v1alpha1/ProjectPipelineRuntimeList":
		resourceDB, _ := c.getResourceDB(&nautescrd.ProjectPipelineRuntime{})
		objList := &nautescrd.ProjectPipelineRuntimeList{
			Items: []nautescrd.ProjectPipelineRuntime{},
		}

		for _, pipelineRuntime := range resourceDB.resource {
			obj := &nautescrd.ProjectPipelineRuntime{}
			copier.Copy(obj, pipelineRuntime)
			objList.Items = append(objList.Items, *obj)
		}

		copier.Copy(list, objList)
	default:
		return fmt.Errorf("resource type %s is not supported", resourceType)
	}
	return nil
}

func (c *mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	resourceDB, err := c.getResourceDB(obj)
	if err != nil {
		return err
	}

	return resourceDB.AddResource(obj)
}

func (c *mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	resourceDB, err := c.getResourceDB(obj)
	if err != nil {
		return err
	}

	return resourceDB.DeleteResource(obj)
}

func (c *mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	resourceDB, err := c.getResourceDB(obj)
	if err != nil {
		return err
	}

	return resourceDB.UpdateResource(obj)
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

// SubResourceClientConstructor returns a subresource client for the named subResource. Known
// upstream subResources usages are:
//
//   - ServiceAccount token creation:
//     sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     token := &authenticationv1.TokenRequest{}
//     c.SubResourceClient("token").Create(ctx, sa, token)
//
//   - Pod eviction creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     c.SubResourceClient("eviction").Create(ctx, pod, &policyv1.Eviction{})
//
//   - Pod binding creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     binding := &corev1.Binding{Target: corev1.ObjectReference{Name: "my-node"}}
//     c.SubResourceClient("binding").Create(ctx, pod, binding)
//
//   - CertificateSigningRequest approval:
//     csr := &certificatesv1.CertificateSigningRequest{
//     ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"},
//     Status: certificatesv1.CertificateSigningRequestStatus{
//     Conditions: []certificatesv1.[]CertificateSigningRequestCondition{{
//     Type: certificatesv1.CertificateApproved,
//     Status: corev1.ConditionTrue,
//     }},
//     },
//     }
//     c.SubResourceClient("approval").Update(ctx, csr)
//
//   - Scale retrieval:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{}
//     c.SubResourceClient("scale").Get(ctx, dep, scale)
//
//   - Scale update:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 2}}
//     c.SubResourceClient("scale").Update(ctx, dep, client.WithSubResourceBody(scale))
func (c *mockClient) SubResource(subResource string) client.SubResourceClient {
	panic("not implemented") // TODO: Implement
}
