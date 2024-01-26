// Copyright 2024 Nautes Authors
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

package middlewareruntime_test

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/transformer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/componentmock"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	performer "github.com/nautes-labs/nautes/app/runtime-operator/pkg/performer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("MiddlewareRuntimePerformer", func() {
	var (
		ctx             context.Context
		seed            string
		product         *v1alpha1.Product
		env             *v1alpha1.Environment
		cluster         *v1alpha1.Cluster
		runtime         *v1alpha1.MiddlewareRuntime
		multiTenant     component.MultiTenant
		snapshot        *componentmock.Snapshot
		initInfo        performer.PerformerInitInfos
		middlewareType  = "redis"
		implementation  = "redis-operator"
		tenantK8sClient client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		seed = testutils.RandNum()

		product = &v1alpha1.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name: "product-" + seed,
			},
		}

		cluster = &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-" + seed,
			},
			Spec: v1alpha1.ClusterSpec{
				ApiServer:      "",
				ClusterType:    v1alpha1.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:    v1alpha1.CLUSTER_KIND_KUBERNETES,
				Usage:          v1alpha1.CLUSTER_USAGE_WORKER,
				HostCluster:    "",
				PrimaryDomain:  "",
				WorkerType:     v1alpha1.ClusterWorkTypeMiddleware,
				ComponentsList: v1alpha1.ComponentsList{},
				MiddlewareProvider: &v1alpha1.MiddlewareProvider{
					Type: providerType,
					Cache: map[string][]v1alpha1.MiddlewareDeploymentImplementation{
						middlewareType: {
							{
								Type:      implementation,
								Namespace: "operators",
							},
						},
					},
				},
				ReservedNamespacesAllowedProducts: map[string][]string{},
				ProductAllowedClusterResources:    map[string][]v1alpha1.ClusterResourceInfo{},
			},
		}

		env = &v1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "env-" + seed,
			},
			Spec: v1alpha1.EnvironmentSpec{
				EnvType: v1alpha1.EnvironmentTypeCluster,
				Cluster: cluster.Name,
			},
		}
		runtime = &v1alpha1.MiddlewareRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "runtime-" + seed,
				Namespace: product.Name,
			},
			Spec: v1alpha1.MiddlewareRuntimeSpec{
				Destination: v1alpha1.MiddlewareRuntimeDestination{
					Environment: env.Name,
				},
				DataBases: []v1alpha1.Middleware{
					{
						Name:                 "middleware-" + seed,
						Type:                 middlewareType,
						Implementation:       implementation,
						Labels:               map[string]string{},
						CommonMiddlewareInfo: map[string]string{},
					},
				},
			},
		}

		snapshot = &componentmock.Snapshot{
			Product: *product,
			Clusters: map[string]v1alpha1.Cluster{
				cluster.Name: *cluster,
			},
			Environments: map[string]v1alpha1.Environment{
				env.Name: *env,
			},
			MiddlewareRuntimes: map[string]v1alpha1.MiddlewareRuntime{
				runtime.Name: *runtime,
			},
		}

		mt := &componentmock.MultiTenant{}
		mt.Init()
		multiTenant = mt

		tenantK8sClient = &mockKubernetesClient{
			SubResourceWriter: &mockKubernetesSubResourceClient{},
		}

		initInfo = performer.PerformerInitInfos{
			ComponentInitInfo: &component.ComponentInitInfo{
				ClusterConnectInfo:     component.ClusterConnectInfo{},
				ClusterName:            cluster.Name,
				NautesResourceSnapshot: snapshot,
				RuntimeName:            runtime.Name,
				NautesConfig:           configs.Config{},
				Components: &component.ComponentList{
					MultiTenant: multiTenant,
					SecretManagement: &componentmock.SecretManagement{
						AccessInfo: testDataKubeconfig,
					},
				},
				PipelinePluginManager:   nil,
				EventSourceSearchEngine: nil,
			},
			Runtime:         runtime,
			TenantK8sClient: tenantK8sClient,
		}

		transformer.AddMiddlewareTransformRule(transformer.MiddlewareTransformRule{
			ProviderType:   providerType,
			MiddlewareType: middlewareType,
			Implementation: implementation,
			Resources:      []string{},
		})
	})

	AfterEach(func() {
		transformer.ClearResourceTransformer()
		transformer.ClearMiddlewareTransformRule()
	})

	Describe("deploy", func() {
		It("can deploy middleware runtime", func() {
			performer, err := NewPerformer(initInfo)
			Expect(err).ShouldNot(HaveOccurred())

			status, err := performer.Deploy(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			wanted := &v1alpha1.MiddlewareRuntimeStatus{
				Environment:    *env,
				Cluster:        cluster,
				AccessInfoName: runtime.Spec.AccessInfoName,
				Spaces:         []string{runtime.Name},
				Middlewares: []v1alpha1.MiddlewareStatus{
					{
						Middleware: runtime.Spec.DataBases[0],
					},
				},
			}
			wanted.Middlewares[0].Middleware.Space = runtime.Name
			Expect(status).Should(Equal(wanted))
		})
	})

	Describe("Delete", func() {
		It("can delete middleware runtime", func() {
			performer, err := NewPerformer(initInfo)
			Expect(err).ShouldNot(HaveOccurred())

			status, err := performer.Deploy(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			runtimeStatus := status.(*v1alpha1.MiddlewareRuntimeStatus)
			runtime.Status = *runtimeStatus

			performer, err = NewPerformer(initInfo)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = performer.Delete(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			mt := multiTenant.(*componentmock.MultiTenant)
			Expect(len(mt.Products)).Should(Equal(0))
		})
	})
})
