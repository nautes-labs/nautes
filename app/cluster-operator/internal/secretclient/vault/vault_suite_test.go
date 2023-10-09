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

	vault "github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"path/filepath"

	"github.com/golang/mock/gomock"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	vaultclient "github.com/nautes-labs/nautes/app/cluster-operator/internal/secretclient/vault"
	kubeClientMock "github.com/nautes-labs/nautes/app/cluster-operator/mock/kubeclient"
	vproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	vproxyMock "github.com/nautes-labs/nautes/test/mock/vaultproxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault Suite")
}

var secClient *vproxyMock.MockSecretHTTPClient
var authClient *vproxyMock.MockAuthHTTPClient
var grantClient *vproxyMock.MockAuthGrantHTTPClient
var kubeClient *kubeClientMock.MockKubernetesClient
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var tenantName = "dev-tenant"
var tenantCodeRepoName = "repo-01"
var logger = logf.Log.WithName("vault-client-test")

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../../..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme
	err = nautescrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "nautes"}}
	err = k8sClient.Create(context.Background(), ns)
	Expect(err).NotTo(HaveOccurred())

	tenantCodeRepo := &nautescrd.CodeRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantCodeRepoName,
			Namespace: "nautes",
			Labels:    map[string]string{nautescrd.LABEL_TENANT_MANAGEMENT: tenantName},
		},
		Spec: nautescrd.CodeRepoSpec{
			URL: "ssh://127.0.0.1/nautes/dev-tenant",
		},
	}
	err = k8sClient.Create(context.Background(), tenantCodeRepo)
	Expect(err).NotTo(HaveOccurred())

	vaultCfg := vault.DefaultConfig()
	vaultCfg.Address = "http://127.0.0.1:8200"

	vaultRootClient, err = vault.NewClient(vaultCfg)
	Expect(err).Should(BeNil())
	vaultRootClient.SetToken("test")

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
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func initMock() {
	ctl := gomock.NewController(GinkgoT())
	secClient = vproxyMock.NewMockSecretHTTPClient(ctl)
	secClient.EXPECT().CreateCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.ClusterRequest, optxs ...interface{}) (*vproxy.CreateClusterReply, error) {
			cluster, err := req.ConvertRequest()
			Expect(err).Should(BeNil())
			sec, err := vaultRawClient.KVv2(cluster.SecretName).Put(ctx, cluster.SecretPath, map[string]interface{}{
				"kubeconfig": req.GetAccount().Kubeconfig,
			})
			Expect(err).Should(BeNil())

			return &vproxy.CreateClusterReply{
				Secret: &vproxy.SecretInfo{
					Name:    cluster.SecretName,
					Path:    cluster.SecretPath,
					Version: int32(sec.VersionMetadata.Version),
				},
			}, nil
		})
	secClient.EXPECT().DeleteCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.ClusterRequest, optxs ...interface{}) (*vproxy.DeleteClusterReply, error) {
			cluster, err := req.ConvertRequest()
			Expect(err).Should(BeNil())
			err = vaultRawClient.KVv2(cluster.SecretName).DeleteMetadata(ctx, cluster.SecretPath)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)

	authClient = vproxyMock.NewMockAuthHTTPClient(ctl)
	authClient.EXPECT().CreateAuth(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthRequest, optxs ...interface{}) (*vproxy.CreateAuthReply, error) {
			configPath := fmt.Sprintf("auth/%s/config", req.ClusterName)

			cfg, err := vaultRawClient.Logical().Read(configPath)
			Expect(err).Should(BeNil())

			if cfg == nil {
				err := vaultRawClient.Sys().EnableAuthWithOptions(cluster.Name, &vault.MountInput{Type: "kubernetes"})
				Expect(err).Should(BeNil())
			}

			opts := map[string]interface{}{
				"kubernetes_host":    req.Kubernetes.Url,
				"kubernetes_ca_cert": req.Kubernetes.Cabundle,
				"token_reviewer_jwt": req.Kubernetes.Token,
			}
			_, err = vaultRawClient.Logical().Write(configPath, opts)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	authClient.EXPECT().DeleteAuth(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthRequest, optxs ...interface{}) (*vproxy.DeleteAuthReply, error) {
			err := vaultRawClient.Sys().DisableAuth(req.ClusterName)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	authClient.EXPECT().CreateAuthrole(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthroleRequest, optxs ...interface{}) (*vproxy.CreateAuthReply, error) {
			path := fmt.Sprintf("auth/%s/role/%s", req.ClusterName, req.DestUser)
			opts := map[string]interface{}{
				"bound_service_account_namespaces": req.GetKubernetes().Namespaces,
				"bound_service_account_names":      req.GetKubernetes().ServiceAccounts,
			}
			_, err := vaultRawClient.Logical().Write(path, opts)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)

	grantClient = vproxyMock.NewMockAuthGrantHTTPClient(ctl)
	grantClient.EXPECT().GrantAuthroleGitPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(grantPermission)
	grantClient.EXPECT().RevokeAuthroleGitPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(revokePermission)
	grantClient.EXPECT().GrantAuthroleClusterPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(grantPermission)
	grantClient.EXPECT().RevokeAuthroleClusterPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(revokePermission)

	kubeClient = kubeClientMock.NewMockKubernetesClient(ctl)

	cfg, err := configs.NewConfig(fmt.Sprintf(`
nautes:
  tenantName: "%s"
secret:
  vault:
    mountPath: "tenant"
`, tenantName))
	Expect(err).Should(BeNil())

	ctx = context.Background()
	ctx = context.WithValue(ctx, vaultclient.ContextKeyConfig, *cfg)
	initOpts := func(vs *vaultclient.VaultClient) error {
		vs.Vault = vaultRawClient
		vs.VaultProxy = secClient
		vs.VaultAuth = authClient
		vs.VaultAuthGrant = grantClient
		vs.GetKubeClient = func(kubeconfig string) (vaultclient.KubernetesClient, error) {
			return kubeClient, nil
		}
		vs.Client = k8sClient
		return nil
	}
	vaultStore, err = vaultclient.NewVaultClientWithOpts(ctx, cfg, initOpts)
	Expect(err).Should(BeNil())
}

var parentCluster, hostCluster *nautescrd.Cluster
var cluster *nautescrd.Cluster
var vaultStore *vaultclient.VaultClient
var vclusterSec corev1.Secret
var vaultAuthServiceAccount corev1.ServiceAccount
var vaultAuthSecret corev1.Secret
var nautesConfig *configs.Config
var testKubeconfig string
var CAPublic string
var KeyPrivate string
var KeyPublic string
var InitError error

func initData() {
	hostCluster = &nautescrd.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host",
			Namespace: "nautes",
		},
		Spec: nautescrd.ClusterSpec{
			ApiServer:   "https://127.0.0.1:6443",
			ClusterType: nautescrd.CLUSTER_TYPE_PHYSICAL,
			Usage:       nautescrd.CLUSTER_USAGE_HOST,
			HostCluster: "",
		},
	}

	parentCluster = &nautescrd.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant",
			Namespace: "nautes",
		},
		Spec: nautescrd.ClusterSpec{
			ApiServer:   "https://127.0.0.1:20001",
			ClusterType: nautescrd.CLUSTER_TYPE_VIRTUAL,
			Usage:       nautescrd.CLUSTER_USAGE_WORKER,
			HostCluster: hostCluster.Name,
		},
	}

	cluster = &nautescrd.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipeline",
			Namespace: "nautes",
		},
		Spec: nautescrd.ClusterSpec{
			ApiServer:   "https://127.0.0.1:30001",
			ClusterType: nautescrd.CLUSTER_TYPE_VIRTUAL,
			Usage:       nautescrd.CLUSTER_USAGE_WORKER,
			HostCluster: hostCluster.Name,
		},
	}

	vaultAuthServiceAccount = corev1.ServiceAccount{
		Secrets: []corev1.ObjectReference{{
			Name: "vault-auth",
		}},
	}

	vaultAuthSecret = corev1.Secret{
		Data: map[string][]byte{
			"token": []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"),
		},
	}

	testKubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUyTmpBMk16YzVOVEF3SGhjTk1qSXdPREUyTURneE9URXdXaGNOTXpJd09ERXpNRGd4T1RFdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUyTmpBMk16YzVOVEF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFTMG9JQTdqOUhmbUtTN1M4d2RmaHBDdUFmWVVWUGVjQ1hJdjQwb0FZTWQKTC85TFduMEVkSDB3cS9zaW8zSW9DNnc0a0tpbklXN3V6aU1XTGYrNkhjZXpvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVW5IVUx2SVRmUEYwRkE1NEVwcE1XCmhYdVRCM0F3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnUlEveXUvUEFUYXdtTW51Q0orM0IxeEl3c0JhTHRXUk4KNDJKVDR1ZWhvdFVDSVFDUk1pVFJ3MXNjeEpnSnV1YmF2YXQrRVBOWXpQcGpmSDJlR2gyNk1HU0NOdz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://localhost:8443
  name: my-vcluster
