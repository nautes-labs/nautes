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

package task_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	component "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	nautesconst "github.com/nautes-labs/nautes/pkg/const"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1 "k8s.io/api/core/v1"
)

func init() {
	NewFunctionMapDeployment["mock"] = NewMockDeployment
	NewFunctionMapMultiTenant["mock"] = NewMockMultiTenant
	NewFunctionMapSecretManagement["mock"] = NewMockSecretManagement
	NewFunctionMapEventListener["mock"] = NewMockEventListener
	NewFunctionMapSecretSync["mock"] = NewMockSecretSync
	ComponentFactoryMapPipeline["mock"] = &mockPipelineFactory{}

	NewSnapshot = newMockDB
}

func TestV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V2 Suite")
}

var testEnv *envtest.Environment
var k8sClient client.Client
var restCFG *rest.Config
var nautesNamespace = "nautes"
var db *mockDB
var testSyncer Syncer
var homePath = "/tmp/unittest"
var result *deployResult

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	err := os.Setenv(nautesconst.EnvNautesHome, homePath)
	Expect(err).NotTo(HaveOccurred())

	err = os.MkdirAll(path.Join(homePath, "config"), os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	configString := `
secret:
  repoType: mock
`
	err = os.WriteFile(path.Join(homePath, "config/config"), []byte(configString), 0600)
	Expect(err).Should(BeNil())

	err = os.WriteFile(path.Join(homePath, "config/base"), []byte(""), 0600)
	Expect(err).Should(BeNil())

	By("bootstrapping test environment")
	var NotUseExistingCluster bool
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "..", "..", "config", "crd", "bases"),
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

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	testSyncer = Syncer{
		KubernetesClient: k8sClient,
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nautesNamespace,
		},
	}
	err = k8sClient.Create(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(homePath)
	Expect(err).NotTo(HaveOccurred())
})

type deployResult struct {
	product    *deployResultProduct
	deployment *deployResultDeployment
}

type deployResultProduct struct {
	name     string
	spaces   []deployResultSpace
	accounts []string
}

type deployResultSpace struct {
	name     string
	accounts []string
}

type deployResultDeployment struct {
	product deployResultDeploymentProduct
	apps    []component.Application
}

type deployResultDeploymentProduct struct {
	name     string
	accounts []string
}

// md *mockDB database.Database
type mockDB struct {
	cluster  *v1alpha1.Cluster
	product  *v1alpha1.Product
	coderepo *v1alpha1.CodeRepo
	provider *v1alpha1.CodeRepoProvider
	runtimes map[string]v1alpha1.Runtime
}

func newMockDB(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (database.Snapshot, error) {
	return db, nil
}

func (md *mockDB) GetProduct(name string) (*v1alpha1.Product, error) {
	return md.product.DeepCopy(), nil
}

func (md *mockDB) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	return md.provider.DeepCopy(), nil
}

func (md *mockDB) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	return md.coderepo.DeepCopy(), nil
}

func (md *mockDB) GetCodeRepoByURL(url string) (*v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) GetCluster(name string) (*v1alpha1.Cluster, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) GetClusterByRuntime(runtime v1alpha1.Runtime) (*v1alpha1.Cluster, error) {
	return db.cluster.DeepCopy(), nil
}

func (md *mockDB) ListPipelineRuntimes() ([]v1alpha1.ProjectPipelineRuntime, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
	runtime, ok := md.runtimes[name]
	if !ok {
		return nil, fmt.Errorf("runtime %s not found", name)
	}
	return runtime, nil
}

// ListUsedNamespces should return all namespaces used by product
func (md *mockDB) ListUsedNamespaces(opts ...database.ListOption) (database.NamespaceUsage, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) ListUsedCodeRepos(opts ...database.ListOption) ([]v1alpha1.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDB) ListUsedURLs(opts ...database.ListOption) ([]string, error) {
	panic("not implemented") // TODO: Implement
}

// mmt *mockMultiTenant MultiTenant
type mockMultiTenant struct{}

func NewMockMultiTenant(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.MultiTenant, error) {
	return &mockMultiTenant{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mmt *mockMultiTenant) CleanUp() error {
	return nil
}

func (mmt *mockMultiTenant) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (mmt *mockMultiTenant) CreateProduct(ctx context.Context, name string) error {
	if result.product == nil {
		result.product = &deployResultProduct{
			name:     name,
			spaces:   []deployResultSpace{},
			accounts: []string{},
		}
	}
	return nil
}

func (mmt *mockMultiTenant) DeleteProduct(ctx context.Context, name string) error {
	if result.product != nil && result.product.name == name {
		result.product = nil
	}
	return nil
}

// Space use to manager a logical space in environment.
func (mmt *mockMultiTenant) CreateSpace(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return fmt.Errorf("product not found")
	}

	for i := range result.product.spaces {
		if name == result.product.spaces[i].name {
			return nil
		}
	}

	result.product.spaces = append(result.product.spaces, deployResultSpace{
		name:     name,
		accounts: []string{},
	})

	return nil
}

