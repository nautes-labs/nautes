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

package configs_test

import (
	"os"
	"testing"

	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNautesconfigs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nautesconfigs Suite")
}

var configString string
var configPath = "/tmp/config.yaml"

var _ = Describe("New nautes config", func() {
	BeforeEach(func() {
		configString = `
nautes:
  namespace: "testNamespace"
`
		err := os.WriteFile(configPath, []byte(configString), 0600)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		err := os.Remove(configPath)
		Expect(err).Should(BeNil())
		os.Unsetenv(configs.EnvNautesConfigPath)
	})

	It("will read from input if input is not empty.", func() {
		cfg, err := configs.NewNautesConfigFromFile(configs.FilePath(configPath))
		Expect(err).Should(BeNil())
		Expect(cfg.Nautes.Namespace).Should(Equal("testNamespace"))
	})

	It("will read from env when env is set.", func() {
		os.Setenv(configs.EnvNautesConfigPath, configPath)
		cfg, err := configs.NewNautesConfigFromFile()
		Expect(err).Should(BeNil())
		Expect(cfg.Nautes.Namespace).Should(Equal("testNamespace"))
	})

	It("will throw an error if file not exist", func() {
		_, err := configs.NewNautesConfigFromFile()
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})
})