contexts:
- context:
    cluster: my-vcluster
    namespace: default
    user: my-vcluster
  name: my-vcluster
current-context: my-vcluster
kind: Config
preferences: {}
users:
- name: my-vcluster
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJY2NWWjlHQkJEZUF3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOall3TmpNM09UVXdNQjRYRFRJeU1EZ3hOakE0TVRreE1Gb1hEVEl6TURneApOakE0TVRreE1Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJDemJ6K2V5VVNVMWNNKzgKSGlyWkFBVXhsTDg0QW9xRm5OZTJ6cEZxYW9YRFZwT1AwSGlMcDJOZ1JYMzNPUWVOY0NROUhYbFRrQW9qSVF5dQpZNExsVkhTalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUWxYenZBUHNJUUQxbnJtN2s0bWdHcjQrYnVhVEFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlBWXpOSWU1dVVYSDNkZG1TbW1NSEJSSVA0RDNGOGVlZkhqZ3lYRGJuL01Md0loQUs0eDk1NGhlS0NQMlRpeApObUUyYlE1VmNVU1hJdWI3RUd1WHhtcjNENmN0Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUyTmpBMk16YzVOVEF3SGhjTk1qSXdPREUyTURneE9URXdXaGNOTXpJd09ERXpNRGd4T1RFdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUyTmpBMk16YzVOVEF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFUZERUbFlQQk1Qell5czMzYytkWmdBWERLQW16Vk1pcUZTYVd1c1lCVFUKdTlBZE1IWmJ5dENtSURJbkx4WHd2OUg3eEtSN2luY3dRMyttelkrV3h0UG5vMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUpWODd3RDdDRUE5WjY1dTVPSm9CCnErUG03bWt3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUloQU1rUVZVQlJ6Nm9PNFJFOHJZeWJGcFUrVklLdVZRS3cKY0lQN2dWOCsrNHdPQWlBZ24xY0tHQjd0aVgzdTBvb0liV1VDQlRFYnBhN0UxUEt3dEIrRVVGcCtKZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSURwOUd1VHZheWNqRC9zSXpMbUlNWms1d3ZLcS9SWElYODdVUTZsaDluNW1vQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFTE52UDU3SlJKVFZ3ejd3ZUt0a0FCVEdVdnpnQ2lvV2MxN2JPa1dwcWhjTldrNC9RZUl1bgpZMkJGZmZjNUI0MXdKRDBkZVZPUUNpTWhESzVqZ3VWVWRBPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
`
	vclusterSec = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
		Data: map[string][]byte{
			"config": []byte(testKubeconfig)},
	}

	CAPublic = `-----BEGIN CERTIFICATE-----
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
	KeyPrivate = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAyTpx4uLcoKb14vIdp9WN2mvkkNwa75Nas+HZYEXb1FNXUabJ
