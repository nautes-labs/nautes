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

package externalsecret_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	externalsecretv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/secretsync/externalsecret"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("External secret", func() {
	var ctx context.Context
	var secReq component.SecretRequest
	var secSync component.SecretSync
	var productName string
	var nsName string = "default"
	var userName string
	var clusterName string
	var seed string
	var vaultURL string = "https://127.0.0.1:8000"
	var vaultCAPath string = "/opt/nautes/cert/127.0.0.1_8000.crt"
	var ca string = `-----BEGIN CERTIFICATE-----
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
	var es *externalsecretv1alpha1.ExternalSecret
	var ss *externalsecretv1alpha1.SecretStore

	BeforeEach(func() {
		var err error
		ctx = context.TODO()
		seed = RandNum()
		productName = fmt.Sprintf("product-%s", seed)
		userName = fmt.Sprintf("user-%s", seed)
		clusterName = fmt.Sprintf("cluster-%s", seed)
		secReq = component.SecretRequest{
			Name: fmt.Sprintf("reqname-%s", seed),
			Source: component.SecretInfo{
				Type: component.SecretTypeCodeRepo,
				CodeRepo: &component.CodeRepo{
					ProviderType: "1",
					ID:           "2",
					User:         "3",
					Permission:   "4",
				},
			},
			AuthInfo: &component.AuthInfo{
				OriginName:      "vault",
				AccountName:     userName,
				AuthType:        component.AuthTypeKubernetesServiceAccount,
				ServiceAccounts: []component.AuthInfoServiceAccount{},
			},
			Destination: component.SecretRequestDestination{
				Name: fmt.Sprintf("secret-%s", seed),
				Space: component.Space{
					ResourceMetaData: component.ResourceMetaData{
						Product: productName,
						Name:    nsName,
					},
					SpaceType: component.SpaceTypeKubernetes,
					Kubernetes: &component.SpaceKubernetes{
						Namespace: nsName,
					},
				},
				Format: "",
			},
		}

		err = os.WriteFile(vaultCAPath, []byte(ca), fs.ModeSticky)
		Expect(err).Should(BeNil())

		initInfo := &component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &component.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName: clusterName,
			NautesConfig: configs.Config{
				Secret: configs.SecretRepo{
					RepoType: configs.SECRET_STORE_VAULT,
					Vault: configs.Vault{
						Addr: vaultURL,
					},
				},
			},
		}
		secSync, err = externalsecret.NewExternalSecret(v1alpha1.Component{}, initInfo)
		Expect(err).Should(BeNil())

		es = &externalsecretv1alpha1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("es-%s", secReq.Name),
				Namespace: nsName,
			},
		}

		ss = &externalsecretv1alpha1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ss-%s", secReq.Name),
				Namespace: nsName,
			},
		}
	})

	AfterEach(func() {
		err := os.Remove(vaultCAPath)
		Expect(err).Should(BeNil())
	})

	It("can create secret store and external secret by secret request", func() {
		err := secSync.CreateSecret(ctx, secReq)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(es), es)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ss), ss)
		Expect(err).Should(BeNil())
	})

	It("can delete secret store and external secret by secret request", func() {
		err := secSync.CreateSecret(ctx, secReq)
		Expect(err).Should(BeNil())

		err = secSync.RemoveSecret(ctx, secReq)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(es), es)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ss), ss)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

	})
})
