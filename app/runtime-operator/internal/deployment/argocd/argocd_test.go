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
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var codeRepoProvider *nautescrd.CodeRepoProvider
var product *nautescrd.Product
var baseCodeRepo *nautescrd.CodeRepo
var baseEnvironment *nautescrd.Environment
var baseCluster *nautescrd.Cluster
var targetURL string

func updateURL() {
	targetURL = fmt.Sprintf("%s/%s/%s.git", codeRepoProvider.Spec.SSHAddress, product.Spec.Name, baseCodeRepo.Spec.RepoName)
}

var _ = Describe("Deploy app argocd", func() {
	var cleanQueue []client.Object
	var productName string
	var productPath string
	var runtimeName string
	var environmentName string
	var clusterName string
	var repoID string
	var repoName string
	var baseRuntime *nautescrd.DeploymentRuntime
	var err error
	var ns *corev1.Namespace
	var groupPolicy string
	var task interfaces.RuntimeSyncTask

	BeforeEach(func() {
		seed := randNum()

		cleanQueue = []client.Object{}
		_ = cleanQueue

		productName = fmt.Sprintf("product-%s", seed)
		productPath = fmt.Sprintf("productPath-%s", seed)
		runtimeName = fmt.Sprintf("runtime-%s", seed)
		environmentName = fmt.Sprintf("environment-%s", seed)
		clusterName = fmt.Sprintf("cluster-%s", seed)
		repoID = fmt.Sprintf("repo-%s", seed)
		repoName = fmt.Sprintf("repoName-%s", seed)

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		err = k8sClient.Create(context.Background(), ns)
		Expect(err).Should(BeNil())

		codeRepoProvider = &nautescrd.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("gitlab-%s", seed),
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.CodeRepoProviderSpec{
				HttpAddress:  "https://127.0.0.1",
				SSHAddress:   "ssh://127.0.0.1",
				ApiServer:    "",
				ProviderType: "gitlab",
			},
		}

		product = &nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ProductSpec{
				Name:         productPath,
				MetaDataPath: "",
			},
		}

		baseCluster = &nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:   "https://127.0.0.1:6443",
				ClusterType: nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind: nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:       nautescrd.CLUSTER_USAGE_WORKER,
				WorkerType:  nautescrd.ClusterWorkTypeDeployment,
				ComponentsList: nautescrd.ComponentsList{
					Deployment: &nautescrd.Component{
						Name:      "argocd",
						Namespace: "argocd",
					},
				},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]nautescrd.ClusterResourceInfo{},
			},
		}

		baseEnvironment = &nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      environmentName,
				Namespace: productName,
			},
			Spec: nautescrd.EnvironmentSpec{
				Product: productName,
				Cluster: clusterName,
				EnvType: "",
			},
		}

		baseCodeRepo = &nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      repoID,
				Namespace: productName,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  codeRepoProvider.Name,
				Product:           productName,
				Project:           "",
				RepoName:          repoName,
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
			},
		}

		baseRuntime = &nautescrd.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeName,
				Namespace: productName,
			},
			Spec: nautescrd.DeploymentRuntimeSpec{
				Product: productName,
				ManifestSource: nautescrd.ManifestSource{
					CodeRepo:       baseCodeRepo.Name,
					TargetRevision: "HEAD",
					Path:           "/production",
				},
				Destination: nautescrd.DeploymentRuntimesDestination{
					Environment: environmentName,
					Namespaces:  []string{runtimeName},
				},
			},
		}

		mockcli.provider = codeRepoProvider
		mockcli.product = product
		mockcli.codeRepo = baseCodeRepo
		mockcli.environment = []nautescrd.Environment{*baseEnvironment}
		mockcli.cluster = []nautescrd.Cluster{*baseCluster}
		mockcli.deployments = []nautescrd.DeploymentRuntime{*baseRuntime}
		updateURL()

		task = interfaces.RuntimeSyncTask{
			AccessInfo:  accessInfo,
			Product:     *product,
			NautesCfg:   *nautesCFG,
			Runtime:     baseRuntime,
			RuntimeType: interfaces.RUNTIME_TYPE_DEPLOYMENT,
		}
		groupPolicy = fmt.Sprintf("g, %s, role:%s", product.Spec.Name, product.Name)
	})

	AfterEach(func() {
		for i, runtime := range mockcli.deployments {
			if runtime.Name == baseRuntime.Name {
				mockcli.deployments[i].DeletionTimestamp = &metav1.Time{Time: time.Now()}
			}
		}
		err = mgr.UnDeploy(ctx, task)
		Expect(err).Should(BeNil())

		repo := &nautescrd.CodeRepo{}
		key := types.NamespacedName{
			Namespace: nautesNamespace,
			Name:      baseCodeRepo.Name,
		}
		err = k8sClient.Get(ctx, key, repo)
		Expect(err).ShouldNot(BeNil())
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		app := &argocrd.Application{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      baseRuntime.Name,
		}
		err = k8sClient.Get(ctx, key, app)
		Expect(err).ShouldNot(BeNil())
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		project := &argocrd.AppProject{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      productName,
		}
		err = k8sClient.Get(ctx, key, project)
		Expect(err).ShouldNot(BeNil())
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = k8sClient.Delete(context.Background(), ns)
		Expect(err).Should(BeNil())

		cm := &corev1.ConfigMap{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      argocdRBACConfigMapName,
		}
		err = k8sClient.Get(ctx, key, cm)
		Expect(err).Should(BeNil())
		Expect(strings.Contains(cm.Data[argocdRBACConfigMapKey], groupPolicy)).Should(BeFalse())
	})

	It("deploy new runtime", func() {
		_, err = mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		repo := &nautescrd.CodeRepo{}
		key := types.NamespacedName{
			Namespace: nautesNamespace,
			Name:      baseCodeRepo.Name,
		}
		err := k8sClient.Get(ctx, key, repo)
		Expect(err).Should(BeNil())
		Expect(repo.Spec.URL).Should(Equal(targetURL))

		app := &argocrd.Application{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      baseRuntime.Name,
		}
		err = k8sClient.Get(ctx, key, app)
		Expect(err).Should(BeNil())
		Expect(app.Spec.Source.RepoURL).Should(Equal(targetURL))

		project := &argocrd.AppProject{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      productName,
		}
		err = k8sClient.Get(ctx, key, project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.SourceRepos)).Should(Equal(1))
		Expect(project.Spec.SourceRepos[0]).Should(Equal(targetURL))
		Expect(len(project.Spec.Destinations)).Should(Equal(2))
		Expect(project.Spec.Destinations[0].Namespace).Should(Equal(baseRuntime.Name))

		baseRuntime.Status.DeployHistory = &nautescrd.DeployHistory{
			Destination: nautescrd.DeploymentRuntimesDestination{
				Environment: "",
				Namespaces:  []string{},
			},
			Source: targetURL,
		}

		cm := &corev1.ConfigMap{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      argocdRBACConfigMapName,
		}
		err = k8sClient.Get(ctx, key, cm)
		Expect(err).Should(BeNil())
		Expect(strings.Contains(cm.Data[argocdRBACConfigMapKey], groupPolicy)).Should(BeTrue())
	})

	It("code repo url change, runtime will update dest argo app, coderepo, app project", func() {
		source, err := mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		codeRepoProvider.Spec.SSHAddress = "ssh://127.0.1.1"
		product.Spec.Name = "tifa"
		baseCodeRepo.Spec.RepoName = "Rec"

		updateURL()
		baseRuntime.Status.DeployHistory = &nautescrd.DeployHistory{
			Source: source.Source,
		}
		_, err = mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		repo := &nautescrd.CodeRepo{}
		key := types.NamespacedName{
			Namespace: nautesNamespace,
			Name:      baseCodeRepo.Name,
		}
		err = k8sClient.Get(ctx, key, repo)
		Expect(err).Should(BeNil())
		Expect(repo.Spec.URL).Should(Equal(targetURL))

		app := &argocrd.Application{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      baseRuntime.Name,
		}
		err = k8sClient.Get(ctx, key, app)
		Expect(err).Should(BeNil())
		Expect(app.Spec.Source.RepoURL).Should(Equal(targetURL))

		project := &argocrd.AppProject{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      productName,
		}
		err = k8sClient.Get(ctx, key, project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.SourceRepos)).Should(Equal(1))
		Expect(project.Spec.SourceRepos[0]).Should(Equal(targetURL))
		Expect(len(project.Spec.Destinations)).Should(Equal(2))
		Expect(project.Spec.Destinations[0].Namespace).Should(Equal(baseRuntime.Name))

		baseRuntime.Status.DeployHistory = &nautescrd.DeployHistory{
			Destination: nautescrd.DeploymentRuntimesDestination{
				Environment: "",
				Namespaces:  []string{},
			},
			Source: targetURL,
		}
	})

	It("if code repo is using, runtime operator will not remove coderepo", func() {
		source, err := mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		baseRuntime.Status.DeployHistory = &nautescrd.DeployHistory{
			ManifestSource: baseRuntime.Spec.ManifestSource,
			Destination:    baseRuntime.Spec.Destination,
			Source:         source.Source,
		}

		runtime2 := baseRuntime.DeepCopy()
		runtime2.Name = fmt.Sprintf("%s-2", baseRuntime.Name)
		task.Runtime = runtime2

		source, err = mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		mockcli.deployments = []nautescrd.DeploymentRuntime{*baseRuntime, *runtime2}

		task.Runtime = baseRuntime
		err = mgr.UnDeploy(ctx, task)
		Expect(err).Should(BeNil())

		repo := &nautescrd.CodeRepo{}
		key := types.NamespacedName{
			Namespace: nautesNamespace,
			Name:      baseCodeRepo.Name,
		}
		err = k8sClient.Get(ctx, key, repo)
		Expect(err).Should(BeNil())
		Expect(repo.Spec.URL).Should(Equal(targetURL))

		project := &argocrd.AppProject{}
		key = types.NamespacedName{
			Namespace: argoNamespace,
			Name:      productName,
		}
		err = k8sClient.Get(ctx, key, project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.SourceRepos)).Should(Equal(1))
		Expect(project.Spec.SourceRepos[0]).Should(Equal(targetURL))
		Expect(len(project.Spec.Destinations)).Should(Equal(2))
		Expect(project.Spec.Destinations[0].Namespace).Should(Equal(runtimeName))

		mockcli.deployments = []nautescrd.DeploymentRuntime{}
		task.Runtime = runtime2
		err = mgr.UnDeploy(ctx, task)
		Expect(err).Should(BeNil())

		baseRuntime.Status.DeployHistory = &nautescrd.DeployHistory{
			Destination: nautescrd.DeploymentRuntimesDestination{
				Environment: "",
				Namespaces:  []string{},
			},
			Source: targetURL,
		}
	})

	It("if cluster allow product use namespace, app project has permission to deploy on it.", func() {
		baseCluster.Spec.ReservedNamespacesAllowedProducts = map[string][]string{
			"argocd": {productPath},
		}
		baseCluster.Status.ProductIDMap = map[string]string{
			productPath: productName,
		}
		mockcli.cluster = []nautescrd.Cluster{*baseCluster}
		_, err := mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		project := &argocrd.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: argoNamespace,
				Name:      productName,
			}}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.SourceRepos)).Should(Equal(1))
		Expect(project.Spec.SourceRepos[0]).Should(Equal(targetURL))
		Expect(len(project.Spec.Destinations)).Should(Equal(3))
	})

	It("if cluster allow product create cluster resources, app project has permission to deploy on it.", func() {
		baseCluster.Spec.ProductAllowedClusterResources = map[string][]nautescrd.ClusterResourceInfo{
			productPath: {
				{
					Kind:  "ClusterRole",
					Group: "authorization.k8s.io",
				},
			},
		}
		baseCluster.Status.ProductIDMap = map[string]string{
			productPath: productName,
		}
		mockcli.cluster = []nautescrd.Cluster{*baseCluster}
		_, err := mgr.Deploy(ctx, task)
		Expect(err).Should(BeNil())

		project := &argocrd.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: argoNamespace,
				Name:      productName,
			}}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.SourceRepos)).Should(Equal(1))
		Expect(project.Spec.SourceRepos[0]).Should(Equal(targetURL))
		Expect(len(project.Spec.Destinations)).Should(Equal(2))
		Expect(len(project.Spec.ClusterResourceWhitelist)).Should(Equal(1))
	})
})