I092xCdRnIwyaZSPGs3FWIF6RBifZaBM/1zaJprOojFb9brMU2At2Eau8/eTu0aS
nXkOsjPAKkfU8PkxNlk6tRBL2wg2niid2FvAPbN3axdQlR1Zx44IrHtQSkPsE8O5
m+EJPWDJvqZ7m0M4QJwV3tVtSOzWMaptFDQJabM+g5IJ+XGNNI7QSU7D+/k4dT+j
aT9x59LS/7PPDacLCVN0dhkPXAN7CH+gPLvriVKJPe71ye9ztBqqGtBpH5KfBzPQ
oVSoKZmxxsf89JLLfZKm99202d5uF/PhelH+SwIDAQABAoIBAQCoX9vtYbAESM/T
zo0b4yfnzIGa6GEtd5ncjCzsTmfrmLSmoK0Ke7I/3Tp/iBuilmjLn8PyE5zvn764
NVJYFiR/Sud9dVmiGmRfm0mg/zvi7ZTSjfGeDC5M09qGRkaaP5h7BlyGJpWiN5Qj
8I5q/BK2ThWtKPwHWWDHBkShtijvibqbe9QoUNkSGJ7M5lHHDdr+MzQQphZ//Vc+
GZjDH7tLv8tlWtkEC+6/4kAiiKIom2bI3e4V6m3QyfA1FQjvT3jdj63OPocKozes
btD4IMw+VrDIGRN8B7sXEE7TP7rBIkymtgg1DZUIN7ujo08AqfyuUhPkN2wAB3k/
alCJ9fohAoGBAPePifsEWYg5635KNAYz2D27/dgTX6B4OM6/jAkpOH6zkDWhe/3Y
6hgQiATXAi4cvuxljrf1wbSmyk/Y94+DBXWWtJ91u8xUtc+3AKtBCk0wnw4cPh7h
BonPyn0wReFNhA4+jwWfvh/fgtqzZcVsJCWC3nhuqdnpg1MyH8LQYLqbAoGBANAW
kDH7+WMmb2T3E8HhMIyFeef40UMk8xfZMVEKT0380zbLxSblY+w9xv8cyh1uaedH
btICWb4d3hOlnbsd2yOxcelCBNPNt7u+S4AMwgHq8W87nqAbcHpqonheXaFKtzKk
IkBecDy8vbd820w3+jZw4Cz058J14PI9yaAfFW4RAn8vYkoGwc5hRLTOd2V9ym6Z
YmIz+YFUNa6p4//pwPoPRk9T9JTHAb3M3V0rj/va16Wzmby3eVKaQVJ39g9saKei
2jW4T9CiS5SBLYXzQX+3RpcrHDzHrEqUFjGrxJGbjjq4f0Dg0rKRZzakpbHVF93T
UDlE0+muzANW6UErCLd7AoGBAJmTDXjWbogupafucjZ07E/Jct8xU8AqVP8U3MDi
ywTTw059tVOvmL+SGHvP05tFEgQPREraUUFu6ae2Y2Ll9gWxwFBW2Rk4ipGVMEOh
Js4jh2yAo+GmXqz6Zk5P1upjKjHF0UGQcWViJuJ006S8632icNC9Lw7l0M73qwbx
6e8BAoGAdnrO3gLVUmfcs8PAvtHrXEs7COFUolTRyBy5EG4LoBnxhsIahal0JVBw
kL8tqeRkFjpX4Paek2bZpiVg7dBXpW/Glhg6WBB/8tLHoXyuKfoWs8IsZclKFL/1
ENGKZpZgvBtJH4IMMEdhikUc93q5t4DLStYA8JrFT8/u1+Wc0KY=
-----END RSA PRIVATE KEY-----`
	KeyPublic = `-----BEGIN CERTIFICATE-----
