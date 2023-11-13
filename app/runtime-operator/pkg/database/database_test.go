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

package database_test

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database", func() {
	var testObjects []client.Object
	var testNamespace []corev1.Namespace
	var productName string
	var seed string
	var clusterNames []string
	var envNames []string
	var deployRuntimeNames []string
	var pipelineRuntimeNames []string
	var product *v1alpha1.Product
	var codeRepoProvider *v1alpha1.CodeRepoProvider
	var productCodeRepo *v1alpha1.CodeRepo
	var codeRepo01 *v1alpha1.CodeRepo
	var deployRuntime01 *v1alpha1.DeploymentRuntime
	var pipelineRuntime01 *v1alpha1.ProjectPipelineRuntime

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

		codeRepoProvider = &v1alpha1.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("codeprovider-%s", seed),
				Namespace: nautesNamespaceName,
			},
			Spec: v1alpha1.CodeRepoProviderSpec{
				HttpAddress:  "https://127.0.0.1",
				SSHAddress:   "ssh://git@127.0.0.1",
				ApiServer:    "https://127.0.0.1",
				ProviderType: "gitlab",
			},
		}
		testObjects = append(testObjects, codeRepoProvider)

		cluster01 := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterNames[1],
				Namespace: nautesNamespaceName,
			},
			Spec: v1alpha1.ClusterSpec{
				ApiServer:                         "https://127.0.0.1:6443",
				ClusterType:                       v1alpha1.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       v1alpha1.CLUSTER_KIND_KUBERNETES,
				Usage:                             v1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:                        v1alpha1.ClusterWorkTypeDeployment,
				ComponentsList:                    v1alpha1.ComponentsList{},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]v1alpha1.ClusterResourceInfo{},
			},
		}
		testObjects = append(testObjects, &cluster01)

		cluster02 := v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterNames[2],
				Namespace: nautesNamespaceName,
			},
			Spec: v1alpha1.ClusterSpec{
				ApiServer:                         "https://127.0.0.1:6443",
				ClusterType:                       v1alpha1.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       v1alpha1.CLUSTER_KIND_KUBERNETES,
				Usage:                             v1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:                        v1alpha1.ClusterWorkTypePipeline,
				ComponentsList:                    v1alpha1.ComponentsList{},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]v1alpha1.ClusterResourceInfo{},
			},
		}
		testObjects = append(testObjects, &cluster02)

		product = &v1alpha1.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: nautesNamespaceName,
			},
			Spec: v1alpha1.ProductSpec{
				Name:         fmt.Sprintf("productName-%s", seed),
				MetaDataPath: "",
			},
		}
		testObjects = append(testObjects, product)
		testNamespace = append(testNamespace, corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: product.Name,
			},
		})

		productCodeRepo = &v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("default-repo-%s", seed),
				Namespace: nautesNamespaceName,
				Labels: map[string]string{
					v1alpha1.LABEL_FROM_PRODUCT: product.Name,
				},
			},
			Spec: v1alpha1.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          "",
				RepoName:         "default.project",
				URL:              "ssh://127.0.0.1/nautes/default.project.git",
			},
		}
		testObjects = append(testObjects, productCodeRepo)

		env01 := v1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envNames[1],
				Namespace: product.Name,
			},
			Spec: v1alpha1.EnvironmentSpec{
				Product: product.Name,
				Cluster: cluster01.Name,
				EnvType: "",
			},
		}
		testObjects = append(testObjects, &env01)

		env02 := v1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envNames[2],
				Namespace: product.Name,
			},
			Spec: v1alpha1.EnvironmentSpec{
				Product: product.Name,
				Cluster: cluster02.Name,
				EnvType: "",
			},
		}
		testObjects = append(testObjects, &env02)

		project01 := v1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: v1alpha1.ProjectSpec{
				Product:  product.Name,
				Language: "",
			},
		}
		testObjects = append(testObjects, &project01)

		repoName := fmt.Sprintf("repoName-01-%s", seed)
		codeRepo01 = &v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("coderepo-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: v1alpha1.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          project01.Name,
				RepoName:         repoName,
			},
		}
		testObjects = append(testObjects, codeRepo01)

		additionalRepoName := fmt.Sprintf("add-repoName-01-%s", seed)
		additionalCodeRepo01 := v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("add-coderepo-01-%s", seed),
				Namespace: product.Name,
			},
			Spec: v1alpha1.CodeRepoSpec{
				CodeRepoProvider: codeRepoProvider.Name,
				Product:          product.Name,
				Project:          project01.Name,
				RepoName:         additionalRepoName,
			},
		}
		testObjects = append(testObjects, &additionalCodeRepo01)

		deployRuntime01 = &v1alpha1.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployRuntimeNames[1],
				Namespace: product.Name,
			},
			Spec: v1alpha1.DeploymentRuntimeSpec{
				Product:     product.Name,
				ProjectsRef: []string{},
				ManifestSource: v1alpha1.ManifestSource{
					CodeRepo:       codeRepo01.Name,
					TargetRevision: "main",
					Path:           "deploy",
				},
				Destination: v1alpha1.DeploymentRuntimesDestination{
					Environment: env01.Name,
					Namespaces: []string{
						deployRuntimeNames[1],
						"otherNamespace"},
				},
			},
		}
		testObjects = append(testObjects, deployRuntime01)

		pipelineRuntime01 = &v1alpha1.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineRuntimeNames[1],
				Namespace: product.Name,
			},
			Spec: v1alpha1.ProjectPipelineRuntimeSpec{
				Project:        project01.Name,
				PipelineSource: codeRepo01.Name,
				Pipelines:      []v1alpha1.Pipeline{},
				Destination: v1alpha1.ProjectPipelineDestination{
					Environment: env02.Name,
				},
				EventSources: []v1alpha1.EventSource{
					{
						Name: "test",
						Gitlab: &v1alpha1.Gitlab{
							RepoName: codeRepo01.Name,
							Revision: "main",
							Events:   []string{},
						},
					},
				},
				Isolation: "",
				PipelineTriggers: []v1alpha1.PipelineTrigger{
					{
						EventSource: "test",
						Pipeline:    "test",
					},
				},
				AdditionalResources: &v1alpha1.ProjectPipelineRuntimeAdditionalResources{
					Git: &v1alpha1.ProjectPipelineRuntimeAdditionalResourcesGit{
						CodeRepo: additionalCodeRepo01.Name,
						Revision: "main",
						Path:     "customize",
					},
				},
			},
		}
		testObjects = append(testObjects, pipelineRuntime01)

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
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		requestResult := database.NamespaceUsage{
			clusterNames[1]: {deployRuntimeNames[1], "otherNamespace", productName},
			clusterNames[2]: {pipelineRuntimeNames[1], productName},
		}
		nsUsage, err := db.ListUsedNamespaces()
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list namespaces by cluster", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		requestResult := database.NamespaceUsage{
			clusterNames[1]: {deployRuntimeNames[1], "otherNamespace", productName},
		}
		nsUsage, err := db.ListUsedNamespaces(database.InCluster(clusterNames[1]))
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list namespaces in without specify runtime", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtime := &v1alpha1.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployRuntimeNames[1],
				Namespace: productName,
			},
			Spec: v1alpha1.DeploymentRuntimeSpec{
				Product: productName,
			},
		}

		requestResult := database.NamespaceUsage{
			clusterNames[2]: {
				pipelineRuntimeNames[1],
				productName,
			},
		}
		nsUsage, err := db.ListUsedNamespaces(database.ExcludeRuntimes([]v1alpha1.Runtime{runtime}))
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(requestResult, nsUsage)).Should(BeTrue())
	})

	It("can list code repo", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		codeRepos, err := db.ListUsedCodeRepos()
		Expect(err).Should(BeNil())
		Expect(len(codeRepos)).Should(Equal(3))
	})

	It("can list code repo by cluster", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		codeRepos, err := db.ListUsedCodeRepos(database.InCluster(clusterNames[2]))
		Expect(err).Should(BeNil())
		Expect(len(codeRepos)).Should(Equal(2))
	})

	It("will remove trigger and event source if pipeline has no permission", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtimes, _ := db.ListPipelineRuntimes()
		Expect(len(runtimes[0].Spec.EventSources)).Should(Equal(1))
		Expect(len(runtimes[0].Spec.PipelineTriggers)).Should(Equal(1))

		runtime := &v1alpha1.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineRuntimeNames[1],
				Namespace: productName,
			}}

		_, err = controllerutil.CreateOrUpdate(context.TODO(), k8sClient, runtime, func() error {
			runtime.Spec.EventSources[0].Gitlab.RepoName = "x"
			return nil
		})
		Expect(err).Should(BeNil())

		db, err = database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtimes, _ = db.ListPipelineRuntimes()
		Expect(len(runtimes[0].Spec.EventSources)).Should(Equal(0))
		Expect(len(runtimes[0].Spec.PipelineTriggers)).Should(Equal(0))
	})

	It("can get deploy runtime", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtime, err := db.GetRuntime(deployRuntime01.Name, v1alpha1.RuntimeTypeDeploymentRuntime)
		Expect(err).Should(BeNil())
		dRuntime, ok := runtime.(*v1alpha1.DeploymentRuntime)
		Expect(ok).Should(BeTrue())
		Expect(dRuntime.Name == deployRuntime01.Name).Should(BeTrue())
		Expect(dRuntime.Namespace == deployRuntime01.Namespace).Should(BeTrue())
		ok = reflect.DeepEqual(dRuntime.Spec, deployRuntime01.Spec)
		Expect(ok).Should(BeTrue())
	})

	It("can get pipeline runtime", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		runtime, err := db.GetRuntime(pipelineRuntime01.Name, v1alpha1.RuntimeTypePipelineRuntime)
		Expect(err).Should(BeNil())
		dRuntime, ok := runtime.(*v1alpha1.ProjectPipelineRuntime)
		Expect(ok).Should(BeTrue())
		Expect(dRuntime.Name == pipelineRuntime01.Name).Should(BeTrue())
		Expect(dRuntime.Namespace == pipelineRuntime01.Namespace).Should(BeTrue())
		ok = reflect.DeepEqual(dRuntime.Spec, pipelineRuntime01.Spec)
		Expect(ok).Should(BeTrue())
	})

	It("can get code repo by url", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		repoURL := fmt.Sprintf("%s/%s/%s.git", codeRepoProvider.Spec.SSHAddress, product.Spec.Name, codeRepo01.Spec.RepoName)
		repo, err := db.GetCodeRepoByURL(repoURL)
		Expect(err).Should(BeNil())
		Expect(repo.Name == codeRepo01.Name).Should(BeTrue())
		Expect(repo.Namespace == "").Should(BeTrue())
		repo.Spec.URL = ""
		ok := reflect.DeepEqual(repo.Spec, codeRepo01.Spec)
		Expect(ok).Should(BeTrue())
	})

	It("can get code repo provider", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		provider, err := db.GetCodeRepoProvider(codeRepoProvider.Name)
		Expect(err).Should(BeNil())

		Expect(provider.Name == codeRepoProvider.Name).Should(BeTrue())
		Expect(provider.Name == codeRepoProvider.Name).Should(BeTrue())
		ok := reflect.DeepEqual(provider.Spec, codeRepoProvider.Spec)
		Expect(ok).Should(BeTrue())
	})

	It("can get product", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		productFromDB, err := db.GetProduct(product.Name)
		Expect(err).Should(BeNil())

		Expect(productFromDB.Name == product.Name).Should(BeTrue())
		Expect(productFromDB.Name == product.Name).Should(BeTrue())
		ok := reflect.DeepEqual(productFromDB.Spec, product.Spec)
		Expect(ok).Should(BeTrue())
	})

	It("can get product code repo", func() {
		db, err := database.NewRuntimeSnapshot(context.TODO(), k8sClient, productName, nautesNamespaceName)
		Expect(err).Should(BeNil())

		repo, err := db.GetProductCodeRepo(productName)
		Expect(err).Should(BeNil())
		Expect(repo.Name == productCodeRepo.Name).Should(BeTrue())
		Expect(repo.Namespace == "").Should(BeTrue())
		ok := reflect.DeepEqual(repo.Spec, productCodeRepo.Spec)
		Expect(ok).Should(BeTrue())

	})
})
