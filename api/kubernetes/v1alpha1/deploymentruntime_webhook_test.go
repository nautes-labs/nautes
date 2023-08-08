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
	"k8s.io/apimachinery/pkg/fields"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("cluster webhook", func() {
	var runtime *DeploymentRuntime
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
	var useNamespace string
	BeforeEach(func() {
		ctx = context.Background()
		cleanBox = []client.Object{}
		seed := randNum()

		productName = fmt.Sprintf("product-%s", seed)
		projectName = fmt.Sprintf("project-%s", seed)
		useNamespace = fmt.Sprintf("ns-%s", seed)
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
				WorkerType:    ClusterWorkTypeDeployment,
			},
		}

		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).Should(BeNil())

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

		eventRepo = source.DeepCopyObject().(*CodeRepo)

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

		runtime = &DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", seed),
				Namespace: ns.Name,
			},
			Spec: DeploymentRuntimeSpec{
				Product:     productName,
				ProjectsRef: []string{projectName},
				ManifestSource: ManifestSource{
					CodeRepo:       source.Name,
					TargetRevision: "HEAD",
					Path:           "/",
				},
				Destination: DeploymentRuntimesDestination{
					Environment: env.Name,
					Namespaces:  []string{useNamespace},
				},
			},
		}

		err = k8sClient.Create(ctx, env)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())

		cleanBox = append(cleanBox, env, cluster, runtime, codeRepoBinding, eventRepo, source)

		logger.V(1).Info("=====Case start=====")
		logger.V(1).Info("product", "Name", productName)
		logger.V(1).Info("project", "Name", projectName)
		logger.V(1).Info("souce repo", "Name", source.Name, "Project", source.Spec.Project)
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

	It("if project has permission to use coderepo, create will successed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("if project has no permission to use coderepo , create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime.Spec.ProjectsRef = []string{"fake"}

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when an identical runtime has already been deployed, create will failed", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Name = fmt.Sprintf("%s-2", runtime.Name)
		err = k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())
		err = waitForCacheUpdate(k8sClient, runtime)
		Expect(err).Should(BeNil())

		err = runtime2.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when an identical runtime has already been deployed, the deployed runtime should pass validate", func() {
		err := k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexFieldUpdateCodeRepo(1, source.Name)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Name = fmt.Sprintf("%s-2", runtime.Name)
		runtime2.Status.DeployHistory = &DeployHistory{
			ManifestSource: runtime2.Spec.ManifestSource,
			Destination:    runtime2.Spec.Destination,
		}
		err = k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		err = runtime2.ValidateCreate()
		Expect(err).Should(BeNil())
	})

	It("when an identical runtime has already been deployed, object meta change should be ignore ", func() {
		runtime.Spec.ProjectsRef = []string{"fake"}

		runtime2 := runtime.DeepCopyObject().(*DeploymentRuntime)
		runtime2.Finalizers = []string{"one two"}

		err := runtime.ValidateUpdate(runtime2)
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
			cluster.Spec.WorkerType = ClusterWorkTypePipeline
			return nil
		})
		Expect(err).Should(BeNil())
		err = waitForCacheUpdateCluster(k8sClient, cluster)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when namespace is a conponent's namespace, create will failed", func() {
		cluster.Spec.ComponentsList.Deployment = &Component{
			Name:      "x",
			Namespace: useNamespace,
		}
		err := k8sClient.Update(ctx, cluster)
		Expect(err).Should(BeNil())

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when namespace is used by other product, create will failed", func() {
		var err error

		product2Name := fmt.Sprintf("%s-2", runtime.Spec.Product)

		namespace2 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: product2Name,
			},
		}
		err = k8sClient.Create(ctx, namespace2)
		Expect(err).Should(BeNil())
		cleanBoxNamespace = append(cleanBoxNamespace, namespace2)

		env2 := env.DeepCopy()
		env2.ResourceVersion = ""
		env2.Namespace = product2Name
		env2.Spec.Product = product2Name
		err = k8sClient.Create(ctx, env2)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, env2)

		runtime2 := runtime.DeepCopy()
		runtime2.Namespace = tmpNamespaceName
		runtime2.Spec.Product = product2Name
		runtime2.Spec.Destination.Environment = env2.Name
		err = k8sClient.Create(ctx, runtime2)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, runtime2)

		err = runtime.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when namespace is same by cluster is not the same, create will successed", func() {
		var err error

		err = k8sClient.Create(ctx, source)
		Expect(err).Should(BeNil())
		err = waitForIndexUpdated(&CodeRepoList{}, fields.OneTermEqualSelector(SelectFieldMetaDataName, source.Name))

		Expect(err).Should(BeNil())
		product2Name := fmt.Sprintf("%s-2", runtime.Spec.Product)

		namespace2 := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: product2Name,
			},
		}
		err = k8sClient.Create(ctx, namespace2)
		Expect(err).Should(BeNil())
		cleanBoxNamespace = append(cleanBoxNamespace, namespace2)

		env2 := env.DeepCopy()
		env2.ResourceVersion = ""
		env2.Namespace = product2Name
		env2.Spec.Product = product2Name
		env2.Spec.Cluster = fmt.Sprintf("%s-2", env.Spec.Cluster)
		err = k8sClient.Create(ctx, env2)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, env2)

		runtime2 := runtime.DeepCopy()
		runtime2.Namespace = tmpNamespaceName
		runtime2.Spec.Product = product2Name
		runtime2.Spec.Destination.Environment = env2.Name
		err = k8sClient.Create(ctx, runtime2)
		Expect(err).Should(BeNil())
		cleanBox = append(cleanBox, runtime2)

		err = runtime.ValidateCreate()
		Expect(err).Should(BeNil())
	})
})