MIIECTCCAvGgAwIBAgIJAPVSPX/HLe42MA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCTEyNy4wLjAuMTAgFw0yMzAyMDExMDAwMThaGA8yMDUwMDYxOTEwMDAxOFow
VTELMAkGA1UEBhMCeHgxCzAJBgNVBAgMAnh4MQswCQYDVQQHDAJ4eDELMAkGA1UE
CgwCeHgxCzAJBgNVBAsMAnh4MRIwEAYDVQQDDAkxMjcuMC4wLjEwggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDJOnHi4tygpvXi8h2n1Y3aa+SQ3Brvk1qz
4dlgRdvUU1dRpskjT3bEJ1GcjDJplI8azcVYgXpEGJ9loEz/XNomms6iMVv1usxT
YC3YRq7z95O7RpKdeQ6yM8AqR9Tw+TE2WTq1EEvbCDaeKJ3YW8A9s3drF1CVHVnH
jgise1BKQ+wTw7mb4Qk9YMm+pnubQzhAnBXe1W1I7NYxqm0UNAlpsz6Dkgn5cY00
jtBJTsP7+Th1P6NpP3Hn0tL/s88NpwsJU3R2GQ9cA3sIf6A8u+uJUok97vXJ73O0
Gqoa0Gkfkp8HM9ChVKgpmbHGx/z0kst9kqb33bTZ3m4X8+F6Uf5LAgMBAAGjggEZ
MIIBFTBEBgNVHSMEPTA7gBTZ5PolU1zgia6jSzDczp42iUE6mqEYpBYwFDESMBAG
A1UEAwwJMTI3LjAuMC4xggkAzQISQAv6bC0wCQYDVR0TBAIwADALBgNVHQ8EBAMC
BDAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMIGVBgNVHREEgY0wgYqC
Cmt1YmVybmV0ZXOCEmt1YmVybmV0ZXMuZGVmYXVsdIIWa3ViZXJuZXRlcy5kZWZh
dWx0LnN2Y4Iea3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVygiRrdWJlcm5l
dGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWyHBH8AAAGHBH8AAAEwDQYJKoZI
hvcNAQELBQADggEBADKXj1/ctCEU4lMiHhQ6Ki6vI6/UflCcU5csCBSeDuRSDlLw
cx5jLj86OIJFPp3o0FNeRHm8T23RWofd5kkQMtoXzfCSTEA5HVE0pE5vXSa1pPyE
Iu1sBsO6X06tqR+ef98ioEa3ZyKNIJQ/EKBkbKa3gdq0Cvp3N330104C6JDtzJ2T
bbkWC1Ehnw/08prqZTMIix3YmSh3X73dU5NFxGkoIUjMRaqJhfRLtdYBDGFwFQon
cfZUnYiiu5Rhbe3vo8V32JfeWt3555hZMvwWYkaBuM7H5LC/0HOjCchcFDjtdvVv
Ne1AR0JiouXncOD3fUji7EjgMLTDYQR89DraG0o=
-----END CERTIFICATE-----`
	nautesConfig, InitError = configs.NewConfig(`