func (mmt *mockMultiTenant) DeleteSpace(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return nil
	}

	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName == name {
			result.product.spaces = append(result.product.spaces[:i], result.product.spaces[i+1:]...)
			break
		}
	}
	return nil
}

func (mmt *mockMultiTenant) GetSpace(ctx context.Context, productName string, name string) (*component.SpaceStatus, error) {
	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName == name {
			return &component.SpaceStatus{
				Space: component.Space{
					ResourceMetaData: component.ResourceMetaData{
						Product: productName,
						Name:    name,
					},
					SpaceType: component.DestinationTypeKubernetes,
					Kubernetes: &component.SpaceKubernetes{
						Namespace: name,
					},
				},
				Accounts: result.product.spaces[i].accounts,
			}, nil
		}
	}

	return nil, fmt.Errorf("space not found")
}

func (mmt *mockMultiTenant) ListSpaces(ctx context.Context, productName string) ([]component.SpaceStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (mmt *mockMultiTenant) AddSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	if result.product == nil {
		return fmt.Errorf("product not found")
	}

	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName == request.Resource.Name {
			result.product.spaces[i].accounts = append(result.product.spaces[i].accounts, request.User)
			break
		}
	}

	return nil
}

func (mmt *mockMultiTenant) DeleteSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	if result.product == nil {
		return nil
	}

	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName != request.Resource.Name {
			continue
		}
		for j, accountName := range result.product.spaces[i].accounts {
			if accountName != request.User {
				continue
			}
			result.product.spaces[i].accounts = append(
				result.product.spaces[i].accounts[:j],
				result.product.spaces[i].accounts[j+1:]...)
		}
	}

	return nil
}

func (mmt *mockMultiTenant) CreateAccount(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return fmt.Errorf("product not found")
	}

	for _, accountName := range result.product.accounts {
		if accountName == name {
			return nil
		}
	}

	result.product.accounts = append(result.product.accounts, name)
	return nil
}

func (mmt *mockMultiTenant) DeleteAccount(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return nil
	}

	for i, accountName := range result.product.accounts {
		if accountName == name {
			result.product.accounts = append(result.product.accounts[:i], result.product.accounts[i+1:]...)
			break
		}
	}

	for _, space := range result.product.spaces {
		mmt.DeleteSpaceUser(ctx, component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    space.name,
			},
			User: name,
		})
	}

	return nil
}

func (mmt *mockMultiTenant) GetAccount(ctx context.Context, productName string, name string) (*component.MachineAccount, error) {
	if result.product == nil {
		return nil, component.AccountNotFound(errors.New("product not found"), name)
	}

	var account *component.MachineAccount
	for _, accountName := range result.product.accounts {
		if accountName == name {
			account = &component.MachineAccount{
				Product: productName,
				Name:    name,
			}
			break
		}
	}

	if account == nil {
		return nil, component.AccountNotFound(errors.New(""), name)
	}

	for _, ns := range result.product.spaces {
		if utils.NewStringSet(ns.accounts...).Has(name) {
			account.Spaces = append(account.Spaces, ns.name)
		}
	}
	return account, nil
}

// md *mockDeployment Deployment
type mockDeployment struct{}

