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
	"fmt"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	MockCodeRepoBinding1Name = "codeRepoBinding1"
)

var _ = Describe("Get CodeRepoBinding", func() {
	var (
		resourceName          = MockCodeRepoBinding1Name
		fakeResource          = createFakeCodeRepoBindingResource(resourceName, "project1", MockCodeRepoName, string(ReadOnly))
		fakeNode              = createFakeCodeRepoBindingNode(fakeResource)
		fakeNodes             = createFakeContainingCodeRepoBindingNodes(fakeNode)
		fakeErrorKindResource = createFakeCodeRepoBindingErrorKindResource(resourceName, "project1")
		fakeErrorKindNode     = createFakeCodeRepoNode(fakeErrorKindResource)
		fakeErrorKindNodes    = createFakeContainingCodeRepoBindingNodes(fakeErrorKindNode)
		gid, _                = utilstrings.ExtractNumber(ProductPrefix, fakeResource.Spec.Product)
		project               = &Project{ID: MockProject1ID, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", "codeRepo1")}
		projectDeployKey      = &ProjectDeployKey{
			ID:  MockProject3ID,
			Key: "FingerprintData",
		}
		cloneRepositoryParam = &CloneRepositoryParam{
			URL:   project.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		deployKeySecretData = &DeployKeySecretData{
			ID:          MockProject3ID,
			Fingerprint: "Fingerprint",
		}
	)

	It("get CodeRepoBinding successfully", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeNode).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		item, err := biz.GetCodeRepoBinding(ctx, options)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(item.Spec).Should(Equal(fakeResource.Spec))
	})

	It("failed to get CodeRepoBinding", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeErrorKindNodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeErrorKindNode).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		_, err := biz.GetCodeRepoBinding(ctx, options)
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("List CodeRepoBinding", func() {
	var (
		resourceName          = MockCodeRepoBinding1Name
		fakeResource          = createFakeCodeRepoBindingResource(resourceName, MockProject, MockCodeRepoName, string(ReadOnly))
		fakeNode              = createFakeCodeRepoBindingNode(fakeResource)
		fakeNodes             = createFakeContainingCodeRepoBindingNodes(fakeNode)
		fakeErrorKindResource = createFakeCodeRepoBindingErrorKindResource(resourceName, "project1")
		fakeErrorKindNode     = createFakeCodeRepoNode(fakeErrorKindResource)
		fakeErrorKindNodes    = createFakeContainingCodeRepoBindingNodes(fakeErrorKindNode)
		gid, _                = utilstrings.ExtractNumber(ProductPrefix, fakeResource.Spec.Product)
		project               = &Project{ID: MockProject1ID, HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", "codeRepo1")}
		projectDeployKey      = &ProjectDeployKey{
			ID:  MockProject3ID,
			Key: "FingerprintData",
		}
		cloneRepositoryParam = &CloneRepositoryParam{
			URL:   project.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		deployKeySecretData = &DeployKeySecretData{
			ID:          MockProject3ID,
			Fingerprint: "Fingerprint",
		}
	)

	It("list CodeRepoBinding successfully", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeNode).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		_, err := biz.ListCodeRepoBindings(ctx, options)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to get CodeRepoBinding", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeErrorKindNodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeErrorKindNode).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		_, err := biz.GetCodeRepoBinding(ctx, options)
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("Save CodeRepoBinding", func() {
	var (
		resourceName  = MockCodeRepoBinding1Name
		fakeResource1 = createFakeCodeRepoBindingResource(resourceName, MockProject, MockCodeRepoName, string(ReadOnly))
		fakeNode1     = createFakeCodeRepoBindingNode(fakeResource1)
		fakeNodes1    = createFakeContainingCodeRepoBindingNodes(fakeNode1)
		fakeResource2 = createFakeCodeRepoBindingResource(resourceName, "", MockCodeRepoName, string(ReadOnly))
		fakeNode2     = createFakeCodeRepoBindingNode(fakeResource2)
		fakeNodes2    = createFakeContainingCodeRepoBindingNodes(fakeNode2)
		repoName1     = "codeRepo1"
		repoName2     = "codeRepo2"
		fakeResource3 = createFakeCodeRepoResource(repoName2)
		fakeNode3     = createFakeCodeRepoNode(fakeResource3)
		project1      = &Project{
			ID:            MockProject1ID,
			HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", repoName1),
			Namespace: &ProjectNamespace{
				ID: ProductID,
			},
		}
		project2 = &Project{
			ID:            MockProject2ID,
			HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", repoName2),
			Namespace: &ProjectNamespace{
				ID: ProductID,
			},
		}
		cloneRepositoryParam = &CloneRepositoryParam{
			URL:   project1.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		projectDeployKey = &ProjectDeployKey{
			ID:  MockProject3ID,
			Key: "FingerprintData",
		}
		deployKeySecretData = &DeployKeySecretData{
			ID:          MockProject3ID,
			Fingerprint: "Fingerprint",
		}
		listProjectDeployKeys = []*ProjectDeployKey{projectDeployKey}
		gid, _                = utilstrings.ExtractNumber(ProductPrefix, fakeResource1.Spec.Product)
		codeRepoKind          = nodestree.CodeRepo
		CodeRepoBindingKind   = nodestree.CodeRepoBinding
	)
	It("will create CodeRepoBinding according to the product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource1.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), fmt.Sprintf("%s/%s", defaultGroupName, fakeResource1.Spec.CodeRepo)).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject2ID).Return(project2, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(listProjectDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		codeRepo.EXPECT().EnableProjectDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().UpdateDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(projectDeployKey, nil).AnyTimes().AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil).AnyTimes()
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes2, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(CodeRepoBindingKind), gomock.Any()).Return(fakeNode1).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode3).AnyTimes()
		nodestree.EXPECT().GetNodes().Return(&fakeNodes2, nil)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&fakeNodes2, nil).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		data := &CodeRepoBindingData{
			Name: resourceName,
			Spec: fakeResource2.Spec,
		}
		err := biz.SaveCodeRepoBinding(ctx, options, data)
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("will update CodeRepoBinding according to the product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource1.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), fmt.Sprintf("%s/%s", defaultGroupName, fakeResource1.Spec.CodeRepo)).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject2ID).Return(project2, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(listProjectDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		codeRepo.EXPECT().EnableProjectDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().UpdateDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(projectDeployKey, nil).AnyTimes().AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil).AnyTimes()
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes2, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(CodeRepoBindingKind), gomock.Any()).Return(fakeNode1).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode3).AnyTimes()
		nodestree.EXPECT().GetNodes().Return(&fakeNodes2, nil)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&fakeNodes2, nil).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		data := &CodeRepoBindingData{
			Name: resourceName,
			Spec: fakeResource2.Spec,
		}
		err := biz.SaveCodeRepoBinding(ctx, options, data)
		Expect(err).ShouldNot(HaveOccurred())
	})
	It("will save CodeRepoBinding when specifying project", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource1.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), fmt.Sprintf("%s/%s", defaultGroupName, fakeResource1.Spec.CodeRepo)).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject1ID).Return(project1, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), MockProject2ID).Return(project2, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(listProjectDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		codeRepo.EXPECT().EnableProjectDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().UpdateDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(projectDeployKey, nil).AnyTimes().AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil).AnyTimes()
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes1, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(CodeRepoBindingKind), gomock.Any()).Return(fakeNode2).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNode3).AnyTimes()
		nodestree.EXPECT().GetNodes().Return(&fakeNodes2, nil)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&fakeNodes1, nil).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		k8sClient := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, k8sClient)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		data := &CodeRepoBindingData{
			Name: resourceName,
			Spec: fakeResource1.Spec,
		}
		err := biz.SaveCodeRepoBinding(ctx, options, data)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("check reference by resources", func() {
		It("failed to find resource referenct project", func() {
			codeRepo := NewMockCodeRepo(ctl)

			gitRepo := NewMockGitRepo(ctl)

			nodeOperator := nodestree.NewMockNodesTree(ctl)
			nodeOperator.EXPECT().AppendOperators(gomock.Any()).AnyTimes()

			secretRepo := NewMockSecretrepo(ctl)
			resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodeOperator, nautesConfigs)
			k8sClient := kubernetes.NewMockClient(ctl)

			biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodeOperator, resourcesUsecase, nautesConfigs, k8sClient)

			options := nodestree.CompareOptions{
				Nodes:       fakeNodes1,
				ProductName: defaultProductId,
			}
			_, err := biz.CheckReference(options, fakeNode1, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

var _ = Describe("Delete CodeRepoBinding", func() {
	var (
		resourceName  = MockCodeRepoBinding1Name
		fakeResource  = createFakeCodeRepoBindingResource(resourceName, MockProject, MockCodeRepoName, string(ReadOnly))
		fakeNode      = createFakeCodeRepoBindingNode(fakeResource)
		fakeNodes     = createFakeContainingCodeRepoBindingNodes(fakeNode)
		repoName      = "codeRepo2"
		fakeResource2 = createFakeCodeRepoResource(repoName)
		fakeNodes2    = createFakeCodeRepoNode(fakeResource2)
		gid, _        = utilstrings.ExtractNumber(ProductPrefix, fakeResource.Spec.Product)
		project       = &Project{
			ID:            MockProject1ID,
			HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", "codeRepo1"),
			Namespace: &ProjectNamespace{
				ID: MockProject2ID,
			},
		}
		project2 = &Project{
			ID:            1225,
			HttpUrlToRepo: fmt.Sprintf("ssh://git@gitlab.io/nautes-labs/%s.git", "codeRepo2"),
			Namespace: &ProjectNamespace{
				ID: 1236,
			},
		}
		projectDeployKey = &ProjectDeployKey{
			Title: fmt.Sprintf("repo-%d-readonly", project.ID),
			ID:    1,
			Key:   "FingerprintData",
		}
		projectDeployKey2 = &ProjectDeployKey{
			Title: fmt.Sprintf("repo-%d-readonly", project2.ID),
			ID:    2,
			Key:   "FingerprintData",
		}
		cloneRepositoryParam = &CloneRepositoryParam{
			URL:   project.HttpUrlToRepo,
			User:  GitUser,
			Email: GitEmail,
		}
		deployKeySecretData = &DeployKeySecretData{
			ID:          MockProject3ID,
			Fingerprint: "Fingerprint",
		}
		listProjectDeployKeys = []*ProjectDeployKey{projectDeployKey2}
		codeRepoKind          = nodestree.CodeRepo
		CodeRepoBindingKind   = nodestree.CodeRepoBinding
	)

	It("delete CodeRepoBinding successfully", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), fakeResource.Spec.Product).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), gid).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetGroup(gomock.Any(), defaultGroupName).Return(defaultProductGroup, nil).AnyTimes()

		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(project, nil).AnyTimes()
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(listProjectDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		codeRepo.EXPECT().EnableProjectDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(projectDeployKey, nil).AnyTimes()
		codeRepo.EXPECT().UpdateDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(projectDeployKey, nil).AnyTimes().AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Any()).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(fakeNodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(CodeRepoBindingKind), gomock.Any()).Return(fakeNode).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Eq(codeRepoKind), gomock.Any()).Return(fakeNodes2).AnyTimes()
		nodestree.EXPECT().GetNodes().Return(&fakeNodes, nil)
		nodestree.EXPECT().RemoveNode(&fakeNodes, fakeNode).Return(&emptyNodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		biz := NewCodeRepoCodeRepoBindingUsecase(logger, codeRepo, secretRepo, nodestree, resourcesUsecase, nautesConfigs, client)
		options := &BizOptions{
			ProductName: defaultGroupName,
			ResouceName: resourceName,
		}
		err := biz.DeleteCodeRepoBinding(ctx, options)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func createFakeCodeRepoBindingErrorKindResource(name, project string) *resourcev1alpha1.CodeRepo {
	crd := &resourcev1alpha1.CodeRepo{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		TypeMeta: v1.TypeMeta{
			Kind: nodestree.CodeRepo,
		},
	}

	return crd
}

func createFakeCodeRepoBindingResource(name, project, authRepo, permission string) *resourcev1alpha1.CodeRepoBinding {
	crd := &resourcev1alpha1.CodeRepoBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		TypeMeta: v1.TypeMeta{
			Kind: nodestree.CodeRepoBinding,
		},
		Spec: resourcev1alpha1.CodeRepoBindingSpec{
			Product:     defaultProductId,
			CodeRepo:    authRepo,
			Projects:    []string{},
			Permissions: permission,
		},
	}
	if project != "" {
		crd.Spec.Projects = append(crd.Spec.Projects, project)
	}
	return crd
}

func createFakeCodeRepoBindingNode(resource *resourcev1alpha1.CodeRepoBinding) *nodestree.Node {
	return &nodestree.Node{
		Name:    resource.Name,
		Path:    fmt.Sprintf("%s/%s/%s/%s.yaml", localRepositoryPath, CodeReposSubDir, resource.Name, resource.Name),
		Level:   4,
		Content: resource,
		Kind:    nodestree.CodeRepoBinding,
	}
}

func createFakeContainingCodeRepoBindingNodes(node *nodestree.Node) nodestree.Node {
	projectID1 := fmt.Sprintf("%s%d", RepoPrefix, MockProject1ID)
	codeRepoNode1 := createFakeCodeRepoNode(createFakeCodeRepoResource(projectID1))
	projectID2 := fmt.Sprintf("%s%d", RepoPrefix, MockProject2ID)
	codeRepoNod2 := createFakeCodeRepoNode(createFakeCodeRepoResource(projectID2))
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
						Path:  fmt.Sprintf("%s/%s/%s", localRepositoryPath, CodeReposSubDir, codeRepoNode1.Name),
						IsDir: true,
						Level: 3,
						Children: []*nodestree.Node{
							node,
							codeRepoNode1,
							codeRepoNod2,
						},
					},
				},
			},
		},
	}
}