secret:
  vault:
    addr: "https://127.0.0.1:8200"
    mountPath: "tenant"
    token: "test"
    PKIPath: "/tmp"
    CABundle: |
      -----BEGIN CERTIFICATE-----
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
      -----END CERTIFICATE-----
`)
	Expect(InitError).Should(BeNil())
}

func initVault(data, roleBinding map[string]interface{}) {
	err := watiForSecretEngineInit()
	Expect(err).Should(BeNil())
	_, err = vaultRootClient.KVv2(CLUSTER).Put(ctx, "kubernetes/host/default/admin", data)
	Expect(err).Should(BeNil())
	err = vaultRootClient.Sys().EnableAuthWithOptions(parentCluster.Name, &vault.MountInput{Type: "kubernetes"})
	Expect(err).Should(BeNil())
	_, err = vaultRootClient.Logical().Write(fmt.Sprintf("auth/%s/role/RUNTIME", parentCluster.Name), roleBinding)
	Expect(err).Should(BeNil())
}

func cleanVault() {
	err := vaultRootClient.Sys().Unmount(CLUSTER)
	Expect(err).Should(BeNil())
	err = vaultRootClient.Sys().DisableAuth(parentCluster.Name)
	Expect(err).Should(BeNil())
}

func watiForSecretEngineInit() error {
	for i := 0; i < 5; i++ {
		_, err := vaultRootClient.KVv2(CLUSTER).Put(ctx, "wait-for-start", map[string]interface{}{"test": "test"})
		if err == nil {
			return nil
		}
		logger.V(1).Info("try to put secret failed", "error", err.Error())
		time.Sleep(time.Second * 3)
	}
	return fmt.Errorf("wait for secret engine start timeout")
}

func initK8SCreateMock() {
	kubeClient.EXPECT().GetCluster(gomock.Any(), hostCluster.Name, hostCluster.Namespace).Return(hostCluster, nil).AnyTimes()
	kubeClient.EXPECT().ListStatefulSets(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&appsv1.StatefulSetList{
			Items: []appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vcluster-tenant",
					},
				},
			},
		}, nil,
	).AnyTimes()
	kubeClient.EXPECT().GetSecret(gomock.Any(), "vc-vcluster-tenant", cluster.Name).Return(&vclusterSec, nil).AnyTimes()
	kubeClient.EXPECT().GetServiceAccount(gomock.Any(), "vault-auth", "vault").Return(&vaultAuthServiceAccount, nil).AnyTimes()
	kubeClient.EXPECT().GetSecret(gomock.Any(), "vault-auth", "vault").Return(&vaultAuthSecret, nil).AnyTimes()

}

func grantPermission(ctx context.Context, req vproxy.AuthGrantRequest, optxs ...interface{}) (*vproxy.GrantAuthrolePolicyReply, error) {
	authMeta, secretMeta, err := req.ConvertToAuthPolicyReqeuest()
	Expect(err).Should(BeNil())
	role, err := vaultRawClient.Logical().Read(authMeta.RolePath)
	Expect(err).Should(BeNil())
	Expect(role).ShouldNot(BeNil())
	hasKey := false
	for _, policy := range role.Data["token_policies"].([]interface{}) {
		if policy == secretMeta.PolicyName {
			hasKey = true
			break
		}
	}
	if !hasKey {
		role.Data["token_policies"] = append(role.Data["token_policies"].([]interface{}), secretMeta.PolicyName)
	}
	_, err = vaultRawClient.Logical().Write(authMeta.RolePath, role.Data)
	Expect(err).Should(BeNil())
	return nil, nil
}

func revokePermission(ctx context.Context, req vproxy.AuthGrantRequest, optxs ...interface{}) (*vproxy.RevokeAuthrolePolicyReply, error) {
	policyMeta, secretMeta, err := req.ConvertToAuthPolicyReqeuest()
	Expect(err).Should(BeNil())
	role, err := vaultRawClient.Logical().Read(policyMeta.RolePath)
	Expect(err).Should(BeNil())
	if role == nil {
		return nil, vproxy.ErrorResourceNotFound("resource not found")
	}
	var newPolicyList []interface{}
	for _, v := range role.Data["token_policies"].([]interface{}) {
		if v != secretMeta.PolicyName {
			newPolicyList = append(newPolicyList, v)
		}
	}
	role.Data["token_policies"] = newPolicyList
	_, err = vaultRawClient.Logical().Write(policyMeta.RolePath, role.Data)
	Expect(err).Should(BeNil())
	return nil, nil
}
