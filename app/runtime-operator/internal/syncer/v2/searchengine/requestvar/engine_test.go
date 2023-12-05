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

package requestvar_test

import (
	"os"
	"path/filepath"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/searchengine/requestvar"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/pkg/nautesconst"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event source search engine", func() {
	tmpHome := "/tmp/testhome"

	BeforeEach(func() {
		err := os.Setenv(nautesconst.EnvNautesHome, tmpHome)
		Expect(err).Should(BeNil())
		err = os.MkdirAll(tmpHome, os.ModeDir)
		Expect(err).Should(BeNil())
		err = os.Mkdir(filepath.Join(tmpHome, "rules"), os.ModeDir)
		Expect(err).Should(BeNil())
		rules := `
rule test "test" {
	when
	  req.Filter.RequestVar == "ref"
	then
	  req.Result = "body.ref";
	  Retract("test");
}
`
		err = os.WriteFile(filepath.Join(tmpHome, requestvar.RuleFilePath), []byte(rules), os.ModeSticky)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpHome)
		Expect(err).Should(BeNil())
	})

	It("can get path by filter", func() {
		engine, err := requestvar.NewSearchEngine(nil)
		Expect(err).Should(BeNil())
		path, err := engine.GetTargetPathInEventSource(component.RequestDataConditions{
			RequestVar: "ref",
		})
		Expect(err).Should(BeNil())
		Expect(path).Should(Equal("body.ref"))
	})
})
