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

func createDeploymentRuntimeResource(name, repoID string) *resourcev1alpha1.DeploymentRuntime {
	return &resourcev1alpha1.DeploymentRuntime{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1alpha1.DeploymentRuntimeSpec{
			Product:     defaultProductId,
			ProjectsRef: []string{MockProject1Name},
			Destination: resourcev1alpha1.DeploymentRuntimesDestination{
				Environment: "env1",
				Namespaces:  []string{},
			},
			ManifestSource: resourcev1alpha1.ManifestSource{
				CodeRepo:       repoID,
				TargetRevision: "main",
				Path:           "production",
			},
		},
	}
}

func createFakeDeploymentRuntimeNode(resource *resourcev1alpha1.DeploymentRuntime) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Path:    fmt.Sprintf("%s/%s/%s.yaml", localRepositoryPath, RuntimesDir, resource.Name),
		Level:   3,
		Content: resource,
		Kind:    nodestree.DeploymentRuntime,
	}
}

func createFakeDeployRuntimeNodes(node *nodestree.Node) nodestree.Node {
	fakeSubNode := &nodestree.Node{
		Name:     nodestree.DeploymentRuntime,
		Path:     fmt.Sprintf("%s/%s", localRepositoryPath, RuntimesDir),
		IsDir:    true,
		Level:    2,
		Content:  node,
		Children: []*nodestree.Node{node},
	}
	fakeNodes := nodestree.Node{
		Name:     defaultProjectName,
		Path:     defaultProjectName,
		IsDir:    true,
		Level:    1,
		Children: []*nodestree.Node{fakeSubNode},
	}

	return fakeNodes
}

var _ = Describe("Get deployment runtime", func() {
	var (
		resourceName = "runtime1"
		toGetProject = &Project{ID: MockID1, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		repoID       = fmt.Sprintf("%s%d", RepoPrefix, int(toGetProject.ID))
		fakeResource = createDeploymentRuntimeResource(resourceName, repoID)
		fakeNode     = createFakeDeploymentRuntimeNode(fakeResource)
		fakeNodes    = createFakeDeployRuntimeNodes(fakeNode)
	)

	It("will get successed", testUseCase.GetResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourcesUsecase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourcesUsecase, client, nautesConfigs)
		result, err := biz.GetDeploymentRuntime(context.Background(), resourceName, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result).Should(Equal(fakeResource))
	}))

	It("will fail when resource is not found", testUseCase.GetResourceFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		_, err := biz.GetDeploymentRuntime(context.Background(), resourceName, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	}))
})

