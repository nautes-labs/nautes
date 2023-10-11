package syncer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("pipeline runtime deployer", func() {
	var seed string
	var ctx context.Context
	var cluster *v1alpha1.Cluster
	var repoProvider *v1alpha1.CodeRepoProvider
	var codeRepo *v1alpha1.CodeRepo
	var runtime *v1alpha1.ProjectPipelineRuntime
	var product *v1alpha1.Product
	var productIDs []string
	var projectNames []string
	var namespaceNames []string
	var runtimeNames []string
	BeforeEach(func() {
		seed = RandNum()
		ctx = context.Background()
		result = &deployResult{}

		productIDs = GenerateNames(fmt.Sprintf("product-%s-%%d", seed), 3)
		projectNames = GenerateNames(fmt.Sprintf("project-%s-%%d", seed), 3)
		namespaceNames = GenerateNames(fmt.Sprintf("ns-%s-%%d", seed), 3)
		runtimeNames = GenerateNames(fmt.Sprintf("runtime-%s-%%d", seed), 3)

		cluster = &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", seed),
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.ClusterSpec{
				ApiServer:   "https://127.0.0.1:6443",
				ClusterType: v1alpha1.CLUSTER_TYPE_PHYSICAL,
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Usage:       v1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  v1alpha1.ClusterWorkTypePipeline,
				ComponentsList: v1alpha1.ComponentsList{
					Deployment: &v1alpha1.Component{
						Name:      "mock",
						Namespace: "mock",
					},
					MultiTenant: &v1alpha1.Component{
						Name:      "mock",
						Namespace: "mock",
					},
					SecretManagement: &v1alpha1.Component{
						Name:      "mock",
						Namespace: "mock",
					},
					SecretSync: &v1alpha1.Component{
						Name:      "mock",
						Namespace: "mock",
					},
					EventListener: &v1alpha1.Component{
						Name:      "mock",
						Namespace: "mock",
					},
				},
			},
		}

		productName := fmt.Sprintf("productName-%s", seed)
		product = &v1alpha1.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productIDs[0],
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.ProductSpec{
				Name:         productName,
				MetaDataPath: "",
			},
		}

		repoProvider = &v1alpha1.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("codeProvider-%s", seed),
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.CodeRepoProviderSpec{
				ProviderType: "gitlab",
			},
		}

		repoName := fmt.Sprintf("repoName-%s", seed)
		codeRepo = &v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("codeRepo-%s", seed),
				Namespace: productIDs[0],
			},
			Spec: v1alpha1.CodeRepoSpec{
				CodeRepoProvider: repoProvider.Name,
				Product:          productIDs[0],
				Project:          "",
				RepoName:         repoName,
				URL:              fmt.Sprintf("https://127.0.0.1:8443/%s/%s.git", productName, repoName),
				Webhook:          &v1alpha1.Webhook{},
			},
		}

		runtime = &v1alpha1.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeNames[0],
				Namespace: productIDs[0],
			},
			Spec: v1alpha1.ProjectPipelineRuntimeSpec{
				Project:        projectNames[0],
				PipelineSource: codeRepo.Name,
				Pipelines: []v1alpha1.Pipeline{
					{
						Name:  "main",
						Label: "main",
						Path:  "pipeline/main.yaml",
					},
					{
						Name:  "dev",
						Label: "dev",
						Path:  "pipeline/dev.yaml",
					},
				},
				Destination: v1alpha1.ProjectPipelineDestination{
					Environment: "",
					Namespace:   namespaceNames[0],
				},
				EventSources: []v1alpha1.EventSource{
					{
						Name: "main",
						Gitlab: &v1alpha1.Gitlab{
							RepoName: codeRepo.Name,
							Revision: "main",
							Events: []string{
								"push_events",
							},
						},
					},
					{
						Name: "daily",
						Calendar: &v1alpha1.Calendar{
							Schedule: "1 0 * * *",
						},
					},
				},
				Isolation: "",
				PipelineTriggers: []v1alpha1.PipelineTrigger{
					{
						EventSource: "main",
						Pipeline:    "main",
					},
					{
						EventSource: "daily",
						Pipeline:    "main",
						Revision:    "main",
					},
				},
				AdditionalResources: &v1alpha1.ProjectPipelineRuntimeAdditionalResources{
					Git: &v1alpha1.ProjectPipelineRuntimeAdditionalResourcesGit{
						CodeRepo: codeRepo.Name,
						Revision: "main",
						Path:     "deploy/",
					},
				},
			},
		}

		db = &mockDB{
			cluster:  cluster,
			product:  product,
			coderepo: codeRepo,
			provider: repoProvider,
			runtimes: map[string]v1alpha1.Runtime{
				runtime.Name: runtime,
			},
		}
	})

	AfterEach(func() {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      syncer.ConfigMapNameClustersUsageCache,
				Namespace: nautesNamespace,
			},
		}
		err := k8sClient.Delete(ctx, cm)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
	})

	It("can deploy pipeline runtime", func() {
		task, err := testSyncer.NewTasks(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.PipelineRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product: &deployResultProduct{
				name: productIDs[0],
				spaces: []deployResultSpace{
					{
						name:  runtime.Spec.Destination.Namespace,
						users: []string{runtime.Name},
					},
				},
				users: []string{runtime.Name},
			},
			deployment: &deployResultDeployment{
				product: deployResultDeploymentProduct{
					name:  runtime.GetProduct(),
					users: []string{product.Spec.Name},
				},
				apps: []syncer.Application{
					{
						Resource: syncer.Resource{
							Product: runtime.GetProduct(),
							Name:    fmt.Sprintf("%s-additional", runtime.Name),
						},
						Git: &syncer.ApplicationGit{
							URL:      codeRepo.Spec.URL,
							Revision: runtime.Spec.AdditionalResources.Git.Revision,
							Path:     runtime.Spec.AdditionalResources.Git.Path,
							CodeRepo: codeRepo.Name,
						},
						Destinations: []syncer.Space{
							{
								Resource: syncer.Resource{
									Product: runtime.GetProduct(),
									Name:    runtime.Spec.Destination.Namespace,
								},
								SpaceType: syncer.SpaceTypeKubernetes,
								Kubernetes: syncer.SpaceKubernetes{
									Namespace: runtime.Spec.Destination.Namespace,
								},
							},
						},
					},
				},
			},
		}
		ok := reflect.DeepEqual(destResult, result)
		Expect(ok).Should(BeTrue())
	})

	It("can delete pipeline runtime", func() {
		task, err := testSyncer.NewTasks(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		task, err = testSyncer.NewTasks(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())

		rawCache, err = task.Delete(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.PipelineRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product:    nil,
			deployment: nil,
		}
		ok := reflect.DeepEqual(destResult, result)
		Expect(ok).Should(BeTrue())
	})
})
