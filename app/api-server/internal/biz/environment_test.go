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
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_TestDeploymentClusterName = "deployment-test"
	_TestPipelineClusterName   = "pipeline-test"
	_TestClusterHostEnvType    = "host"
	_TestClusterWorkerEnvType  = "worker"
)

func createEnvironmentResource(name, _TestClusterHostEnvType, _TestDeploymentClusterName string) *resourcev1alpha1.Environment {
	return &resourcev1alpha1.Environment{
		TypeMeta: v1.TypeMeta{
			Kind: nodestree.Environment,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1alpha1.EnvironmentSpec{
			Product: defaultProductId,
			Cluster: _TestDeploymentClusterName,
			EnvType: _TestClusterHostEnvType,
		},
	}
}

func createEnvironmentNode(resource *resourcev1alpha1.Environment) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Kind:    nodestree.Environment,
		Path:    fmt.Sprintf("%s/%s/%s.yaml", localRepositoryPath, EnvSubDir, resource.Name),
		Level:   3,
		Content: resource,
	}
}

func createContainEnvironmentNodes(node *nodestree.Node) nodestree.Node {
	return nodestree.Node{
		Name:  defaultProjectName,
		Path:  defaultProjectName,
		IsDir: true,
		Level: 1,
		Children: []*nodestree.Node{
			{
				Name:  EnvSubDir,
				Path:  fmt.Sprintf("%v/%v", defaultProjectName, EnvSubDir),
				IsDir: true,
				Level: 2,
				Children: []*nodestree.Node{
					node,
				},
			},
		},
	}
}

var _ = Describe("Get environment", func() {
	var (
		resourceName = "env1"
		fakeResource = createEnvironmentResource(resourceName, _TestClusterHostEnvType, _TestDeploymentClusterName)
		fakeNode     = createEnvironmentNode(fakeResource)
		fakeNodes    = createContainEnvironmentNodes(fakeNode)
	)
	It("will get environment success", testUseCase.GetResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourcesUsecase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		id, _ := utilstrings.ExtractNumber(ProductPrefix, fakeResource.Spec.Product)
		codeRepo.EXPECT().GetGroup(gomock.Any(), id).Return(defaultProductGroup, nil)

		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourcesUsecase)
		result, err := biz.GetEnvironment(context.Background(), resourceName, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result).Should(Equal(fakeResource))
	}))

	It("will fail when resource is not found", testUseCase.GetResourceFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		_, err := biz.GetEnvironment(context.Background(), resourceName, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	}))
})

var _ = Describe("List enviroments", func() {
	var (
		resourceName = "env1"
		fakeResource = createEnvironmentResource(resourceName, _TestClusterHostEnvType, _TestDeploymentClusterName)
		fakeNode     = createEnvironmentNode(fakeResource)
		fakeNodes    = createContainEnvironmentNodes(fakeNode)
	)
	It("will list successfully", testUseCase.ListResourceSuccess(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {

		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		results, err := biz.ListEnvironments(ctx, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		for _, result := range results {
			Expect(result).Should(Equal(fakeNode))
		}
	}))
})

var _ = Describe("Save environment", func() {
	var (
		resourceName   = "env1"
		fakeResource   = createEnvironmentResource(resourceName, _TestClusterHostEnvType, _TestDeploymentClusterName)
		fakeNode       = createEnvironmentNode(fakeResource)
		fakeNodes      = createContainEnvironmentNodes(fakeNode)
		enviromentData = &EnviromentData{
			Name: resourceName,
			Spec: resourcev1alpha1.EnvironmentSpec{
				Product: defaultGroupName,
				Cluster: "test-cluster",
				EnvType: _TestClusterHostEnvType,
			},
		}
		bizOptions = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
	)

	It("failed to get product info", testUseCase.GetProductFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to get default project info", testUseCase.GetDefaultProjectFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	It("will created successfully", testUseCase.CreateResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("will updated successfully", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("auto merge conflict, updated successfully", testUseCase.UpdateResourceAndAutoMerge(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to auto merge conflict", testUseCase.MergeConflictFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to push code retry three times", testUseCase.CreateResourceAndAutoRetry(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	It("modify resource but non compliant layout", testUseCase.UpdateResourceButNotConformTemplate(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to save config", testUseCase.SaveConfigFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.SaveEnvironment(context.Background(), bizOptions, enviromentData)
		Expect(err).Should(HaveOccurred())
	}))

	Describe("check reference by resources", func() {
		It("incorrect product name", testUseCase.CheckReferenceButIncorrectProduct(fakeNodes, func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree) {
			biz := NewEnviromentUsecase(logger, nautesConfigs, nil, nodestree, nil)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		}))
		It("cluster reference not found", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			objKey := client.ObjectKey{
				Namespace: nautesConfigs.Nautes.Namespace,
				Name:      fakeResource.Spec.Cluster,
			}
			client := kubernetes.NewMockClient(ctl)
			client.EXPECT().Get(gomock.Any(), objKey, &resourcev1alpha1.Cluster{}).Return(ErrorResourceNoFound)

			biz := NewEnviromentUsecase(logger, nautesConfigs, nil, nodestree, nil)
			ok, err := biz.CheckReference(options, fakeNode, client)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})
})

var _ = Describe("Delete environment", func() {
	var (
		resourceName = "env1"
		fakeResource = createEnvironmentResource(resourceName, _TestClusterHostEnvType, _TestDeploymentClusterName)
		fakeNode     = createEnvironmentNode(fakeResource)
		fakeNodes    = createContainEnvironmentNodes(fakeNode)
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
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.DeleteEnvironment(context.Background(), bizOptions)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("modify resource but non compliant layout standards", testUseCase.DeleteResourceErrorLayout(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewEnviromentUsecase(logger, nautesConfigs, codeRepo, nodestree, resourceUseCase)
		err := biz.DeleteEnvironment(context.Background(), bizOptions)
		Expect(err).Should(HaveOccurred())
	}))
})
