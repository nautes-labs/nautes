package argocd_test

import (
	"context"
	"path/filepath"
	"testing"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployment/argocd"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
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
			filepath.Join("..", "..", "..", "..", "..", "..", "test", "hnc"),
			filepath.Join("..", "..", "..", "..", "..", "..", "test", "argocd"),
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

// Generate by m *mockSecMgr syncer.SecretManagement
type mockSecMgr struct{}

// GetAccessInfo should return the infomation on how to access the cluster
func (m *mockSecMgr) GetAccessInfo(ctx context.Context, clusterName string) (string, error) {
	panic("not implemented") // TODO: Implement
}

// // The cache will be stored and passed based on the product name + user name.
func (m *mockSecMgr) CreateUser(ctx context.Context, user syncer.User, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) DeleteUser(ctx context.Context, user syncer.User, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockSecMgr) GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	return nil
}

func (m *mockSecMgr) RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	return nil
}

// Generate by m *mockMultiTenant syncer.MultiTenant
type mockMultiTenant struct {
	spaces []string
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (m *mockMultiTenant) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) CreateProduct(ctx context.Context, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteProduct(ctx context.Context, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetProduct(ctx context.Context, name string) (*syncer.ProductStatus, error) {
	panic("not implemented") // TODO: Implement
}

// Space use to manager a logical space in environment.
// It can be a namespace in kubernetes, a folder in host or some else.
// The cache will be stored and passed based on the product name + space name.
func (m *mockMultiTenant) CreateSpace(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteSpace(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetSpace(ctx context.Context, productName string, name string) (*syncer.SpaceStatus, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) ListSpaces(ctx context.Context, productName string, opts ...syncer.ListOption) ([]syncer.SpaceStatus, error) {
	spaces := make([]syncer.SpaceStatus, len(m.spaces))
	for i := 0; i < len(m.spaces); i++ {
		spaces[i] = syncer.SpaceStatus{
			Space: syncer.Space{
				Resource: syncer.Resource{
					Product: productName,
					Name:    m.spaces[i],
				},
				SpaceType: "",
				Kubernetes: syncer.SpaceKubernetes{
					Namespace: m.spaces[i],
				},
			},
			Users: []string{},
		}
	}
	return spaces, nil
}

func (m *mockMultiTenant) AddSpaceUser(ctx context.Context, request syncer.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteSpaceUser(ctx context.Context, request syncer.PermissionRequest) error {
	panic("not implemented") // TODO: Implement
}

// The cache will be stored and passed based on the product name + user name.
func (m *mockMultiTenant) CreateUser(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) DeleteUser(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error) {
	panic("not implemented") // TODO: Implement
}

func (m *mockMultiTenant) GetUser(ctx context.Context, productName string, name string) (*syncer.User, error) {
	panic("not implemented") // TODO: Implement
}
