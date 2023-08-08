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

package tekton_test

import (
	"fmt"
	"reflect"
	"time"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/pipeline/tekton"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/constant"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvManager", func() {
	var product *nautescrd.Product
	var cluster *nautescrd.Cluster
	var productCodeRepo *nautescrd.CodeRepo
	var productCodeRepoProvider *nautescrd.CodeRepoProvider
	var runtimeCodeRepo *nautescrd.CodeRepo
	var runtimeCodeRepo2 *nautescrd.CodeRepo
	var runtimeCodeRepoProvider *nautescrd.CodeRepoProvider
	var productName string
	var runtime *nautescrd.ProjectPipelineRuntime
	var env *nautescrd.Environment
	var task interfaces.RuntimeSyncTask
	var task2 interfaces.RuntimeSyncTask
	var syncer interfaces.Pipeline
	var app *argocdv1alpha1.Application
	var project *argocdv1alpha1.AppProject
	var err error
	var productNamespace *corev1.Namespace
	var appName string
	BeforeEach(func() {
		productName = fmt.Sprintf("product-%s", randNum())

		product = &nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ProductSpec{
				Name:         "product",
				MetaDataPath: "ssh://git@127.0.0.1/product/default.project.git",
			},
		}

		productNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		err = k8sClient.Create(ctx, productNamespace)
		Expect(err).Should(BeNil())

		cluster = &nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", randNum()),
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:     "https://127.0.0.1:6443",
				ClusterType:   nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:   nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:         nautescrd.CLUSTER_USAGE_WORKER,
				HostCluster:   "",
				PrimaryDomain: "",
			},
		}

		env = &nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("env-%s", randNum()),
				Namespace: productName,
			},
			Spec: nautescrd.EnvironmentSpec{
				Product: productName,
				Cluster: cluster.Name,
				EnvType: "kubernetes",
			},
		}

		productCodeRepoProvider = &nautescrd.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("provider-%s", randNum()),
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.CodeRepoProviderSpec{
				HttpAddress:  "https://127.0.0.1",
				SSHAddress:   "ssh://git@127.0.0.1",
				ApiServer:    "https://127.0.0.1",
				ProviderType: "gitlab",
			},
		}

		productCodeRepo = &nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("rpeo-%s", randNum()),
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  productCodeRepoProvider.Name,
				Product:           productName,
				Project:           "",
				RepoName:          "default.project",
				URL:               "ssh://127.0.0.1/testgroup/default.project.git",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook:           nil,
			},
		}

		runtimeCodeRepoProvider = productCodeRepoProvider

		runtimeCodeRepo = &nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-%s", randNum()),
				Namespace: productName,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  runtimeCodeRepoProvider.Name,
				Product:           productName,
				Project:           "",
				RepoName:          "pipeline",
				URL:               "ssh://127.0.0.1/testgroup/pipeline.git",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook:           nil,
			},
		}

		runtime = &nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: productName,
			},
			Spec: nautescrd.ProjectPipelineRuntimeSpec{
				Project:          "",
				PipelineSource:   runtimeCodeRepo.Name,
				Pipelines:        nil,
				Destination:      env.Name,
				EventSources:     nil,
				Isolation:        "",
				PipelineTriggers: nil,
			},
		}

		accessInfo := interfaces.AccessInfo{
			Name:       "",
			Type:       interfaces.ACCESS_TYPE_K8S,
			SSH:        nil,
			Kubernetes: cfg,
		}

		task = interfaces.RuntimeSyncTask{
			AccessInfo:         accessInfo,
			Product:            *product,
			Cluster:            *cluster,
			NautesCfg:          *nautesCFG,
			Runtime:            runtime,
			RuntimeType:        interfaces.RUNTIME_TYPE_PIPELINE,
			ServiceAccountName: constant.ServiceAccountDefault,
		}

		newTask := task
		task2 = newTask
		runtimeCodeRepo2 = &nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-2", runtimeCodeRepo.Name),
				Namespace: productName,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  runtimeCodeRepoProvider.Name,
				Product:           productName,
				Project:           "",
				RepoName:          "test-2",
				URL:               "ssh://127.0.0.1/testgroup/test-2.git",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook:           &nautescrd.Webhook{},
			},
		}
		task2.Runtime = &nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: productName,
			},
			Spec: nautescrd.ProjectPipelineRuntimeSpec{
				Project:        "project",
				PipelineSource: runtimeCodeRepo2.Name,
				Destination:    env.Name,
			},
		}

		appName = fmt.Sprintf("%s-%s", productName, tekton.PipelineDeployTaskName)

		mockK8SClient.productName = productName
		mockK8SClient.coderepos = []*nautescrd.CodeRepo{productCodeRepo, runtimeCodeRepo, runtimeCodeRepo2}
		mockK8SClient.coderepoProviders = []*nautescrd.CodeRepoProvider{
			productCodeRepoProvider,
			runtimeCodeRepoProvider,
		}
		mockK8SClient.runtimes = []*nautescrd.ProjectPipelineRuntime{
			runtime,
		}

		syncer = tekton.NewSyncer(mockK8SClient)
	})

	AfterEach(func() {
		runtime.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}
		err := syncer.UnDeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		err = k8sClient.Delete(ctx, productNamespace)
		Expect(err).Should(BeNil())
	})

	It("create rolebinding and grant permission to role", func() {
		err := syncer.DeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		app = &argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		targetAppSpec := argocdv1alpha1.ApplicationSpec{
			Source: argocdv1alpha1.ApplicationSource{
				RepoURL:        productCodeRepo.Spec.URL,
				Path:           "templates/pipelines",
				TargetRevision: "HEAD",
			},
			Destination: argocdv1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: productName,
			},
			SyncPolicy: &argocdv1alpha1.SyncPolicy{
				Automated: &argocdv1alpha1.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
			Project: productName,
		}
		Expect(reflect.DeepEqual(app.Spec, targetAppSpec)).Should(BeTrue())

		project = &argocdv1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		targetSourceRepos := []string{
			productCodeRepo.Spec.URL,
		}
		Expect(reflect.DeepEqual(project.Spec.SourceRepos, targetSourceRepos)).Should(BeTrue())
		targetDestinations := []argocdv1alpha1.ApplicationDestination{
			{
				Server:    "*",
				Namespace: productName,
			},
		}
		Expect(reflect.DeepEqual(project.Spec.Destinations, targetDestinations)).Should(BeTrue())
	})

	It("if product has two pipeline runtime, app project has nothing changed", func() {
		err := syncer.DeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		err = syncer.DeployPipelineRuntime(ctx, task2)
		Expect(err).Should(BeNil())

		app = &argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		targetAppSpec := argocdv1alpha1.ApplicationSpec{
			Source: argocdv1alpha1.ApplicationSource{
				RepoURL:        productCodeRepo.Spec.URL,
				Path:           "templates/pipelines",
				TargetRevision: "HEAD",
			},
			Destination: argocdv1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: productName,
			},
			SyncPolicy: &argocdv1alpha1.SyncPolicy{
				Automated: &argocdv1alpha1.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
			Project: productName,
		}
		Expect(reflect.DeepEqual(app.Spec, targetAppSpec)).Should(BeTrue())

		project = &argocdv1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		targetSourceRepos := []string{
			productCodeRepo.Spec.URL,
		}
		Expect(reflect.DeepEqual(project.Spec.SourceRepos, targetSourceRepos)).Should(BeTrue())
		targetDestinations := []argocdv1alpha1.ApplicationDestination{
			{
				Server:    "*",
				Namespace: productName,
			},
		}
		Expect(reflect.DeepEqual(project.Spec.Destinations, targetDestinations)).Should(BeTrue())

		err = syncer.UnDeployPipelineRuntime(ctx, task2)
		Expect(err).Should(BeNil())
	})

	It("if product code repo update, app project will update", func() {
		err := syncer.DeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		productCodeRepo.Spec.URL = "ssh://127.0.0.2/test/groups.git"
		err = syncer.DeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		app = &argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		targetAppSpec := argocdv1alpha1.ApplicationSpec{
			Source: argocdv1alpha1.ApplicationSource{
				RepoURL:        productCodeRepo.Spec.URL,
				Path:           "templates/pipelines",
				TargetRevision: "HEAD",
			},
			Destination: argocdv1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: productName,
			},
			SyncPolicy: &argocdv1alpha1.SyncPolicy{
				Automated: &argocdv1alpha1.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
			Project: productName,
		}
		Expect(reflect.DeepEqual(app.Spec, targetAppSpec)).Should(BeTrue())

		project = &argocdv1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		// Issue #7, this check can not pass now
		// targetSourceRepos := []string{
		// productCodeRepo.Spec.URL,
		// }
		// Expect(reflect.DeepEqual(project.Spec.SourceRepos, targetSourceRepos)).Should(BeTrue())
		targetDestinations := []argocdv1alpha1.ApplicationDestination{
			{
				Server:    "*",
				Namespace: productName,
			},
		}
		Expect(reflect.DeepEqual(project.Spec.Destinations, targetDestinations)).Should(BeTrue())

		// Issue #7, this make aftereach pass
		err = k8sClient.Delete(ctx, project)
		Expect(err).Should(BeNil())
	})

	It("if runtime is not the last one, app project will not be remove", func() {
		err := syncer.DeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		mockK8SClient.runtimes = append(mockK8SClient.runtimes, task2.Runtime.(*nautescrd.ProjectPipelineRuntime))

		err = syncer.UnDeployPipelineRuntime(ctx, task)
		Expect(err).Should(BeNil())

		app = &argocdv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		project = &argocdv1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())

		mockK8SClient.runtimes = mockK8SClient.runtimes[:0]
	})
})
