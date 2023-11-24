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

package handler_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/handler"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/pkg/nautesconst"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArgoCD", func() {
	var ctx context.Context
	var seed string
	var space component.Space
	var initInfo *component.ComponentInitInfo

	BeforeEach(func() {
		ctx = context.TODO()
		seed = RandNum()

		initInfo = &component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &component.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			Components: &component.ComponentList{
				SecretSync: &mockSecretSyncer{},
			},
		}
		space = component.Space{
			ResourceMetaData: component.ResourceMetaData{
				Product: fmt.Sprintf("product-%s", seed),
				Name:    "default",
			},
			SpaceType: component.SpaceTypeKubernetes,
			Kubernetes: &component.SpaceKubernetes{
				Namespace: "default",
			},
		}

	})
	It("can create ssh key", func() {
		hdl, err := handler.NewHandler(initInfo)
		Expect(err).Should(BeNil())

		req := component.RequestResource{
			Type:         component.ResourceTypeCodeRepoSSHKey,
			ResourceName: fmt.Sprintf("rs-%s", seed),
			SSHKey: &component.ResourceRequestSSHKey{
				SecretInfo: component.SecretInfo{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: "1",
						ID:           "2",
						User:         "3",
						Permission:   "readonly",
					},
				},
			},
		}
		err = hdl.CreateResource(ctx, space, component.AuthInfo{}, req)
		Expect(err).Should(BeNil())

		err = hdl.DeleteResource(ctx, space, req)
		Expect(err).Should(BeNil())
	})

	It("can create access token", func() {
		hdl, err := handler.NewHandler(initInfo)
		Expect(err).Should(BeNil())

		req := component.RequestResource{
			Type:         component.ResourceTypeCodeRepoAccessToken,
			ResourceName: fmt.Sprintf("rs-%s", seed),
			AccessToken: &component.ResourceRequestAccessToken{
				SecretInfo: component.SecretInfo{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: "1",
						ID:           "2",
						User:         "3",
						Permission:   "accesstoken",
					},
				},
			},
		}
		err = hdl.CreateResource(ctx, space, component.AuthInfo{}, req)
		Expect(err).Should(BeNil())

		err = hdl.DeleteResource(ctx, space, req)
		Expect(err).Should(BeNil())
	})

	Context("CA file", func() {
		var fakeURL = "https://127.0.0.1"
		var nautesHome = "/tmp/handle-test"
		var caFilePath = "/tmp/handle-test/cert/127.0.0.1.crt"
		BeforeEach(func() {
			fakeCA := `
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
`
			os.Setenv(nautesconst.EnvNautesHome, nautesHome)
			err := os.MkdirAll(filepath.Join(nautesHome, "cert"), 0700)
			Expect(err).Should(BeNil())
			err = os.WriteFile(caFilePath, []byte(fakeCA), 0400)
			Expect(err).Should(BeNil())
		})
		AfterEach(func() {
			err := os.RemoveAll(nautesHome)
			Expect(err).Should(BeNil())
			err = os.Unsetenv(nautesconst.EnvNautesHome)
			Expect(err).Should(BeNil())
		})

		It("can create ca", func() {
			hdl, err := handler.NewHandler(initInfo)
			Expect(err).Should(BeNil())

			req := component.RequestResource{
				Type:         component.ResourceTypeCAFile,
				ResourceName: fmt.Sprintf("rs-%s", seed),
				CAFile: &component.ResourceRequestCAFile{
					URL: fakeURL,
				},
			}
			err = hdl.CreateResource(ctx, space, component.AuthInfo{}, req)
			Expect(err).Should(BeNil())

			err = hdl.DeleteResource(ctx, space, req)
			Expect(err).Should(BeNil())
		})
	})
})
