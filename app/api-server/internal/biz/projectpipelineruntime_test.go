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
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createProjectPiepeLineResource(name string) *resourcev1alpha1.ProjectPipelineRuntime {
	return &resourcev1alpha1.ProjectPipelineRuntime{
		TypeMeta: v1.TypeMeta{
			Kind: nodestree.Project,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1alpha1.ProjectPipelineRuntimeSpec{
			Project:        MockProject1Name,
			PipelineSource: "pipelineCodeRepo",
			Destination: resourcev1alpha1.ProjectPipelineDestination{
				Environment: "env1",
				Namespace:   name,
			},
			Isolation: "shared",
			Pipelines: []resourcev1alpha1.Pipeline{
				{
					Name:  "pipeline1",
					Label: "test",
					Path:  "dev",
				},
			},
			EventSources: []resourcev1alpha1.EventSource{
				{
					Name: "event1",
					Gitlab: &resourcev1alpha1.Gitlab{
						RepoName: "repo-2123",
						Revision: "main",
						Events:   []string{"push_events"},
					},
					Calendar: &resourcev1alpha1.Calendar{
						Schedule:       "0 0 * * *",
						Interval:       "24h",
						ExclusionDates: []string{"2023-05-01"},
						Timezone:       "America/Los_Angeles",
					},
				},
			},
			PipelineTriggers: []resourcev1alpha1.PipelineTrigger{
				{
					EventSource: "event1",
					Pipeline:    "pipeline1",
					Revision:    "main",
				},
			},
		},
	}
}

func createFakeProjectPipelineRuntimeNode(resource *resourcev1alpha1.ProjectPipelineRuntime) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Kind:    nodestree.ProjectPipelineRuntime,
		Path:    fmt.Sprintf("%s/%s/%s/%s.yaml", localRepositoryPath, ProjectsDir, MockProject1Name, resource.Name),
		Level:   4,
		Content: resource,
	}
}

func createFakeProjectPipelineRuntimeNodes(node *nodestree.Node) nodestree.Node {
	return nodestree.Node{
		Name:  defaultProjectName,
		Path:  defaultProjectName,
		IsDir: true,
		Level: 1,
		Children: []*nodestree.Node{
			{
				Name:  ProjectsDir,
				Path:  fmt.Sprintf("%s/%s", defaultProjectName, ProjectsDir),
				IsDir: true,
				Level: 2,
				Children: []*nodestree.Node{
					{
						Name:  MockProject1Name,
						Path:  fmt.Sprintf("%s/%s/%s", defaultProjectName, ProjectsDir, MockProject1Name),
						IsDir: true,
						Level: 3,
						Children: []*nodestree.Node{
							node,
						},
					},
				},
			},
		},
	}
}

var _ = Describe("Get project pipeline runtime", func() {
	var (
		resourceName = "runtime1"
		fakeResource = createProjectPiepeLineResource(resourceName)
		fakeNode     = createFakeProjectPipelineRuntimeNode(fakeResource)
		fakeNodes    = createFakeProjectPipelineRuntimeNodes(fakeNode)
	)
	It("will get success", testUseCase.GetResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		result, err := biz.GetProjectPipelineRuntime(context.Background(), resourceName, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result).Should(Equal(fakeNode))
	}))

	It("will fail when resource is not found", testUseCase.GetResourceFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		_, err := biz.GetProjectPipelineRuntime(context.Background(), resourceName, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	}))
})

var _ = Describe("List project pipeline runtimes", func() {
	var (
		resourceName = "projectpipelineruntime1"
		fakeResource = createProjectPiepeLineResource(resourceName)
		fakeNode     = createFakeProjectPipelineRuntimeNode(fakeResource)
		fakeNodes    = createFakeProjectPipelineRuntimeNodes(fakeNode)
	)
	It("will list successfully", testUseCase.ListResourceSuccess(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		results, err := biz.ListProjectPipelineRuntimes(ctx, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		for _, result := range results {
			Expect(result).Should(Equal(fakeNode))
		}
	}))
})

