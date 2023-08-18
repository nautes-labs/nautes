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
	"regexp"
	"strconv"
	"strings"
	"sync"

	errors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilerror "github.com/nautes-labs/nautes/app/api-server/util/error"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"golang.org/x/sync/singleflight"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CodeRepoBindingUsecase struct {
	log              *log.Helper
	codeRepo         CodeRepo
	secretRepo       Secretrepo
	nodestree        nodestree.NodesTree
	resourcesUsecase *ResourcesUsecase
	config           *nautesconfigs.Config
	client           client.Client
	groupName        string
	lock             sync.RWMutex
	wg               sync.WaitGroup
	cacheStore       *CacheStore
}

type CacheStore struct {
	// cache the deploykey information of the repository.
	// the key is the repository id, and the value is a deploykey infomation map.
	// this map key is the deploykey id, and the value is the deploykey infomation.
	projectDeployKeyMap map[int]map[int]*ProjectDeployKey
	// cache the deploykey and CodeRepo information of the repository.
	// the key is the deploykey title, and the value is deploykey and association repository infomation.
	deploykeyInAllProjectsMap map[string]*deployKeyMapValue
}

type CodeRepoBindingData struct {
	Name string
	Spec resourcev1alpha1.CodeRepoBindingSpec
}

type deployKeyMapValue struct {
	deployKey *ProjectDeployKey
	codeRepos []*resourcev1alpha1.CodeRepo
}

type applyDeploykeyFunc func(ctx context.Context, pid interface{}, deployKey int) error

func NewCodeRepoCodeRepoBindingUsecase(logger log.Logger, codeRepo CodeRepo, secretRepo Secretrepo, nodestree nodestree.NodesTree, resourcesUsecase *ResourcesUsecase, config *nautesconfigs.Config, client client.Client) *CodeRepoBindingUsecase {
	cacheStore := createCacheStore()

	codeRepoBindingUsecase := &CodeRepoBindingUsecase{
		log:              log.NewHelper(log.With(logger)),
		codeRepo:         codeRepo,
		secretRepo:       secretRepo,
		nodestree:        nodestree,
		resourcesUsecase: resourcesUsecase,
		config:           config,
		client:           client,
		cacheStore:       cacheStore,
	}
	nodestree.AppendOperators(codeRepoBindingUsecase)

	return codeRepoBindingUsecase
}

func (c *CodeRepoBindingUsecase) GetCodeRepoBinding(ctx context.Context, options *BizOptions) (*resourcev1alpha1.CodeRepoBinding, error) {
	node, err := c.resourcesUsecase.Get(ctx, nodestree.CodeRepoBinding, options.ProductName, c, func(nodes nodestree.Node) (string, error) {
		return options.ResouceName, nil
	})
	if err != nil {
		return nil, err
	}

	codeRepoBinding, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
	if !ok {
		return nil, fmt.Errorf("failed to get instance when get %s coderepoBinding", node.Name)
	}

	return codeRepoBinding, nil
}

func (c *CodeRepoBindingUsecase) ListCodeRepoBindings(ctx context.Context, options *BizOptions) ([]*nodestree.Node, error) {
	nodes, err := c.resourcesUsecase.List(ctx, options.ProductName, c)
	if err != nil {
		return nil, err
	}

	codeRepoBindingNodes := nodestree.ListsResourceNodes(*nodes, nodestree.CodeRepoBinding)

	return codeRepoBindingNodes, nil
}

func (c *CodeRepoBindingUsecase) SaveCodeRepoBinding(ctx context.Context, options *BizOptions, data *CodeRepoBindingData) error {
	c.groupName = options.ProductName

	resourceOptions := &resourceOptions{
		resourceName:      options.ResouceName,
		resourceKind:      nodestree.CodeRepoBinding,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          c,
	}
	if err := c.resourcesUsecase.Save(ctx, resourceOptions, data); err != nil {
		return err
	}

	latestNodes, err := c.resourcesUsecase.GetNodes()
	if err != nil {
		return err
	}

	return c.refreshAuthorization(ctx, latestNodes, data.Spec.CodeRepo)
}

func (c *CodeRepoBindingUsecase) DeleteCodeRepoBinding(ctx context.Context, options *BizOptions) error {
	resourceOptions := &resourceOptions{
		resourceName:      options.ResouceName,
		resourceKind:      nodestree.CodeRepoBinding,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          c,
	}

	nodes, err := c.resourcesUsecase.loadDefaultProjectNodes(ctx, options.ProductName)
	if err != nil {
		return err
	}
	node := c.nodestree.GetNode(nodes, nodestree.CodeRepoBinding, options.ResouceName)
	if node == nil {
		return fmt.Errorf("%s resource %s not found or invalid. Please check whether the resource exists under the default project", resourceOptions.resourceKind, options.ResouceName)
	}
	lastCodeRepoBinding, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
	if !ok {
		return fmt.Errorf("resource type is inconsistent, please check if this resource %s is legal", options.ResouceName)
	}

	if err := c.resourcesUsecase.Delete(ctx, resourceOptions, func(nodes nodestree.Node) (string, error) {
		return resourceOptions.resourceName, nil
	}); err != nil {
		return err
	}

	nodes, err = c.resourcesUsecase.GetNodes()
	if err != nil {
		return err
	}

	return c.refreshAuthorization(ctx, nodes, lastCodeRepoBinding.Spec.CodeRepo)
}

