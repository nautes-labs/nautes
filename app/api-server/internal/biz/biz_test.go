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
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
)

const (
	MockID1 = iota + 1
	MockID2
	MockID3
)

var (
	MockProject1Name  = fmt.Sprintf("project-%d", MockID1)
	MockEnv1Name      = fmt.Sprintf("env-%d", MockID1)
	MockCodeRepo1Name = fmt.Sprintf("repo-%d", MockID1)
)

type BizFunc func(codeRepo *MockCodeRepo, secretRepo *MockSecretrepo, resourceUseCase *ResourcesUsecase, nodestree *nodestree.MockNodesTree, gitRepo *MockGitRepo, client *kubernetes.MockClient)
type CompareFunc func(options nodestree.CompareOptions, nodestree *nodestree.MockNodesTree)

type GetRequestTestCases interface {
	GetResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	GetResourceFail(fn BizFunc) interface{}
	GetResourceNoMatch(fn BizFunc) interface{}
}

type ListRequestTestCases interface {
	ListResourceSuccess(nodes nodestree.Node, fn BizFunc) interface{}
}

type SaveRequestTestCases interface {
	GetProductFail(fn BizFunc) interface{}
	GetDefaultProjectFail(fn BizFunc) interface{}
	CreateResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	CreateResourceAndAutoRetry(nodes nodestree.Node, fn BizFunc) interface{}
	CreateResourceButNotConformTemplate(fn BizFunc) interface{}
	UpdateResoureSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	UpdateResourceButNotConformTemplate(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	UpdateResourceAndAutoMerge(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	MergeConflictFail(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	SaveConfigFail(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	CheckReferenceButIncorrectProduct(nodes nodestree.Node, fn CompareFunc) interface{}
}

type DeleteRequestTestCases interface {
	DeleteResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
	DeleteResourceErrorLayout(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{}
}

type TestUseCases interface {
	GetRequestTestCases
	ListRequestTestCases
	SaveRequestTestCases
	DeleteRequestTestCases
}

type testBiz struct{}

func NewTestUseCase() TestUseCases {
	return &testBiz{}
}

var (
	cloneRepositoryParam = &CloneRepositoryParam{
		URL:   defautlProject.HttpUrlToRepo,
		User:  GitUser,
		Email: GitEmail,
	}
)

func (t *testBiz) GetResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, secretRepo, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) GetResourceFail(fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		fn(codeRepo, nil, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) GetResourceNoMatch(fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, nil, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) ListResourceSuccess(nodes nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, nil, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) ListResourceNotMatch(nodes nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, nil, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) GetProductFail(fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(nil, ErrorGroupNotFound)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, nil, nil, nautesConfigs)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		secretrepo := NewMockSecretrepo(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, nil, nil)
	}
}

func (t *testBiz) GetDefaultProjectFail(fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(nil, fmt.Errorf("default.project is not found"))

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, nil, nil, nautesConfigs)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()

		secretrepo := NewMockSecretrepo(ctl)
		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, nil, nil)
	}
}

func (t *testBiz) CreateResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretrepo := NewMockSecretrepo(ctl)
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) UpdateResoureSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretrepo := NewMockSecretrepo(ctl)
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretrepo, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) UpdateResourceAndAutoMerge(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		fetchRemoteOrigin := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any()).Return("any", nil).After(fetchRemoteOrigin)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("any", nil)
		gitRepo.EXPECT().Merge(gomock.Any(), localRepositoryPath).Return("any", nil)
		gitRepo.EXPECT().Commit(gomock.Any(), localRepositoryPath)
		gitRepo.EXPECT().Push(gomock.Any(), localRepositoryPath).Return(nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretrepo := NewMockSecretrepo(ctl)
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) MergeConflictFail(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		fetchRemoteOrigin := gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any()).Return("any", nil).After(fetchRemoteOrigin)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("any", nil)
		gitRepo.EXPECT().Merge(gomock.Any(), localRepositoryPath).Return("", fmt.Errorf("unabled to auto merge"))
		gitRepo.EXPECT().Commit(gomock.Any(), localRepositoryPath)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		secretrepo := NewMockSecretrepo(ctl)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) SaveConfigFail(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(fmt.Errorf("failed to save config"))
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		secretrepo := NewMockSecretrepo(ctl)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) CheckReferenceButIncorrectProduct(nodes nodestree.Node, fn CompareFunc) interface{} {
	return func() {
		options := nodestree.CompareOptions{
			Nodes:       nodes,
			ProductName: "test",
		}
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		fn(options, nodestree)
	}
}

func (t *testBiz) CreateResourceAndAutoRetry(nodes nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
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

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		secretrepo := NewMockSecretrepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretrepo, gitRepo, nodestree, nautesConfigs)

		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) CreateResourceButNotConformTemplate(fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(emptyNodes, nil)
		nodestree.EXPECT().Compare(gomock.Any()).Return(fmt.Errorf("resource project is no match"))
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		secretrepo := NewMockSecretrepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) UpdateResourceButNotConformTemplate(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(fmt.Errorf("resource project is no match"))
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node)
		nodestree.EXPECT().InsertNodes(gomock.Any(), gomock.Any()).Return(&nodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		secretrepo := NewMockSecretrepo(ctl)
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)
		client := kubernetes.NewMockClient(ctl)

		fn(codeRepo, secretrepo, resourcesUsecase, nodestree, gitRepo, client)
	}
}

func (t *testBiz) DeleteResourceSuccess(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()

		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(nil).AnyTimes()
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node).AnyTimes()
		nodestree.EXPECT().RemoveNode(gomock.Any(), node).Return(&emptyNodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, secretRepo, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) DeleteResouceNoMatch(nodes nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(ErrorResourceNoMatch)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil).AnyTimes()

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, secretRepo, resourcesUsecase, nodestree, gitRepo, nil)
	}
}

func (t *testBiz) DeleteResourceErrorLayout(nodes nodestree.Node, node *nodestree.Node, fn BizFunc) interface{} {
	return func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(defaultGroupName)).Return(defaultProductGroup, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Eq(defaultProjectPath)).Return(defautlProject, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(GitUser, GitEmail, nil).AnyTimes()

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil).AnyTimes()

		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any()).AnyTimes()
		nodestree.EXPECT().Load(gomock.Eq(localRepositoryPath)).Return(nodes, nil).AnyTimes()
		nodestree.EXPECT().Compare(gomock.Any()).Return(ErrorResourceNoMatch)
		nodestree.EXPECT().GetNode(gomock.Any(), gomock.Any(), gomock.Any()).Return(node).AnyTimes()
		nodestree.EXPECT().RemoveNode(gomock.Any(), node).Return(&emptyNodes, nil)
		nodestree.EXPECT().FilterIgnoreByLayout(localRepositoryPath).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nodestree, nautesConfigs)

		fn(codeRepo, secretRepo, resourcesUsecase, nodestree, gitRepo, nil)
	}
}