var _ = Describe("Save project pipeline runtime", func() {
	var (
		resourceName             = "projectpipelineruntime1"
		projectForPipeline       = &Project{Name: "pipeline", ID: MockID1, HttpUrlToRepo: "ssh://git@gitlab.io/nautes-labs/pipeline.git"}
		projectForPipelineRepoID = fmt.Sprintf("%s%d", RepoPrefix, int(projectForPipeline.ID))
		projectForBase           = &Project{Name: "base", ID: MockID2, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		projectForBaseRepoID     = fmt.Sprintf("%s%d", RepoPrefix, int(projectForBase.ID))
		fakeResource             = createProjectPiepeLineResource(resourceName)
		fakeNode                 = createFakeProjectPipelineRuntimeNode(fakeResource)
		fakeNodes                = createFakeProjectPipelineRuntimeNodes(fakeNode)
		data                     = &ProjectPipelineRuntimeData{
			Name: fakeResource.Name,
			Spec: resourcev1alpha1.ProjectPipelineRuntimeSpec{
				Project:        MockProject1Name,
				PipelineSource: projectForPipeline.Name,
				Destination: resourcev1alpha1.ProjectPipelineDestination{
					Environment: MockEnv1Name,
					Namespace:   fakeResource.Name,
				},
				Isolation: "shared",
				Pipelines: []resourcev1alpha1.Pipeline{
					{
						Name:  "pipeline1",
						Label: "test",
						Path:  "dev",
					},
				},
				EventSources: []resourcev1alpha1.EventSource{
					{
						Name: "event1",
						Gitlab: &resourcev1alpha1.Gitlab{
							RepoName: fmt.Sprintf("repo-%d", MockID3),
							Revision: "main",
							Events:   []string{"push_events"},
						},
						Calendar: &resourcev1alpha1.Calendar{
							Schedule:       "0 0 * * *",
							Interval:       "24h",
							ExclusionDates: []string{"2023-05-01"},
							Timezone:       "America/Los_Angeles",
						},
					},
				},
				PipelineTriggers: []resourcev1alpha1.PipelineTrigger{
					{
						EventSource: "event1",
						Pipeline:    "pipeline1",
						Revision:    "main",
					},
				},
				AdditionalResources: &resourcev1alpha1.ProjectPipelineRuntimeAdditionalResources{
					Git: &resourcev1alpha1.ProjectPipelineRuntimeAdditionalResourcesGit{
						CodeRepo: MockCodeRepo1Name,
						URL:      "",
						Revision: "main",
						Path:     "deployment",
					},
				},
			},
		}
		bizOptions = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
	)

	var (
		pipelineSourceCodeRepoPath, eventSourcesGitlabRepoPath, additionalCodeRepoPath string
		projectFromPipelineSouce, projectFromEventSources, projectFromAdditional       *Project
	)

	BeforeEach(func() {
		pipelineSourceCodeRepoPath = fmt.Sprintf("%s/%s", defaultProductGroup.Path, projectForPipeline.Name)
		eventSourcesGitlabRepoPath = fmt.Sprintf("%s/%s", defaultProductGroup.Path, data.Spec.EventSources[0].Gitlab.RepoName)
		additionalCodeRepoPath = fmt.Sprintf("%s/%s", defaultProductGroup.Path, data.Spec.AdditionalResources.Git.CodeRepo)
		projectFromPipelineSouce = &Project{ID: MockID2}
		projectFromEventSources = &Project{ID: MockID3}
		projectFromAdditional = &Project{ID: MockID1}
	})

	AfterEach(func() {
		data.Spec.PipelineSource = projectForPipeline.Name
		data.Spec.EventSources[0].Gitlab.RepoName = projectForBase.Name
	})

	It("failed to get product info", testUseCase.GetProductFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(projectForBase, nil).AnyTimes()

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to get default project info", testUseCase.GetDefaultProjectFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	It("will created successfully", testUseCase.CreateResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("will updated successfully", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("auto merge conflict, updated successfully", testUseCase.UpdateResourceAndAutoMerge(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to auto merge conflict", testUseCase.MergeConflictFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to push code retry three times", testUseCase.CreateResourceAndAutoRetry(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	It("modify resource but non compliant layout", testUseCase.UpdateResourceButNotConformTemplate(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to save config", testUseCase.SaveConfigFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(pipelineSourceCodeRepoPath)).Return(projectFromPipelineSouce, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(additionalCodeRepoPath)).Return(projectFromAdditional, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(eventSourcesGitlabRepoPath)).Return(projectFromEventSources, nil)

		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveProjectPipelineRuntime(context.Background(), bizOptions, data)
		Expect(err).Should(HaveOccurred())
	}))

	Describe("check reference by resources", func() {
		It("incorrect product name", testUseCase.CheckReferenceButIncorrectProduct(fakeNodes, func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree) {
			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		}))

		It("project reference not found", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("repeated reference code repository", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			newResouce := fakeResource.DeepCopy()
			newResouce.Spec.PipelineSource = projectForPipelineRepoID
			newResouce.Spec.EventSources[0].Gitlab.RepoName = projectForBaseRepoID
			fakeNode.Content = newResouce

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("environment reference not found", func() {
			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("pipeline source reference not found", func() {
			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			env := fakeResource.Spec.Destination.Environment
			envProjects := createContainEnvironmentNodes(createEnvironmentNode(createEnvironmentResource(env, TestClusterHostEnvType, TestDeploymentClusterName)))
			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, envProjects.Children...)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("pipeline repository not found in event sources", func() {
			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			env := fakeResource.Spec.Destination.Environment
			envProjects := createContainEnvironmentNodes(createEnvironmentNode(createEnvironmentResource(env, TestClusterHostEnvType, TestDeploymentClusterName)))
			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, envProjects.Children...)
			codeRepoNodes := createFakeCcontainingCodeRepoNodes(createFakeCodeRepoNode(createFakeCodeRepoResource(projectForPipelineRepoID)))
			fakeNodes.Children = append(fakeNodes.Children, codeRepoNodes.Children...)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			newResouce := fakeResource.DeepCopy()
			newResouce.Spec.PipelineSource = projectForPipelineRepoID
			fakeNode.Content = newResouce

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, nil, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("will successed", func() {
			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			env := fakeResource.Spec.Destination.Environment
			envProjects := createContainEnvironmentNodes(createEnvironmentNode(createEnvironmentResource(env, TestClusterHostEnvType, TestDeploymentClusterName)))
			codeRepoNodes := createFakeCcontainingCodeRepoNodes(createFakeCodeRepoNode(createFakeCodeRepoResource(projectForPipelineRepoID)))
			codeRepoNodes.Children = append(codeRepoNodes.Children, createFakeCodeRepoNode(createFakeCodeRepoResource(projectForBaseRepoID)))
			codeRepoBinding1 := createFakeCodeRepoBindingResource(MockCodeRepo1Name, MockProject1Name, projectForPipelineRepoID, string(ReadOnly))
			codeRepoBinding2 := createFakeCodeRepoBindingResource(MockCodeRepo1Name, MockProject1Name, projectForBaseRepoID, string(ReadOnly))
			codeRepoBindingNode1 := createFakeCodeRepoBindingNode(codeRepoBinding1)
			codeRepoBindingNode2 := createFakeCodeRepoBindingNode(codeRepoBinding2)
			codeRepoBindingNodes := createFakeContainingCodeRepoBindingNodes(codeRepoBindingNode1)
			codeRepoBindingNodes.Children = append(codeRepoBindingNodes.Children, codeRepoBindingNode2)

			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, envProjects.Children...)
			fakeNodes.Children = append(fakeNodes.Children, codeRepoNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, codeRepoNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, codeRepoBindingNodes.Children...)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}

			codeRepoKind := nodestree.CodeRepo
			environmentKind := nodestree.Environment
			projectPipelineRuntimeKind := nodestree.ProjectPipelineRuntime
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())
			nodestree.EXPECT().GetNode(gomock.Any(), codeRepoKind, gomock.Any()).Return(createFakeCodeRepoNode(createFakeCodeRepoResource(projectForBaseRepoID))).AnyTimes()
			nodestree.EXPECT().GetNode(gomock.Any(), environmentKind, gomock.Any()).Return(createEnvironmentNode(createEnvironmentResource(MockEnv1Name, TestClusterHostEnvType, TestPipelineClusterName))).AnyTimes()
			nodestree.EXPECT().GetNode(gomock.Any(), projectPipelineRuntimeKind, gomock.Any()).Return(createFakeProjectPipelineRuntimeNode(createProjectPiepeLineResource(resourceName))).AnyTimes()

			newResouce := fakeResource.DeepCopy()
			newResouce.Spec.PipelineSource = projectForPipelineRepoID
			newResouce.Spec.EventSources[0].Gitlab.RepoName = projectForBaseRepoID
			fakeNode.Content = newResouce

			client := kubernetes.NewMockClient(ctl)
			client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			biz := NewProjectPipelineRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})
})

var _ = Describe("Delete project pipeline runtime", func() {
	var (
		resourceName = "projectpipelineruntime1"
		fakeResource = createProjectPiepeLineResource(resourceName)
		fakeNode     = createFakeProjectPipelineRuntimeNode(fakeResource)
		fakeNodes    = createFakeProjectPipelineRuntimeNodes(fakeNode)
		bizOptions   = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
	)

	BeforeEach(func() {
		err := os.MkdirAll(filepath.Dir(fakeNode.Path), 0644)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = os.Create(fakeNode.Path)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("will deleted successfully", testUseCase.DeleteResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.DeleteProjectPipelineRuntime(context.Background(), bizOptions)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("modify resource but non compliant layout standards", testUseCase.DeleteResourceErrorLayout(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectPipelineRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.DeleteProjectPipelineRuntime(context.Background(), bizOptions)
		Expect(err).Should(HaveOccurred())
	}))
})
