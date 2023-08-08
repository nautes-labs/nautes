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
	"strconv"

	"github.com/golang/mock/gomock"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get product", func() {
	var (
		ProductName    = "test-1"
		defaultProduct = &GroupAndProjectItem{
			Group:   defaultProductGroup,
			Project: defautlProject,
		}
	)

	It("will get product successed", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(ProductName)).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(defautlProject, nil)

		secretRepo := NewMockSecretrepo(ctl)
		gitRepo := NewMockGitRepo(ctl)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nil, nautesConfigs)
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		p, err := product.GetProduct(ctx, ProductName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(p).To(Equal(defaultProduct))
	})

	It("will fail when product is not found", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(ProductName)).Return(nil, ErrorGroupNotFound)

		secretRepo := NewMockSecretrepo(ctl)
		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)

		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		result, err := product.GetProduct(ctx, ProductName)
		Expect(err).Should(HaveOccurred())
		Expect(result).ToNot(Equal(defaultProductGroup))
	})

	It("no default project found under the defaultProductGroup", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), gomock.Eq(ProductName)).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)

		secretRepo := NewMockSecretrepo(ctl)
		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		result, err := product.GetProduct(ctx, ProductName)
		Expect(err).Should(HaveOccurred())
		Expect(result).ToNot(Equal(defaultProductGroup))
	})
})

var _ = Describe("List products", func() {
	var (
		defaultItems = []*GroupAndProjectItem{
			{
				Group:   defaultProductGroup,
				Project: defautlProject,
			},
		}
	)

	It("will list products success", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().ListCodeRepos(gomock.Any(), DefaultProject).Return([]*Project{defautlProject}, nil)
		codeRepo.EXPECT().GetGroup(gomock.Any(), defautlProject.Namespace.ID).Return(defaultProductGroup, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		result, err := product.ListProducts(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result).To(Equal(defaultItems))
	})

	It("project is empty under the product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().ListCodeRepos(gomock.Any(), DefaultProject).Return([]*Project{}, nil)

		secretRepo := NewMockSecretrepo(ctl)
		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)
		result, err := product.ListProducts(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(result).ToNot(Equal([]*Group{}))
	})
})

