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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	vault "github.com/hashicorp/vault/api"
	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	vaultclient "github.com/nautes-labs/nautes/app/cluster-operator/internal/secretclient/vault"
	nautesconst "github.com/nautes-labs/nautes/pkg/nautesconst"
	loadcert "github.com/nautes-labs/nautes/pkg/util/loadcerts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	CLUSTER = "cluster"
)

var ctx context.Context
var vaultServer *exec.Cmd
var vaultRootClient *vault.Client
var vaultRawClient *vault.Client

func setVaultToken() {
	vaultCfg := vault.DefaultConfig()
	vaultCfg.Address = "http://127.0.0.1:8200"

	token, err := vaultRootClient.Auth().Token().Create(&vault.TokenCreateRequest{
		Policies: []string{"root"},
	})
	Expect(err).Should(BeNil())
	vaultRawClient, err = vault.NewClient(vaultCfg)
	Expect(err).Should(BeNil())
	vaultRawClient.SetToken(token.Auth.ClientToken)
}

var _ = Describe("VaultStore", func() {
	var tenantRepoPolicy string
	BeforeEach(func() {
		var err error
		mountPath := CLUSTER
		mountInput := &vault.MountInput{
			Type:                  "kv",
			Description:           "",
			Config:                vault.MountConfigInput{},
			Local:                 false,
			SealWrap:              false,
			ExternalEntropyAccess: false,
			Options:               map[string]string{"version": "2"},
			PluginName:            "",
		}

		setVaultToken()
		err = vaultRawClient.Sys().Mount(mountPath, mountInput)
		Expect(err).Should(BeNil())

		tenantRepoPolicy = fmt.Sprintf("github-%s-default-readonly", tenantCodeRepoName)

		initData()
		initMock()
	})

	AfterEach(func() {
		cleanVault()
	})

	Describe("when a new cluster", func() {
		var newCluster *clustercrd.Cluster
		Context("if it usage is runtime", func() {
			BeforeEach(func() {
				data := map[string]interface{}{
					"kubeconfig": "thisisconfig",
				}
				roleBinding := map[string]interface{}{
					"bound_service_account_namespaces": "nautes",
					"bound_service_account_names":      "nautes",
				}
				initVault(data, roleBinding)
				initK8SCreateMock()

				newCluster = cluster.DeepCopy()
			})

			It("will do nothing when cluster name is tenant name", func() {
				newCluster.Name = "tenant"
				_, err := vaultStore.Sync(ctx, newCluster, nil)
				Expect(err).ShouldNot(BeNil())
			})

			It("will create vault secret, vault auth, vault role and grant permission", func() {
				rst, err := vaultStore.Sync(ctx, newCluster, nil)
				Expect(err).Should(BeNil())
				Expect(rst.SecretID).Should(Equal("1"))

				sec, err := vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
				Expect(err).Should(BeNil())
				Expect(strings.Contains(sec.Data["kubeconfig"].(string), newCluster.Spec.ApiServer)).Should(BeTrue())

				mnt, err := vaultRootClient.Logical().Read(fmt.Sprintf("sys/auth/%s", newCluster.Name))
				Expect(err).Should(BeNil())
				Expect(mnt).ShouldNot(BeNil())

				roleList, err := vaultRootClient.Logical().List(fmt.Sprintf("auth/%s/role", newCluster.Name))
				Expect(err).Should(BeNil())
				Expect(len(roleList.Data["keys"].([]interface{}))).Should(Equal(6))

				role, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/ARGO", newCluster.Name))
				Expect(err).Should(BeNil())
				Expect(role.Data["token_policies"].([]interface{})[0]).Should(Equal(tenantRepoPolicy))

				role, err = vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name))
				Expect(err).Should(BeNil())
				Expect(role.Data["token_policies"].([]interface{})[0]).Should(Equal("kubernetes-pipeline-default-admin"))
			})

			It("when the clustertype is host cluster, will not create secret", func() {
				newCluster.Spec.ClusterType = clustercrd.CLUSTER_TYPE_PHYSICAL

				ver, err := vaultStore.SyncSecret(newCluster, nil)
				Expect(err).ShouldNot(BeNil())
				Expect(ver).ShouldNot(Equal(1))

				_, err = vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
				Expect(err).ShouldNot(BeNil())
			})

			It("cluster clean up when sync failed", func() {})

		})
	})

	Describe("when update cluster", func() {
		var oldCluster, newCluster *clustercrd.Cluster
		var policyName string
		BeforeEach(func() {
			data := map[string]interface{}{
				"kubeconfig": "thisisconfig",
			}
			roleBinding := map[string]interface{}{
				"bound_service_account_namespaces": "nautes",
				"bound_service_account_names":      "nautes",
				"token_policies":                   []interface{}{"kubernetes-pipeline-default-admin"},
			}
			initVault(data, roleBinding)
			initK8SCreateMock()

			oldCluster = cluster.DeepCopy()
			newCluster = cluster.DeepCopy()

			_, err := vaultStore.SyncCluster(ctx, newCluster, nil)
			Expect(err).Should(BeNil())
			policyName = "kubernetes-pipeline-default-admin"
			_ = policyName
		})

		AfterEach(func() {
			err := vaultRootClient.Sys().DisableAuth(parentCluster.Name)
			Expect(err).Should(BeNil())
			err = vaultRootClient.Sys().DisableAuth(oldCluster.Name)
			Expect(err).Should(BeNil())
			err = vaultRootClient.Sys().DisableAuth(newCluster.Name)
			Expect(err).Should(BeNil())
		})

		It("if cluster type change to host, will do nothing", func() {
			initK8SCreateMock()
			_, err := vaultStore.SyncCluster(ctx, newCluster, oldCluster)
			Expect(err).Should(BeNil())

			newCluster.Spec.ClusterType = clustercrd.CLUSTER_TYPE_PHYSICAL
			_, err = vaultStore.SyncSecret(newCluster, nil)
			Expect(err).Should(BeNil())

			sec, err := vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
			Expect(err).Should(BeNil())
			Expect(sec).ShouldNot(BeNil())
		})

		It("if api server change, will update secret and auth", func() {
			initK8SCreateMock()
			newCluster.Spec.ApiServer = "https://127.0.0.1:40001"
			_, err := vaultStore.SyncCluster(ctx, newCluster, oldCluster)
			Expect(err).Should(BeNil())

			sec, err := vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
			Expect(err).Should(BeNil())
			Expect(strings.Contains(sec.Data["kubeconfig"].(string), newCluster.Spec.ApiServer)).Should(BeTrue())

			authMeta, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/config", newCluster.Name))
			Expect(err).Should(BeNil())
			Expect(authMeta.Data["kubernetes_host"].(string)).Should(Equal(newCluster.Spec.ApiServer))
		})

		It("change cluster type from virtual to physic", func() {
			initK8SCreateMock()

			newCluster.Spec.ClusterType = clustercrd.CLUSTER_TYPE_PHYSICAL
			_, err := vaultStore.SyncCluster(ctx, newCluster, oldCluster)
			Expect(err).Should(BeNil())

			_, err = vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
			Expect(err).ShouldNot(BeNil())

			authMeta, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/config", newCluster.Name))
			Expect(err).Should(BeNil())
			Expect(authMeta.Data["kubernetes_host"].(string)).Should(Equal(newCluster.Spec.ApiServer))
		})

		It("change cluster usage from host to worker", func() {
			newCluster.Spec.Usage = clustercrd.CLUSTER_USAGE_HOST
			_, err := vaultStore.SyncCluster(ctx, newCluster, oldCluster)
			Expect(err).Should(BeNil())

			authMeta, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/config", newCluster.Name))
			Expect(err).Should(BeNil())
			Expect(authMeta).Should(BeNil())

			role, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name))
			Expect(err).Should(BeNil())
			Expect(len(role.Data["token_policies"].([]interface{}))).Should(Equal(0))

			_, err = vaultStore.SyncCluster(ctx, oldCluster, newCluster)
			Expect(err).Should(BeNil())

			authMeta, err = vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/config", oldCluster.Name))
			Expect(err).Should(BeNil())
			Expect(authMeta).ShouldNot(BeNil())

			role, err = vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name))
			Expect(err).Should(BeNil())
			Expect(len(role.Data["token_policies"].([]interface{}))).Should(Equal(1))
		})

		It("cluster clean up", func() {
			newCluster.Spec.Usage = clustercrd.CLUSTER_USAGE_WORKER
			newCluster.Spec.ClusterType = clustercrd.CLUSTER_TYPE_VIRTUAL
			oldCluster.Spec.Usage = clustercrd.CLUSTER_USAGE_HOST
			oldCluster.Spec.ClusterType = clustercrd.CLUSTER_TYPE_PHYSICAL

			err := vaultStore.CleanCluster(ctx, newCluster, oldCluster, &vaultclient.SyncResult{})
			Expect(err).Should(BeNil())

			_, err = vaultRootClient.KVv2(CLUSTER).Get(ctx, "kubernetes/pipeline/default/admin")
			Expect(err).ShouldNot(BeNil())

			auth, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s", newCluster.Name))
			Expect(err).Should(BeNil())
			Expect(auth).Should(BeNil())

			role, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name))
			Expect(err).Should(BeNil())
			Expect(len(role.Data["token_policies"].([]interface{}))).Should(Equal(0))
		})

	})

	Describe("Remove Cluster", func() {
		var deleteCluster, lastCluster *clustercrd.Cluster
		BeforeEach(func() {
			data := map[string]interface{}{
				"kubeconfig": "thisisconfig",
			}
			roleBinding := map[string]interface{}{
				"bound_service_account_namespaces": "nautes",
				"bound_service_account_names":      "nautes",
				"token_policies":                   []interface{}{"kubernetes-pipeline-default-admin"},
			}
			initVault(data, roleBinding)
			initK8SCreateMock()

			deleteCluster = cluster.DeepCopy()
			lastCluster = cluster.DeepCopy()

			_, err := vaultStore.SyncCluster(ctx, cluster, nil)
			Expect(err).Should(BeNil())
		})

		AfterEach(func() {
			err := vaultRootClient.Sys().DisableAuth(parentCluster.Name)
			Expect(err).Should(BeNil())
		})

		It("clean up vault auth", func() {
			err := vaultStore.Delete(ctx, lastCluster)
			Expect(err).Should(BeNil())

			auth, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s", deleteCluster.Name))
			Expect(err).Should(BeNil())
			Expect(auth).Should(BeNil())

			role, err := vaultRootClient.Logical().Read(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name))
			Expect(err).Should(BeNil())
			Expect(len(role.Data["token_policies"].([]interface{}))).Should(Equal(0))
		})

	})

	Describe("new object method", func() {
		const nautesKeypairPath = "/tmp/keypair"
		const nautesHomePath = "/tmp/cluster-test/"
		BeforeEach(func() {
			var err error
			err = os.Setenv(nautesconst.EnvNautesHome, nautesHomePath)
			err = os.Mkdir(nautesHomePath, 0755)
			certPath := filepath.Join(nautesHomePath, loadcert.DefaultCertsPath)
			err = os.Mkdir(certPath, 0755)
			err = os.WriteFile(filepath.Join(certPath, "./ca.crt"), []byte(CAPublic), 0600)
			Expect(err).Should(BeNil())
			err = os.Mkdir(nautesKeypairPath, 0755)
			Expect(err).Should(BeNil())
			err = os.Setenv("VAULT_PROXY_CLIENT_KEYPAIR_PATH", nautesKeypairPath)
			Expect(err).Should(BeNil())
			err = os.WriteFile(filepath.Join(nautesKeypairPath, "client.key"), []byte(KeyPrivate), 0600)
			Expect(err).Should(BeNil())
			err = os.WriteFile(filepath.Join(nautesKeypairPath, "client.crt"), []byte(KeyPublic), 0600)
			Expect(err).Should(BeNil())
		})

		AfterEach(func() {
			err := os.RemoveAll(nautesHomePath)
			Expect(err).Should(BeNil())
			err = os.RemoveAll(nautesKeypairPath)
			Expect(err).Should(BeNil())
		})

		It("new vault client", func() {
			secretClient, err := vaultclient.NewVaultClient(context.Background(), nautesConfig, k8sClient)
			Expect(err).Should(BeNil())
			Expect(secretClient).ShouldNot(BeNil())
		})

		It("new k8s client", func() {
			k8sClient, err := vaultclient.NewK8SClient(testKubeconfig)
			Expect(err).Should(BeNil())
			Expect(k8sClient).ShouldNot(BeNil())
		})
	})
})