func (c *CodeRepoBindingUsecase) authorizeDeployKey(ctx context.Context, codeRepos []*resourcev1alpha1.CodeRepo, authorizationpid interface{}, permissions string) error {
	if err := c.applyDeploykey(ctx, authorizationpid, permissions, codeRepos, func(ctx context.Context, authorizationpid interface{}, deployKeyID int) error {
		value, ok := authorizationpid.(int)
		if !ok {
			return fmt.Errorf("the ID of the authorized repository is not of type int during applyDeploykey: %v", authorizationpid)
		}

		projectDeployKey, err := c.codeRepo.EnableProjectDeployKey(ctx, value, deployKeyID)
		if err != nil {
			return err
		}

		if permissions != string(ReadWrite) {
			return nil
		}

		_, err = c.codeRepo.UpdateDeployKey(ctx, value, projectDeployKey.ID, projectDeployKey.Title, true)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *CodeRepoBindingUsecase) applyDeploykey(ctx context.Context, authorizationpid interface{}, permissions string, codeRepos []*resourcev1alpha1.CodeRepo, fn applyDeploykeyFunc) error {
	for _, codeRepo := range codeRepos {
		pid, err := utilstrings.ExtractNumber(RepoPrefix, codeRepo.Name)
		if err != nil {
			return err
		}

		val, ok := authorizationpid.(int)
		if !ok {
			return fmt.Errorf("the ID of the authorized repository is not of type int during applyDeploykey: %v", authorizationpid)
		}
		if val == pid {
			continue
		}

		deployKeyTitle := fmt.Sprintf("%s%d-%s", RepoPrefix, pid, permissions)
		tmpMap, exists := c.cacheStore.deploykeyInAllProjectsMap[deployKeyTitle]
		if !exists {
			secretData, err := c.GetDeployKeyFromSecretRepo(ctx, fmt.Sprintf("%s%d", RepoPrefix, pid), DefaultUser, permissions)
			if err != nil {
				if commonv1.IsDeploykeyNotFound(err) {
					return nil
				}
				return err
			}
			tmpMap = &deployKeyMapValue{}
			tmpMap.deployKey = &ProjectDeployKey{}
			tmpMap.deployKey.ID = secretData.ID
		}

		deploykey, exists := c.cacheStore.projectDeployKeyMap[pid][tmpMap.deployKey.ID]
		if exists {
			err := fn(ctx, authorizationpid, deploykey.ID)
			if err != nil {
				return err
			}
			continue
		}

		deploykey, err = c.codeRepo.GetDeployKey(ctx, pid, tmpMap.deployKey.ID)
		if err != nil {
			if commonv1.IsDeploykeyNotFound(err) {
				c.log.Debugf("failed to get deploykey during applyDeploykey, repo id: %d, err: %w", pid, err)
				continue
			}
			return fmt.Errorf("failed to get deploykey during applyDeploykey, repo id: %d, err: %w", pid, err)
		}

		err = fn(ctx, authorizationpid, deploykey.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// refreshAuthorization is responsible for refreshing the authorization of a given code repository binding use case.
// It clears any invalid deploy keys, authorizes the same project repo, and processes authorization for each repo with each permission.
func (c *CodeRepoBindingUsecase) refreshAuthorization(ctx context.Context, nodes *nodestree.Node, skipRepositories ...string) error {
	err := c.initialCacheStore(*nodes, ctx)
	if err != nil {
		return err
	}
	// clear cache store.
	defer func() {
		c.cacheStore = createCacheStore()
	}()

	err = c.performDeployKeyValidationAndCleanup(ctx, *nodes)
	if err != nil {
		return err
	}

	err = c.authorizeForSameProjectRepo(ctx, *nodes)
	if err != nil {
		return err
	}

	err = c.processPermissionRefresh(ctx, nodes)
	if err != nil {
		return err
	}

	return nil
}

func (c *CodeRepoBindingUsecase) processPermissionRefresh(ctx context.Context, nodes *nodestree.Node) error {
	permissionToReposMap := c.mapPermissionsToRepositories(nodes)
	err := c.processPermissionMapAuthorization(ctx, permissionToReposMap, nodes)
	if err != nil {
		return err
	}

	return nil
}

func (c *CodeRepoBindingUsecase) processPermissionMapAuthorization(ctx context.Context, permissionToReposMap map[string][]string, nodes *nodestree.Node) error {
	var err error
	errorMessages := make([]string, 0)

	for permission, authRepos := range permissionToReposMap {
		for _, repo := range authRepos {
			err = c.processAuthorization(ctx, *nodes, permission, repo)
			if err != nil {
				if commonv1.IsNoAuthorization(err) {
					errorMessages = append(errorMessages, err.Error())
				} else {
					return err
				}
			}
		}
	}

	if len(errorMessages) > 0 {
		return commonv1.ErrorRefreshPermissionsAccessDenied("failed to refersh permission, err: %s", strings.Join(errorMessages, "|"))
	}
	return nil
}

func (*CodeRepoBindingUsecase) mapPermissionsToRepositories(nodes *nodestree.Node) map[string][]string {
	permissionToRepos := make(map[string][]string)
	codeRepoBindingNodes := nodestree.ListsResourceNodes(*nodes, nodestree.CodeRepoBinding)

	for _, codeRepoBindingNode := range codeRepoBindingNodes {
		codeRepoBinding, ok := codeRepoBindingNode.Content.(*resourcev1alpha1.CodeRepoBinding)
		if !ok {
			continue
		}

		permission := codeRepoBinding.Spec.Permissions
		reposWithPermission := permissionToRepos[permission]
		reposWithPermission = utilstrings.AddIfNotExists(reposWithPermission, codeRepoBinding.Spec.CodeRepo)
		permissionToRepos[permission] = reposWithPermission
	}

	return permissionToRepos
}

// processAuthorization Calculate the authorization scopes, Perform corresponding operations based on the authorization scopes.
func (c *CodeRepoBindingUsecase) processAuthorization(ctx context.Context, nodes nodestree.Node, permissions, authRepoName string) error {
	pid, err := utilstrings.ExtractNumber(RepoPrefix, authRepoName)
	if err != nil {
		return err
	}

	codeRepoBindings := c.getCodeRepoBindingsInAuthorizedRepo(ctx, nodes, authRepoName, permissions)
	scopes := c.calculateAuthorizationScopes(ctx, codeRepoBindings, permissions)
	for _, scope := range scopes {
		if scope.isProductPermission {
			err = c.updateAllAuthorization(ctx, nodes, pid, permissions, scope.ProductName)
			if err != nil {
				return err
			}
		} else {
			err = c.recycleAuthorizationByProjectScopes(ctx, scope.ProjectScopes, nodes, pid, permissions, scope.ProductName)
			if err != nil {
				return err
			}

			err = c.updateAuthorization(ctx, scope.ProjectScopes, nodes, pid, permissions, scope.ProductName)
			if err != nil {
				return err
			}
		}
	}

	if len(codeRepoBindings) == 0 {
		err := c.recycleAuthorizationByProjectScopes(ctx, nil, nodes, pid, permissions, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CodeRepoBindingUsecase) authorizeForSameProjectRepo(ctx context.Context, nodes nodestree.Node) error {
	codeRepos := c.getCodeRepos(nodes)
	errChan := make(chan interface{})
	semaphore := make(chan struct{}, 10)

	wg := sync.WaitGroup{}
	for i := 0; i < len(codeRepos); i++ {
		for j := i + 1; j < len(codeRepos); j++ {
			repo1 := codeRepos[i]
			repo2 := codeRepos[j]

			ok := isEqualProject(repo1, repo2)
			if !ok {
				continue
			}

			wg.Add(1)
			go func(repo1, repo2 *resourcev1alpha1.CodeRepo) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() {
					<-semaphore
				}()

				c.authorizeRepositories(ctx, repo1, repo2, errChan)
			}(repo1, repo2)
		}
	}

	go func() {
		wg.Wait()
		close(semaphore)
	}()

	return utilerror.CollectAndCombineErrors(errChan)
}

func isEqualProject(repo1 *resourcev1alpha1.CodeRepo, repo2 *resourcev1alpha1.CodeRepo) bool {
	if repo1.Name == repo2.Name {
		return false
	}

	if repo1.Spec.Project == "" && repo2.Spec.Project == "" {
		return false
	} else if repo1.Spec.Project != repo2.Spec.Project {
		return false
	}

	return true
}

func (*CodeRepoBindingUsecase) getCodeRepos(nodes nodestree.Node) []*resourcev1alpha1.CodeRepo {
	codeRepoNodes := nodestree.ListsResourceNodes(nodes, nodestree.CodeRepo)
	tmpCodeRepos := make([]*resourcev1alpha1.CodeRepo, 0)

	for _, codeRepoNode := range codeRepoNodes {
		codeRepo, ok := codeRepoNode.Content.(*resourcev1alpha1.CodeRepo)
		if ok {
			tmpCodeRepos = append(tmpCodeRepos, codeRepo)
		}
	}
	return tmpCodeRepos
}

func (c *CodeRepoBindingUsecase) authorizeRepositories(ctx context.Context, repo1, repo2 *resourcev1alpha1.CodeRepo, errChan chan interface{}) {
	c.lock.Lock()

	tmpSecretDeploykeyMap := make(map[string]*DeployKeySecretData, 0)

	roDeployKey1Info, err := c.getDeployKey(ctx, repo1, tmpSecretDeploykeyMap, ReadOnly)
	if err != nil {
		c.lock.Unlock()
		if commonv1.IsDeploykeyNotFound(err) {
			return
		}
		errChan <- err
		return
	}

	rwDeployKey1Info, err := c.getDeployKey(ctx, repo1, tmpSecretDeploykeyMap, ReadWrite)
	if err != nil {
		c.lock.Unlock()
		if commonv1.IsDeploykeyNotFound(err) {
			return
		}
		errChan <- err
		return
	}

	roDeployKey2Info, err := c.getDeployKey(ctx, repo2, tmpSecretDeploykeyMap, ReadOnly)
	if err != nil {
		c.lock.Unlock()
		if commonv1.IsDeploykeyNotFound(err) {
			return
		}
		errChan <- err
		return
	}

	rwDeployKey2Info, err := c.getDeployKey(ctx, repo2, tmpSecretDeploykeyMap, ReadWrite)
	if err != nil {
		c.lock.Unlock()
		if commonv1.IsDeploykeyNotFound(err) {
			return
		}
		errChan <- err
		return
	}

	c.lock.Unlock()

	err = c.enableProjectDeployKey(ctx, repo1, roDeployKey2Info, rwDeployKey2Info)
	if err != nil {
		errChan <- err
		return
	}

	err = c.enableProjectDeployKey(ctx, repo2, roDeployKey1Info, rwDeployKey1Info)
	if err != nil {
		errChan <- err
		return
	}
}

func (c *CodeRepoBindingUsecase) getDeployKey(ctx context.Context, repo *resourcev1alpha1.CodeRepo, tmpSecretDeploykeyMap map[string]*DeployKeySecretData, permission DeployKeyType) (*ProjectDeployKey, error) {
	var err error

	pid, err := utilstrings.ExtractNumber(RepoPrefix, repo.Name)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s-%s", repo.Name, permission)

	// Check if there is a deploy key in the cache, and if it does not exist, obtain it from the keystore.
	deployKey, ok := tmpSecretDeploykeyMap[key]
	if !ok {
		deployKey, err = c.GetDeployKeyFromSecretRepo(ctx, repo.Name, DefaultUser, string(permission))
		if err != nil {
			if commonv1.IsDeploykeyNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		tmpSecretDeploykeyMap[key] = deployKey
	}

	// Check if there is a deploy key for the project in the cache, and if it does not exist, obtain it from codeRepo.
	projectDeploykey, ok := c.cacheStore.projectDeployKeyMap[pid][deployKey.ID]
	if !ok {
		projectDeploykey, err = c.codeRepo.GetDeployKey(ctx, pid, deployKey.ID)
		if err != nil {
			return nil, err
		}
		c.cacheStore.projectDeployKeyMap[pid] = map[int]*ProjectDeployKey{
			deployKey.ID: projectDeploykey,
		}
	}

	return projectDeploykey, nil
}
func (c *CodeRepoBindingUsecase) enableProjectDeployKey(ctx context.Context, repo *resourcev1alpha1.CodeRepo, roDeployKeyInfo *ProjectDeployKey, rwDeployKeyInfo *ProjectDeployKey) error {
	pid, err := utilstrings.ExtractNumber(RepoPrefix, repo.Name)
	if err != nil {
		return err
	}

	if _, ok := c.cacheStore.projectDeployKeyMap[pid][roDeployKeyInfo.ID]; !ok {
		err = c.enableDeployKey(ctx, pid, roDeployKeyInfo.ID)
		if err != nil {
			return err
		}
	}

	if _, ok := c.cacheStore.projectDeployKeyMap[pid][rwDeployKeyInfo.ID]; !ok {
		err = c.enableDeployKey(ctx, pid, rwDeployKeyInfo.ID)
		if err != nil {
			return err
		}

		err = c.updateDeployKey(ctx, pid, rwDeployKeyInfo.ID, rwDeployKeyInfo.Title, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CodeRepoBindingUsecase) enableDeployKey(ctx context.Context, pid int, deployKeyID int) error {
	_, err := c.codeRepo.EnableProjectDeployKey(ctx, pid, deployKeyID)
	if err != nil {
		return err
	}
	return nil
}

func (c *CodeRepoBindingUsecase) updateDeployKey(ctx context.Context, pid int, deployKeyID int, title string, enabled bool) error {
	_, err := c.codeRepo.UpdateDeployKey(ctx, pid, deployKeyID, title, enabled)
	if err != nil {
		return err
	}
	return nil
}

func Contains(arr []int, target int) bool {
	for _, element := range arr {
		if element == target {
			return true
		}
	}
	return false
}

func (c *CodeRepoBindingUsecase) getCodeRepoBindingsInAuthorizedRepo(ctx context.Context, nodes nodestree.Node, codeRepoName, permissions string) []*resourcev1alpha1.CodeRepoBinding {
	codeRepoBindingNodes := nodestree.ListsResourceNodes(nodes, nodestree.CodeRepoBinding, func(node *nodestree.Node) bool {
		val, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
		if !ok {
			return false
		}

		if val.Spec.CodeRepo == codeRepoName && val.Spec.Permissions == permissions {
			return true
		}

		return false
	})

	codeRepoBindings := make([]*resourcev1alpha1.CodeRepoBinding, 0)
	for _, node := range codeRepoBindingNodes {
		val, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
		if ok {
			codeRepoBindings = append(codeRepoBindings, val)
		}
	}

	return codeRepoBindings
}

type ProductAuthorization struct {
	ProductName         string
	ProjectScopes       map[string]bool
	isProductPermission bool
}

// calculateAuthorizationScopes calculate the permissions for each product based on CodeRepobindings.
// Return a list of ProductAuthorization entities, recording the authorization scope for each product.
// Each entity contains permissions for product name, product level, and project scopes.
// Projectscopes have merged all project permissions under CodeRepoBinding for the product, This is a reserved data.
func (c *CodeRepoBindingUsecase) calculateAuthorizationScopes(ctx context.Context, codeRepoBindings []*resourcev1alpha1.CodeRepoBinding, permissions string) []*ProductAuthorization {
	scopes := []*ProductAuthorization{}

	for _, codeRepoBinding := range codeRepoBindings {
		isProductPermission := len(codeRepoBinding.Spec.Projects) == 0
		projectScopes := make(map[string]bool)

		for _, project := range codeRepoBinding.Spec.Projects {
			projectScopes[project] = true
		}

		existingScope := findExistingScope(scopes, codeRepoBinding.Spec.Product)
		if existingScope != nil {
			if isProductPermission {
				existingScope.isProductPermission = isProductPermission
			}
			for key, _ := range projectScopes {
				existingScope.ProjectScopes[key] = true
			}
		} else {
			// authorizedCodeRepoProject is project of the authorized repository, when isProductPermission is false that it is required.
			scopes = append(scopes, &ProductAuthorization{
				ProductName:         codeRepoBinding.Spec.Product,
				isProductPermission: isProductPermission,
				ProjectScopes:       projectScopes,
			})
		}
	}

	return scopes
}

func findExistingScope(scopes []*ProductAuthorization, productName string) *ProductAuthorization {
	for _, scope := range scopes {
		if scope.ProductName == productName {
			return scope
		}
	}
	return nil
}

// updateAllAuthorization Authorize all Codebase under the specified product.
func (c *CodeRepoBindingUsecase) updateAllAuthorization(ctx context.Context, nodes nodestree.Node, pid interface{}, permissions, productName string) error {
	codeRepos, err := nodesToCodeRepoists(nodes)
	if err != nil {
		return err
	}

	err = c.authorizeDeployKey(ctx, codeRepos, pid, permissions)
	if err != nil {
		return err
	}

	return nil
}

// updateAuthorization Authorize according to the authorization scope of the project
func (c *CodeRepoBindingUsecase) updateAuthorization(ctx context.Context, projectScopes map[string]bool, nodes nodestree.Node, pid interface{}, permissions, product string) error {
	if len(projectScopes) == 0 {
		return nil
	}

	authorizedCodeRepos := []*resourcev1alpha1.CodeRepo{}
	codeRepos, err := nodesToCodeRepoists(nodes)
	if err != nil {
		return fmt.Errorf("failed to convert codeRepo nodes when update authorization")
	}
	for _, codeRepo := range codeRepos {
		for project, _ := range projectScopes {
			if codeRepo.Spec.Project == project {
				authorizedCodeRepos = append(authorizedCodeRepos, codeRepo)
			}
		}
	}

	err = c.authorizeDeployKey(ctx, authorizedCodeRepos, pid, permissions)
	if err != nil {
		return err
	}

	return nil
}

// recycleAuthorization Recycle according to the authorization scope of the project.
func (c *CodeRepoBindingUsecase) recycleAuthorizationByProjectScopes(ctx context.Context, projectScopes map[string]bool, nodes nodestree.Node, authorizationpid interface{}, permissions, productName string) error {
	if len(projectScopes) == 0 {
		return nil
	}

	authorizedRepository, err := c.codeRepo.GetCodeRepo(ctx, authorizationpid)
	if err != nil {
		return err
	}

	codeRepos := []*resourcev1alpha1.CodeRepo{}
	currentProductName := fmt.Sprintf("%s%d", ProductPrefix, authorizedRepository.Namespace.ID)
	if productName == currentProductName {
		codeRepos, err = nodesToCodeRepoists(nodes, func(codeRepo *resourcev1alpha1.CodeRepo) bool {
			pid, err := utilstrings.ExtractNumber(RepoPrefix, codeRepo.Name)
			if err != nil {
				return false
			}

			if pid == authorizationpid {
				return false
			}

			if codeRepo.Spec.Project == "" {
				return true
			}

			_, ok := projectScopes[codeRepo.Spec.Project]
			return !ok
		})
		if err != nil {
			return err
		}
	} else {
		// TODO: Increase cross-product processing
	}

	if err := c.RevokeDeployKey(ctx, codeRepos, authorizationpid, permissions); err != nil {
		return err
	}
	return nil
}

// clearUnauthorizedDeployKeys Search for Deploykey without permission based on cached data 'projectDeployKeyMap', If it is found, delete it.
func (c *CodeRepoBindingUsecase) clearUnauthorizedDeployKeys(ctx context.Context, nodes nodestree.Node) error {
	re := regexp.MustCompile(`repo-(\d+)-(readonly|readwrite)`)

	for projectID, deploykeyMapValue := range c.cacheStore.projectDeployKeyMap {
		codeRepoName := fmt.Sprintf("%s%d", RepoPrefix, projectID)
		totalPermission := c.calculateCodeRepoTotalPermission(nodes, codeRepoName)
		if totalPermission == nil {
			continue
		}

		for deployKeyID, deploykey := range deploykeyMapValue {
			matches := re.FindStringSubmatch(deploykey.Title)
			if matches == nil || strconv.Itoa(projectID) == matches[1] {
				continue
			}

			pid := matches[1]
			permission := matches[2]
			codeRepoName := fmt.Sprintf("repo-%s", pid)

			if !c.shouldDeleteDeployKey(&nodes, totalPermission, codeRepoName, permission) {
				continue
			}

			err := c.codeRepo.DeleteDeployKey(ctx, projectID, deployKeyID)
			if err != nil {
				if commonv1.IsDeploykeyNotFound(err) {
					return nil
				}

				return err
			}
		}
	}

	return nil
}

// calculateCodeRepoTotalPermission Calculate the total permissions for the current authorization code repository.
func (c *CodeRepoBindingUsecase) calculateCodeRepoTotalPermission(nodes nodestree.Node, authorizationCodeRepoName string) *CodeRepoTotalPermission {

	// Initialize the totalPermission structure
	totalPermission := &CodeRepoTotalPermission{
		CodeRepoName: authorizationCodeRepoName,
		ReadWrite:    &CodeRepoPermission{},
		ReadOnly:     &CodeRepoPermission{},
	}

	// Helper function to update permissions
	updatePermission := func(codeRepoPermission *CodeRepoPermission, codeRepoBinding *resourcev1alpha1.CodeRepoBinding) {
		if len(codeRepoBinding.Spec.Projects) == 0 {
			codeRepoPermission.IsProduct = true
			return
		}
		for _, project := range codeRepoBinding.Spec.Projects {
			codeRepoPermission.Projects = utilstrings.AddIfNotExists(codeRepoPermission.Projects, project)
		}
	}

	// Iterate over the nodes to extract and process the CodeRepoBinding ones
	codeRepoBindingNodes := nodestree.ListsResourceNodes(nodes, nodestree.CodeRepoBinding)
	for _, codeRepoBindingNode := range codeRepoBindingNodes {
		codeRepoBinding, ok := codeRepoBindingNode.Content.(*resourcev1alpha1.CodeRepoBinding)
		if !ok || codeRepoBinding.Spec.CodeRepo != authorizationCodeRepoName {
			continue
		}

		switch codeRepoBinding.Spec.Permissions {
		case string(ReadOnly):
			updatePermission(totalPermission.ReadOnly, codeRepoBinding)
		case string(ReadWrite):
			updatePermission(totalPermission.ReadWrite, codeRepoBinding)
		}
	}

	// Update permissions based on the project of the authorization code library
	node := c.nodestree.GetNode(&nodes, nodestree.CodeRepo, authorizationCodeRepoName)
	if node != nil {
		codeRepo, ok := node.Content.(*resourcev1alpha1.CodeRepo)
		if ok && codeRepo.Spec.Project != "" {
			totalPermission.ReadOnly.Projects = utilstrings.AddIfNotExists(totalPermission.ReadOnly.Projects, codeRepo.Spec.Project)
			totalPermission.ReadWrite.Projects = utilstrings.AddIfNotExists(totalPermission.ReadWrite.Projects, codeRepo.Spec.Project)
		}
	}

	// Return the total permissions or nil based on the presence of any permissions
	if len(totalPermission.ReadOnly.Projects) > 0 || len(totalPermission.ReadWrite.Projects) > 0 {
		return totalPermission
	}

	return nil
}

func (c *CodeRepoBindingUsecase) shouldDeleteDeployKey(nodes *nodestree.Node, totalPermission *CodeRepoTotalPermission, repoName, permission string) bool {
	node := c.nodestree.GetNode(nodes, nodestree.CodeRepo, repoName)
	if node == nil {
		return true
	}

	codeRepo, ok := node.Content.(*resourcev1alpha1.CodeRepo)
	if !ok {
		return true
	}

	var relevantPermission *CodeRepoPermission
	switch permission {
	case string(ReadOnly):
		relevantPermission = totalPermission.ReadOnly
	case string(ReadWrite):
		relevantPermission = totalPermission.ReadWrite
	default:
		return false
	}

	if relevantPermission.IsProduct {
		return false
	}

	return !utilstrings.ContainsString(relevantPermission.Projects, codeRepo.Spec.Project)
}

// PerformDeployKeyValidationAndCleanup executes a sequence of tasks to validate and clean up
// Deploy Keys within the given context and nodes. It first initializes the cache store with the
// provided nodes, then proceeds to clear any unauthorized Deploy Keys, followed by the removal
// of invalid Deploy Keys. This function helps ensure the security and integrity of the Deploy Keys
// associated with the specified nodes.
func (c *CodeRepoBindingUsecase) performDeployKeyValidationAndCleanup(ctx context.Context, nodes nodestree.Node) error {
	err := c.clearUnauthorizedDeployKeys(ctx, nodes)
	if err != nil {
		return err
	}

	err = c.clearInvalidDeployKeys(ctx, nodes)
	if err != nil {
		return err
	}

	return nil
}

// clearInvalidDeployKeys Query the validity of deploykey based on cached data 'deploykeyInAllProjectsMap',
// If invalid data is found, mark for deletion.
func (c *CodeRepoBindingUsecase) clearInvalidDeployKeys(ctx context.Context, nodes nodestree.Node) error {
	repoMap := make(map[int]*Project)
	re := regexp.MustCompile(`repo-(\d+)-`)
	errChan := make(chan interface{})

	for key, val := range c.cacheStore.deploykeyInAllProjectsMap {
		match := re.FindStringSubmatch(key)
		if len(match) == 0 {
			continue
		}
		pid, err := strconv.Atoi(match[1])
		if err != nil {
			return err
		}

		deployKeyID := val.deployKey.ID

		c.wg.Add(1)

		go c.checkDeployKey(ctx, pid, deployKeyID, val.codeRepos, repoMap, errChan)
	}

	c.wg.Wait()

	return utilerror.CollectAndCombineErrors(errChan)
}

func (c *CodeRepoBindingUsecase) checkDeployKey(ctx context.Context, pid, deployKeyID int, codeRepos []*resourcev1alpha1.CodeRepo, repoMap map[int]*Project, errChan chan<- interface{}) {
	defer c.wg.Done()

	isDelete, err := c.isDeleteDeployKey(ctx, pid, deployKeyID, repoMap)
	if isDelete && err == nil {
		err = c.deleteAssociatedRepositoryDeployKey(ctx, codeRepos, deployKeyID)
	}

	if err != nil {
		errChan <- err
	}
}

func (c *CodeRepoBindingUsecase) isDeleteDeployKey(ctx context.Context, pid, deployKeyID int, repoMap map[int]*Project) (bool, error) {
	deleteDeployKeys := make(map[int]bool)

	isDeleteDeployKey, err := c.checkRepositoryExistence(ctx, repoMap, pid)
	if err != nil || !isDeleteDeployKey {
		return isDeleteDeployKey, err
	}

	isDeleteDeployKey, err = c.checkDeployKeyExistence(ctx, pid, deployKeyID, deleteDeployKeys)
	return isDeleteDeployKey, err
}

func (c *CodeRepoBindingUsecase) initialCacheStore(nodes nodestree.Node, ctx context.Context) error {
	codeRepoNodes := nodestree.ListsResourceNodes(nodes, nodestree.CodeRepo)
	errChan := make(chan interface{}, len(codeRepoNodes))

	for _, codeRepoNode := range codeRepoNodes {
		codeRepo, ok := codeRepoNode.Content.(*resourcev1alpha1.CodeRepo)
		if !ok {
			continue
		}

		c.wg.Add(1)

		go func(codeRepo *resourcev1alpha1.CodeRepo) {
			defer c.wg.Done()
			err := c.deduplicateAndCacheDeployKeys(ctx, codeRepo)
			if err != nil {
				errChan <- err
			}
		}(codeRepo)
	}

	c.wg.Wait()

	return utilerror.CollectAndCombineErrors(errChan)
}

func (c *CodeRepoBindingUsecase) deleteAssociatedRepositoryDeployKey(ctx context.Context, codeRepos []*resourcev1alpha1.CodeRepo, deployKeyID int) error {
	for _, codeRepo := range codeRepos {
		pid, err := utilstrings.ExtractNumber(RepoPrefix, codeRepo.Name)
		if err != nil {
			return err
		}

		if err := c.codeRepo.DeleteDeployKey(ctx, pid, deployKeyID); err != nil {
			return err
		}
	}
	return nil
}

func (c *CodeRepoBindingUsecase) deduplicateAndCacheDeployKeys(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo) error {
	pid, err := utilstrings.ExtractNumber(RepoPrefix, codeRepo.Name)
	if err != nil {
		return err
	}

	projectDeployKeys, err := GetAllDeployKeys(ctx, c.codeRepo, pid)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cacheStore.projectDeployKeyMap[pid] == nil {
		c.cacheStore.projectDeployKeyMap[pid] = make(map[int]*ProjectDeployKey)
	}

	for _, projectDeployKey := range projectDeployKeys {
		c.cacheStore.projectDeployKeyMap[pid][projectDeployKey.ID] = projectDeployKey

		if _, ok := c.cacheStore.deploykeyInAllProjectsMap[projectDeployKey.Title]; !ok {
			c.cacheStore.deploykeyInAllProjectsMap[projectDeployKey.Title] = &deployKeyMapValue{
				deployKey: projectDeployKey,
				codeRepos: []*resourcev1alpha1.CodeRepo{codeRepo},
			}
		} else {
			c.cacheStore.deploykeyInAllProjectsMap[projectDeployKey.Title].codeRepos = append(c.cacheStore.deploykeyInAllProjectsMap[projectDeployKey.Title].codeRepos, codeRepo)
		}
	}

	return nil
}

func (c *CodeRepoBindingUsecase) checkRepositoryExistence(ctx context.Context, repoMap map[int]*Project, pid int) (bool, error) {
	var repository *Project
	var ok bool
	var err error

	c.lock.Lock()

	repository, ok = repoMap[pid]
	if !ok {
		sg := &singleflight.Group{}
		sg.Do(fmt.Sprintf("%d", pid), func() (interface{}, error) {
			repository, err = c.codeRepo.GetCodeRepo(ctx, pid)
			if err != nil {
				if !commonv1.IsProjectNotFound(err) {
					c.lock.Unlock()
					return false, err
				}

				return true, nil
			}

			repoMap[pid] = repository

			return false, nil
		})
	}
	c.lock.Unlock()

	return false, nil
}

func (c *CodeRepoBindingUsecase) checkDeployKeyExistence(ctx context.Context, pid int, deployKeyID int, DeleteDeployKeys map[int]bool) (bool, error) {
	_, err := c.codeRepo.GetDeployKey(ctx, pid, deployKeyID)
	if err != nil {
		if !commonv1.IsDeploykeyNotFound(err) {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (c *CodeRepoBindingUsecase) RevokeDeployKey(ctx context.Context, codeRepos []*resourcev1alpha1.CodeRepo, authorizationpid interface{}, permissions string) error {
	for _, codeRepo := range codeRepos {
		pid, err := utilstrings.ExtractNumber(RepoPrefix, codeRepo.Name)
		if err != nil {
			return err
		}

		value, ok := authorizationpid.(int)
		if !ok {
			return fmt.Errorf("the ID of the authorized repository is not of type int during applyDeploykey: %v", authorizationpid)
		}
		if value == pid {
			continue
		}

		deployKeyTitle := fmt.Sprintf("%s%d-%s", RepoPrefix, pid, permissions)
		tmpMap, exists := c.cacheStore.deploykeyInAllProjectsMap[deployKeyTitle]
		if !exists {
			secretData, err := c.GetDeployKeyFromSecretRepo(ctx, codeRepo.Name, DefaultUser, permissions)
			if err != nil {
				if commonv1.IsDeploykeyNotFound(err) {
					return nil
				}
				return err
			}

			tmpMap = &deployKeyMapValue{}
			tmpMap.deployKey = &ProjectDeployKey{}
			tmpMap.deployKey.ID = secretData.ID
		}

		deploykey, exists := c.cacheStore.projectDeployKeyMap[value][tmpMap.deployKey.ID]
		if exists {
			err = c.codeRepo.DeleteDeployKey(ctx, value, deploykey.ID)
			if err != nil && !commonv1.IsDeploykeyNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func (c *CodeRepoBindingUsecase) GetDeployKeyFromSecretRepo(ctx context.Context, repoName, user, permissions string) (*DeployKeySecretData, error) {
	gitType := c.config.Git.GitType
	secretsEngine := SecretsGitEngine
	secretsKey := SecretsDeployKey
	secretPath := fmt.Sprintf("%s/%s/%s/%s", gitType, repoName, user, permissions)
	secretOptions := &SecretOptions{
		SecretPath:   secretPath,
		SecretEngine: secretsEngine,
		SecretKey:    secretsKey,
	}

	deployKeySecretData, err := c.secretRepo.GetDeployKey(ctx, secretOptions)
	if err != nil {
		return nil, err
	}

	return deployKeySecretData, nil
}

func (c *CodeRepoBindingUsecase) CheckReference(options nodestree.CompareOptions, node *nodestree.Node, k8sClient client.Client) (bool, error) {
	if node.Kind != nodestree.CodeRepoBinding {
		return false, nil
	}

	codeRepoBinding, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
	if !ok {
		return true, fmt.Errorf("wrong type found for %s node when checking CodeRepoBinding type", node.Name)
	}

	ok = nodestree.IsResourceExist(options, codeRepoBinding.Spec.CodeRepo, nodestree.CodeRepo)
	if !ok {
		return true, fmt.Errorf("the authorization repository %s does not exist when checking CodeRepoBinding %s", codeRepoBinding.Spec.CodeRepo, codeRepoBinding.Name)
	}

	// TODO:
	// Query through k8s cluster when crossing products.
	for _, project := range codeRepoBinding.Spec.Projects {
		ok = nodestree.IsResourceExist(options, project, nodestree.Project)
		if !ok {
			return true, fmt.Errorf("project resource %s does not exist or is invalid", project)
		}
	}

	return true, nil
}

func (c *CodeRepoBindingUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*CodeRepoBindingData)
	if !ok {
		return nil, fmt.Errorf("failed to creating specify node, the path: %s", path)
	}

	if len(val.Spec.Projects) == 0 {
		val.Spec.Projects = make([]string, 0)
	}

	codeRepo := &resourcev1alpha1.CodeRepoBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       nodestree.CodeRepoBinding,
			APIVersion: resourcev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, CodeReposSubDir)
	resourcePath := fmt.Sprintf("%s/%s/%s.yaml", resourceDirectory, val.Spec.CodeRepo, val.Name)

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourcePath,
		Content: codeRepo,
		Kind:    nodestree.CodeRepoBinding,
		Level:   4,
	}, nil
}

func (c *CodeRepoBindingUsecase) UpdateNode(resourceNode *nodestree.Node, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*CodeRepoBindingData)
	if !ok {
		return nil, fmt.Errorf("failed to get conderepo %s data when updating node", resourceNode.Name)
	}

	if len(val.Spec.Projects) == 0 {
		val.Spec.Projects = make([]string, 0)
	}

	codeRepoBinding, ok := resourceNode.Content.(*resourcev1alpha1.CodeRepoBinding)
	if !ok {
		return nil, fmt.Errorf("failed to get coderepo %s when updating node", resourceNode.Name)
	}

	if val.Spec.CodeRepo != codeRepoBinding.Spec.CodeRepo {
		return nil, errors.New(500, "NOT_ALLOWED_MODIFY", "It is not allowed to modify the authorized repository. If you want to change the authorized repository, please delete the authorization")
	}

	codeRepoBinding.Spec = val.Spec
	resourceNode.Content = codeRepoBinding

	return resourceNode, nil
}

func (c *CodeRepoBindingUsecase) CreateResource(kind string) interface{} {
	if kind != nodestree.CodeRepoBinding {
		return nil
	}

	return &resourcev1alpha1.CodeRepoBinding{}
}

func createCacheStore() *CacheStore {
	return &CacheStore{
		projectDeployKeyMap:       make(map[int]map[int]*ProjectDeployKey),
		deploykeyInAllProjectsMap: make(map[string]*deployKeyMapValue),
	}
}
