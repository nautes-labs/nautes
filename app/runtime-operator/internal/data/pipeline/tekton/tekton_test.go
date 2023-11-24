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

package tekton_test

import (
	"encoding/json"
	"fmt"

	nautesv1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/pipeline/tekton"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tekton", func() {
	var seed string
	var initInfo *component.ComponentInitInfo
	var builtinVars map[component.BuiltinVar]string
	BeforeEach(func() {
		seed = RandNum()

		initInfo = &component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: nautesv1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes:  &component.ClusterConnectInfoKubernetes{},
			},
			ClusterName:            "",
			NautesResourceSnapshot: &mockSnapShot{},
			RuntimeName:            fmt.Sprintf("runtime-%s", seed),
			NautesConfig:           configs.Config{},
			Components: &component.ComponentList{
				SecretSync: &mockSecretSync{},
			},
			PipelinePluginManager: &mockPluginManager{},
		}
		builtinVars = map[component.BuiltinVar]string{
			component.VarCodeRepoProviderType: fmt.Sprintf("provider-%s", seed),
			component.VarPipelineFilePath:     fmt.Sprintf("pipeline/%s.yaml", seed),
			component.VarPipelineCodeRepoName: fmt.Sprintf("codeRepo-%s", seed),
		}
		fakeHook = &component.Hook{}
	})

	It("can create init pipeline", func() {
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, nil)
		Expect(err).Should(BeNil())

		hooks, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars:       builtinVars,
			Hooks:             nautesv1alpha1.Hooks{},
			EventSource:       component.EventSource{},
			EventSourceType:   "",
			EventListenerType: "",
		})
		Expect(err).Should(BeNil())
		Expect(resources[0]).Should(Equal(component.RequestResource{
			Type:         component.ResourceTypeCodeRepoSSHKey,
			ResourceName: tekton.SecretNamePipelineReadOnlySSHKey,
			SSHKey: &component.ResourceRequestSSHKey{
				SecretInfo: component.SecretInfo{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: builtinVars[component.VarCodeRepoProviderType],
						ID:           builtinVars[component.VarPipelineCodeRepoName],
						User:         "default",
						Permission:   component.CodeRepoPermissionReadOnly,
					},
				},
			},
		}))
		_ = hooks
	})

	It("if old resource is unused, it will remove old resources", func() {
		req := component.RequestResource{
			Type:         component.ResourceTypeCodeRepoAccessToken,
			ResourceName: "ttp",
			AccessToken: &component.ResourceRequestAccessToken{
				SecretInfo: component.SecretInfo{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: builtinVars[component.VarCodeRepoProviderType],
						ID:           builtinVars[component.VarEventSourceCodeRepoName],
						User:         "default",
						Permission:   component.CodeRepoPermissionAccessToken,
					},
				},
			},
		}
		Expect(nil).Should(BeNil())
		pr, _ := json.Marshal(v1alpha1.PipelineTask{
			Name: "test",
		})
		fakeHook = &component.Hook{
			RequestResources: []component.RequestResource{req},
			Resource:         pr,
		}

		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, nil)
		Expect(err).Should(BeNil())

		hookInitInfo := component.HooksInitInfo{
			BuiltinVars: builtinVars,
			Hooks: nautesv1alpha1.Hooks{
				PreHooks: []nautesv1alpha1.Hook{
					{
						Name:  "fake",
						Alias: new(string),
						Vars:  map[string]string{},
					},
				},
				PostHooks: []nautesv1alpha1.Hook{},
			},
			EventSource: component.EventSource{},
		}

		hooks, resources, err := pipeline.GetHooks(hookInitInfo)
		Expect(err).Should(BeNil())

		Expect(resources[1]).Should(Equal(req))
		_ = hooks
	})
})
