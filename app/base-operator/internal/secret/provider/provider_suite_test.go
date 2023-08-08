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

package provider_test

import (
	"context"
	"testing"

	"github.com/nautes-labs/nautes/app/base-operator/internal/secret/provider"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}

var _ = BeforeSuite(func() {
	provider.SecretProviders["mock"] = NewClient
})

var _ = Describe("Secret provider", func() {
	It("get exist provider", func() {
		_, err := provider.GetSecretStore(context.Background(), nautescfg.SecretRepo{
			RepoType: "mock",
		})
		Expect(err).Should(BeNil())
	})
	It("get unknow provider, will failed", func() {
		_, err := provider.GetSecretStore(context.Background(), nautescfg.SecretRepo{
			RepoType: "other",
		})
		Expect(err).ShouldNot(BeNil())
	})
})

func NewClient(cfg nautescfg.SecretRepo) (baseinterface.SecretClient, error) {
	return nil, nil
}
