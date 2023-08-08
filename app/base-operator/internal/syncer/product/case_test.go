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

package product

import (
	"context"
	"fmt"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/base-operator/internal/syncer/productprovider"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("K8S syncer", func() {
	var product *nautescrd.Product
	var selector *client.MatchingLabels
	var ctx context.Context
	var urlFormat string
	var repoName string

	BeforeEach(func() {
		urlFormat = "https://www.github.com/%s/default.project"
		productID := randNum()
		repoID := randNum()
		product = &nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("product-%s", productID),
				Namespace: "nautes",
			},
			Spec: nautescrd.ProductSpec{
				Name:         "new-operator",
				MetaDataPath: fmt.Sprintf(urlFormat, "new-operator"),
			},
		}
		productprovider.MockProvider = &productprovider.MockProductProvider{
			ProductMeta: baseinterface.ProductMeta{
				ID:     productID,
				MetaID: repoID,
			},
		}
		repoName = fmt.Sprintf("repo-%s", repoID)

		ctx = context.Background()
		ctx = context.WithValue(ctx, "productName", product.Name)

		err := k8sClient.Create(ctx, product)
		Expect(err).NotTo(HaveOccurred())

		selector = &client.MatchingLabels{nautescrd.LABEL_FROM_PRODUCT: product.Name}

	})

	It("create a new product", func() {
		err := syncInstance.Sync(ctx, *product)
		Expect(err).Should(BeNil())

		appList := &argocrd.ApplicationList{}
		err = k8sClient.List(ctx, appList, selector)
		Expect(err).Should(BeNil())
		Expect(len(appList.Items)).Should(Equal(1))

		nsList := &corev1.NamespaceList{}
		err = k8sClient.List(ctx, nsList, selector)
		Expect(err).Should(BeNil())
		Expect(len(nsList.Items)).Should(Equal(1))

		coderepos := &nautescrd.CodeRepoList{}
		err = k8sClient.List(ctx, coderepos, selector)
		Expect(err).Should(BeNil())
		Expect(len(coderepos.Items)).Should(Equal(1))
		Expect(coderepos.Items[0].Spec.URL).Should(Equal(product.Spec.MetaDataPath))
		Expect(coderepos.Items[0].Name).Should(Equal(repoName))
	})

	It("update a product", func() {
		err := syncInstance.Sync(ctx, *product)
		Expect(err).Should(BeNil())
		product.Spec.MetaDataPath = fmt.Sprintf(urlFormat, "new-operator2")

		err = syncInstance.Sync(ctx, *product)
		Expect(err).Should(BeNil())

		app := &argocrd.Application{}
		key := types.NamespacedName{
			Name:      product.Name,
			Namespace: "default",
		}
		err = k8sClient.Get(ctx, key, app)
		Expect(err).Should(BeNil())
		Expect(app.Spec.Source.RepoURL).Should(Equal(product.Spec.MetaDataPath))

		coderepos := &nautescrd.CodeRepoList{}
		err = k8sClient.List(ctx, coderepos, selector)
		Expect(err).Should(BeNil())
		Expect(len(coderepos.Items)).Should(Equal(1))
		Expect(coderepos.Items[0].Spec.URL).Should(Equal(product.Spec.MetaDataPath))
	})

	It("delete product", func() {
		err := syncInstance.Sync(ctx, *product)
		Expect(err).Should(BeNil())

		// namespace can not be delete in envtest, it must return error
		_ = syncInstance.Delete(ctx, *product)

		appList := &argocrd.ApplicationList{}
		err = k8sClient.List(ctx, appList, selector)
		Expect(err).Should(BeNil())
		Expect(len(appList.Items)).Should(Equal(0))

		nsList := &corev1.NamespaceList{}
		err = k8sClient.List(ctx, nsList, selector)
		Expect(err).Should(BeNil())
		Expect(nsList.Items[0].DeletionTimestamp.IsZero()).Should(BeFalse())
	})
})
