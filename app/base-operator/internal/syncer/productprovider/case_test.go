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

package productprovider

import (
	"context"
	"fmt"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ProviderSyncer", func() {
	var ctx context.Context
	var providerCRD *nautescrd.ProductProvider
	BeforeEach(func() {
		ctx = context.Background()
		providerCRD = &nautescrd.ProductProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("provider-%s", randNum()),
				Namespace: "default",
			},
			Spec: nautescrd.ProductProviderSpec{
				Type: "mock",
				Name: "noname",
			},
		}
		err := k8sClient.Create(context.Background(), providerCRD)
		Expect(err).Should(BeNil())

		MockProvider = &MockProductProvider{
			ProductList: []nautescrd.Product{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "product-01",
					},
					Spec: nautescrd.ProductSpec{
						Name:         "01",
						MetaDataPath: "ssh://127.0.0.1/group/id01",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "product-03",
					},
					Spec: nautescrd.ProductSpec{
						Name:         "03",
						MetaDataPath: "ssh://127.0.0.1/group/id03",
					},
				},
			},
		}
	})
	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), providerCRD)
		Expect(err).Should(BeNil())
		key := types.NamespacedName{
			Namespace: providerCRD.Namespace,
			Name:      providerCRD.Name,
		}
		isClean := false
		for i := 0; i < 5; i++ {
			err := k8sClient.Get(context.Background(), key, providerCRD)
			if err != nil && client.IgnoreNotFound(err) == nil {
				isClean = true
				break
			}
			time.Sleep(time.Second * 5)
		}
		Expect(isClean).Should(BeTrue())

		products := &nautescrd.ProductList{}
		err = k8sClient.List(ctx, products)
		for _, product := range products.Items {
			k8sClient.Delete(context.Background(), &product)
		}
		err = k8sClient.List(ctx, products)
		Expect(len(products.Items)).Should(Equal(0))
	})

	It("create product by product provider", func() {
		err := syncInstance.Sync(ctx, *providerCRD)
		Expect(err).Should(BeNil())
		products := &nautescrd.ProductList{}
		err = k8sClient.List(ctx, products)
		Expect(len(products.Items)).Should(Equal(2))
	})

	It("delete product by product provider", func() {
		err := syncInstance.Sync(ctx, *providerCRD)
		Expect(err).Should(BeNil())

		MockProvider.ProductList = nil
		err = syncInstance.Sync(ctx, *providerCRD)
		Expect(err).Should(BeNil())

		products := &nautescrd.ProductList{}
		err = k8sClient.List(ctx, products)
		Expect(len(products.Items)).Should(Equal(0))
	})

	It("update product by product provider", func() {
		err := syncInstance.Sync(ctx, *providerCRD)
		Expect(err).Should(BeNil())

		newUrl := "ssh://127.0.0.2/product-02/default.project.git"
		MockProvider.ProductList[0].Spec.MetaDataPath = newUrl
		err = syncInstance.Sync(ctx, *providerCRD)
		Expect(err).Should(BeNil())

		product := &nautescrd.Product{}
		key := types.NamespacedName{
			Namespace: "default",
			Name:      MockProvider.ProductList[0].Name,
		}
		err = k8sClient.Get(ctx, key, product)
		Expect(product.Spec.MetaDataPath).Should(Equal(newUrl))
	})
})