func NewMockDeployment(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.Deployment, error) {
	return &mockDeployment{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (md *mockDeployment) CleanUp() error {
	return nil
}

func (md *mockDeployment) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployment) CreateProduct(ctx context.Context, name string) error {
	if result.deployment == nil {
		result.deployment = &deployResultDeployment{
			product: deployResultDeploymentProduct{
				name: name,
			},
		}
	}
	return nil
}

func (md *mockDeployment) DeleteProduct(ctx context.Context, name string) error {
	if result.deployment != nil {
		result.deployment = nil
	}

	return nil
}

func (md *mockDeployment) AddProductUser(ctx context.Context, request component.PermissionRequest) error {
	if result.deployment == nil {
		return fmt.Errorf("product not found")
	}

	accountSet := sets.New(result.deployment.product.accounts...)
	if !accountSet.Has(request.User) {
		result.deployment.product.accounts = append(result.deployment.product.accounts, request.User)
	}

	return nil
}

func (md *mockDeployment) DeleteProductUser(ctx context.Context, request component.PermissionRequest) error {
	if result.deployment == nil {
		return nil
	}

	accountSet := sets.New(result.deployment.product.accounts...)
	if accountSet.Has(request.User) {
		result.deployment.product.accounts = accountSet.Delete(request.User).UnsortedList()
	}

	if len(result.deployment.product.accounts) == 0 {
		result.deployment.product.accounts = nil
	}

	return nil
}

func (md *mockDeployment) CreateApp(ctx context.Context, app component.Application) error {
	if result.deployment == nil {
		return fmt.Errorf("product not found")
	}

	if result.deployment.apps == nil {
		result.deployment.apps = []component.Application{}
	}

	var isExisted bool
	for i := range result.deployment.apps {
		rApp := result.deployment.apps[i]
		if rApp.Name == app.Name {
			isExisted = true
			result.deployment.apps[i] = app
		}
	}

	if !isExisted {
		result.deployment.apps = append(result.deployment.apps, app)
	}

	return nil
}

func (md *mockDeployment) DeleteApp(ctx context.Context, app component.Application) error {
	if result.deployment == nil || result.deployment.apps == nil {
		return nil
	}

	var newApps []component.Application
	for i := range result.deployment.apps {
		rApp := result.deployment.apps[i]
		if rApp.Name != app.Name {
			newApps = append(newApps, rApp)
		}
	}

	result.deployment.apps = newApps

	return nil
}

// msm *mockSecretManager SecretManagement
type mockSecretManager struct{}

func NewMockSecretManagement(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.SecretManagement, error) {
	return &mockSecretManager{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (msm *mockSecretManager) CleanUp() error {
	return nil
}

func (msm *mockSecretManager) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

// GetAccessInfo should return the information on how to access the cluster
func (msm *mockSecretManager) GetAccessInfo(ctx context.Context) (string, error) {
	return `apiVersion: v1
clusters:
- cluster:
    server: http://127.0.0.1:8080
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
users:
- name: default
  user:
    token: "000"
`, nil
}

func (msm *mockSecretManager) CreateAccount(ctx context.Context, account component.MachineAccount) (*component.AuthInfo, error) {
	authInfo := &component.AuthInfo{
		OriginName:  "mock",
		AccountName: account.Name,
		AuthType:    component.AuthTypeKubernetesServiceAccount,
	}

	for _, space := range account.Spaces {
		authInfo.ServiceAccounts = append(authInfo.ServiceAccounts, component.AuthInfoServiceAccount{
			ServiceAccount: account.Name,
			Namespace:      space,
		})
	}
	return authInfo, nil
}

func (msm *mockSecretManager) DeleteAccount(ctx context.Context, account component.MachineAccount) error {
	return nil
}

func (msm *mockSecretManager) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	return nil
}

func (msm *mockSecretManager) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	return nil
}

// mss *mockSecretSyncer SecretSync
type mockSecretSyncer struct{}

func NewMockSecretSync(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.SecretSync, error) {
	return &mockSecretSyncer{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mss *mockSecretSyncer) CleanUp() error {
	return nil
}

func (mss *mockSecretSyncer) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

// CreateSecret will create a secret object (sercret in kubernetes, file in host) from secret database to dest environment.
func (mss *mockSecretSyncer) CreateSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

func (mss *mockSecretSyncer) RemoveSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

// mel *mockEventListener EventListener
type mockEventListener struct{}

func NewMockEventListener(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.EventListener, error) {
	return &mockEventListener{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mel *mockEventListener) CleanUp() error {
	return nil
}

func (mel *mockEventListener) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

func (mel *mockEventListener) CreateEventSource(ctx context.Context, eventSource component.EventSourceSet) error {
	return nil
}

func (mel *mockEventListener) DeleteEventSource(ctx context.Context, UniqueID string) error {
	return nil
}

func (mel *mockEventListener) CreateConsumer(ctx context.Context, consumer component.ConsumerSet) error {
	return nil
}

func (mel *mockEventListener) DeleteConsumer(ctx context.Context, productName string, name string) error {
	return nil
}

// mpf *mockPipelineFactory PipelineFactory
type mockPipelineFactory struct{}

func (mpf *mockPipelineFactory) NewComponent(opt v1alpha1.Component, info *component.ComponentInitInfo, status interface{}) (component.Pipeline, error) {
	return &mockPipeline{}, nil
}
func (mpf *mockPipelineFactory) NewStatus(rawStatus []byte) (interface{}, error) {
	return nil, nil
}

// mp *mockPipeline component.Pipeline
type mockPipeline struct{}

func (mp *mockPipeline) CleanUp() error {
	return nil
}

func (mp *mockPipeline) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

func (mp *mockPipeline) GetHooks(info component.HooksInitInfo) (component.Hooks, []interface{}, error) {
	return component.Hooks{}, nil, nil
}

func (mp *mockPipeline) CreateHookSpace(ctx context.Context, authInfo component.AuthInfo, space component.HookSpace) error {
	return nil
}

func (mp *mockPipeline) CleanUpHookSpace(ctx context.Context) error {
	return nil
}
