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

package cache_test

import (
	"fmt"

	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Account usage cache", func() {
	var seed string
	var accountNames []string

	BeforeEach(func() {
		seed = RandNum()
		accountNames = GenerateNames(fmt.Sprintf("account-%s-%%d", seed), 5)

		Expect(len(accountNames)).Should(Equal(5))
	})

})
