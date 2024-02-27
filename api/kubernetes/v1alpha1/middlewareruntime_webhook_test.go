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

package v1alpha1_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/pkg/middlewareinfo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Middlewareruntime", func() {
	var (
		ctx                        context.Context
		seed                       string
		instance                   MiddlewareRuntime
		validateClient             ValidateClient
		cleanBox                   []client.Object
		productNamespace           *corev1.Namespace
		env                        *Environment
		cluster                    *Cluster
		middlewareProvider         string
		middlewareMetadataFileName = filepath.Join(homePath, "config", "middlewares.yaml")
	)

	BeforeEach(func() {
		var err error

		ctx = context.TODO()
		middlewareProvider = "operatorhub"
		seed = testutils.RandNum()

		Expect(os.WriteFile(middlewareMetadataFileName, []byte(`redis:
  hasAuthenticationSystem: true
  defaultAuthType: UserPassword
  availableAuthTypes:
    UserPassword: {}
  providers:
    operatorhub:
      defaultImplementation: redis-operator
      implementations:
        redis-operator: {}
test:
  hasAuthenticationSystem: true
  defaultAuthType: UserPassword
  availableAuthTypes:
    UserPassword: {}
  providers:
    operatorhub:
      defaultImplementation: test-operator
      implementations:
        test-operator: {}
`), 0644)).Should(Succeed())
		infos, err := middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath(middlewareMetadataFileName))
		Expect(err).NotTo(HaveOccurred())
		middlewareinfo.MiddlewareMetadata = *infos

		productNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-namespace-%s", seed),
			},
		}
		Expect(k8sClient.Create(ctx, productNamespace)).Should(Succeed())

		cluster = &Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-cluster-%s", seed),
				Namespace: nautesNamespaceName,
			},
			Spec: ClusterSpec{
				ApiServer:   "https://127.0.0.1:6443",
				ClusterType: CLUSTER_TYPE_PHYSICAL,
				ClusterKind: CLUSTER_KIND_KUBERNETES,
				Usage:       CLUSTER_USAGE_WORKER,
				WorkerType:  ClusterWorkTypeMiddleware,
				MiddlewareProvider: &MiddlewareProvider{
					Type: middlewareProvider,
					Cache: map[string][]MiddlewareDeploymentImplementation{
						"test": {
							{
								Type:      "test-operator",
								Namespace: "",
								Additions: map[string]string{},
							},
						},
					},
				},
			},
		}

		Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
		cleanBox = append(cleanBox, cluster)

		env = &Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-env-%s", seed),
				Namespace: productNamespace.Name,
			},
			Spec: EnvironmentSpec{
				EnvType: EnvironmentTypeCluster,
				Cluster: cluster.Name,
			},
		}
		Expect(k8sClient.Create(ctx, env)).Should(Succeed())
		cleanBox = append(cleanBox, env)

		instance = MiddlewareRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-middlewareruntime-%s", seed),
				Namespace: productNamespace.Name,
			},
			Spec: MiddlewareRuntimeSpec{
				Destination: MiddlewareRuntimeDestination{
					Environment: env.Name,
				},
			},
		}
		validateClient = NewValidateClientFromK8s(k8sClient)
	})

	AfterEach(func() {
		for _, obj := range cleanBox {
			err := k8sClient.Delete(ctx, obj)
			Expect(client.IgnoreNotFound(err)).Should(Succeed())
			Expect(waitForDelete(obj)).Should(Succeed())
		}
		Expect(k8sClient.Delete(ctx, productNamespace)).Should(Succeed())
		Expect(os.Remove(middlewareMetadataFileName)).Should(Succeed())
	})

	It("can validate a valid instance", func() {
		instance.Spec.Caches = []Middleware{
			{
				Name: "test",
				Type: "test",
			},
		}
		err := instance.Validate(ctx, validateClient, k8sClient)
		Expect(err).NotTo(HaveOccurred())
	})

	It("will return an error if the middleware type does not supported", func() {
		instance.Spec.Caches = []Middleware{
			{
				Name: "test",
				Type: "fake-type",
			},
		}
		err := instance.Validate(ctx, validateClient, k8sClient)
		Expect(err.Error()).Should(ContainSubstring("middleware %s does not found", "fake-type"))
	})

	It("will return an error if space is used by other runtime", func() {
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster)).Should(Succeed())
		cluster.Status.ResourceUsage = &ResourceUsage{
			RuntimeUsage: map[string][]RuntimeUsage{
				"other-product": {
					{
						Name:        "other-runtime",
						AccountName: "",
						Namespaces:  []string{instance.Name},
					},
				},
			},
		}
		Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())
		instance.Spec.Caches = []Middleware{
			{
				Name: "test",
				Type: "test",
			},
		}
		err := instance.Validate(ctx, validateClient, k8sClient)
		Expect(err.Error()).Should(ContainSubstring("namespaces %v are used by other runtime", []string{instance.Name}))
	})
})
