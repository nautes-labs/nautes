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

package biz

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/golang/mock/gomock"
	"github.com/nautes-labs/nautes/pkg/log/zap"
	nautesconst "github.com/nautes-labs/nautes/pkg/nautesconst"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	logger = log.With(zap.NewLogger(),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	ctx         = context.Background()
	ctl         *gomock.Controller
	testUseCase = NewTestUseCase()
)

func TestBiz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Biz Suite")
}

var homePath = "/tmp/unittest-api"

var _ = BeforeSuite(func() {
	err := os.MkdirAll(path.Join(homePath, "config"), os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	err = os.Setenv(nautesconst.EnvNautesHome, homePath)
	Expect(err).NotTo(HaveOccurred())

	configString := `
secret:
  repoType: mock
`
	err = os.WriteFile(path.Join(homePath, "config/config"), []byte(configString), 0600)
	Expect(err).Should(BeNil())
})

var _ = AfterSuite(func() {
	err := os.RemoveAll(homePath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = BeforeEach(func() {
	ctl = gomock.NewController(GinkgoT())
})

var _ = AfterEach(func() {
	ctl.Finish()
})
