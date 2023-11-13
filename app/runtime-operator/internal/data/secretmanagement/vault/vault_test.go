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

package vault_test

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/secretmanagement/vault"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vault", func() {
	var secMgr syncer.SecretManagement
	var ctx context.Context
	var kubeconfig = "thisisconfig"
	var seed string
	var user syncer.MachineAccount
	BeforeEach(func() {
		var err error
		seed = RandNum()
		ctx = context.Background()
		data := map[string]interface{}{
			"kubeconfig": kubeconfig,
		}
		roleBinding := map[string]interface{}{
			"bound_service_account_namespaces": "nautes",
			"bound_service_account_names":      "nautes",
		}
		initVault(data, roleBinding)
		initMock()

		user = syncer.MachineAccount{
			Name:   fmt.Sprintf("user-%s", seed),
			Spaces: []string{fmt.Sprintf("ns-%s", seed)},
		}

		opt := v1alpha1.Component{}
		initInfo := syncer.ComponentInitInfo{
			ClusterConnectInfo: syncer.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes:  nil,
			},
			ClusterName:            authName,
			NautesResourceSnapshot: nil,
			NautesConfig: configs.Config{
				Secret: configs.SecretRepo{
					RepoType: "",
					Vault: configs.Vault{
						Addr:  "http://127.0.0.1:8200",
						Token: "test",
					},
					OperatorName: map[string]string{
						configs.OperatorNameRuntime: "runtime",
					},
				},
			},
			Components: nil,
		}
		secMgr, err = vault.NewVaultClient(opt, &initInfo, vault.SetNewVaultProxyClientFunction(getMockVaultProxyClient))
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		cleanVault()
	})

	It("can get cluster info", func() {
		connectInfo, err := secMgr.GetAccessInfo(ctx)
		Expect(err).Should(BeNil())
		Expect(connectInfo).Should(Equal(kubeconfig))
	})

	It("can create user", func() {
		authInfo, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		sa := sec.Data["bound_service_account_names"].([]interface{})[0].(string)
		Expect(sa).Should(Equal(authInfo.ServiceAccounts[0].ServiceAccount))
		namespace := sec.Data["bound_service_account_namespaces"].([]interface{})[0].(string)
		Expect(namespace).Should(Equal(authInfo.ServiceAccounts[0].Namespace))
	})

	It("can delete user", func() {
		_, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		err = secMgr.DeleteAccount(ctx, user)
		Expect(err).Should(BeNil())
		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		Expect(sec).Should(BeNil())

	})

	It("grant code repo permission to user", func() {
		_, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		repo := syncer.SecretInfo{
			Type: syncer.SecretTypeCodeRepo,
			CodeRepo: &syncer.CodeRepo{
				ProviderType: "a",
				ID:           "b",
				User:         "c",
				Permission:   "d",
			},
		}
		policy := fmt.Sprintf("%s-%s-%s-%s",
			repo.CodeRepo.ProviderType,
			repo.CodeRepo.ID,
			repo.CodeRepo.User,
			repo.CodeRepo.Permission)
		err = secMgr.GrantPermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		rolePolicy := sec.Data["token_policies"].([]interface{})[0].(string)
		Expect(rolePolicy).Should(Equal(policy))
	})

	It("revoke code repo permission from user", func() {
		_, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		repo := syncer.SecretInfo{
			Type: syncer.SecretTypeCodeRepo,
			CodeRepo: &syncer.CodeRepo{
				ProviderType: "a",
				ID:           "b",
				User:         "c",
				Permission:   "d",
			},
		}
		err = secMgr.GrantPermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		err = secMgr.RevokePermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		policyNum := len(sec.Data["token_policies"].([]interface{}))
		Expect(policyNum).Should(Equal(0))
	})

	It("grant artifact account permission to user", func() {
		_, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		repo := syncer.SecretInfo{
			Type: syncer.SecretTypeArtifactRepo,
			ArtifactAccount: &syncer.ArtifactAccount{
				ProviderName: "a",
				Product:      "b",
				Project:      "c",
			},
		}
		policy := fmt.Sprintf("%s-%s-%s",
			repo.ArtifactAccount.ProviderName,
			repo.ArtifactAccount.Product,
			repo.ArtifactAccount.Project,
		)
		err = secMgr.GrantPermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		rolePolicy := sec.Data["token_policies"].([]interface{})[0].(string)
		Expect(rolePolicy).Should(Equal(policy))
	})

	It("revoke artifact account permission from user", func() {
		_, err := secMgr.CreateAccount(ctx, user)
		Expect(err).Should(BeNil())

		repo := syncer.SecretInfo{
			Type: syncer.SecretTypeArtifactRepo,
			ArtifactAccount: &syncer.ArtifactAccount{
				ProviderName: "a",
				Product:      "b",
				Project:      "c",
			},
		}
		err = secMgr.GrantPermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		err = secMgr.RevokePermission(ctx, repo, user)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		policyNum := len(sec.Data["token_policies"].([]interface{}))
		Expect(policyNum).Should(Equal(0))
	})
})
