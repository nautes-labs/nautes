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
	"strconv"

	"github.com/golang/mock/gomock"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createFakeCcontainingCodeRepoNodes(node *nodestree.Node) nodestree.Node {
	return nodestree.Node{
		Name:  defaultProjectName,
		Path:  defaultProjectName,
		IsDir: true,
		Level: 1,
		Children: []*nodestree.Node{
			{
				Name:  CodeReposSubDir,
				Path:  fmt.Sprintf("%v/%v", defaultProjectName, CodeReposSubDir),
				IsDir: true,
				Level: 2,
				Children: []*nodestree.Node{
					{
						Name:  node.Name,
						Path:  fmt.Sprintf("%s/%s/%s", localRepositoryPath, CodeReposSubDir, node.Name),
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

var _ = Describe("Get codeRepo", func() {
	var (
		resourceName      = "toGetCodeRepo"
		fakeResource      = createFakeCodeRepoResource(resourceName)
		fakeNode          = createFakeCodeRepoNode(fakeResource)
		fakeNodes         = createFakeCcontainingCodeRepoNodes(fakeNode)
		gid, _            = utilstrings.ExtractNumber(ProductPrefix, fakeResource.Spec.Product)
		group             = &Group{ID: int32(gid), Name: fakeResource.Spec.Product, Path: fakeResource.Spec.Product}
		project           = &Project{ID: MockID1, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", resourceName)}
		productPrefixPath = fmt.Sprintf("%s/%s", fakeResource.Spec.Product, resourceName)
		groupPath         = fmt.Sprintf("%s/%s", defaultGroupName, resourceName)
	)

	It("will get successfully", testUseCase.GetResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourcesUsecase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(productPrefixPath)).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(groupPath)).Return(project, nil)

		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(gid)).Return(group, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		projectCodeRepo, err := biz.GetCodeRepo(context.Background(), resourceName, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(projectCodeRepo.CodeRepo).To(Equal(fakeResource))
	}))

	It("will fail when resource is not found", testUseCase.GetResourceFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(productPrefixPath)).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(groupPath)).Return(project, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, nil, nil)
		_, err := biz.GetCodeRepo(context.Background(), resourceName, defaultGroupName)
		Expect(err).Should(HaveOccurred())
	}))
})

var _ = Describe("List coderepos", func() {
	var (
		resourceName = MockCodeRepo1Name
		fakeResource = createFakeCodeRepoResource(resourceName)
		fakeNode     = createFakeCodeRepoNode(fakeResource)
		fakeNodes    = createFakeCcontainingCodeRepoNodes(fakeNode)
		pid, _       = utilstrings.ExtractNumber(RepoPrefix, resourceName)
		project      = &Project{ID: int32(pid), Name: fakeResource.Spec.RepoName, Path: fakeResource.Spec.RepoName}
	)

	It("will list successfully", testUseCase.ListResourceSuccess(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().ListGroupCodeRepos(gomock.Any(), gomock.Eq(defaultGroupName), gomock.Any()).Return([]*Project{project}, nil).AnyTimes()

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, nil, nil)
		_, err := biz.ListCodeRepos(ctx, defaultGroupName)
		Expect(err).ShouldNot(HaveOccurred())
	}))

})

