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

package task_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	component "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Deploy runtime deployer", func() {
	var seed string
	var ctx context.Context
	var cluster *v1alpha1.Cluster
	var repoProvider *v1alpha1.CodeRepoProvider
	var codeRepo *v1alpha1.CodeRepo
	var runtime *v1alpha1.DeploymentRuntime
	var product *v1alpha1.Product
	var productIDs []string
	var namespaceNames []string
	var runtimeNames []string
	BeforeEach(func() {
		seed = RandNum()
		ctx = context.Background()
		result = &deployResult{}

		productIDs = GenerateNames(fmt.Sprintf("product-%s-%%d", seed), 5)
		namespaceNames = GenerateNames(fmt.Sprintf("ns-%s-%%d", seed), 5)
		runtimeNames = GenerateNames(fmt.Sprintf("runtime-%s-%%d", seed), 5)

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
				WorkerType:  v1alpha1.ClusterWorkTypeDeployment,
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
				},
			},
		}
		err := k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())

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

		runtime = &v1alpha1.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeNames[0],
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.DeploymentRuntimeSpec{
				Product: productIDs[0],
				ManifestSource: v1alpha1.ManifestSource{
					CodeRepo:       codeRepo.Name,
					TargetRevision: "main",
					Path:           "path",
				},
				Destination: v1alpha1.DeploymentRuntimesDestination{
					Environment: "",
					Namespaces: []string{
						namespaceNames[0],
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
		err := k8sClient.Delete(ctx, cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      syncer.ConfigMapNameClustersUsageCache,
				Namespace: nautesNamespace,
			},
		}
		err = k8sClient.Delete(ctx, cm)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
	})

	It("can deploy runtime", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destCache := &syncer.DeploymentRuntimeSyncHistory{
			Cluster: "",
			Spaces:  utils.NewStringSet(runtime.GetNamespaces()...),
			Account: &component.MachineAccount{
				Name:    runtime.Name,
				Product: runtime.Spec.Product,
				Spaces:  runtime.GetNamespaces(),
			},
			Permissions: []component.SecretInfo{
				{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: repoProvider.Spec.ProviderType,
						ID:           runtime.Spec.ManifestSource.CodeRepo,
						User:         "default",
						Permission:   component.CodeRepoPermissionReadOnly,
					},
				},
			},
		}

		ok := reflect.DeepEqual(destCache, cache)
		Expect(ok).Should(BeTrue())

		destResult := &deployResult{
			product: &deployResultProduct{
				name: productIDs[0],
				spaces: []deployResultSpace{
					{
						name:     runtime.Spec.Destination.Namespaces[0],
						accounts: []string{runtime.Name},
					},
				},
				accounts: []string{runtime.Name},
			},
			deployment: &deployResultDeployment{
				product: deployResultDeploymentProduct{
					name:     runtime.GetProduct(),
					accounts: []string{product.Spec.Name},
				},
				apps: []component.Application{
					{
						ResourceMetaData: component.ResourceMetaData{
							Product: runtime.GetProduct(),
							Name:    runtime.Name,
						},
						Git: &component.ApplicationGit{
							URL:      codeRepo.Spec.URL,
							Revision: runtime.Spec.ManifestSource.TargetRevision,
							Path:     runtime.Spec.ManifestSource.Path,
							CodeRepo: codeRepo.Name,
						},
						Destinations: []component.Space{
							{
								ResourceMetaData: component.ResourceMetaData{
									Product: runtime.GetProduct(),
									Name:    runtime.Spec.Destination.Namespaces[0],
								},
								SpaceType: component.SpaceTypeKubernetes,
								Kubernetes: &component.SpaceKubernetes{
									Namespace: runtime.Spec.Destination.Namespaces[0],
								},
							},
						},
					},
				},
			},
		}
		ok = reflect.DeepEqual(destResult, result)
		Expect(ok).Should(BeTrue())
	})

	It("will update space when namespace changed", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		runtime.Spec.Destination.Namespaces = []string{namespaceNames[1]}
		task, err = testSyncer.NewTask(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())

		rawCache, err = task.Run(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product: &deployResultProduct{
				name: productIDs[0],
				spaces: []deployResultSpace{
					{
						name:     runtime.Spec.Destination.Namespaces[0],
						accounts: []string{runtime.Name},
					},
				},
				accounts: []string{runtime.Name},
			},
			deployment: &deployResultDeployment{
				product: deployResultDeploymentProduct{
					name:     runtime.GetProduct(),
					accounts: []string{product.Spec.Name},
				},
				apps: []component.Application{
					{
						ResourceMetaData: component.ResourceMetaData{
							Product: runtime.GetProduct(),
							Name:    runtime.Name,
						},
						Git: &component.ApplicationGit{
							URL:      codeRepo.Spec.URL,
							Revision: runtime.Spec.ManifestSource.TargetRevision,
							Path:     runtime.Spec.ManifestSource.Path,
							CodeRepo: codeRepo.Name,
						},
						Destinations: []component.Space{
							{
								ResourceMetaData: component.ResourceMetaData{
									Product: runtime.GetProduct(),
									Name:    runtime.Spec.Destination.Namespaces[0],
								},
								SpaceType: component.SpaceTypeKubernetes,
								Kubernetes: &component.SpaceKubernetes{
									Namespace: runtime.Spec.Destination.Namespaces[0],
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

	It("can delete runtime", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		task, err = testSyncer.NewTask(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())

		rawCache, err = task.Delete(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product:    nil,
			deployment: nil,
		}
		ok := reflect.DeepEqual(destResult, result)
		Expect(ok).Should(BeTrue())
	})

	It("if space is used by other runtime, it will not be delete", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopy()
		runtime2.Name = runtimeNames[1]
		db.runtimes[runtime2.Name] = runtime2
		task2, err := testSyncer.NewTask(ctx, runtime2, nil)
		Expect(err).Should(BeNil())

		_, err = task2.Run(ctx)
		Expect(err).Should(BeNil())

		task, err = testSyncer.NewTask(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())
		rawCache, err = task.Delete(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product: &deployResultProduct{
				name: productIDs[0],
				spaces: []deployResultSpace{
					{
						name:     runtime.Spec.Destination.Namespaces[0],
						accounts: []string{runtime2.Name},
					},
				},
				accounts: []string{runtime2.Name},
			},
			deployment: &deployResultDeployment{
				product: deployResultDeploymentProduct{
					name:     runtime2.GetProduct(),
					accounts: []string{product.Spec.Name},
				},
				apps: []component.Application{
					{
						ResourceMetaData: component.ResourceMetaData{
							Product: runtime2.GetProduct(),
							Name:    runtime2.Name,
						},
						Git: &component.ApplicationGit{
							URL:      codeRepo.Spec.URL,
							Revision: runtime2.Spec.ManifestSource.TargetRevision,
							Path:     runtime2.Spec.ManifestSource.Path,
							CodeRepo: codeRepo.Name,
						},
						Destinations: []component.Space{
							{
								ResourceMetaData: component.ResourceMetaData{
									Product: runtime2.GetProduct(),
									Name:    runtime2.Spec.Destination.Namespaces[0],
								},
								SpaceType: component.SpaceTypeKubernetes,
								Kubernetes: &component.SpaceKubernetes{
									Namespace: runtime2.Spec.Destination.Namespaces[0],
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

	It("can delete runtime, when runtime doesn't deployed", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		cache, err := task.Delete(ctx)
		Expect(err).Should(BeNil())
		_ = cache
	})

	It("can specify account", func() {
		accountName := fmt.Sprintf("account-%s", seed)
		runtime.Spec.Account = accountName
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResult{
			product: &deployResultProduct{
				name: productIDs[0],
				spaces: []deployResultSpace{
					{
						name:     runtime.Spec.Destination.Namespaces[0],
						accounts: []string{accountName},
					},
				},
				accounts: []string{accountName},
			},
			deployment: &deployResultDeployment{
				product: deployResultDeploymentProduct{
					name:     runtime.GetProduct(),
					accounts: []string{product.Spec.Name},
				},
				apps: []component.Application{
					{
						ResourceMetaData: component.ResourceMetaData{
							Product: runtime.GetProduct(),
							Name:    runtime.Name,
						},
						Git: &component.ApplicationGit{
							URL:      codeRepo.Spec.URL,
							Revision: runtime.Spec.ManifestSource.TargetRevision,
							Path:     runtime.Spec.ManifestSource.Path,
							CodeRepo: codeRepo.Name,
						},
						Destinations: []component.Space{
							{
								ResourceMetaData: component.ResourceMetaData{
									Product: runtime.GetProduct(),
									Name:    runtime.Spec.Destination.Namespaces[0],
								},
								SpaceType: component.SpaceTypeKubernetes,
								Kubernetes: &component.SpaceKubernetes{
									Namespace: runtime.Spec.Destination.Namespaces[0],
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

	It("if account is used by other runtime, account will not be deleted", func() {
		accountName := fmt.Sprintf("account-%s", seed)
		runtime.Spec.Account = accountName
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		runtime2 := runtime.DeepCopy()
		runtime2.Name = runtimeNames[1]
		runtime2.Spec.Destination.Namespaces = []string{namespaceNames[3]}
		db.runtimes[runtime2.Name] = runtime2
		task2, err := testSyncer.NewTask(ctx, runtime2, nil)
		Expect(err).Should(BeNil())

		_, err = task2.Run(ctx)
		Expect(err).Should(BeNil())

		task, err = testSyncer.NewTask(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())
		rawCache, err = task.Delete(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destResult := &deployResultProduct{
			name: productIDs[0],
			spaces: []deployResultSpace{
				{
					name:     namespaceNames[3],
					accounts: []string{accountName},
				},
			},
			accounts: []string{accountName},
		}
		ok := reflect.DeepEqual(destResult, result.product)
		Expect(ok).Should(BeTrue())
	})

	It("if account is changed, account will update", func() {
		task, err := testSyncer.NewTask(ctx, runtime, nil)
		Expect(err).Should(BeNil())

		rawCache, err := task.Run(ctx)
		Expect(err).Should(BeNil())

		accountName := fmt.Sprintf("account-%s", seed)
		runtime.Spec.Account = accountName
		task, err = testSyncer.NewTask(ctx, runtime, rawCache)
		Expect(err).Should(BeNil())
		rawCache, err = task.Run(ctx)
		Expect(err).Should(BeNil())

		cache := &syncer.DeploymentRuntimeSyncHistory{}
		err = json.Unmarshal(rawCache.Raw, cache)
		Expect(err).Should(BeNil())

		destCache := &syncer.DeploymentRuntimeSyncHistory{
			Cluster: "",
			Spaces:  utils.NewStringSet(runtime.GetNamespaces()...),
			Account: &component.MachineAccount{
				Name:    accountName,
				Product: productIDs[0],
				Spaces:  runtime.GetNamespaces(),
			},
			Permissions: []component.SecretInfo{
				{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: repoProvider.Spec.ProviderType,
						ID:           runtime.Spec.ManifestSource.CodeRepo,
						User:         "default",
						Permission:   component.CodeRepoPermissionReadOnly,
					},
				},
			},
		}

		ok := reflect.DeepEqual(destCache, cache)
		Expect(ok).Should(BeTrue())

		destResult := &deployResultProduct{
			name: productIDs[0],
			spaces: []deployResultSpace{
				{
					name:     namespaceNames[0],
					accounts: []string{accountName},
				},
			},
			accounts: []string{accountName},
		}
		ok = reflect.DeepEqual(destResult, result.product)
		Expect(ok).Should(BeTrue())
	})
})