var _ = Describe("List deployment runtimes", func() {
	var (
		resourceName = "runtime1"
		toGetProject = &Project{ID: MockID1, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		repoID       = fmt.Sprintf("%s%d", RepoPrefix, int(toGetProject.ID))
		fakeResource = createDeploymentRuntimeResource(resourceName, repoID)
		fakeNode     = createFakeDeploymentRuntimeNode(fakeResource)
		fakeNodes    = createFakeDeployRuntimeNodes(fakeNode)
		client       = kubernetes.NewMockClient(ctl)
	)

	It("will successfully", testUseCase.ListResourceSuccess(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		results, err := biz.ListDeploymentRuntimes(ctx, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		for _, result := range results {
			Expect(result).Should(Equal(fakeNode))
		}
	}))

	It("will fail when project is not found", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(nil, ErrorProjectNotFound)

		gitRepo := NewMockGitRepo(ctl)

		in := nodestree.NewMockNodesTree(ctl)
		in.EXPECT().AppendOperators(gomock.Any())

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, in, nautesConfigs)
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, in, resourcesUsecase, client, nautesConfigs)
		_, err := biz.ListDeploymentRuntimes(ctx, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("Save deployment runtime", func() {
	var (
		resourceName          = "runtime1"
		toGetProject          = &Project{ID: MockID1, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		repoID                = fmt.Sprintf("%s%d", RepoPrefix, int(toGetProject.ID))
		fakeResource          = createDeploymentRuntimeResource(resourceName, repoID)
		fakeNode              = createFakeDeploymentRuntimeNode(fakeResource)
		fakeNodes             = createFakeDeployRuntimeNodes(fakeNode)
		pid                   = fmt.Sprintf("%s/%s", defaultGroupName, fakeResource.Spec.ManifestSource.CodeRepo)
		project               = &Project{ID: MockID1}
		deploymentRuntimeData = &DeploymentRuntimeData{
			Name: fakeResource.Name,
			Spec: fakeResource.Spec,
		}
		bizOptions = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
	)

	It("failed to get product info", testUseCase.GetProductFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to get default project info", testUseCase.GetDefaultProjectFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))

	It("will created successfully", testUseCase.CreateResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("will updated successfully", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("auto merge conflict, updated successfully", testUseCase.UpdateResourceAndAutoMerge(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to save config", testUseCase.SaveConfigFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to push code retry three times", testUseCase.CreateResourceAndAutoRetry(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))
	It("auto merge code push successed and retry three times when the remote code changes", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		first := codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil).After(first)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil)
		firstFetch := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		secondFetch := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any()).Return("any", nil).After(firstFetch)
		thirdFetch := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil).After(secondFetch)
		fouthFetch := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any()).Return("any", nil).After(thirdFetch)
		fifthFetch := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil).After(fouthFetch)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any()).Return("any", nil).After(fifthFetch)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("any", nil).AnyTimes()
		gitRepo.EXPECT().Merge(gomock.Any(), localRepositoryPath).Return("successfully auto merge", nil).AnyTimes()
		gitRepo.EXPECT().Push(gomock.Any(), gomock.Any()).Return(fmt.Errorf("unable to push code")).AnyTimes()
		gitRepo.EXPECT().Commit(gomock.Any(), localRepositoryPath).AnyTimes()

		in := nodestree.NewMockNodesTree(ctl)
		in.EXPECT().AppendOperators(gomock.Any())
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, in, nautesConfigs)

		client := kubernetes.NewMockClient(ctl)
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, in, resourcesUsecase, client, nautesConfigs)
		in.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		in.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		in.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeNode)
		in.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&fakeNodes, nil)
		in.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to auto merge conflict", testUseCase.MergeConflictFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))

	It("modify resource but non compliant layout", testUseCase.UpdateResourceButNotConformTemplate(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), pid).Return(project, nil)

		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.SaveDeploymentRuntime(context.Background(), bizOptions, deploymentRuntimeData)
		Expect(err).Should(HaveOccurred())
	}))

	Describe("check reference by resources", func() {
		var (
			projectForBase       = &Project{Name: "base", ID: MockID2, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
			projectForBaseRepoID = fmt.Sprintf("%s%d", RepoPrefix, int(projectForBase.ID))
			environmentName      = "env1"
		)
		It("incorrect product name", testUseCase.CheckReferenceButIncorrectProduct(fakeNodes, func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree) {
			client := kubernetes.NewMockClient(ctl)
			biz := NewDeploymentRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
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

			client := kubernetes.NewMockClient(ctl)
			biz := NewDeploymentRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
		It("environment reference not found", func() {
			projectName := fakeResource.Spec.ProjectsRef[0]
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			client := kubernetes.NewMockClient(ctl)
			biz := NewDeploymentRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
		It("codeRepo reference not found", func() {
			projectName := fakeResource.Spec.ProjectsRef[0]
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

			client := kubernetes.NewMockClient(ctl)
			biz := NewDeploymentRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
		It("will successed", func() {
			projectName := fakeResource.Spec.ProjectsRef[0]
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			env := fakeResource.Spec.Destination.Environment
			envProjects := createContainEnvironmentNodes(createEnvironmentNode(createEnvironmentResource(env, TestClusterHostEnvType, TestDeploymentClusterName)))
			fakeNodes.Children = append(fakeNodes.Children, projectNodes.Children...)
			fakeNodes.Children = append(fakeNodes.Children, envProjects.Children...)
			codeRepoNodes := createFakeCcontainingCodeRepoNodes(createFakeCodeRepoNode(createFakeCodeRepoResource(repoID)))
			fakeNodes.Children = append(fakeNodes.Children, codeRepoNodes.Children...)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}

			codeRepoKind := nodestree.CodeRepo
			environmentKind := nodestree.Environment
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())
			nodestree.EXPECT().GetNode(gomock.Any(), codeRepoKind, gomock.Any()).Return(createFakeCodeRepoNode(createFakeCodeRepoResource(projectForBaseRepoID))).AnyTimes()
			nodestree.EXPECT().GetNode(gomock.Any(), environmentKind, gomock.Any()).Return(createEnvironmentNode(createEnvironmentResource(environmentName, TestClusterHostEnvType, TestDeploymentClusterName))).AnyTimes()

			client := kubernetes.NewMockClient(ctl)
			client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			client.EXPECT().List(gomock.Any(), gomock.Eq(&resourcev1alpha1.DeploymentRuntimeList{})).Return(nil)

			biz := NewDeploymentRuntimeUsecase(logger, nil, nodestree, nil, client, nautesConfigs)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})
})

var _ = Describe("Delete deployment runtime", func() {
	var (
		resourceName = "runtime1"
		toGetProject = &Project{ID: MockID1, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		repoID       = fmt.Sprintf("%s%d", RepoPrefix, int(toGetProject.ID))
		fakeResource = createDeploymentRuntimeResource(resourceName, repoID)
		fakeNode     = createFakeDeploymentRuntimeNode(fakeResource)
		fakeNodes    = createFakeDeployRuntimeNodes(fakeNode)
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
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.DeleteDeploymentRuntime(context.Background(), bizOptions)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("modify resource but non compliant layout standards", testUseCase.DeleteResourceErrorLayout(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewDeploymentRuntimeUsecase(logger, codeRepo, nodestree, resourceUseCase, client, nautesConfigs)
		err := biz.DeleteDeploymentRuntime(context.Background(), bizOptions)
		Expect(err).Should(HaveOccurred())
	}))
})
