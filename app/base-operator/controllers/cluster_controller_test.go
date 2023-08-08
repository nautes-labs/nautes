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

package controllers

import (
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Reconcile", func() {
	var cleanQueue []client.Object
	var product *nautescrd.Product
	var cluster *nautescrd.Cluster
	var productID string
	var productName string
	var clusterName string
	var err error

	BeforeEach(func() {
		seed := randNum()
		logger.Info("Case start", "seed", seed)

		cleanQueue = []client.Object{}
		productID = fmt.Sprintf("productid-%s", seed)
		productName = fmt.Sprintf("productName-%s", seed)
		clusterName = fmt.Sprintf("cluster-%s", seed)
		product = &nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ProductSpec{
				Name:         productName,
				MetaDataPath: "ssh://127.0.0.1",
			},
		}
		cluster = &nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: nautesNamespace,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:                         "http://127.0.0.1:6443",
				ClusterType:                       nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:                             nautescrd.CLUSTER_USAGE_WORKER,
				HostCluster:                       "",
				PrimaryDomain:                     "",
				WorkerType:                        nautescrd.ClusterWorkTypeDeployment,
				ComponentsList:                    nautescrd.ComponentsList{},
				ReservedNamespacesAllowedProducts: map[string][]string{"argocd": {productName}},
				ProductAllowedClusterResources:    map[string][]nautescrd.ClusterResourceInfo{},
			},
		}

		cleanQueue = append(cleanQueue, product, cluster)
		err = nil
	})

	AfterEach(func() {
		for _, obj := range cleanQueue {
			err := k8sClient.Delete(ctx, obj)
			Expect(client.IgnoreNotFound(err)).Should(BeNil())
			err = waitForDelete(obj)
			Expect(err).Should(BeNil())
		}
	})

	It("update product info in cluster", func() {
		err = k8sClient.Create(ctx, product)
		Expect(err).Should(BeNil())
		err = waitForIndexUpdate(&nautescrd.ProductList{}, fields.OneTermEqualSelector(indexFieldProductName, product.Spec.Name))
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())

		clusterAfterReconcile := cluster.DeepCopy()
		_, err = waitForCondition(clusterAfterReconcile, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		Expect(clusterAfterReconcile.Status.ProductIDMap[productName]).Should(Equal(productID))
	})

	It("if product is removed from cluster, productID should be cleared", func() {
		err = k8sClient.Create(ctx, product)
		Expect(err).Should(BeNil())
		err = waitForIndexUpdate(&nautescrd.ProductList{}, fields.OneTermEqualSelector(indexFieldProductName, product.Spec.Name))
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())
		_, err = waitForCondition(cluster, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())

		cluster.Spec.ReservedNamespacesAllowedProducts = nil
		err = k8sClient.Update(ctx, cluster)
		Expect(err).Should(BeNil())

		clusterAfterReconcile := cluster.DeepCopy()
		_, err = waitForCondition(clusterAfterReconcile, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		_, ok := clusterAfterReconcile.Status.ProductIDMap[productName]
		Expect(ok).Should(BeFalse())
	})

	It("if product name is changed, add warning message in cluster", func() {
		err = k8sClient.Create(ctx, product)
		Expect(err).Should(BeNil())
		err = waitForIndexUpdate(&nautescrd.ProductList{}, fields.OneTermEqualSelector(indexFieldProductName, product.Spec.Name))
		Expect(err).Should(BeNil())

		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())
		_, err = waitForCondition(cluster, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		err = waitForIndexUpdate(&nautescrd.ClusterList{}, fields.OneTermEqualSelector(indexFieldProductsInCluster, product.Spec.Name))
		Expect(err).Should(BeNil())

		product.Spec.Name = "fakeName"
		err = k8sClient.Update(ctx, product)
		Expect(err).Should(BeNil())

		clusterAfterReconcile := cluster.DeepCopy()
		_, err = waitForCondition(clusterAfterReconcile, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		Expect(clusterAfterReconcile.Status.ProductIDMap[productName]).Should(Equal(productID))
		Expect(clusterAfterReconcile.Status.Warnings[0].ID).Should(Equal(productName))
	})

	It("if product not found, add warning message in cluster", func() {
		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())

		clusterAfterReconcile := cluster.DeepCopy()
		_, err := waitForCondition(clusterAfterReconcile, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		Expect(len(clusterAfterReconcile.Status.ProductIDMap)).Should(Equal(0))
		Expect(clusterAfterReconcile.Status.Warnings[0].ID).Should(Equal(productName))
	})

	It("if product is removed from cluster, warning should be cleared", func() {
		err = k8sClient.Create(ctx, cluster)
		Expect(err).Should(BeNil())
		_, err = waitForCondition(cluster, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())

		cluster.Spec.ReservedNamespacesAllowedProducts = nil
		err = k8sClient.Update(ctx, cluster)
		Expect(err).Should(BeNil())

		clusterAfterReconcile := cluster.DeepCopy()
		_, err = waitForCondition(clusterAfterReconcile, clusterConditionTypeProductInfoUpdated)
		Expect(err).Should(BeNil())
		Expect(len(clusterAfterReconcile.Status.Warnings)).Should(Equal(0))
	})

})
