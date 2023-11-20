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

package hnc_test

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/multitenant/hnc"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"

	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	hierarchyConfigurationName = "hierarchy"
)

const (
	keyProductUserList = "productUsers"
	keySpaceUserList   = "spaceUsers"
)

var _ = Describe("HNC", func() {
	var mt component.MultiTenant
	var err error
	var ctx context.Context
	var productName string
	var spaces []string
	var users []string
	var seed string
	BeforeEach(func() {
		ctx = context.Background()
		seed = RandNum()
		productName = fmt.Sprintf("product-%s", seed)
		spaces = GenerateNames(fmt.Sprintf("space-%s-%%d", seed), 5)
		users = GenerateNames(fmt.Sprintf("user-%s-%%d", seed), 5)
		opts := v1alpha1.Component{
			Name:      productName,
			Namespace: "",
			Additions: map[string]string{
				hnc.OptKeyProductResourceKustomizeFileFolder: "./pipeline/template",
				hnc.OptKeyProductResourceRevision:            "main",
				hnc.OptKeySyncResourceTypes:                  "v1/ConfigMap, v1/Pod",
			},
		}

		db := &mockDB{
			product: v1alpha1.Product{
				Spec: v1alpha1.ProductSpec{
					Name:         fmt.Sprintf("name-%s", productName),
					MetaDataPath: "",
				}},
			cluster: v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1alpha1.ClusterSpec{
					ApiServer:                         "",
					ClusterType:                       "",
					ClusterKind:                       "",
					Usage:                             "",
					HostCluster:                       "",
					PrimaryDomain:                     "",
					WorkerType:                        v1alpha1.ClusterWorkTypePipeline,
					ComponentsList:                    v1alpha1.ComponentsList{},
					ReservedNamespacesAllowedProducts: map[string][]string{},
					ProductAllowedClusterResources:    map[string][]v1alpha1.ClusterResourceInfo{},
				}},
		}

		initInfo := component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &component.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName:            "",
			NautesResourceSnapshot: db,
			NautesConfig:           configs.Config{},
			Components: &component.ComponentList{
				Deployment: &mockDeployer{},
			},
		}
		mt, err = hnc.NewHNC(opts, &initInfo)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		for _, space := range spaces {
			err := mt.DeleteSpace(ctx, productName, space)
			Expect(err).Should(BeNil())

			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: space,
				},
			}

			ok, err := NamespaceIsNotExist(k8sClient, ns)
			Expect(err).Should(BeNil())
			Expect(ok).Should(BeTrue())
		}

		err := mt.DeleteProduct(ctx, productName)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}
		ok, err := NamespaceIsNotExist(k8sClient, ns)
		Expect(err).Should(BeNil())
		Expect(ok).Should(BeTrue())
	})

	It("create product", func() {
		err := mt.CreateProduct(ctx, productName)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: productName,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
		Expect(err).Should(BeNil())
		Expect(utils.IsBelongsToProduct(ns, productName)).Should(BeTrue())

		hncCFG := &hncv1alpha2.HNCConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config",
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(hncCFG), hncCFG)
		Expect(err).Should(BeNil())
		Expect(len(hncCFG.Spec.Resources)).Should(Equal(2))

	})

	It("can create space", func() {
		err := mt.CreateSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: spaces[0],
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
		Expect(err).Should(BeNil())
		Expect(utils.IsBelongsToProduct(ns, productName)).Should(BeTrue())

		hierarchyCfg := &hncv1alpha2.HierarchyConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hierarchyConfigurationName,
				Namespace: ns.Name,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(hierarchyCfg), hierarchyCfg)
		Expect(err).Should(BeNil())
		Expect(hierarchyCfg.Spec.Parent).Should(Equal(productName))

		spaceStatus, err := mt.GetSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())
		Expect(spaceStatus.Product).Should(Equal(productName))
		Expect(spaceStatus.Name).Should(Equal(spaces[0]))
	})

	It("can create user and add user to space", func() {
		err := mt.CreateProduct(ctx, productName)
		Expect(err).Should(BeNil())

		err = mt.CreateSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		err = mt.CreateAccount(ctx, productName, users[0])
		Expect(err).Should(BeNil())

		err = mt.AddSpaceUser(ctx, component.PermissionRequest{
			RequestScope: component.RequestScopeAccount,
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    spaces[0],
			},
			User:       users[0],
			Permission: component.Permission{},
		})
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: spaces[0],
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
		Expect(err).Should(BeNil())

		userList := ns.Annotations[keySpaceUserList]
		Expect(userList).Should(Equal(users[0]))

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      users[0],
				Namespace: spaces[0],
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		Expect(err).Should(BeNil())
	})

	It("can delete user from space", func() {
		err := mt.CreateProduct(ctx, productName)
		Expect(err).Should(BeNil())

		err = mt.CreateSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		err = mt.CreateAccount(ctx, productName, users[0])
		Expect(err).Should(BeNil())

		request := component.PermissionRequest{
			RequestScope: component.RequestScopeAccount,
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    spaces[0],
			},
			User:       users[0],
			Permission: component.Permission{},
		}

		err = mt.AddSpaceUser(ctx, request)
		Expect(err).Should(BeNil())

		err = mt.DeleteSpaceUser(ctx, request)
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: spaces[0],
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
		Expect(err).Should(BeNil())

		userList := ns.Annotations[keySpaceUserList]
		Expect(userList).Should(Equal(""))

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      users[0],
				Namespace: spaces[0],
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())
	})

	It("will remove space permission when user is deleted", func() {
		err := mt.CreateProduct(ctx, productName)
		Expect(err).Should(BeNil())

		err = mt.CreateSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		hncParentConfig := &hncv1alpha2.HierarchyConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hierarchyConfigurationName,
				Namespace: productName,
			},
		}

		err = k8sClient.Create(ctx, hncParentConfig)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(hncParentConfig), hncParentConfig)
		Expect(err).Should(BeNil())

		hncParentConfig.Status.Children = []string{spaces[0]}
		err = k8sClient.Update(ctx, hncParentConfig)
		Expect(err).Should(BeNil())

		err = mt.CreateAccount(ctx, productName, users[0])
		Expect(err).Should(BeNil())

		request := component.PermissionRequest{
			RequestScope: component.RequestScopeAccount,
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    spaces[0],
			},
			User:       users[0],
			Permission: component.Permission{},
		}

		err = mt.AddSpaceUser(ctx, request)
		Expect(err).Should(BeNil())

		err = mt.DeleteAccount(ctx, productName, users[0])
		Expect(err).Should(BeNil())

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: spaces[0],
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
		Expect(err).Should(BeNil())

		userList := ns.Annotations[keySpaceUserList]
		Expect(userList).Should(Equal(""))

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      users[0],
				Namespace: spaces[0],
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(hncParentConfig), hncParentConfig)
		Expect(err).Should(BeNil())

		hncParentConfig.Status.Children = []string{}
		err = k8sClient.Update(ctx, hncParentConfig)
		Expect(err).Should(BeNil())
	})

	It("can list spaces", func() {
		err := mt.CreateProduct(ctx, productName)
		Expect(err).Should(BeNil())

		err = mt.CreateSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		err = mt.CreateAccount(ctx, productName, users[0])
		Expect(err).Should(BeNil())

		request := component.PermissionRequest{
			RequestScope: component.RequestScopeAccount,
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    spaces[0],
			},
			User:       users[0],
			Permission: component.Permission{},
		}

		err = mt.AddSpaceUser(ctx, request)
		Expect(err).Should(BeNil())

		spaceStatus, err := mt.ListSpaces(ctx, productName)
		Expect(len(spaceStatus)).Should(Equal(2))
		Expect(spaceStatus[1].Accounts[0]).Should(Equal(users[0]))

		account, err := mt.GetAccount(ctx, productName, users[0])
		Expect(len(account.Spaces)).Should(Equal(1))
		Expect(account.Spaces[0]).Should(Equal(spaces[0]))

		err = mt.DeleteSpace(ctx, productName, spaces[0])
		Expect(err).Should(BeNil())

		spaceStatus, err = mt.ListSpaces(ctx, productName)
		Expect(len(spaceStatus)).Should(Equal(1))
		Expect(spaceStatus[0].Name).Should(Equal(productName))
	})
})
