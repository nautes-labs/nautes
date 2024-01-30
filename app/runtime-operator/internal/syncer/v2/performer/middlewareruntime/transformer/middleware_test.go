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

package transformer_test

import (
	"fmt"
	"strings"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/transformer"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("MiddlewareTransformRule", func() {
	var (
		providerType   string
		middlewareType string
		implementation string
		rule           []byte
	)

	AfterEach(func() {
		ClearMiddlewareTransformRule()
	})

	Context("NewMiddlewareTransformRule", func() {
		BeforeEach(func() {
			providerType = "test-provider"
			middlewareType = "test-middlewareType"
			implementation = "test-implementation"
			ruleTmpl := `
providerType: %s
middlewareType: %s
implementation: %s
resources: |
  - path: spec.replicas
    value: 1
`
			rule = []byte(fmt.Sprintf(ruleTmpl, providerType, middlewareType, implementation))

		})

		It("should return a new MiddlewareTransformRule", func() {
			ruleString := string(rule)
			ruleString = strings.TrimSpace(ruleString)
			wanted := MiddlewareTransformRule{
				ProviderType:   providerType,
				MiddlewareType: middlewareType,
				Implementation: implementation,
				Resources:      []string{"- path: spec.replicas\n  value: 1"},
			}
			tr := &MiddlewareTransformRule{}
			err := yaml.Unmarshal(rule, tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(*tr).To(Equal(wanted))
		})
	})

	var _ = Describe("ConvertMiddlewareToResources", func() {
		BeforeEach(func() {
			// Prepare the transform rule.
			rule := MiddlewareTransformRule{
				ProviderType:   providerType,
				MiddlewareType: middlewareType,
				Implementation: implementation,
				Resources: []string{
					`metadata:
  name: test-config
  type: ConfigMap
spec:
  key: value`,
					// Add other resources as needed.
				},
			}
			err := AddMiddlewareTransformRule(rule)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should convert middleware to resources successfully", func() {
			middleware := v1alpha1.Middleware{
				Type:           middlewareType,
				Implementation: implementation,
				// Add other fields as needed.
			}
			res, err := ConvertMiddlewareToResources(providerType, middleware)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(res)).To(Equal(1))

			// Check the first resource.
			resource := res[0]
			Expect(resource.GetName()).To(Equal("test-config"))
			Expect(resource.GetType()).To(Equal("ConfigMap"))
			Expect(resource.GetResourceAttributes()).To(Equal(map[string]string{
				"key": "value",
			}))
			Expect(resource.GetStatus()).To(BeNil())
		})

		It("should return error when no rule found", func() {
			middleware := v1alpha1.Middleware{
				Type:           middlewareType,
				Implementation: implementation,
				// Add other fields as needed.
			}
			_, err := ConvertMiddlewareToResources("non-existent-provider", middleware)
			Expect(err).To(HaveOccurred())
			Expect(runtimeerr.IsTransformRuleNotFoundError(err)).To(BeTrue())
		})
	})

	var _ = Describe("DeleteMiddlewareTransformRule", func() {
		BeforeEach(func() {
			providerType = "test-provider"
			middlewareType = "test-middlewareType"
			implementation = "test-implementation"

			// Add a rule to the middlewareTransformRules map.
			rule := MiddlewareTransformRule{
				ProviderType:   providerType,
				MiddlewareType: middlewareType,
				Implementation: implementation,
				Resources: []string{
					`metadata:
	  name: test-config
	  type: ConfigMap
	spec:
	  key: value`,
				},
			}
			err := AddMiddlewareTransformRule(rule)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete the middleware transform rule successfully", func() {
			err := RemoveMiddlewareTransformRule(providerType, middlewareType, implementation)
			Expect(err).ToNot(HaveOccurred())

			// Check that the rule has been deleted.
			_, err = GetMiddlewareTransformRule(providerType, middlewareType, implementation)
			Expect(err).To(HaveOccurred())
			Expect(runtimeerr.IsTransformRuleNotFoundError(err)).To(BeTrue())
		})

		It("should return error when the provider does not exist", func() {
			err := RemoveMiddlewareTransformRule("non-existent-provider", middlewareType, implementation)
			Expect(err).To(HaveOccurred())
			Expect(runtimeerr.IsTransformRuleNotFoundError(err)).To(BeTrue())
		})

		It("should return error when the middleware type does not exist", func() {
			err := RemoveMiddlewareTransformRule(providerType, "non-existent-middlewareType", implementation)
			Expect(err).To(HaveOccurred())
			Expect(runtimeerr.IsTransformRuleNotFoundError(err)).To(BeTrue())
		})

		It("should return error when the implementation does not exist", func() {
			err := RemoveMiddlewareTransformRule(providerType, middlewareType, "non-existent-implementation")
			Expect(err).To(HaveOccurred())
			Expect(runtimeerr.IsTransformRuleNotFoundError(err)).To(BeTrue())
		})
	})
})
