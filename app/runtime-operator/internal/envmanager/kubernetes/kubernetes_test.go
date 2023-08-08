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

package kubernetes_test

import (
	"fmt"
	"os"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	envmgr "github.com/nautes-labs/nautes/app/runtime-operator/internal/envmanager/kubernetes"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/constant"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

const (
	_HNC_CONFIG_NAME = "hierarchy"
)

var _ = Describe("EnvManager", func() {
	var err error
	var mgr envmgr.Syncer
	var productName string
	var groupName string
	var runtimeName string
	var artifactRepoName string
	var secretDBName string
	var secretKey string

	var accessInfo *interfaces.AccessInfo
	var baseRuntime *nautescrd.DeploymentRuntime
	var productNamespaceIsTerminating bool
	var artifactRepos []nautescrd.ArtifactRepo

	var productCodeRepo *nautescrd.CodeRepo
	var task interfaces.RuntimeSyncTask

	var destNamespace string
	BeforeEach(func() {
		seed := randNum()
		productName = fmt.Sprintf("test-project-%s", seed)
		groupName = fmt.Sprintf("group-%s", seed)
		runtimeName = fmt.Sprintf("runtime-%s", seed)
		artifactRepoName = fmt.Sprintf("artifact-repo-%s", seed)
		secretDBName = "repo"
		productCodeRepoName := fmt.Sprintf("repo-%s", seed)
		mockcli = newMockClient()
		destNamespace = fmt.Sprintf("dest-namespace-%s", seed)
		envName := fmt.Sprintf("environment-%s", seed)

		accessInfo, err = mgr.GetAccessInfo(ctx, nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aa",
				Namespace: "bb",
			},
		})
		Expect(err).Should(BeNil())

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}

		productCodeRepo = &nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productCodeRepoName,
				Namespace: "nautes",
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider: "test",
				Product:          productName,
				Project:          "",
				RepoName:         artifactRepoName,
				URL:              "ssh://127.0.0.1/test/default.project.git",
			},
		}

		env := &nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      envName,
				Namespace: productName,
			},
		}

		baseRuntime = &nautescrd.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runtimeName,
				Namespace: productName,
			},
			Spec: nautescrd.DeploymentRuntimeSpec{
				Product:        productName,
				ManifestSource: nautescrd.ManifestSource{},
				Destination: nautescrd.DeploymentRuntimesDestination{
					Environment: envName,
					Namespaces:  []string{destNamespace},
				},
			},
		}

		task = interfaces.RuntimeSyncTask{
			AccessInfo: *accessInfo,
			Product: nautescrd.Product{
				ObjectMeta: metav1.ObjectMeta{
					Name:      productName,
					Namespace: nautesCFG.Nautes.Namespace,
				},
				Spec: nautescrd.ProductSpec{
					Name: groupName,
				},
			},
			NautesCfg:          *nautesCFG,
			Runtime:            baseRuntime,
			RuntimeType:        interfaces.RUNTIME_TYPE_DEPLOYMENT,
			ServiceAccountName: constant.ServiceAccountDefault,
		}

		artifactProvier := &nautescrd.ArtifactRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stander",
				Namespace: nautesCFG.Nautes.Namespace,
			},
			Spec: nautescrd.ArtifactRepoProviderSpec{
				URL:          "",
				APIServer:    "",
				ProviderType: "harbor",
			},
		}

		artifactRepos = []nautescrd.ArtifactRepo{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      artifactRepoName,
					Namespace: productName,
				},
				Spec: nautescrd.ArtifactRepoSpec{
					ArtifactRepoProvider: artifactProvier.Name,
					Product:              productName,
					Projects:             []string{},
					RepoName:             artifactRepoName,
					RepoType:             "",
					PackageType:          "",
				},
			},
		}

		mockcli.Create(ctx, namespace)
		mockcli.Create(ctx, productCodeRepo)
		mockcli.Create(ctx, artifactProvier)
		mockcli.Create(ctx, baseRuntime)
		mockcli.Create(ctx, env)

		for _, repo := range artifactRepos {
			mockcli.Create(ctx, &repo)
		}
		productNamespaceIsTerminating = true

		secretKey = fmt.Sprintf("%s/%s/%s/default/readonly", artifactProvier.Name, artifactProvier.Spec.ProviderType, artifactRepos[0].Name)
		err = os.Setenv("TEST_SECRET_DB", secretDBName)
		Expect(err).Should(BeNil())
		err = os.Setenv("TEST_SECRET_KEY", secretKey)
		Expect(err).Should(BeNil())

		mgr = envmgr.Syncer{mockcli}
	})

	AfterEach(func() {
		err = mockcli.Delete(ctx, baseRuntime)
		Expect(err).Should(BeNil())

		err = mgr.Remove(ctx, task)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{}
		key := types.NamespacedName{
			Name: productName,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok := isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).ShouldNot(Equal(productNamespaceIsTerminating))

		for _, namespace := range baseRuntime.Spec.Destination.Namespaces {
			key = types.NamespacedName{
				Name: namespace,
			}
			err = k8sClient.Get(ctx, key, ns)
			Expect(err).Should(BeNil())
			ok = isNotTerminatingAndBelongsToProduct(ns, productName)
			Expect(ok).Should(BeFalse())
		}

	})

	It("init a new env", func() {
		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())
		ns := &corev1.Namespace{}
		key := types.NamespacedName{
			Name: productName,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok := isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeTrue())

		key = types.NamespacedName{
			Name: destNamespace,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok = isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeTrue())
	})

	It("update exited env", func() {
		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())
		ns := &corev1.Namespace{}
		key := types.NamespacedName{
			Name: productName,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok := isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeTrue())

		key = types.NamespacedName{
			Name: destNamespace,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok = isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeTrue())
	})

	It("if namespace parent is not product, change it back to product", func() {
		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		key := types.NamespacedName{
			Namespace: destNamespace,
			Name:      _HNC_CONFIG_NAME,
		}
		hnc := &hncv1alpha2.HierarchyConfiguration{}
		err = k8sClient.Get(ctx, key, hnc)
		Expect(err).Should(BeNil())

		hnc.Spec.Parent = fmt.Sprintf("%s-2", productName)
		err = k8sClient.Update(ctx, hnc)
		Expect(err).Should(BeNil())

		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, key, hnc)
		Expect(err).Should(BeNil())
		Expect(hnc.Spec.Parent).Should(Equal(productName))

	})

	It("do not delete product namespace if deployment has other runtime", func() {
		runtime2 := baseRuntime.DeepCopy()
		runtime2.Name = fmt.Sprintf("%s-2", baseRuntime.Name)
		runtime2.Spec.Destination.Namespaces = []string{fmt.Sprintf("%s-2", baseRuntime.Spec.Destination.Namespaces[0])}
		err = mockcli.Create(ctx, runtime2)
		Expect(err).Should(BeNil())

		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		productNamespaceIsTerminating = false

		err := mockcli.Delete(ctx, baseRuntime)
		Expect(err).Should(BeNil())
		err = mgr.Remove(ctx, task)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{}
		key := types.NamespacedName{
			Name: productName,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok := isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeTrue())

		key = types.NamespacedName{
			Name: destNamespace,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ok = isNotTerminatingAndBelongsToProduct(ns, productName)
		Expect(ok).Should(BeFalse())
	})

	It("if namespace is not belongs to runtime, it should not be delete", func() {
		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{}
		key := types.NamespacedName{
			Name: destNamespace,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		ns.Labels = map[string]string{}
		err = k8sClient.Update(ctx, ns)
		Expect(err).Should(BeNil())

		err = mgr.Remove(ctx, task)
		Expect(err).Should(BeNil())

		ns = &corev1.Namespace{}
		key = types.NamespacedName{
			Name: destNamespace,
		}
		err = k8sClient.Get(ctx, key, ns)
		Expect(err).Should(BeNil())
		Expect(ns.DeletionTimestamp.IsZero()).Should(BeTrue())

		err = k8sClient.Delete(ctx, ns)
		Expect(err).Should(BeNil())
	})

	It("if destination has many namesapces, it will create many namespaces", func() {
		baseRuntime.Spec.Destination.Namespaces = append(baseRuntime.Spec.Destination.Namespaces, fmt.Sprintf("%s-2", destNamespace))
		err = mockcli.Update(ctx, baseRuntime)
		Expect(err).Should(BeNil())

		task.Runtime = baseRuntime
		_, err = mgr.Sync(ctx, task)
		Expect(err).Should(BeNil())

		nsList := &corev1.NamespaceList{}
		err = k8sClient.List(ctx, nsList, client.MatchingLabels(task.GetLabel()))
		Expect(err).Should(BeNil())
		Expect(len(nsList.Items)).Should(Equal(3))
	})
})
