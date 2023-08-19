package datasource_test

import (
	"context"
	"fmt"
	"reflect"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/datasource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Datasource", func() {
	var testObjects []client.Object
	var testNamespace []corev1.Namespace
	var productName string
	var seed string
	var clusterNames []string
	var envNames []string
	var deployRuntimeNames []string
	var pipelineRuntimeNames []string

	BeforeEach(func() {
		seed = RandNum()
		ctx := context.Background()
		testObjects = []client.Object{}
		testNamespace = []corev1.Namespace{}
		productName = fmt.Sprintf("product-%s", seed)
		clusterNames = GenerateNames(fmt.Sprintf("cluster-%%d-%s", seed), 10)
		envNames = GenerateNames(fmt.Sprintf("env-%%d-%s", seed), 10)
		deployRuntimeNames = GenerateNames(fmt.Sprintf("dr-%%d-%s", seed), 10)
		pipelineRuntimeNames = GenerateNames(fmt.Sprintf("pr-%%d-%s", seed), 10)

		codeRepoProvider := nautescrd.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("codeprovider-%s", seed),
				Namespace: nautesNamespaceName,
			},
			Spec: nautescrd.CodeRepoProviderSpec{
				HttpAddress:  "https://127.0.0.1",
				SSHAddress:   "ssh://git@127.0.0.1",
				ApiServer:    "https://127.0.0.1",
				ProviderType: "gitlab",
			},
		}
		testObjects = append(testObjects, &codeRepoProvider)

		cluster01 := nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterNames[1],
				Namespace: nautesNamespaceName,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:                         "https://127.0.0.1:6443",
				ClusterType:                       nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:                             nautescrd.CLUSTER_USAGE_WORKER,
				WorkerType:                        nautescrd.ClusterWorkTypeDeployment,
				ComponentsList:                    nautescrd.ComponentsList{},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]nautescrd.ClusterResourceInfo{},
			},
		}
		testObjects = append(testObjects, &cluster01)

		cluster02 := nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterNames[2],
				Namespace: nautesNamespaceName,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:                         "https://127.0.0.1:6443",
				ClusterType:                       nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:                             nautescrd.CLUSTER_USAGE_WORKER,
				WorkerType:                        nautescrd.ClusterWorkTypePipeline,
				ComponentsList:                    nautescrd.ComponentsList{},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]nautescrd.ClusterResourceInfo{},
			},
		}
		testObjects = append(testObjects, &cluster02)

		product := nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: nautesNamespaceName,
			},
			Spec: nautescrd.ProductSpec{
				Name:         fmt.Sprintf("productName-%s", seed),
				MetaDataPath: "",
			},
		}
		testObjects = append(testObjects, &product)
		testNamespace = append(testNamespace, corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: product.Name,
			},
		})

		productCodeRepo := nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("default-repo-%s", seed),
				Namespace: nautesNamespaceName,
				Labels: map[string]string{
					nautescrd.LABEL_FROM_PRODUCT: product.Name,
				},
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          "",
				RepoName:         "default.project",
				URL:              "ssh://127.0.0.1/nautes/default.project.git",
			},
		}
		testObjects = append(testObjects, &productCodeRepo)

		env01 := nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envNames[1],
				Namespace: product.Name,
			},
			Spec: nautescrd.EnvironmentSpec{
				Product: product.Name,
				Cluster: cluster01.Name,
				EnvType: "",
			},
		}
		testObjects = append(testObjects, &env01)

		env02 := nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envNames[2],
				Namespace: product.Name,
			},
			Spec: nautescrd.EnvironmentSpec{
				Product: product.Name,
				Cluster: cluster02.Name,
				EnvType: "",
			},
		}
		testObjects = append(testObjects, &env02)

		project01 := nautescrd.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: nautescrd.ProjectSpec{
				Product:  product.Name,
				Language: "",
			},
		}
		testObjects = append(testObjects, &project01)

		repoName := fmt.Sprintf("repoName-01-%s", seed)
		codeRepo01 := nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("coderepo-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          project01.Name,
				RepoName:         repoName,
			},
		}
		testObjects = append(testObjects, &codeRepo01)

		additionalRepoName := fmt.Sprintf("add-repoName-01-%s", seed)
		additionalCodeRepo01 := nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("add-coderepo-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          project01.Name,
				RepoName:         additionalRepoName,
			},
		}
		testObjects = append(testObjects, &additionalCodeRepo01)

		deployRuntime01 := nautescrd.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployRuntimeNames[1],
				Namespace: product.Name,
			},
			Spec: nautescrd.DeploymentRuntimeSpec{
				Product:     product.Name,
				ProjectsRef: []string{},
				ManifestSource: nautescrd.ManifestSource{
					CodeRepo:       codeRepo01.Name,
					TargetRevision: "main",
					Path:           "deploy",
				},
				Destination: nautescrd.DeploymentRuntimesDestination{
					Environment: env01.Name,
					Namespaces: []string{
						deployRuntimeNames[1],
						"otherNamespace"},
				},
			},
		}
		testObjects = append(testObjects, &deployRuntime01)

		pipelineRuntime01 := nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineRuntimeNames[1],
				Namespace: product.Name,
			},
			Spec: nautescrd.ProjectPipelineRuntimeSpec{
				Project:        project01.Name,
				PipelineSource: codeRepo01.Name,
				Pipelines:      []nautescrd.Pipeline{},
				Destination: nautescrd.ProjectPipelineDestination{
					Environment: env02.Name,
				},
				EventSources: []nautescrd.EventSource{
					{
						Name: "test",
						Gitlab: &nautescrd.Gitlab{
							RepoName: codeRepo01.Name,
							Revision: "main",
							Events:   []string{},
						},
					},
				},
				Isolation: "",
				PipelineTriggers: []nautescrd.PipelineTrigger{
					{
						EventSource: "test",
						Pipeline:    "test",
					},
				},
				AdditionalResources: &nautescrd.ProjectPipelineRuntimeAdditionalResources{
					Git: &nautescrd.ProjectPipelineRuntimeAdditionalResourcesGit{
						CodeRepo: additionalCodeRepo01.Name,
						Revision: "main",
						Path:     "customize",
					},
				},
			},
		}
		testObjects = append(testObjects, &pipelineRuntime01)

		for _, ns := range testNamespace {
			err := k8sClient.Create(ctx, ns.DeepCopy())
			Expect(err).Should(BeNil())
		}

		for _, obj := range testObjects {
			err := k8sClient.Create(ctx, obj)
			Expect(err).Should(BeNil())
		}

	})

	AfterEach(func() {
		ctx := context.Background()
		for _, obj := range testObjects {
			err := k8sClient.Delete(ctx, obj)
			Expect(client.IgnoreNotFound(err)).Should(BeNil())
			err = WaitForDelete(k8sClient, obj)
			Expect(err).Should(BeNil())
		}

		for _, ns := range testNamespace {
			err := k8sClient.Delete(ctx, ns.DeepCopy())
			Expect(err).Should(BeNil())
		}
	})

	It("can list namespaces in product", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		requestResult := datasource.NamespaceUsage{
			clusterNames[1]: {deployRuntimeNames[1], "otherNamespace", productName},
			clusterNames[2]: {pipelineRuntimeNames[1], productName},
		}
		nsUsage, err := db.ListUsedNamespaces()
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list namespaces by cluster", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		requestResult := datasource.NamespaceUsage{
			clusterNames[1]: {deployRuntimeNames[1], "otherNamespace", productName},
		}
		nsUsage, err := db.ListUsedNamespaces(datasource.InCluster(clusterNames[1]))
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list namespaces in without specify runtime", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtime := &nautescrd.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployRuntimeNames[1],
				Namespace: productName,
			},
			Spec: nautescrd.DeploymentRuntimeSpec{
				Product: productName,
			},
		}

		requestResult := datasource.NamespaceUsage{
			clusterNames[2]: {
				pipelineRuntimeNames[1],
				productName,
			},
		}
		nsUsage, err := db.ListUsedNamespaces(datasource.ExcludeRuntimes([]nautescrd.Runtime{runtime}))
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list code repo", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		codeRepos, err := db.ListUsedCodeRepos()
		Expect(err).Should(BeNil())
		Expect(len(codeRepos)).Should(Equal(3))
	})

	It("can list code repo by cluster", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		codeRepos, err := db.ListUsedCodeRepos(datasource.InCluster(clusterNames[2]))
		Expect(err).Should(BeNil())
		Expect(len(codeRepos)).Should(Equal(2))
	})

	It("will remove trigger and event source if pipeline has no permission", func() {
		db, err := datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtimes, _ := db.ListPipelineRuntimes()
		Expect(len(runtimes[0].Spec.EventSources)).Should(Equal(1))
		Expect(len(runtimes[0].Spec.PipelineTriggers)).Should(Equal(1))

		runtime := &nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineRuntimeNames[1],
				Namespace: productName,
			}}

		_, err = controllerutil.CreateOrUpdate(context.TODO(), k8sClient, runtime, func() error {
			runtime.Spec.EventSources[0].Gitlab.RepoName = "x"
			return nil
		})
		Expect(err).Should(BeNil())

		db, err = datasource.NewRuntimeDataSource(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtimes, _ = db.ListPipelineRuntimes()
		Expect(len(runtimes[0].Spec.EventSources)).Should(Equal(0))
		Expect(len(runtimes[0].Spec.PipelineTriggers)).Should(Equal(0))
	})
})