var _ = Describe("Save codeRepo", func() {
	var (
		resourceName  = MockCodeRepo1Name
		pid, _        = utilstrings.ExtractNumber(RepoPrefix, resourceName)
		fakeResource  = createFakeCodeRepoResource(resourceName)
		fakeNode      = createFakeCodeRepoNode(fakeResource)
		fakeNodes     = createFakeCcontainingCodeRepoNodes(fakeNode)
		toSaveProject = &Project{
			ID:            MockID1,
			HttpUrlToRepo: "https://gitlab.com/nautes-labs/test.git",
			Namespace: &ProjectNamespace{
				ID:   MockID2,
				Path: defaultGroupName,
			},
		}
		toSaveProjectDeployKey = &ProjectDeployKey{
			ID:  2013,
			Key: "FingerprintData",
		}
		extendKVs         = make(map[string]string)
		toGetCodeRepoPath = fmt.Sprintf("%s/%s", defaultProductGroup.Path, resourceName)
		repoName          = getCodeRepoResourceName(int(toSaveProject.ID))
		data              = &CodeRepoData{
			Name: resourceName,
			Spec: resourcev1alpha1.CodeRepoSpec{
				CodeRepoProvider:  "provider",
				Product:           "product",
				Project:           MockProject1Name,
				RepoName:          repoName,
				DeploymentRuntime: true,
				Webhook: &resourcev1alpha1.Webhook{
					Events: []string{"push_events"},
				},
			},
		}
		options = &GitCodeRepoOptions{
			Gitlab: &GitlabCodeRepoOptions{
				Name:        repoName,
				Path:        repoName,
				NamespaceID: defaultProductGroup.ID,
			},
		}
		bizOptions = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
		listDeployKeys = []*ProjectDeployKey{
			{
				ID:    2013,
				Key:   "Key1",
				Title: "deploykey1",
			},
			{
				ID:    2014,
				Key:   "Key2",
				Title: "deploykey2",
			},
		}
		sameTitleListDeployKeys = []*ProjectDeployKey{
			{
				ID:    2013,
				Key:   "Key1",
				Title: "deploykey1",
			},
			{
				ID:    2014,
				Key:   "Key2",
				Title: "deploykey2",
			},
			{
				ID:    2015,
				Key:   "Key3",
				Title: fmt.Sprintf("%s%d-%s", RepoPrefix, pid, ReadWrite),
			},
		}
		projectAccessToken  = &ProjectAccessToken{ID: 81, Token: "access token"}
		projectAccessTokens = []*ProjectAccessToken{
			{ID: 81, Name: "token1", Token: "access token"},
			{ID: 82, Name: "token2", Token: "access token"},
			{ID: 83, Name: "token2", Token: "access token"},
		}
		accessTokenSecretData = &AccessTokenSecretData{
			ID: 81,
		}
		accessTokenName                 = AccessTokenName
		scopes                          = []string{"api"}
		accessLevel                     = AccessLevelValue(40)
		createProjectAccessTokenOptions = &CreateProjectAccessTokenOptions{
			Name:        &accessTokenName,
			Scopes:      &scopes,
			AccessLevel: &accessLevel,
		}
		accessTokenExtendKVs = make(map[string]string)
		secretOptions        = &SecretOptions{
			SecretPath:   fmt.Sprintf("%s/%s/%s/%s", string(nautesConfigs.Git.GitType), repoName, DefaultUser, AccessTokenName),
			SecretEngine: SecretsGitEngine,
			SecretKey:    SecretsAccessToken,
		}
		codeRepoKind = nodestree.CodeRepo
	)

	BeforeEach(func() {
		extendKVs["fingerprint"] = toSaveProjectDeployKey.Key
		extendKVs["id"] = strconv.Itoa(toSaveProjectDeployKey.ID)

		accessTokenExtendKVs[AccessTokenID] = strconv.Itoa(projectAccessToken.ID)
	})

	It("failed to get product info", testUseCase.GetProductFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, nil, nil)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to get default project info", testUseCase.GetDefaultProjectFail(func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(toSaveProject, nil)
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, nil, nil)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	It("will created successfully", testUseCase.CreateResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), gomock.Eq(int(defaultProductGroup.ID)), options).Return(toSaveProject, nil)

		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessToken, nil).AnyTimes()
		codeRepo.EXPECT().CreateProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Eq(createProjectAccessTokenOptions)).Return(projectAccessToken, nil)
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(nil, commonv1.ErrorAccesstokenNotFound("failed to get access token from secret repo"))
		secretRepo.EXPECT().SaveProjectAccessToken(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(toSaveProject.ID))), gomock.Eq(projectAccessToken.Token), gomock.Any(), gomock.Any(), gomock.Eq(accessTokenExtendKVs)).Return(nil)

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)

		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("will updated successfully", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)

		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(toSaveProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(toSaveProject.ID), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessToken, nil).AnyTimes()
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(accessTokenSecretData, nil)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("clear same titile invalid deploy key", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)

		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(sameTitleListDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessToken, nil).AnyTimes()
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(nil)

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(accessTokenSecretData, nil)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to get repository access token", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)

		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(toSaveProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(toSaveProject.ID), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(nil, commonv1.ErrorAccesstokenNotFound("failed to get access token")).AnyTimes()
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()
		codeRepo.EXPECT().CreateProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Eq(createProjectAccessTokenOptions)).Return(projectAccessToken, nil)

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(accessTokenSecretData, nil)
		secretRepo.EXPECT().SaveProjectAccessToken(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(toSaveProject.ID))), gomock.Eq(projectAccessToken.Token), gomock.Any(), gomock.Any(), gomock.Eq(accessTokenExtendKVs)).Return(nil)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to get access token from secret repo", testUseCase.UpdateResoureSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)

		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(toSaveProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(toSaveProject.ID), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessToken, nil).AnyTimes()
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()
		codeRepo.EXPECT().CreateProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Eq(createProjectAccessTokenOptions)).Return(projectAccessToken, nil)

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(nil, commonv1.ErrorAccesstokenNotFound("failed to get access token"))
		secretRepo.EXPECT().SaveProjectAccessToken(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(toSaveProject.ID))), gomock.Eq(projectAccessToken.Token), gomock.Any(), gomock.Any(), gomock.Eq(accessTokenExtendKVs)).Return(nil)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, nil)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("auto merge conflict, updated successfully", testUseCase.UpdateResourceAndAutoMerge(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(toSaveProject, nil).AnyTimes()
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(toSaveProjectDeployKey, nil).Times(2)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(toSaveProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(toSaveProject.ID), int(2014)).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessToken, nil).AnyTimes()
		codeRepo.EXPECT().CreateProjectAccessToken(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Eq(createProjectAccessTokenOptions)).Return(projectAccessToken, nil)
		codeRepo.EXPECT().ListAccessTokens(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), gomock.Any()).Return(projectAccessTokens, nil).AnyTimes()

		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found")).Times(2)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), extendKVs).Return(nil).Times(2).Times(2)
		secretRepo.EXPECT().GetProjectAccessToken(gomock.Any(), secretOptions).Return(nil, commonv1.ErrorAccesstokenNotFound("failed to get access token from secret repo"))
		secretRepo.EXPECT().SaveProjectAccessToken(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(toSaveProject.ID))), gomock.Eq(projectAccessToken.Token), gomock.Any(), gomock.Any(), gomock.Eq(accessTokenExtendKVs)).Return(nil)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   toSaveProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).ShouldNot(HaveOccurred())
	}))

	It("failed to auto merge conflict", testUseCase.MergeConflictFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), gomock.Eq(int(defaultProductGroup.ID)), options).Return(toSaveProject, nil)
		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to push code retry three times", testUseCase.CreateResourceAndAutoRetry(fakeNodes, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), gomock.Eq(int(defaultProductGroup.ID)), options).Return(toSaveProject, nil)

		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	It("modify resource but non compliant layout", testUseCase.UpdateResourceButNotConformTemplate(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(toSaveProject, nil)
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)
		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	It("failed to save config", testUseCase.SaveConfigFail(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(toGetCodeRepoPath)).Return(toSaveProject, nil)
		codeRepo.EXPECT().UpdateCodeRepo(gomock.Any(), gomock.Eq(int(toSaveProject.ID)), options).Return(toSaveProject, nil)
		client.EXPECT().List(context.Background(), gomock.Any(), gomock.Any()).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.SaveCodeRepo(context.Background(), bizOptions, data, options)
		Expect(err).Should(HaveOccurred())
	}))

	Describe("check reference by resources", func() {
		It("incorrect product name", testUseCase.CheckReferenceButIncorrectProduct(fakeNodes, func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree) {
			biz := NewCodeRepoUsecase(logger, nil, nil, nodestree, nautesConfigs, nil, nil, nil)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		}))

		It("webhook matching failed ", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())
			newResouce := fakeResource.DeepCopy()
			newResouce.Spec.Webhook.Events = append(newResouce.Spec.Webhook.Events, "errorWebhook")
			node := createFakeCodeRepoNode(newResouce)

			biz := NewCodeRepoUsecase(logger, nil, nil, nodestree, nautesConfigs, nil, nil, nil)
			ok, err := biz.CheckReference(options, node, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("project reference not found", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			biz := NewCodeRepoUsecase(logger, nil, nil, nodestree, nautesConfigs, nil, nil, nil)
			ok, err := biz.CheckReference(options, fakeNode, nil)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("codeRepo provider reference not found", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			objKey := client.ObjectKey{
				Namespace: nautesConfigs.Nautes.Namespace,
				Name:      fakeResource.Spec.CodeRepoProvider,
			}

			client := kubernetes.NewMockClient(ctl)
			client.EXPECT().Get(gomock.Any(), objKey, &resourcev1alpha1.CodeRepoProvider{}).Return(ErrorResourceNoFound)

			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			options.Nodes.Children = append(options.Nodes.Children, projectNodes.Children...)

			biz := NewCodeRepoUsecase(logger, nil, nil, nodestree, nautesConfigs, nil, nil, nil)
			ok, err := biz.CheckReference(options, fakeNode, client)
			Expect(err).Should(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("will successed", func() {
			options := nodestree.CompareOptions{
				Nodes:       fakeNodes,
				ProductName: defaultProductId,
			}
			nodestree := nodestree.NewMockNodesTree(ctl)
			nodestree.EXPECT().AppendOperators(gomock.Any())

			objKey := client.ObjectKey{
				Namespace: nautesConfigs.Nautes.Namespace,
				Name:      fakeResource.Spec.CodeRepoProvider,
			}

			client := kubernetes.NewMockClient(ctl)
			client.EXPECT().Get(gomock.Any(), objKey, &resourcev1alpha1.CodeRepoProvider{}).Return(nil)

			projectName := fakeResource.Spec.Project
			projectNodes := createProjectNodes(createProjectNode(createProjectResource(projectName)))
			options.Nodes.Children = append(options.Nodes.Children, projectNodes.Children...)

			biz := NewCodeRepoUsecase(logger, nil, nil, nodestree, nautesConfigs, nil, nil, nil)
			ok, err := biz.CheckReference(options, fakeNode, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})
})

var _ = Describe("Delete codeRepo", func() {
	var (
		resourceName   = MockCodeRepo1Name
		fakeResource   = createFakeCodeRepoResource(resourceName)
		fakeNode       = createFakeCodeRepoNode(fakeResource)
		fakeNodes      = createFakeCcontainingCodeRepoNodes(fakeNode)
		deletedProject = &Project{ID: MockID1}
		bizOptions     = &BizOptions{
			ResouceName: resourceName,
			ProductName: defaultGroupName,
		}
		codeRepoKind   = nodestree.CodeRepo
		listDeployKeys = []*ProjectDeployKey{
			{
				ID:  2013,
				Key: "Key1",
			},
			{
				ID:  2014,
				Key: "Key2",
			},
		}
	)

	BeforeEach(func() {
		err := os.MkdirAll(filepath.Dir(fakeNode.Path), 0644)
		Expect(err).ShouldNot(HaveOccurred())
		_, err = os.Create(fakeNode.Path)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("will deleted successfully", testUseCase.DeleteResourceSuccess(fakeNodes, fakeNode, func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient) {
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(deletedProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(deletedProject.ID), gomock.Any()).Return(nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(deletedProject, nil).AnyTimes()
		codeRepo.EXPECT().DeleteCodeRepo(gomock.Any(), gomock.Eq(int(deletedProject.ID))).Return(nil)
		secretRepo.EXPECT().DeleteSecret(gomock.Any(), gomock.Eq(int(deletedProject.ID)), DefaultUser, string(ReadOnly)).Return(nil)
		secretRepo.EXPECT().DeleteSecret(gomock.Any(), gomock.Eq(int(deletedProject.ID)), DefaultUser, string(ReadWrite)).Return(nil)
		secretRepo.EXPECT().DeleteSecret(gomock.Any(), gomock.Eq(int(deletedProject.ID)), DefaultUser, string(AccessTokenName)).Return(nil)
		codeRepoBindingUsecase := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourceUseCase, nautesConfigs, client)

		cloneRepositoryParam := &CloneRepositoryParam{
			URL:   deletedProject.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode).AnyTimes()
		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil).AnyTimes()

		biz := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourceUseCase, codeRepoBindingUsecase, client)
		err := biz.DeleteCodeRepo(context.Background(), bizOptions)
		Expect(err).ShouldNot(HaveOccurred())
	}))
})

func createFakeCodeRepoResource(name string) *resourcev1alpha1.CodeRepo {
	return &resourcev1alpha1.CodeRepo{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		TypeMeta: v1.TypeMeta{
			Kind: nodestree.CodeRepo,
		},
		Spec: resourcev1alpha1.CodeRepoSpec{
			Product:           defaultProductId,
			RepoName:          name,
			Project:           MockProject1Name,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"push_events"},
			},
		},
	}
}

func createFakeCodeRepoNode(resource *resourcev1alpha1.CodeRepo) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Path:    fmt.Sprintf("%s/%s/%s/%s.yaml", localRepositoryPath, CodeReposSubDir, resource.Name, resource.Name),
		Level:   4,
		Content: resource,
		Kind:    nodestree.CodeRepo,
	}
}
