package tekton_test

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/pipeline/tekton"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tekton", func() {
	var ctx context.Context
	var seed string
	var initInfo *syncer.ComponentInitInfo
	var buildInVars map[syncer.BuildInVar]string
	var space syncer.Space
	var authInfo = syncer.AuthInfo{}
	BeforeEach(func() {
		ctx = context.TODO()
		seed = RandNum()

		initInfo = &syncer.ComponentInitInfo{
			ClusterConnectInfo:     syncer.ClusterConnectInfo{},
			ClusterName:            "",
			NautesResourceSnapshot: nil,
			RuntimeName:            fmt.Sprintf("runtime-%s", seed),
			NautesConfig:           configs.Config{},
			Components: &syncer.ComponentList{
				SecretSync: &mockSecretSync{},
			},
		}
		buildInVars = map[syncer.BuildInVar]string{
			syncer.VarCodeRepoProviderType: fmt.Sprintf("provider-%s", seed),
			syncer.VarPipelineFilePath:     fmt.Sprintf("pipeline/%s.yaml", seed),
			syncer.VarPipelineCodeRepoName: fmt.Sprintf("codeRepo-%s", seed),
		}
		spaceName := fmt.Sprintf("space-%s", seed)
		space = syncer.Space{
			ResourceMetaData: syncer.ResourceMetaData{
				Product: "",
				Name:    spaceName,
			},
			SpaceType: syncer.SpaceTypeKubernetes,
			Kubernetes: &syncer.SpaceKubernetes{
				Namespace: spaceName,
			},
		}
	})

	It("can create init pipeline", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())
		Expect(status).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{},
		}))
		Expect(err)
		hooks, resources, err := pipeline.GetHooks(syncer.HooksInitInfo{
			BuildInVars:       buildInVars,
			Hooks:             v1alpha1.Hook{},
			EventSource:       syncer.EventSource{},
			EventSourceType:   "",
			EventListenerType: "",
		})
		Expect(err).Should(BeNil())
		Expect(resources[0].(tekton.ResourceRequest)).Should(Equal(tekton.ResourceRequest{
			Type: tekton.ResourceTypeCodeRepoSSHKey,
			SSHKey: &tekton.ResourceRequestSSHKey{
				ResourceName: tekton.SecretNamePipelineReadOnlySSHKey,
				SecretInfo: syncer.SecretInfo{
					Type: syncer.SecretTypeCodeRepo,
					CodeRepo: &syncer.CodeRepo{
						ProviderType: buildInVars[syncer.VarCodeRepoProviderType],
						ID:           buildInVars[syncer.VarPipelineCodeRepoName],
						User:         "default",
						Permission:   syncer.CodeRepoPermissionReadOnly,
					},
				},
			},
		}))
		_ = hooks
	})

	It("can create hook space", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(syncer.HooksInitInfo{
			BuildInVars: buildInVars,
			Hooks:       v1alpha1.Hook{},
			EventSource: syncer.EventSource{},
		})
		Expect(err).Should(BeNil())

		err = pipeline.CreateHookSpace(ctx, authInfo, syncer.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		})
		Expect(err).Should(BeNil())

		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space: space,
					Resources: map[tekton.ResourceType][]tekton.ResourceRequest{
						tekton.ResourceTypeCodeRepoSSHKey: ConvertInterfaceToRequest(resources),
					},
				},
			},
		}))
	})

	It("can merge duplicate requests", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(syncer.HooksInitInfo{
			BuildInVars: buildInVars,
			Hooks:       v1alpha1.Hook{},
			EventSource: syncer.EventSource{},
		})
		Expect(err).Should(BeNil())

		resources2 := append(resources, resources...)

		err = pipeline.CreateHookSpace(ctx, authInfo, syncer.HookSpace{
			BaseSpace:       space,
			DeployResources: resources2,
		})
		Expect(err).Should(BeNil())
		Expect(status.(*tekton.TektonStatus)).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{
				initInfo.RuntimeName: {
					Space: space,
					Resources: map[tekton.ResourceType][]tekton.ResourceRequest{
						tekton.ResourceTypeCodeRepoSSHKey: ConvertInterfaceToRequest(resources),
					},
				},
			},
		}))
	})

	It("can clean up hook space", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(syncer.HooksInitInfo{
			BuildInVars: buildInVars,
			Hooks:       v1alpha1.Hook{},
			EventSource: syncer.EventSource{},
		})
		Expect(err).Should(BeNil())

		err = pipeline.CreateHookSpace(ctx, authInfo, syncer.HookSpace{
			BaseSpace:       space,
			DeployResources: resources,
		})
		Expect(err).Should(BeNil())

		pipeline, err = tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		err = pipeline.CleanUpHookSpace(ctx)
		Expect(err).Should(BeNil())
		Expect(status).Should(Equal(&tekton.TektonStatus{
			Runtimes: map[string]tekton.RuntimeResource{},
		}))
	})

	It("if space is changed, it will remove old resources and deploy new resources", func() {
		status, _ := tekton.TektonFactory.NewStatus(nil)
		pipeline, err := tekton.TektonFactory.NewComponent(v1alpha1.Component{}, initInfo, status)
		Expect(err).Should(BeNil())

		_, resources, err := pipeline.GetHooks(syncer.HooksInitInfo{
			BuildInVars: buildInVars,
			Hooks:       v1alpha1.Hook{},
			EventSource: syncer.EventSource{},
		})
		Expect(err).Should(BeNil())

		hookSpace := syncer.HookSpace{
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
					Space: space2,
					Resources: map[tekton.ResourceType][]tekton.ResourceRequest{
						tekton.ResourceTypeCodeRepoSSHKey: ConvertInterfaceToRequest(resources),
					},
				},
			},
		}))
	})
})
