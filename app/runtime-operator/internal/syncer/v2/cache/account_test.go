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
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/cache"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var _ = Describe("User usage cache", func() {
	var deploymentRuntime *v1alpha1.DeploymentRuntime
	var pipelineRuntime *v1alpha1.ProjectPipelineRuntime
	var seed string
	var userNames []string

	BeforeEach(func() {
		seed = RandNum()
		userNames = GenerateNames(fmt.Sprintf("user-%s-%%d", seed), 5)

		deploymentRuntime = &v1alpha1.DeploymentRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dr",
			},
			Spec: v1alpha1.DeploymentRuntimeSpec{
				Destination: v1alpha1.DeploymentRuntimesDestination{
					Environment: "",
					Namespaces: []string{
						"ns01",
						"ns02",
					},
				},
				ManifestSource: v1alpha1.ManifestSource{
					CodeRepo: "coderepo01",
				},
				Account: userNames[0],
			},
		}

		pipelineRuntime = &v1alpha1.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pr",
			},
			Spec: v1alpha1.ProjectPipelineRuntimeSpec{
				Project:        "",
				PipelineSource: "coderepo02",
				Pipelines:      []v1alpha1.Pipeline{},
				Destination: v1alpha1.ProjectPipelineDestination{
					Environment: "",
					Namespace:   "ns03",
				},
				EventSources:        []v1alpha1.EventSource{},
				Isolation:           "",
				PipelineTriggers:    []v1alpha1.PipelineTrigger{},
				AdditionalResources: &v1alpha1.ProjectPipelineRuntimeAdditionalResources{},
				Account:             "",
			},
		}

	})

	It("can record user is used by deployment runtime", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], deploymentRuntime)
		Expect(err).Should(BeNil())

		destUsage := cache.AccountUsage{
			Accounts: map[string]cache.AccountResource{
				userNames[0]: {
					Runtimes: map[string]cache.RuntimeResource{
						deploymentRuntime.Name: {
							Spaces: utils.StringSet{
								Set: sets.New[string](deploymentRuntime.GetNamespaces()...),
							},
							CodeRepos: utils.StringSet{
								Set: sets.New[string](deploymentRuntime.Spec.ManifestSource.CodeRepo),
							},
						},
					},
				},
			},
		}

		ok := reflect.DeepEqual(usage, destUsage)
		Expect(ok).Should(BeTrue())
	})

	It("can record user is used by pipeline runtime", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], pipelineRuntime)
		Expect(err).Should(BeNil())

		destUsage := cache.AccountUsage{
			Accounts: map[string]cache.AccountResource{
				userNames[0]: {
					Runtimes: map[string]cache.RuntimeResource{
						pipelineRuntime.Name: {
							Spaces: utils.StringSet{
								Set: sets.New[string](pipelineRuntime.GetNamespaces()...),
							},
							CodeRepos: utils.StringSet{
								Set: sets.New[string](pipelineRuntime.Spec.PipelineSource),
							},
						},
					},
				},
			},
		}

		ok := reflect.DeepEqual(usage, destUsage)
		Expect(ok).Should(BeTrue())
	})

	It("can remove runtime in user", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], deploymentRuntime)
		Expect(err).Should(BeNil())
		usage.DeleteRuntime(userNames[0], deploymentRuntime)

		destUsage := cache.AccountUsage{
			Accounts: map[string]cache.AccountResource{},
		}

		ok := reflect.DeepEqual(usage, destUsage)
		Expect(ok).Should(BeTrue())
	})

	It("can get using namespaces", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], deploymentRuntime)
		Expect(err).Should(BeNil())

		err = usage.AddOrUpdateRuntime(userNames[0], pipelineRuntime)
		Expect(err).Should(BeNil())

		account := usage.Accounts[userNames[0]]
		spaces := account.ListAccountSpaces()
		destSpaces := append(deploymentRuntime.GetNamespaces(), pipelineRuntime.GetNamespaces()...)
		ok := reflect.DeepEqual(spaces, utils.NewStringSet(destSpaces...))
		Expect(ok).Should(BeTrue())
	})

	It("can get using coderepos", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], deploymentRuntime)
		Expect(err).Should(BeNil())

		err = usage.AddOrUpdateRuntime(userNames[0], pipelineRuntime)
		Expect(err).Should(BeNil())

		account := usage.Accounts[userNames[0]]
		repos := account.ListAccountCodeRepos()
		destRepos := []string{
			deploymentRuntime.Spec.ManifestSource.CodeRepo,
			pipelineRuntime.Spec.PipelineSource,
		}
		ok := reflect.DeepEqual(repos, utils.NewStringSet(destRepos...))
		Expect(ok).Should(BeTrue())
	})

	It("can get using namespaces without runtime", func() {
		usage := cache.AccountUsage{}
		err := usage.AddOrUpdateRuntime(userNames[0], deploymentRuntime)
		Expect(err).Should(BeNil())

		err = usage.AddOrUpdateRuntime(userNames[0], pipelineRuntime)
		Expect(err).Should(BeNil())

		account := usage.Accounts[userNames[0]]
		spaces := account.ListAccountSpaces(cache.ExcludedRuntimeNames([]string{deploymentRuntime.Name}))
		ok := reflect.DeepEqual(spaces, utils.NewStringSet(pipelineRuntime.GetNamespaces()...))
		Expect(ok).Should(BeTrue())
	})

})
