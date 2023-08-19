package argocd_test

import (
	"path/filepath"
	"testing"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/datasource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var testEnv *envtest.Environment
var k8sClient client.Client
var cfg *rest.Config

func TestArgocd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Argocd Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme
	err = argov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

type mockDB struct {
	Namespaces *datasource.NamespaceUsage
	URLs       []string
}

// ListUsedNamespces should return all namespaces used by product
func (db *mockDB) ListUsedNamespaces(opts ...datasource.ListOption) (datasource.NamespaceUsage, error) {
	return *db.Namespaces, nil
}

func (db *mockDB) ListUsedURLs(opts ...datasource.ListOption) ([]string, error) {
	return db.URLs, nil
}

func (db *mockDB) ListUsedCodeRepos(opts ...datasource.ListOption) ([]nautescrd.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (db *mockDB) GetCodeRepo(name string) (*nautescrd.CodeRepo, error) {
	panic("not implemented") // TODO: Implement
}

func (db *mockDB) GetProduct() (*nautescrd.Product, error) {
	panic("not implemented") // TODO: Implement
}

func (db *mockDB) GetClusterByRuntime(runtime nautescrd.Runtime) (*nautescrd.Cluster, error) {
	panic("not implemented") // TODO: Implement
}

func (db *mockDB) ListPipelineRuntimes() ([]nautescrd.ProjectPipelineRuntime, error) {
	panic("not implemented") // TODO: Implement
}
