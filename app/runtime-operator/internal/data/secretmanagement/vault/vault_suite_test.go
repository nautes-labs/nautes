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
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	vproxyMock "github.com/nautes-labs/nautes/test/mock/vaultproxy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault Suite")
}

var (
	secClient       *vproxyMock.MockSecretHTTPClient
	authClient      *vproxyMock.MockAuthHTTPClient
	grantClient     *vproxyMock.MockAuthGrantHTTPClient
	vaultClientRoot *vaultapi.Client
	vaultServer     *exec.Cmd
	logger          = logf.Log.WithName("runtime-secret-manager-vault-test")
)

var _ = BeforeSuite(func() {
	var err error
	vaultCfg := vaultapi.DefaultConfig()
	vaultCfg.Address = "http://127.0.0.1:8200"

	vaultClientRoot, err = vaultapi.NewClient(vaultCfg)
	Expect(err).Should(BeNil())
	vaultClientRoot.SetToken("test")

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
})

var _ = AfterSuite(func() {
	err := vaultServer.Process.Kill()
	Expect(err).NotTo(HaveOccurred())
})

const (
	engineNameCluster = "cluster"
	authName          = "cluster-01"
)

func initVault(data, roleBinding map[string]interface{}) {
	var err error
	ctx := context.Background()
	mountPath := engineNameCluster
	mountInput := &vaultapi.MountInput{
		Type:    "kv",
		Config:  vaultapi.MountConfigInput{},
		Options: map[string]string{"version": "2"},
	}
	err = vaultClientRoot.Sys().Mount(mountPath, mountInput)
	Expect(err).Should(BeNil())
	err = watiForSecretEngineInit()
	Expect(err).Should(BeNil())

	path := fmt.Sprintf("kubernetes/%s/default/admin", authName)
	_, err = vaultClientRoot.KVv2(engineNameCluster).Put(ctx, path, data)
	Expect(err).Should(BeNil())
	err = vaultClientRoot.Sys().EnableAuthWithOptions(authName, &vaultapi.MountInput{Type: "kubernetes"})
	Expect(err).Should(BeNil())
	_, err = vaultClientRoot.Logical().Write(fmt.Sprintf("auth/%s/role/RUNTIME", authName), roleBinding)
	Expect(err).Should(BeNil())
}

func cleanVault() {
	err := vaultClientRoot.Sys().Unmount(engineNameCluster)
	Expect(err).Should(BeNil())
	err = vaultClientRoot.Sys().DisableAuth(authName)
	Expect(err).Should(BeNil())
}

func watiForSecretEngineInit() error {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := vaultClientRoot.KVv2(engineNameCluster).Put(ctx, "wait-for-start", map[string]interface{}{"test": "test"})
		if err == nil {
			return nil
		}
		logger.V(1).Info("try to put secret failed", "error", err.Error())
		time.Sleep(time.Second * 3)
	}
	return fmt.Errorf("wait for secret engine start timeout")
}
