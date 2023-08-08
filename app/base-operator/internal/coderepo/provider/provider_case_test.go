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

package coderepoprovider_test

import (
	"context"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Code repo provider", func() {
	var ncfg *nautescfg.Config
	var err error
	BeforeEach(func() {
		ncfg, err = nautescfg.NewConfig("")
		Expect(err).Should(BeNil())
		ncfg.Nautes.Namespace = "default"
		ncfg.Secret.RepoType = "mock"
		coderepoProviderCR = &nautescrd.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "icoderepo",
				Namespace: "default",
			},
			Spec: nautescrd.CodeRepoProviderSpec{
				HttpAddress:  "https://127.0.0.1",
				SSHAddress:   "ssh://127.0.0.1",
				ApiServer:    "https://127.0.0.1",
				ProviderType: "gitlab",
			},
		}
		err := k8sClient.Create(context.Background(), coderepoProviderCR)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		key := types.NamespacedName{
			Namespace: coderepoProviderCR.Namespace,
			Name:      coderepoProviderCR.Name,
		}
		err := k8sClient.Delete(context.Background(), coderepoProviderCR)
		Expect(err).Should(BeNil())
		timeout := true
		for i := 0; i < 3; i++ {
			cr := &nautescrd.CodeRepoProvider{}
			err := k8sClient.Get(context.TODO(), key, cr)
			if err != nil && client.IgnoreNotFound(err) == nil {
				timeout = false
				break
			}
			time.Sleep(time.Second * 5)
		}
		Expect(timeout).Should(BeFalse())

	})

	It("get product provider", func() {
		provider, err := crProvider.GetProvider(context.Background(), coderepoProviderCR.Name, k8sClient, *ncfg)
		Expect(err).Should(BeNil())
		Expect(provider).ShouldNot(BeNil())
	})

	It("if coderepo not exist, get provider faild", func() {
		provider, err := crProvider.GetProvider(context.Background(), "notexisted", k8sClient, *ncfg)
		Expect(err).ShouldNot(BeNil())
		Expect(provider).Should(BeNil())
	})

	It("if provider not support, get provider failed", func() {
		newRepoProvider := coderepoProviderCR.DeepCopy()
		newRepoProvider.Spec.ProviderType = "github"
		patch := client.MergeFrom(coderepoProviderCR)
		err = k8sClient.Patch(context.TODO(), newRepoProvider, patch)
		Expect(err).Should(BeNil())

		provider, err := crProvider.GetProvider(context.Background(), "notexisted", k8sClient, *ncfg)
		Expect(err).ShouldNot(BeNil())
		Expect(provider).Should(BeNil())
	})
})
