package syncer_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
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

	NewDatabase = newMockDB
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
			filepath.Join("..", "..", "..", "..", "..", "config", "crd", "bases"),
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
	name   string
	spaces []deployResultSpace
	users  []string
}

type deployResultSpace struct {
	name  string
	users []string
}

type deployResultDeployment struct {
	product deployResultDeploymentProduct
	apps    []Application
}

type deployResultDeploymentProduct struct {
	name  string
	users []string
}

// md *mockDB database.Database
type mockDB struct {
	cluster  *v1alpha1.Cluster
	product  *v1alpha1.Product
	coderepo *v1alpha1.CodeRepo
	provider *v1alpha1.CodeRepoProvider
	runtimes map[string]v1alpha1.Runtime
}

func newMockDB(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (database.Database, error) {
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

func NewMockMultiTenant(opt v1alpha1.Component, info *ComponentInitInfo) (MultiTenant, error) {
	return &mockMultiTenant{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mmt *mockMultiTenant) CleanUp() error {
	return nil
}

func (mmt *mockMultiTenant) CreateProduct(ctx context.Context, name string) error {
	if result.product == nil {
		result.product = &deployResultProduct{
			name:   name,
			spaces: []deployResultSpace{},
			users:  []string{},
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

func (mmt *mockMultiTenant) GetProduct(ctx context.Context, name string) (*ProductStatus, error) {
	panic("not implemented") // TODO: Implement
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
		name:  name,
		users: []string{},
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

func (mmt *mockMultiTenant) GetSpace(ctx context.Context, productName string, name string) (*SpaceStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (mmt *mockMultiTenant) ListSpaces(ctx context.Context, productName string, opts ...ListOption) ([]SpaceStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (mmt *mockMultiTenant) AddSpaceUser(ctx context.Context, request PermissionRequest) error {
	if result.product == nil {
		return fmt.Errorf("product not found")
	}

	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName == request.Resource.Name {
			result.product.spaces[i].users = append(result.product.spaces[i].users, request.User)
			break
		}
	}

	return nil
}

func (mmt *mockMultiTenant) DeleteSpaceUser(ctx context.Context, request PermissionRequest) error {
	if result.product == nil {
		return nil
	}

	for i := range result.product.spaces {
		spaceName := result.product.spaces[i].name
		if spaceName != request.Resource.Name {
			continue
		}
		for j, userName := range result.product.spaces[i].users {
			if userName != request.User {
				continue
			}
			result.product.spaces[i].users = append(
				result.product.spaces[i].users[:j],
				result.product.spaces[i].users[j+1:]...)
		}
	}

	return nil
}

func (mmt *mockMultiTenant) CreateUser(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return fmt.Errorf("product not found")
	}

	for _, userName := range result.product.users {
		if userName == name {
			return nil
		}
	}

	result.product.users = append(result.product.users, name)
	return nil
}

func (mmt *mockMultiTenant) DeleteUser(ctx context.Context, productName string, name string) error {
	if result.product == nil {
		return nil
	}

	for i, userName := range result.product.users {
		if userName == name {
			result.product.users = append(result.product.users[:i], result.product.users[i+1:]...)
			break
		}
	}

	for _, space := range result.product.spaces {
		mmt.DeleteSpaceUser(ctx, PermissionRequest{
			Resource: Resource{
				Product: productName,
				Name:    space.name,
			},
			User: name,
		})
	}

	return nil
}

func (mmt *mockMultiTenant) GetUser(ctx context.Context, productName string, name string) (*User, error) {
	if result.product == nil {
		return nil, fmt.Errorf("product not found")
	}

	authInfo := &Auth{
		Kubernetes: []AuthKubernetes{},
	}

	for i := range result.product.spaces {
		userSet := sets.New(result.product.spaces[i].users...)
		if userSet.Has(name) {
			authInfo.Kubernetes = append(authInfo.Kubernetes, AuthKubernetes{
				ServiceAccount: name,
				Namespace:      result.product.spaces[i].name,
			})
		}
	}

	for _, userName := range result.product.users {
		if userName == name {
			return &User{
				Resource: Resource{
					Product: productName,
					Name:    name,
				},
				UserType: UserTypeMachine,
				AuthInfo: authInfo,
			}, nil
		}
	}
	return nil, UserNotFound(errors.New(""), name)
}

// md *mockDeployment Deployment
type mockDeployment struct{}

func NewMockDeployment(opt v1alpha1.Component, info *ComponentInitInfo) (Deployment, error) {
	return &mockDeployment{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (md *mockDeployment) CleanUp() error {
	return nil
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

func (md *mockDeployment) GetProduct(ctx context.Context, name string) (*ProductStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (md *mockDeployment) AddProductUser(ctx context.Context, request PermissionRequest) error {
	if result.deployment == nil {
		return fmt.Errorf("product not found")
	}

	userSet := sets.New(result.deployment.product.users...)
	if !userSet.Has(request.User) {
		result.deployment.product.users = append(result.deployment.product.users, request.User)
	}

	return nil
}

func (md *mockDeployment) DeleteProductUser(ctx context.Context, request PermissionRequest) error {
	if result.deployment == nil {
		return nil
	}

	userSet := sets.New(result.deployment.product.users...)
	if userSet.Has(request.User) {
		result.deployment.product.users = userSet.Delete(request.User).UnsortedList()
	}

	if len(result.deployment.product.users) == 0 {
		result.deployment.product.users = nil
	}

	return nil
}

func (md *mockDeployment) CreateApp(ctx context.Context, app Application) error {
	if result.deployment == nil {
		return fmt.Errorf("product not found")
	}

	if result.deployment.apps == nil {
		result.deployment.apps = []Application{}
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

func (md *mockDeployment) DeleteApp(ctx context.Context, app Application) error {
	if result.deployment == nil || result.deployment.apps == nil {
		return nil
	}

	var newApps []Application
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

func NewMockSecretManagement(opt v1alpha1.Component, info *ComponentInitInfo) (SecretManagement, error) {
	return &mockSecretManager{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (msm *mockSecretManager) CleanUp() error {
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

func (msm *mockSecretManager) CreateUser(ctx context.Context, user User) error {
	return nil
}

func (msm *mockSecretManager) DeleteUser(ctx context.Context, user User) error {
	return nil
}

func (msm *mockSecretManager) GrantPermission(ctx context.Context, repo SecretInfo, user User) error {
	return nil
}

func (msm *mockSecretManager) RevokePermission(ctx context.Context, repo SecretInfo, user User) error {
	return nil
}

// mss *mockSecretSyncer SecretSync
type mockSecretSyncer struct{}

func NewMockSecretSync(opt v1alpha1.Component, info *ComponentInitInfo) (SecretSync, error) {
	return &mockSecretSyncer{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mss *mockSecretSyncer) CleanUp() error {
	return nil
}

// CreateSecret will create a secret object (sercret in kubernetes, file in host) from secret database to dest environment.
func (mss *mockSecretSyncer) CreateSecret(ctx context.Context, secretReq SecretRequest) error {
	return nil
}

func (mss *mockSecretSyncer) RemoveSecret(ctx context.Context, secretReq SecretRequest) error {
	return nil
}

// mel *mockEventListener EventListener
type mockEventListener struct{}

func NewMockEventListener(opt v1alpha1.Component, info *ComponentInitInfo) (EventListener, error) {
	return &mockEventListener{}, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (mel *mockEventListener) CleanUp() error {
	return nil
}

func (mel *mockEventListener) CreateEventSource(ctx context.Context, eventSource EventSource) error {
	return nil
}

func (mel *mockEventListener) DeleteEventSource(ctx context.Context, UniqueID string) error {
	return nil
}

func (mel *mockEventListener) CreateConsumer(ctx context.Context, consumer Consumers) error {
	return nil
}

func (mel *mockEventListener) DeleteConsumer(ctx context.Context, productName string, name string) error {
	return nil
}
