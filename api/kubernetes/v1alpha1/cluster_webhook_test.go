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
	"time"

	. "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterConfig "github.com/nautes-labs/nautes/pkg/config/cluster"
)

var _ = Describe("cluster webhook", func() {
	var runtime *DeploymentRuntime
	var env *Environment
	var cluster *Cluster
	var ctx context.Context
	var productName string
	var ns *corev1.Namespace
	var componentName = "hnc"
	var componentNamespace = "hnc"

	BeforeEach(func() {
		ctx = context.Background()
		productName = fmt.Sprintf("product-%s", randNum())

		cluster = &Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", randNum()),
				Namespace: nautesNamespaceName,
			},
			Spec: ClusterSpec{
				ApiServer:   "",
				ClusterType: CLUSTER_TYPE_PHYSICAL,
				ClusterKind: CLUSTER_KIND_KUBERNETES,
				Usage:       CLUSTER_USAGE_WORKER,
				HostCluster: "",
				WorkerType:  ClusterWorkTypeDeployment,
				ComponentsList: ComponentsList{
					MultiTenant: &Component{
						Name:      componentName,
						Namespace: componentNamespace,
					},
					Deployment: &Component{
						Name:      "Placeholder",
						Namespace: "Placeholder_Namespace",
					},
				},
				ReservedNamespacesAllowedProducts: map[string][]string{
					componentNamespace: {productName},
				},
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
				Name:      fmt.Sprintf("env-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: EnvironmentSpec{
				Product: productName,
				Cluster: cluster.Name,
				EnvType: "test",
			},
		}

		runtime = &DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: ns.Name,
			},
			Spec: DeploymentRuntimeSpec{
				Product:     productName,
				ProjectsRef: []string{},
				ManifestSource: ManifestSource{
					CodeRepo:       "code",
					TargetRevision: "HEAD",
					Path:           "/basepoint",
				},
				Destination: DeploymentRuntimesDestination{
					Environment: env.Name,
					Namespaces:  []string{},
				},
			},
		}

		err = clusterConfig.SetClusterValidateConfig()
		Expect(err).Should(BeNil())

	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, runtime)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = k8sClient.Delete(ctx, env)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = k8sClient.Delete(ctx, ns)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = k8sClient.Delete(ctx, cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())

		err = clusterConfig.DeleteValidateConfig()
		Expect(err).Should(BeNil())
	})

	It("if cluster is a virtual cluster without host cluster, create will failed", func() {
		cluster.Spec.ClusterType = CLUSTER_TYPE_VIRTUAL
		err := cluster.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("if cluster does not has runtime, delete cluster will successed", func() {
		err := k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		err = cluster.ValidateDelete()
		Expect(err).Should(BeNil())
	})

	It("if cluster has runtime, delete cluster will failed", func() {
		err := k8sClient.Create(ctx, runtime)
		Expect(err).Should(BeNil())

		runtime.Status.Cluster = cluster.Name
		err = k8sClient.Status().Update(ctx, runtime)
		Expect(err).Should(BeNil())

		isTimeout := true
		for i := 0; i < 3; i++ {
			tmpRuntime := &DeploymentRuntime{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(runtime), tmpRuntime)
			Expect(err).Should(BeNil())
			if tmpRuntime.Status.Cluster == cluster.Name {
				isTimeout = false
				break
			}
			time.Sleep(time.Second)
		}
		Expect(isTimeout).Should(BeFalse())

		err = cluster.ValidateDelete()
		Expect(err).ShouldNot(BeNil())
	})

	It("if reserved namespace not in component list, create will failed", func() {
		cluster.Spec.ReservedNamespacesAllowedProducts = map[string][]string{
			"fake": {},
		}
		err := cluster.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("when the cluster does not have a set of components, component check fails", func() {
		err := cluster.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})

	It("component required value not filled in verification failed", func() {
		cluster = &Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", randNum()),
				Namespace: nautesNamespaceName,
			},
			Spec: ClusterSpec{
				ApiServer:   "https://127.0.0.1:31668",
				ClusterType: CLUSTER_TYPE_PHYSICAL,
				Usage:       CLUSTER_USAGE_HOST,
				ComponentsList: ComponentsList{
					Gateway: &Component{
						Name:      "traefik",
						Namespace: "traefik",
					},
					OauthProxy: &Component{
						Name:      "oauth2-proxy",
						Namespace: "oauth2-proxy",
					},
				},
			},
		}

		err := cluster.ValidateCreate()
		Expect(err).ShouldNot(BeNil())
	})
})
