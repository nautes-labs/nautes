package vault_test

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/secretmanagent/vault"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
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
	var user syncer.User
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

		user = syncer.User{
			Resource: syncer.Resource{
				Product: "",
				Name:    fmt.Sprintf("user-%s", seed),
			},
			UserType: syncer.UserTypeMachine,
			AuthInfo: &syncer.Auth{
				Kubernetes: []syncer.AuthKubernetes{
					{
						ServiceAccount: fmt.Sprintf("sa-%s", seed),
						Namespace:      fmt.Sprintf("ns-%s", seed),
					},
				},
			},
		}

		opt := v1alpha1.Component{}
		initInfo := syncer.ComponentInitInfo{
			ClusterConnectInfo: syncer.ClusterConnectInfo{
				Type:       v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: nil,
			},
			ClusterName: authName,
			RuntimeType: "",
			NautesDB:    nil,
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
			Components: syncer.ComponentList{},
		}
		secMgr, err = vault.NewVaultClient(opt, initInfo, vault.SetNewVaultProxyClientFunction(getMockVaultProxyClient))
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
		_, err := secMgr.CreateUser(ctx, user, nil)
		Expect(err).Should(BeNil())

		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		sa := sec.Data["bound_service_account_names"].([]interface{})[0].(string)
		Expect(sa).Should(Equal(user.AuthInfo.Kubernetes[0].ServiceAccount))
		namespace := sec.Data["bound_service_account_namespaces"].([]interface{})[0].(string)
		Expect(namespace).Should(Equal(user.AuthInfo.Kubernetes[0].Namespace))

	})

	It("can delete user", func() {
		_, err := secMgr.CreateUser(ctx, user, nil)
		Expect(err).Should(BeNil())

		_, err = secMgr.DeleteUser(ctx, user, nil)
		Expect(err).Should(BeNil())
		path := fmt.Sprintf("auth/%s/role/%s", authName, user.Name)
		sec, err := vaultClientRoot.Logical().Read(path)
		Expect(err).Should(BeNil())
		Expect(sec).Should(BeNil())

	})

	It("grant code repo permission to user", func() {
		_, err := secMgr.CreateUser(ctx, user, nil)
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
		_, err := secMgr.CreateUser(ctx, user, nil)
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
		_, err := secMgr.CreateUser(ctx, user, nil)
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
		_, err := secMgr.CreateUser(ctx, user, nil)
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
