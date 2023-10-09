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
	"os/exec"
	"testing"

	vault "github.com/hashicorp/vault/api"
	providervault "github.com/nautes-labs/nautes/app/base-operator/internal/secret/vault"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault Suite")
}

var ctx = context.Background()
var vaultServer *exec.Cmd
var vaultRawClient *vault.Client
var provider baseinterface.SecretClient
var gitInstName string
var err error
var caBundle string

var _ = BeforeSuite(func() {
	vaultCfg := vault.DefaultConfig()
	vaultCfg.Address = "http://127.0.0.1:8200"

	gitInstName = "gitlab-007"

	vaultRawClient, err = vault.NewClient(vaultCfg)
	Expect(err).Should(BeNil())
	vaultRawClient.SetToken("test")

	vaultServer = exec.Command("vault", "server", "-dev", "-dev-root-token-id=test")
	err = vaultServer.Start()
	Expect(err).Should(BeNil())

	for {
		vaultServerHealthCheck := exec.Command("vault", "status", "-address=http://127.0.0.1:8200")
		err := vaultServerHealthCheck.Run()
		if err == nil {
			break
		}
	}

	mountPath := providervault.TenantNamespace
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
	vaultRawClient.Sys().Mount(mountPath, mountInput)
	vaultRawClient.KVv2(providervault.TenantNamespace).Put(ctx,
		fmt.Sprintf(providervault.CodeHostingPlatformRootPath, gitInstName),
		map[string]interface{}{
			providervault.CodeHostingPlatformRootKey: "helpme",
		})

	cfg := nautescfg.SecretRepo{
		RepoType: "vault",
		Vault: nautescfg.Vault{
			Addr:     "http://127.0.0.1:8200",
			CABundle: "",
			Token:    "test",
		},
	}
	provider, err = providervault.NewClient(cfg)
	Expect(err).Should(BeNil())

	caBundle = `-----BEGIN CERTIFICATE-----
MIIC/TCCAeWgAwIBAgIJAM0CEkAL+mwtMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCTEyNy4wLjAuMTAgFw0yMzAyMDEwOTU4MDlaGA8yMDUwMDYxOTA5NTgwOVow
FDESMBAGA1UEAwwJMTI3LjAuMC4xMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAn9fykf4Hk0dmEQa2zAjWytzR65oanctqiiX8L/vfstbYscPJm4B+cZkv
pq66ynn5utCF6tGlNq5CDkWLM2gohGenb3n+2fuOk25bP9o6HY4hFAEphxt3CAqs
vV0QEVoWQ+YrqWJe7Qde2hTfI33HM0mOronJPVc4LlCtaQPg6Gh1HWh7HPcxN0Wz
qIeYGenykc+I2q4CwfkZKSifu9Xgr96yJ3ySx8SNhEk4Kqyb0NHUnP/n0gVJo8IO
o/pFoiwi+MskFJzVu0Rm8oGX/GprMCc0k3Dw3l2Yn/LUsQviEl81oJZnjNZWh7WU
rLbWjhsgXsu7ERkIgCRqrKfcbqYSMwIDAQABo1AwTjAdBgNVHQ4EFgQU2eT6JVNc
4Imuo0sw3M6eNolBOpowHwYDVR0jBBgwFoAU2eT6JVNc4Imuo0sw3M6eNolBOpow
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAfOczfYn11VlDLQt6GoY
JFewHBSqFnCAWTTBfZyVH4XI5pydWuii/idwTjOv7UmVIP9EU3SOsyfCXyitQO9g
YqyiLY9qFJRxD6oXJP+M4CCdtXF9YB4akcR+8hvTG96fOSG/Ej92cXT51Hm9VCwI
Y4i1hBjnJNiJojl32hdtl79RvJ5iXk5o4u00rMz07y3t7hX/47izCb2Wo1w1Xaf5
OgD1MEnYUJdatjqb1hq+z9GDgrkc8Xb3CPCAHYk0OKij1A0YgNqrI5v7lEg1oXsx
O37YdSPd7gbdPanLOgvydTvcwuTHJLCSmmfveZHFxHrbjcEwiVzyZHetuai5lEGY
vw==
-----END CERTIFICATE-----`
	cfg.Vault.Addr = "https://127.0.0.1:8200"
	cfg.Vault.CABundle = caBundle

	_, err := providervault.NewClient(cfg)
	Expect(err).Should(BeNil())

})

var _ = AfterSuite(func() {
	err := vaultServer.Process.Kill()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("vault provider", func() {
	It("get exit git instance root token", func() {
		sec, err := provider.GetGitRepoRootToken(ctx, gitInstName)
		Expect(err).Should(BeNil())
		Expect(sec).Should(Equal("helpme"))
	})

	It("when git instance not exit, get token failed", func() {
		_, err := provider.GetGitRepoRootToken(ctx, "bbb")
		Expect(err).ShouldNot(BeNil())
	})
})