var _ = Describe("Save product", func() {
	var (
		productName = "test-1"
		gitOptions  = &GitGroupOptions{
			Gitlab: &GroupOptions{
				Name: "test-1",
				Path: "test-2",
			},
		}
		defaultProjectDeployKey = &ProjectDeployKey{
			ID:  2013,
			Key: "Fingerprint",
		}
		deployKeySecretData = &DeployKeySecretData{
			ID:          2013,
			Fingerprint: "Fingerprint",
		}
		extendKVs      = make(map[string]string)
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
		extendKVs["fingerprint"] = defaultProjectDeployKey.Key
		extendKVs["id"] = strconv.Itoa(defaultProjectDeployKey.ID)
	})

	It("will created product successfully", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorNodetNotFound)
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), int(defaultProductGroup.ID), gomock.Any()).Return(defautlProject, nil)
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), int(defautlProject.ID), gomock.Any(), gomock.Any(), gomock.Any()).Return(defaultProjectDeployKey, nil)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(defautlProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found"))
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(defautlProject.ID))), gomock.Any(), gomock.Any(), gomock.Any(), extendKVs).Return(nil)
		secretRepo.EXPECT().AuthorizationSecret(gomock.Any(), gomock.Eq(int(defautlProject.ID)), ArgoOperator, string(nautesConfigs.Git.GitType), nautesConfigs.Secret.Vault.MountPath).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("will updated product successfully", func() {
		newDefaultProjectDeployKey := &ProjectDeployKey{
			ID:  defaultProjectDeployKey.ID,
			Key: "new fingerprint",
		}
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorGroupNotFound)
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(defautlProject, nil)
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(newDefaultProjectDeployKey, nil)
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), int(defautlProject.ID), defaultProjectDeployKey.ID).Return(newDefaultProjectDeployKey, nil)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(listDeployKeys, nil)
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil)
		extendKVs = make(map[string]string)
		extendKVs["fingerprint"] = newDefaultProjectDeployKey.Key
		extendKVs["id"] = strconv.Itoa(newDefaultProjectDeployKey.ID)
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(defautlProject.ID))), gomock.Any(), gomock.Any(), gomock.Any(), extendKVs).Return(nil)
		secretRepo.EXPECT().AuthorizationSecret(gomock.Any(), gomock.Eq(int(defautlProject.ID)), ArgoOperator, string(nautesConfigs.Git.GitType), nautesConfigs.Secret.Vault.MountPath).Return(nil)

		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to create product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorGroupNotFound)
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("%v path is exists", defaultProductGroup.Path))

		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, nil, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, nil, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to update product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().UpdateGroup(gomock.Any(), int(defaultProductGroup.ID), gitOptions).Return(nil, fmt.Errorf("Unable to update defaultProductGroup"))
		codeRepo.EXPECT().ListGroupCodeRepos(gomock.Any(), int(defaultProductGroup.ID)).Return([]*Project{defautlProject}, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to save deploy key", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorGroupNotFound).AnyTimes()
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), int(defaultProductGroup.ID), gomock.Any()).Return(defautlProject, nil)
		codeRepo.EXPECT().GetDeployKey(gomock.Any(), int(defautlProject.ID), defaultProjectDeployKey.ID).Return(nil, commonv1.ErrorDeploykeyNotFound("the deploy key is not found"))
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), int(defautlProject.ID), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("unable to add deploy key"))
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(defautlProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(deployKeySecretData, nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to save secret", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorGroupNotFound).AnyTimes()
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), int(defaultProductGroup.ID), gomock.Any()).Return(defautlProject, nil)
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), int(defautlProject.ID), gomock.Any(), gomock.Any(), gomock.Any()).Return(defaultProjectDeployKey, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(defautlProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found"))
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(defautlProject.ID))), gomock.Any(), gomock.Any(), gomock.Any(), extendKVs).Return(fmt.Errorf("unable to save secret"))

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to authorization secret", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), productName).Return(nil, ErrorGroupNotFound).AnyTimes()
		codeRepo.EXPECT().CreateGroup(gomock.Any(), gitOptions).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(nil, ErrorProjectNotFound)
		codeRepo.EXPECT().CreateCodeRepo(gomock.Any(), int(defaultProductGroup.ID), gomock.Any()).Return(defautlProject, nil)
		codeRepo.EXPECT().SaveDeployKey(gomock.Any(), int(defautlProject.ID), gomock.Any(), gomock.Any(), gomock.Any()).Return(defaultProjectDeployKey, nil)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil)
		codeRepo.EXPECT().ListDeployKeys(gomock.Any(), int(defautlProject.ID), gomock.Any()).Return(listDeployKeys, nil).AnyTimes()
		codeRepo.EXPECT().DeleteDeployKey(gomock.Any(), int(defautlProject.ID), gomock.Any()).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetDeployKey(gomock.Any(), gomock.Any()).Return(nil, commonv1.ErrorSecretNotFound("secret data is not found"))
		secretRepo.EXPECT().SaveDeployKey(gomock.Any(), gomock.Eq(getCodeRepoResourceName(int(defautlProject.ID))), gomock.Any(), gomock.Any(), gomock.Any(), extendKVs).Return(nil)
		secretRepo.EXPECT().AuthorizationSecret(gomock.Any(), gomock.Eq(int(defautlProject.ID)), ArgoOperator, string(nautesConfigs.Git.GitType), nautesConfigs.Secret.Vault.MountPath).Return(fmt.Errorf("Failed to authorization"))

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), cloneRepositoryParam).Return(localRepositoryPath, nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), gomock.Eq(localRepositoryPath)).Return(nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, secretRepo, nodestree, nautesConfigs, resourcesUsecase, nil, nil)

		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		_, _, err := product.SaveProduct(ctx, productName, gitOptions)
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("Delete product", func() {
	var (
		TestProject = &Project{
			ID:                int32(297),
			Name:              "test",
			Path:              "test",
			WebUrl:            "https://github.com/test-2/test",
			SshUrlToRepo:      "ssh://git@github.com:2222/test-2/test.git",
			HttpUrlToRepo:     "https://github.com:2222/test-2/test.git",
			PathWithNamespace: "test-2/test",
		}
		ProductID = "test-2"
	)

	It("will delete product successfully", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), ProductID).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().ListGroupCodeRepos(gomock.Any(), int(defaultProductGroup.ID)).Return([]*Project{defautlProject}, nil)
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), defaultProjectPath).Return(defautlProject, nil)
		codeRepo.EXPECT().DeleteGroup(gomock.Any(), int(defaultProductGroup.ID)).Return(nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().DeleteSecret(gomock.Any(), int(defautlProject.ID), DefaultUser, string(ReadOnly)).Return(nil)

		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		err := product.DeleteProduct(ctx, ProductID)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to delete product", func() {
		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetGroup(gomock.Any(), ProductID).Return(defaultProductGroup, nil)
		codeRepo.EXPECT().ListGroupCodeRepos(gomock.Any(), int(defaultProductGroup.ID)).Return([]*Project{defautlProject, TestProject}, nil)

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)

		resourcesUsecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		nodestree := nodestree.NewMockNodesTree(ctl)
		nodestree.EXPECT().AppendOperators(gomock.Any())
		codeRepoUsecase := NewCodeRepoUsecase(logger, codeRepo, nil, nodestree, nautesConfigs, resourcesUsecase, nil, nil)
		product := NewProductUsecase(logger, codeRepo, secretRepo, gitRepo, nautesConfigs, resourcesUsecase, codeRepoUsecase)

		err := product.DeleteProduct(ctx, ProductID)
		Expect(err).Should(HaveOccurred())
	})
})
