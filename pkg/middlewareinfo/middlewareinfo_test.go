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

package middlewareinfo_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nautes-labs/nautes/pkg/middlewareinfo"
)

var _ = Describe("NewMiddlewares", func() {
	var (
		tempFile string
	)

	BeforeEach(func() {
		// Create a temporary file for testing
		file, err := os.CreateTemp("", "middlewares.yaml")
		Expect(err).NotTo(HaveOccurred())
		tempFile = file.Name()

		// Write sample middlewares to the temporary file
		middlewares := `
middleware1:
  providers:
    provider1:
      defaultImplementation: implementation1
      implementations:
        implementation1:
          name: implementation1
`
		err = os.WriteFile(tempFile, []byte(middlewares), 0644)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Remove the temporary file
		err := os.Remove(tempFile)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should load middlewares from file", func() {
		middlewares, err := middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath(tempFile))
		Expect(err).NotTo(HaveOccurred())

		// Assert that the middlewares were loaded correctly
		wanted := &middlewareinfo.Middlewares{
			"middleware1": {
				Name: "middleware1",
				Providers: map[string]middlewareinfo.Provider{
					"provider1": {
						Name:                  "provider1",
						DefaultImplementation: "implementation1",
						Implementations: map[string]middlewareinfo.MiddlewareImplementation{
							"implementation1": {
								Name: "implementation1",
							},
						},
					},
				},
			},
		}
		Expect(middlewares).To(Equal(wanted))
	})

	It("should return an error if failed to read middlewares file", func() {
		info, err := middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath("/path/to/nonexistent/file.yaml"))
		Expect(err).Should(BeNil())
		Expect(info).Should(Equal(&middlewareinfo.Middlewares{}))
	})

	It("should return an error if failed to unmarshal middlewares", func() {
		// Write an invalid YAML to the temporary file
		err := os.WriteFile(tempFile, []byte("invalid_yaml"), 0644)
		Expect(err).NotTo(HaveOccurred())

		_, err = middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath(tempFile))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to unmarshal middlewares"))
	})

	It("should return an error if default implementation is empty", func() {
		// Write a middleware with an empty default implementation to the temporary file
		err := os.WriteFile(tempFile, []byte(`
middleware1:
  providers:
    provider1:
      default_implementation:
      implementations:
        implementation1:
          name: implementation1
`), 0644)
		Expect(err).NotTo(HaveOccurred())

		_, err = middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath(tempFile))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("default implementation in provider provider1 of middleware middleware1 is empty"))
	})

	It("should return an error if default implementation is not in implementations", func() {
		err := os.WriteFile(tempFile, []byte(`
middleware1:
  providers:
    provider1:
      defaultImplementation: implementation2
      implementations:
        implementation1:
          name: implementation1
`), 0644)
		Expect(err).NotTo(HaveOccurred())

		_, err = middlewareinfo.NewMiddlewares(middlewareinfo.WithLoadPath(tempFile))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("default implementation implementation2 in provider provider1 of middleware middleware1 does not exist"))
	})
})
