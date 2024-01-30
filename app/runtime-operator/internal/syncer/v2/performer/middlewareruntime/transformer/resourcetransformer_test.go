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
	"os"
	"path/filepath"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/caller/http"
	. "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/transformer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
)

var _ = Describe("LoadResourceTransformers", func() {
	var tempDir string
	var providerName string
	var resourceType string
	var seed string

	BeforeEach(func() {
		var err error
		seed = testutils.RandNum()
		providerName = fmt.Sprintf("provider-%s", seed)
		resourceType = fmt.Sprintf("resource-%s", seed)

		tempDir, err = os.MkdirTemp("", "transformers")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("should load resource transformers correctly", func() {
		rule := `
providerType: %s
callerType: %s
resourceType: %s
create:
  generation:
    request: POST
    body: '{"key": "value"}'
    uri: '/create'
    header:
      Content-Type: application/json
  parse:
  - name: id
    path: /data/id
get:
  generation:
    request: GET
    uri: '/get/{id}'
    header:
      Accept: application/json
  parse:
  - name: data
    path: /data
update:
  generation:
    request: PUT
    body: '{"key": "new_value"}'
    uri: '/update/{id}'
    header:
      Content-Type: application/json
  parse:
  - name: result
    path: /data/result
delete:
  generation:
    request: DELETE
    uri: '/delete/{id}'
    header:
      Content-Type: application/json
`
		rule = fmt.Sprintf(rule, providerName, component.CallerTypeHTTP, resourceType)

		createBody := `{"key": "value"}`
		updateBody := `{"key": "new_value"}`
		wanted := &ResourceTransformer{
			ProviderType: providerName,
			ResourceType: resourceType,
			CallerType:   component.CallerTypeHTTP,
			Create: &RequestTransformer{
				CallerType: component.CallerTypeHTTP,
				HTTP: RequestTransformerHTTP{
					TransformRule: http.RequestTransformerRule{
						RequestGenerationRule: http.RequestGenerationRule{
							Request: "POST",
							Body:    &createBody,
							URI:     "/create",
							Header: map[string]utils.StringOrStringArray{
								"Content-Type": utils.NewStringOrStringArray("application/json"),
							},
						},
						ResponseParseRule: []http.ResponseParseRule{
							{
								KeyName: "id",
								KeyPath: "/data/id",
							},
						},
					},
				},
			},
			Get: &RequestTransformer{
				CallerType: component.CallerTypeHTTP,
				HTTP: RequestTransformerHTTP{
					TransformRule: http.RequestTransformerRule{
						RequestGenerationRule: http.RequestGenerationRule{
							Request: "GET",
							URI:     "/get/{id}",
							Header: map[string]utils.StringOrStringArray{
								"Accept": utils.NewStringOrStringArray("application/json"),
							},
						},
						ResponseParseRule: []http.ResponseParseRule{
							{
								KeyName: "data",
								KeyPath: "/data",
							},
						},
					},
				},
			},
			Update: &RequestTransformer{
				CallerType: component.CallerTypeHTTP,
				HTTP: RequestTransformerHTTP{
					TransformRule: http.RequestTransformerRule{
						RequestGenerationRule: http.RequestGenerationRule{
							Request: "PUT",
							Body:    &updateBody,
							URI:     "/update/{id}",
							Header: map[string]utils.StringOrStringArray{
								"Content-Type": utils.NewStringOrStringArray("application/json"),
							},
						},
						ResponseParseRule: []http.ResponseParseRule{
							{
								KeyName: "result",
								KeyPath: "/data/result",
							},
						},
					},
				},
			},
			Delete: &RequestTransformer{
				CallerType: component.CallerTypeHTTP,
				HTTP: RequestTransformerHTTP{
					TransformRule: http.RequestTransformerRule{
						RequestGenerationRule: http.RequestGenerationRule{
							Request: "DELETE",
							URI:     "/delete/{id}",
							Header: map[string]utils.StringOrStringArray{
								"Content-Type": utils.NewStringOrStringArray("application/json"),
							},
						},
					},
				},
			},
		}

		err := os.WriteFile(tempDir+fmt.Sprintf("/%s-%s-%s", providerName, component.CallerTypeHTTP, resourceType), []byte(rule), 0644)
		Expect(err).NotTo(HaveOccurred())

		err = LoadResourceTransformers(WithTransformRulesFilePath(tempDir))
		Expect(err).NotTo(HaveOccurred())
		tf, err := GetResourceTransformer(providerName, component.CallerTypeHTTP, resourceType)
		Expect(err).NotTo(HaveOccurred())
		Expect(tf.ProviderType).Should(Equal(wanted.ProviderType))
		Expect(tf.ResourceType).Should(Equal(wanted.ResourceType))
		Expect(tf.CallerType).Should(Equal(wanted.CallerType))
		Expect(tf.Create).Should(Equal(wanted.Create))
		Expect(tf.Get).Should(Equal(wanted.Get))
		Expect(tf.Update).Should(Equal(wanted.Update))
		Expect(tf.Delete).Should(Equal(wanted.Delete))

	})

	It("should return an error if the rule file cannot be parsed", func() {
		// Create a temporary directory for the test
		tempDir, err := os.MkdirTemp("", "transformers")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		resourceFile := fmt.Sprintf("%s-%s-%s", providerName, component.CallerTypeHTTP, resourceType)
		path := filepath.Join(tempDir, resourceFile)

		// Write a rule file with invalid content
		err = os.WriteFile(path, []byte("invalid content"), 0644)
		Expect(err).NotTo(HaveOccurred())

		// Load the resource transformers
		err = LoadResourceTransformers(WithTransformRulesFilePath(tempDir))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to unmarshal transform rules"))
	})
})

var _ = Describe("RequestTransformer", func() {
	var seed string
	BeforeEach(func() {
		seed = testutils.RandNum()
	})

	Context("when the request transformer is http", func() {

		Context("GenerateRequest", func() {

			It("should generate the request correctly", func() {
				body := `{"key": "value"}`
				rule := http.RequestTransformerRule{
					RequestGenerationRule: http.RequestGenerationRule{
						Request: "POST",
						Body:    &body,
						URI:     "/create",
						Header: map[string]utils.StringOrStringArray{
							"Content-Type": utils.NewStringOrStringArray("application/json"),
						},
					},
				}

				res := &resources.CommonResource{
					ResourceMetadata: resources.ResourceMetadata{
						Type: "mock",
						Name: fmt.Sprintf("mock-%s", seed),
					},
					Dependencies: []resources.ResourceMetadata{},
					Spec:         map[string]string{},
					Status:       map[string]string{},
				}

				ruleByte, err := yaml.Marshal(rule)
				Expect(err).NotTo(HaveOccurred())
				rt, err := NewRequestTransformer("http", ruleByte)

				req, err := rt.GenerateRequest(res)
				Expect(err).NotTo(HaveOccurred())
				reqHttp, ok := req.(*http.RequestHTTP)
				Expect(ok).To(BeTrue())
				Expect(reqHttp.Request).To(Equal("POST"))
				Expect(*reqHttp.Body).To(Equal(body))
				Expect(reqHttp.Path).To(Equal("/create"))
				Expect(reqHttp.Header).To(Equal(map[string][]string{
					"Content-Type": {"application/json"},
				}))
			})

			It("when the function receives a nil Resource instance, should return an error indicating the invalid input", func() {
				bodyTemplate := `{"key": "{{.Key}}"}`
				headerTemplate := map[string]utils.StringOrStringArray{
					"Content-Type": utils.NewStringOrStringArray("application/json"),
				}
				queryTemplate := map[string]utils.StringOrStringArray{
					"id": utils.NewStringOrStringArray(".ID"),
				}
				transformRule := http.RequestTransformerRule{
					RequestGenerationRule: http.RequestGenerationRule{
						Request: "POST",
						Body:    &bodyTemplate,
						URI:     "/create",
						Header:  headerTemplate,
						Query:   queryTemplate,
					},
				}
				rt := &RequestTransformerHTTP{
					TransformRule: transformRule,
				}

				_, err := rt.GenerateRequest(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid input"))
			})

		})

		Context("ParseResponse", func() {
			var rt *RequestTransformerHTTP

			BeforeEach(func() {
				rt = &RequestTransformerHTTP{
					TransformRule: http.RequestTransformerRule{
						ResponseParseRule: []http.ResponseParseRule{
							{
								KeyName: "id",
								KeyPath: "data.id",
							},
						},
					},
				}
			})

			It("should parse the response correctly", func() {
				response := []byte(`{"data": {"id": "123"}}`)

				state, err := rt.ParseResponse(response)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(map[string]string{
					"id": "123",
				}))
			})

			It("when key path has many levels, should parse the response correctly", func() {
				rt.TransformRule.ResponseParseRule[0].KeyPath = "data.id.value"
				response := []byte(`{"data": {"id": {"value": "123"}}}`)
				state, err := rt.ParseResponse(response)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(Equal(map[string]string{
					"id": "123",
				}))
			})

			It("should return an error if the response cannot be unmarshaled", func() {
				response := []byte(`invalid response`)

				_, err := rt.ParseResponse(response)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to unmarshal response"))
			})

			It("should return an empty state if the key path does not exist in the response", func() {
				response := []byte(`{"data": {}}`)

				state, err := rt.ParseResponse(response)
				Expect(err).NotTo(HaveOccurred())
				Expect(state).To(BeEmpty())
			})
		})
	})
})
