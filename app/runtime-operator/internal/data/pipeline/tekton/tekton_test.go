package tekton_test

import (
	"context"
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

	pluginshared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/shared"
	shared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/tekton"
)

var _ = Describe("Tekton", func() {
	var ctx context.Context
	var seed string
	var initInfo *component.ComponentInitInfo
	var builtinVars map[component.BuiltinVar]string
	var space component.Space
	var authInfo = component.AuthInfo{}
	BeforeEach(func() {
		ctx = context.TODO()
		seed = RandNum()

		initInfo = &component.ComponentInitInfo{
			ClusterConnectInfo:     component.ClusterConnectInfo{},
			ClusterName:            "",
			NautesResourceSnapshot: nil,
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
		spaceName := fmt.Sprintf("space-%s", seed)
		space = component.Space{
			ResourceMetaData: component.ResourceMetaData{
				Product: "",
				Name:    spaceName,
			},
			SpaceType: component.SpaceTypeKubernetes,
			Kubernetes: &component.SpaceKubernetes{
				Namespace: spaceName,
			},
		}

		fakeHook = &pluginshared.Hook{}
	})

	It("can create init pipeline", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())
		Expect(status).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{},
		}))
		Expect(err)
		hooks, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars:       builtinVars,
			Hooks:             nautesv1alpha1.Hooks{},
			EventSource:       component.EventSource{},
			EventSourceType:   "",
			EventListenerType: "",
		})
		Expect(err).Should(BeNil())
		Expect(resources[0].(shared.ResourceRequest)).Should(Equal(shared.ResourceRequest{
			Type: shared.ResourceTypeCodeRepoSSHKey,
			SSHKey: &shared.ResourceRequestSSHKey{
				ResourceName: tekton.SecretNamePipelineReadOnlySSHKey,
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

	It("can create hook space", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars: builtinVars,
			Hooks:       nautesv1alpha1.Hooks{},
			EventSource: component.EventSource{},
		})
		Expect(err).Should(BeNil())

		err = pipeline.CreateHookSpace(ctx, authInfo, component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		})
		Expect(err).Should(BeNil())

		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space:     space,
					Resources: ConvertInterfaceToRequest(resources),
				},
			},
		}))
	})

	It("can merge duplicate requests", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars: builtinVars,
			Hooks:       nautesv1alpha1.Hooks{},
			EventSource: component.EventSource{},
		})
		Expect(err).Should(BeNil())

		resources2 := append(resources, resources...)

		err = pipeline.CreateHookSpace(ctx, authInfo, component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources2,
		})
		Expect(err).Should(BeNil())
		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space:     space,
					Resources: ConvertInterfaceToRequest(resources),
				},
			},
		}))
	})

	It("can clean up hook space", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars: builtinVars,
			Hooks:       nautesv1alpha1.Hooks{},
			EventSource: component.EventSource{},
		})
		Expect(err).Should(BeNil())

		err = pipeline.CreateHookSpace(ctx, authInfo, component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		})
		Expect(err).Should(BeNil())

		pipeline, err = tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		err = pipeline.CleanUpHookSpace(ctx)
		Expect(err).Should(BeNil())
		Expect(status).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{},
		}))
	})

	It("if space is changed, it will remove old resources and deploy new resources", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(component.HooksInitInfo{
			BuiltinVars: builtinVars,
			Hooks:       nautesv1alpha1.Hooks{},
			EventSource: component.EventSource{},
		})
		Expect(err).Should(BeNil())

		hookSpace := component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		}
		err = pipeline.CreateHookSpace(ctx, authInfo, hookSpace)
		Expect(err).Should(BeNil())

		space2 := space
		space2.Kubernetes.Namespace = fmt.Sprintf("%s-2", space.Kubernetes.Namespace)
		hookSpace.BaseSpace = space2

		err = pipeline.CreateHookSpace(ctx, authInfo, hookSpace)
		Expect(err).Should(BeNil())

		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space:     space2,
					Resources: ConvertInterfaceToRequest(resources),
				},
			},
		}))
	})

	It("if old resource is unused, it will remove old resources", func() {
		reqByte, err := json.Marshal(shared.ResourceRequest{
			Type: shared.ResourceTypeCodeRepoAccessToken,
			AccessToken: &shared.ResourceRequestAccessToken{
				ResourceName: "ttp",
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
		})
		Expect(nil).Should(BeNil())
		pr, _ := json.Marshal(v1alpha1.PipelineTask{
			Name: "test",
		})
		fakeHook = &pluginshared.Hook{
			RequestResources: [][]byte{
				reqByte,
			},
			Resource: pr,
		}

		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(nautesv1alpha1.Component{}, initInfo, status)
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

		_, resources, err := pipeline.GetHooks(hookInitInfo)
		Expect(err).Should(BeNil())

		hookSpace := component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		}
		err = pipeline.CreateHookSpace(ctx, authInfo, hookSpace)
		Expect(err).Should(BeNil())

		fakeHook.RequestResources = nil

		_, resources, err = pipeline.GetHooks(hookInitInfo)
		Expect(err).Should(BeNil())

		hookSpace = component.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		}

		err = pipeline.CreateHookSpace(ctx, authInfo, hookSpace)
		Expect(err).Should(BeNil())

		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space:     space,
					Resources: ConvertInterfaceToRequest(resources),
				},
			},
		}))
	})
})
