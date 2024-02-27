// Copyright 2024 Nautes Authors
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

package middlewareruntime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/transformer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/pkg/middlewaresecret"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Middleware deployment", func() {
	var (
		seed                    string
		implementation          = "MockImplementation"
		middlewareType          = "MockMiddleware"
		resourceTypeA           = "MockResourceA"
		resourceTypeB           = "MockResourceB"
		deploymentInfo          *MiddlewareDeploymentInfo
		ctx                     context.Context
		middleware              v1alpha1.Middleware
		middlewareStatus        *v1alpha1.MiddlewareStatus
		mockResourceTransformer transformer.ResourceTransformer
		res                     resources.CommonResource
		resourceStatus          map[string]string
		runtimeInfo             RuntimeInfo
	)

	BeforeEach(func() {
		_ = resourceTypeA
		_ = resourceTypeB

		var err error
		seed = testutils.RandNum()
		ctx = context.Background()

		providerInfo := component.ProviderInfo{
			Type: providerType,
		}

		middleware = v1alpha1.Middleware{
			Name:           "test-middleware-" + seed,
			Type:           middlewareType,
			Implementation: implementation,
		}

		res = resources.CommonResource{
			ResourceMetadata: resources.ResourceMetadata{
				Type: resourceTypeA,
				Name: fmt.Sprintf("resource-%s", middleware.Name),
			},
		}

		resourceTmpl := `
metadata:
  name: resource-{{ .Name }}
  type: %s
`
		middlewareRule := transformer.MiddlewareTransformRule{
			ProviderType:   providerType,
			MiddlewareType: middleware.Type,
			Implementation: implementation,
			Resources:      []string{fmt.Sprintf(resourceTmpl, resourceTypeA)},
		}
		err = transformer.AddMiddlewareTransformRule(middlewareRule)
		Expect(err).To(BeNil())

		resourceStatus = map[string]string{}
		mockResourceTransformer = transformer.ResourceTransformer{
			ProviderType: providerType,
			ResourceType: resourceTypeA,
			CallerType:   callerType,
			Create: &mockRequestTransformer{
				PostResp:  []byte("create"),
				PostErr:   nil,
				ParseResp: &resourceStatus,
			},
			Update: &mockRequestTransformer{
				PostResp:  []byte("update"),
				PostErr:   nil,
				ParseResp: &resourceStatus,
			},
			Delete: &mockRequestTransformer{
				PostResp: []byte("delete"),
				PostErr:  nil,
			},
			Get: &mockRequestTransformer{
				PostResp:  []byte("get"),
				PostErr:   nil,
				ParseResp: &resourceStatus,
			},
		}
		err = transformer.AddResourceTransformer(mockResourceTransformer)
		Expect(err).To(BeNil())

		deploymentInfo, err = NewMiddlewareDeploymentInfo(providerInfo, middleware)
		Expect(err).To(BeNil())

		middlewareStatus = nil

		runtimeInfo = RuntimeInfo{
			Name:            "",
			Product:         "",
			Environment:     "",
			Cluster:         "",
			NautesNamespace: "nautes",
		}
		Expect(err).To(BeNil())
		SecretIndexDirectory = &middlewaresecret.Mock{}
	},
	)
	AfterEach(func() {
		transformer.ClearResourceTransformer()
		transformer.ClearMiddlewareTransformRule()
	})

	Context("when creating a middleware", func() {

		It("should create a middleware", func() {
			newResourceStatus := map[string]string{
				"name":   res.Name,
				"action": "create",
			}

			mockResourceTransformer.Create = &mockRequestTransformer{
				PostResp:  []byte("create"),
				PostErr:   nil,
				ParseResp: &newResourceStatus,
			}
			err := transformer.UpdateResourceTransformer(mockResourceTransformer)
			Expect(err).To(BeNil())

			res.Status = resources.ResourceStatus{Properties: newResourceStatus}
			commonMiddlewareStatus := CommonMiddlewareStatus{
				ResourceStatus: []resources.Resource{&res},
			}
			resByte, err := json.Marshal(commonMiddlewareStatus)
			Expect(err).To(BeNil())

			status, err := DeployMiddleware(ctx, middleware, middlewareStatus, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			wanted := &v1alpha1.MiddlewareStatus{
				Middleware: middleware,
				Status: &runtime.RawExtension{
					Raw: resByte,
				},
			}
			Expect(status).To(Equal(wanted))
		})

		It("if status is changed, should update a middleware", func() {
			resourceStatus["name"] = res.Name
			resourceStatus["action"] = "create"

			status, err := DeployMiddleware(ctx, middleware, middlewareStatus, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			delete(resourceStatus, "action")

			newResourceStatus := map[string]string{
				"name":   res.Name,
				"action": "update",
			}
			mockResourceTransformer.Update = &mockRequestTransformer{
				PostResp:  []byte("update"),
				PostErr:   nil,
				ParseResp: &newResourceStatus,
			}
			err = transformer.UpdateResourceTransformer(mockResourceTransformer)
			Expect(err).To(BeNil())

			res.Status = resources.ResourceStatus{Properties: newResourceStatus}
			commentMiddlewareStatus := CommonMiddlewareStatus{
				ResourceStatus: []resources.Resource{&res},
			}
			resByte, err := json.Marshal(commentMiddlewareStatus)
			Expect(err).To(BeNil())

			status, err = DeployMiddleware(ctx, middleware, status, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			wanted := &v1alpha1.MiddlewareStatus{
				Middleware: middleware,
				Status: &runtime.RawExtension{
					Raw: resByte,
				},
			}
			Expect(status).To(Equal(wanted))
		})

		It("if middleware is changed, should update resources", func() {
			status, err := DeployMiddleware(ctx, middleware, middlewareStatus, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			middleware.Name = fmt.Sprintf("%s-2", middleware.Name)

			status, err = DeployMiddleware(ctx, middleware, status, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			res2 := resources.CommonResource{
				ResourceMetadata: resources.ResourceMetadata{
					Type: resourceTypeA,
					Name: fmt.Sprintf("resource-%s", middleware.Name),
				},
				Status: resources.ResourceStatus{},
			}
			commentMiddlewareStatus := CommonMiddlewareStatus{
				ResourceStatus: []resources.Resource{&res2},
			}
			res2Byte, err := json.Marshal(commentMiddlewareStatus)
			Expect(err).To(BeNil())

			wanted := &v1alpha1.MiddlewareStatus{
				Middleware: middleware,
				Status: &runtime.RawExtension{
					Raw: res2Byte,
				},
			}
			Expect(status).To(Equal(wanted))
		})
	})

	Context("when delete a middleware", func() {
		It("if middleware doesn't deployed, should delete success", func() {
			err := DeleteMiddleware(ctx, middlewareStatus, *deploymentInfo)
			Expect(err).To(BeNil())
		})

		It("should delete middleware success", func() {
			status, err := DeployMiddleware(ctx, middleware, middlewareStatus, *deploymentInfo, runtimeInfo)
			Expect(err).To(BeNil())

			err = DeleteMiddleware(ctx, status, *deploymentInfo)
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("Sort dependencies", func() {
	resA := &resources.CommonResource{
		ResourceMetadata: resources.ResourceMetadata{
			Type: "unknown",
			Name: "a",
		},
		ResourceDependencies: []resources.ResourceMetadata{
			{
				Type: "unknown",
				Name: "b",
			},
		},
	}
	resB := &resources.CommonResource{
		ResourceMetadata: resources.ResourceMetadata{
			Type: "unknown",
			Name: "b",
		},
		ResourceDependencies: []resources.ResourceMetadata{},
	}
	resC := &resources.CommonResource{
		ResourceMetadata: resources.ResourceMetadata{
			Type: "unknown",
			Name: "c",
		},
		ResourceDependencies: []resources.ResourceMetadata{
			{
				Type: "unknown",
				Name: "a",
			},
			{
				Type: "unknown",
				Name: "b",
			},
		},
	}
	resD := &resources.CommonResource{
		ResourceMetadata: resources.ResourceMetadata{
			Type: "unknown",
			Name: "d",
		},
		ResourceDependencies: []resources.ResourceMetadata{
			{
				Type: "unknown",
				Name: "c",
			},
		},
	}
	resD2 := &resources.CommonResource{
		ResourceMetadata: resources.ResourceMetadata{
			Type: "unknown-02",
			Name: "d",
		},
		ResourceDependencies: []resources.ResourceMetadata{
			{
				Type: "unknown",
				Name: "b",
			},
		},
	}

	Context("Create or update resources", func() {

		It("a depended on b, should sort b before a", func() {

			res := map[string]resources.Resource{
				resA.GetUniqueID(): resA,
				resB.GetUniqueID(): resB,
			}
			want := ResourceActions{
				{
					Action:   ResourceActionCreate,
					Resource: resB,
				},
				{
					Action:   ResourceActionCreate,
					Resource: resA,
				},
			}
			actions := NewActionListCreateOrUpdate(res, nil)
			Expect(actions).To(Equal(want))
		})

		It("c depended on a and b, should sort b before a and c after a and b", func() {
			res := map[string]resources.Resource{
				resC.GetUniqueID(): resC,
				resA.GetUniqueID(): resA,
				resB.GetUniqueID(): resB,
			}
			want := ResourceActions{
				{
					Action:   ResourceActionCreate,
					Resource: resB,
				},
				{
					Action:   ResourceActionCreate,
					Resource: resA,
				},
				{
					Action:   ResourceActionCreate,
					Resource: resC,
				},
			}
			actions := NewActionListCreateOrUpdate(res, nil)
			log.Println("Sort result:")
			for _, action := range actions {
				log.Println(action.Resource.GetUniqueID())
			}
			Expect(actions).To(Equal(want))
		})

		It("被依赖的应该会优先创建", func() {
			res := map[string]resources.Resource{
				resD2.GetUniqueID(): resD2,
				resA.GetUniqueID():  resA,
				resB.GetUniqueID():  resB,
				resC.GetUniqueID():  resC,
				resD.GetUniqueID():  resD,
			}

			actions := NewActionListCreateOrUpdate(res, nil)
			log.Println("Sort result:")
			for _, action := range actions {
				log.Println(action.Resource.GetUniqueID())
			}
			var resAIndex int
			var resBIndex int
			var resCIndex int
			var resDIndex int
			var resD2Index int
			for i, action := range actions {
				if action.Resource.GetUniqueID() == resA.GetUniqueID() {
					resAIndex = i
				}
				if action.Resource.GetUniqueID() == resB.GetUniqueID() {
					resBIndex = i
				}
				if action.Resource.GetUniqueID() == resC.GetUniqueID() {
					resCIndex = i
				}
				if action.Resource.GetUniqueID() == resD.GetUniqueID() {
					resDIndex = i
				}
				if action.Resource.GetUniqueID() == resD2.GetUniqueID() {
					resD2Index = i
				}
			}
			Expect(resAIndex > resBIndex).Should(BeTrue())
			Expect(resCIndex > resAIndex && resCIndex > resBIndex).Should(BeTrue())
			Expect(resDIndex > resCIndex).Should(BeTrue())
			Expect(resD2Index > resBIndex).Should(BeTrue())
		})

		It("如果依赖的资源不存在，视为没有依赖, 顺序应该是D，B", func() {
			res := map[string]resources.Resource{
				resD.GetUniqueID(): resD,
			}
			want := ResourceActions{
				{
					Action:   ResourceActionCreate,
					Resource: resD,
				},
			}
			actions := NewActionListCreateOrUpdate(res, nil)
			Expect(actions).To(Equal(want))
		})
	})
	Context("Delete resources", func() {
		It("a depended on b, should sort a before b", func() {
			res := map[string]resources.Resource{
				resA.GetUniqueID(): resA,
				resB.GetUniqueID(): resB,
			}
			want := ResourceActions{
				{
					Action:   ResourceActionDelete,
					Resource: resA,
				},
				{
					Action:   ResourceActionDelete,
					Resource: resB,
				},
			}
			actions := NewActionListDelete(res)
			Expect(actions).To(Equal(want))
		})
	})
})
