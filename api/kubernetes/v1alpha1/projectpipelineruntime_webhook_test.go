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

package v1alpha1_test

import (
	"context"
	"fmt"

	. "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("cluster webhook", func() {
	var runtime *ProjectPipelineRuntime
	var env *Environment
	var cluster *Cluster
	var ctx context.Context
	var productName string
	var projectName string
	var ns *corev1.Namespace
	var source *CodeRepo
	var eventRepo *CodeRepo
	var codeRepoBinding *CodeRepoBinding
	var cleanBox []client.Object
	var cleanBoxNamespace []client.Object
	BeforeEach(func() {
		ctx = context.Background()
		seed := randNum()
		cleanBox = []client.Object{}
		cleanBoxNamespace = []client.Object{}
		productName = fmt.Sprintf("product-%s", seed)
		projectName = fmt.Sprintf("project-%s", seed)

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).Should(BeNil())
		cleanBoxNamespace = append(cleanBoxNamespace, ns)

		cluster = &Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", seed),
				Namespace: nautesNamespaceName,
			},
			Spec: ClusterSpec{
				ApiServer:     "https://127.0.0.1:6443",
				ClusterType:   CLUSTER_TYPE_PHYSICAL,
				ClusterKind:   CLUSTER_KIND_KUBERNETES,
				Usage:         CLUSTER_USAGE_WORKER,
				HostCluster:   "",
				PrimaryDomain: "",
				WorkerType:    ClusterWorkTypePipeline,
				ComponentsList: ComponentsList{
					CertManagement:      &Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &Component{Name: "argo-events", Namespace: "argo-events"},
					Gateway:             &Component{Name: "traefik", Namespace: "traefik"},
					MultiTenant:         &Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}
		cleanBox = append(cleanBox, cluster)

		env = &Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("env-%s", seed),
				Namespace: ns.Name,
			},
			Spec: EnvironmentSpec{
				Product: productName,
				Cluster: cluster.Name,
				EnvType: "test",
			},
		}
		cleanBox = append(cleanBox, env)

		source = &CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-%s", seed),
				Namespace: ns.Name,
			},
			Spec: CodeRepoSpec{
				CodeRepoProvider:  "",
				Product:           productName,
				Project:           projectName,
				RepoName:          "",
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook:           nil,
			},
		}
		cleanBox = append(cleanBox, source)

		eventRepo = source.DeepCopyObject().(*CodeRepo)
		cleanBox = append(cleanBox, eventRepo)

		codeRepoBinding = &CodeRepoBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("binding-%s", seed),
				Namespace: ns.Name,
			},
			Spec: CodeRepoBindingSpec{
				CodeRepo:    eventRepo.Name,
				Product:     productName,
				Projects:    []string{eventRepo.Spec.Project},
				Permissions: "",
			},
		}
		cleanBox = append(cleanBox, codeRepoBinding)

		runtimeName := fmt.Sprintf("runtime-%s", seed)
		runtime = &ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeName,
				Namespace: ns.Name,
			},
			Spec: ProjectPipelineRuntimeSpec{
				Project:        projectName,
				PipelineSource: source.Name,
				Pipelines:      []Pipeline{},
				Destination: ProjectPipelineDestination{
					Environment: env.Name,
					Namespace:   "",
				},
				EventSources: []EventSource{
					{
						Name: "evname",
						Gitlab: &Gitlab{
							RepoName: eventRepo.Name,
							Revision: "main",
							Events:   []string{},
						},
					},
				},
				Isolation:        "",
				PipelineTriggers: []PipelineTrigger{},
			},
		}
		cleanBox = append(cleanBox, runtime)

		err = k8sClient.Create(ctx, env)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())

		logger.V(1).Info("=====Case start=====")
		logger.V(1).Info("product", "Name", productName)
		logger.V(1).Info("project", "Name", projectName)
		logger.V(1).Info("souce repo", "Name", source.Name, "Project", source.Spec.Project)
		logger.V(1).Info("runtime", "Name", runtime.Name, "Project", runtime.Spec.Project)
	})

	AfterEach(func() {
		for _, obj := range cleanBox {
			err := k8sClient.Delete(ctx, obj)
			Expect(client.IgnoreNotFound(err)).Should(BeNil())
			err = waitForDelete(obj)
			Expect(err).Should(BeNil())
		}

		for _, obj := range cleanBoxNamespace {
			err := k8sClient.Delete(ctx, obj)
			Expect(client.IgnoreNotFound(err)).Should(BeNil())
		}
	})

	It("if source and runtime in the same project, create will success", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources = nil
		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if source and runtime not in the same project, it need coderepo binding", func() {
		source.Spec.Project = fmt.Sprintf("%s-2", source.Spec.Project)
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources = nil
		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())

		codeRepoBinding.Spec.CodeRepo = source.Name
		codeRepoBinding.Spec.Projects = []string{runtime.Spec.Project}
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if product is same, and coderepobinding's projects is nil, runtime permission check should pass", func() {
		source.Spec.Project = fmt.Sprintf("%s-2", source.Spec.Project)
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		codeRepoBinding.Spec.CodeRepo = source.Name
		codeRepoBinding.Spec.Projects = nil
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if event source repo and runtime not in the same project, it need coderepo binding", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		eventRepo.Name = fmt.Sprintf("%s-2", eventRepo.Name)
		eventRepo.Spec.Project = fmt.Sprintf("%s-2", eventRepo.Spec.Project)
		err = k8sClient.Create(ctx, eventRepo)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, eventRepo.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.EventSources[0].Gitlab.RepoName = eventRepo.Name
		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())

		codeRepoBinding.Spec.CodeRepo = eventRepo.Name
		codeRepoBinding.Spec.Projects = []string{runtime.Spec.Project}
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, eventRepo.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("when cluster is not a worker cluster, create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		_, err = controllerutil.CreateOrPatch(ctx, k8sClient, cluster, func() error {
			cluster.Spec.Usage = CLUSTER_USAGE_HOST
			return nil
		})
		Expect(err).Should(BeNil())
		err = waitForCacheUpdateCluster(k8sClient, cluster)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when cluster is not a deployment cluster, create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		_, err = controllerutil.CreateOrPatch(ctx, k8sClient, cluster, func() error {
			cluster.Spec.WorkerType = ClusterWorkTypeDeployment
			return nil
		})
		Expect(err).Should(BeNil())
		err = waitForCacheUpdateCluster(k8sClient, cluster)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when namespace is used by other runtime, create will failed", func() {
		var err error
		runtime02 := runtime.DeepCopy()
		runtime02.Name = fmt.Sprintf("%s-02", runtime02.Name)
		runtime02.Spec.Destination.Namespace = runtime.Name

		err = k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, runtime02)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, runtime02)

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when additional resource code repo has not perrmission, create will failed", func() {
		additionalRepo := source.DeepCopy()
		additionalRepo.Name = fmt.Sprintf("%s-addional-repo", source.Name)
		additionalRepo.Spec.Project = "other"

		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, additionalRepo)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, additionalRepo.Name)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, additionalRepo)

		runtime.Spec.AdditionalResources = &ProjectPipelineRuntimeAdditionalResources{
			Git: &ProjectPipelineRuntimeAdditionalResourcesGit{
				CodeRepo: additionalRepo.Name,
				Revision: "main",
				Path:     "/deploy",
			},
		}

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())

		codeRepoBinding.Spec.CodeRepo = additionalRepo.Name
		err = k8sClient.Create(ctx, codeRepoBinding)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateBinding(1, productName, additionalRepo.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})
})
