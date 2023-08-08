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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createProjectResource(name string) *resourcev1alpha1.Project {
	return &resourcev1alpha1.Project{
		TypeMeta: v1.TypeMeta{
			Kind:       nodestree.Project,
			APIVersion: resourcev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1alpha1.ProjectSpec{
			Language: "go",
			Product:  fmt.Sprintf("%s%d", ProductPrefix, int(defaultProductGroup.ID)),
		},
	}
}

func createProjectNode(resource *resourcev1alpha1.Project) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Kind:    nodestree.Project,
		Path:    fmt.Sprintf("%s/%s/%s/%s.yaml", localRepositoryPath, ProjectsDir, resource.Name, resource.Name),
		Level:   4,
		Content: resource,
	}
}

func createProjectNodes(node *nodestree.Node) nodestree.Node {
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
						Name:  node.Name,
						Path:  fmt.Sprintf("%s/%s/%s", defaultProjectName, ProjectsDir, node.Name),
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

var _ = Describe("Get project", func() {
	var (
		resourceName = "project1"
		fakeResource = createProjectResource(resourceName)
		fakeNode     = createProjectNode(fakeResource)
		fakeNodes    = createProjectNodes(fakeNode)
	)
	It("will get project success", testUseCase.GetResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourcesUsecase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		id, _ := utilstrings.ExtractNumber("product-", fakeResource.Spec.Product)
		codeRepo.EXPECT().GetGroup(gomock.Any(), id).Return(defaultProductGroup, nil)

		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourcesUsecase)
		result, err := biz.GetProject(context.Background(), resourceName, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result.Spec.Product).To(Equal(fakeResource.Spec.Product))
		Expect(result.Spec.Language).To(Equal(fakeResource.Spec.Language))
	}))

	It("will fail when resource is not found", testUseCase.GetResourceFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		_, err := biz.GetProject(context.Background(), resourceName, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	}))
})

var _ = Describe("List projects", func() {
	var (
		resourceName = "project1"
		fakeResource = createProjectResource(resourceName)
		fakeNode     = createProjectNode(fakeResource)
		fakeNodes    = createProjectNodes(fakeNode)
	)

	It("will successfully", testUseCase.ListResourceSuccess(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		id, _ := utilstrings.ExtractNumber("product-", fakeResource.Spec.Product)
		codeRepo.EXPECT().GetGroup(gomock.Any(), id).Return(defaultProductGroup, nil)

		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		results, err := biz.ListProjects(ctx, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		for _, result := range results {
			Expect(result).Should(Equal(fakeResource))
		}
	}))
})

var _ = Describe("Save project", func() {
	var (
		resourceName = "project1"
		fakeResource = createProjectResource(resourceName)
		fakeNode     = createProjectNode(fakeResource)
		fakeNodes    = createProjectNodes(fakeNode)
		projectData  = &ProjectData{
			ProductName: fmt.Sprintf("%s%d", ProductPrefix, int(defaultProductGroup.ID)),
			ProjectName: resourceName,
			Language:    "Java",
		}
		bizOptions = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
	)

	It("failed to get product info", testUseCase.GetProductFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to get default project info", testUseCase.GetDefaultProjectFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	It("will created successfully", testUseCase.CreateResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("will updated successfully", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to auto merge conflict", testUseCase.MergeConflictFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to push code retry three times", testUseCase.CreateResourceAndAutoRetry(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	It("modify resource but non compliant layout", testUseCase.UpdateResourceButNotConformTemplate(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to save config", testUseCase.SaveConfigFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.SaveProject(context.Background(), bizOptions, projectData)
		Expect(err).Should(HaveOccurred())
	}))

	Describe("check reference by resources", func() {
		It("incorrect product name", testUseCase.CheckReferenceButIncorrectProduct(fakeNodes, func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree) {
			biz := NewProjectUsecase(logger, nil, nil, nodestree, nautesConfigs, nil)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		}))
	})
})

var _ = Describe("Delete project", func() {
	var (
		resourceName = "project1"
		fakeResource = createProjectResource(resourceName)
		fakeNode     = createProjectNode(fakeResource)
		fakeNodes    = createProjectNodes(fakeNode)
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
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.DeleteProject(context.Background(), bizOptions)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("modify resource but non compliant layout standards", testUseCase.DeleteResourceErrorLayout(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewProjectUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourceUseCase)
		err := biz.DeleteProject(context.Background(), bizOptions)
		Expect(err).Should(HaveOccurred())
	}))
})
