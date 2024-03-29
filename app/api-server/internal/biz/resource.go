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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/tidwall/sjson"

	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	kustomize "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type RretryCountType string
type getResouceName func(nodes nodestree.Node) (string, error)

type ResourcesUsecase struct {
	log          *log.Helper
	codeRepo     CodeRepo
	secretRepo   Secretrepo
	gitRepo      GitRepo
	nodeOperator nodestree.NodesTree
	configs      *nautesconfigs.Config
}

func NewResourcesUsecase(logger log.Logger, codeRepo CodeRepo, secretRepo Secretrepo, gitRepo GitRepo, nodeOperator nodestree.NodesTree, configs *nautesconfigs.Config) *ResourcesUsecase {
	return &ResourcesUsecase{
		log:          log.NewHelper(log.With(logger)),
		codeRepo:     codeRepo,
		secretRepo:   secretRepo,
		gitRepo:      gitRepo,
		nodeOperator: nodeOperator,
		configs:      configs,
	}
}

func (r *ResourcesUsecase) Get(ctx context.Context, resourceKind, productName string, _ nodestree.NodesOperator, getResourceName getResouceName) (*nodestree.Node, error) {
	_, project, err := r.GetGroupAndProjectByGroupID(ctx, productName)
	if err != nil {
		return nil, err
	}

	localPath, err := r.CloneCodeRepo(ctx, project.HttpUrlToRepo)
	if err != nil {
		return nil, err
	}

	defer cleanCodeRepo(localPath)

	err = r.nodeOperator.FilterIgnoreByLayout(localPath)
	if err != nil {
		return nil, err
	}

	nodes, err := r.nodeOperator.Load(localPath)
	if err != nil {
		return nil, err
	}

	resourceName, err := getResourceName(nodes)
	if err != nil {
		return nil, err
	}

	resourceNode := r.GetNode(&nodes, resourceKind, resourceName)
	if resourceNode == nil {
		return nil, ErrorResourceNoFound
	}

	return resourceNode, nil
}

func (r *ResourcesUsecase) List(ctx context.Context, gid interface{}, _ nodestree.NodesOperator) (*nodestree.Node, error) {
	_, project, err := r.GetGroupAndProjectByGroupID(ctx, gid)
	if err != nil {
		if commonv1.IsProjectNotFound(err) {
			return nil, commonv1.ErrorProjectNotFound("%s is non product, missing metadata", gid)
		}
		if commonv1.IsGroupNotFound(err) {
			return nil, commonv1.ErrorGroupNotFound(err.Error())
		}
		return nil, err
	}

	localPath, err := r.CloneCodeRepo(ctx, project.HttpUrlToRepo)
	if err != nil {
		return nil, err
	}

	defer cleanCodeRepo(localPath)

	err = r.nodeOperator.FilterIgnoreByLayout(localPath)
	if err != nil {
		return nil, err
	}

	nodes, err := r.nodeOperator.Load(localPath)
	if err != nil {
		return nil, err
	}

	return &nodes, nil
}

type resourceOptions struct {
	resourceKind      string
	resourceName      string
	productName       string
	insecureSkipCheck bool
	operator          nodestree.NodesOperator
}

// Save create or update config to git platform
func (r *ResourcesUsecase) Save(ctx context.Context, resourceOptions *resourceOptions, data interface{}) error {
	var resourceNode *nodestree.Node

	product, project, err := r.GetGroupAndProjectByGroupID(ctx, resourceOptions.productName)
	if err != nil {
		r.log.Errorf("failed to get product and coderepo data, err: %s", err)
		return err
	}

	localPath, err := r.CloneCodeRepo(ctx, project.HttpUrlToRepo)
	if err != nil {
		r.log.Errorf("failed to clone coderepo %s", project.HttpUrlToRepo)
		return err
	}

	defer cleanCodeRepo(localPath)

	err = r.nodeOperator.FilterIgnoreByLayout(localPath)
	if err != nil {
		return err
	}

	nodes, err := r.nodeOperator.Load(localPath)
	if err != nil {
		r.log.Errorf("failed to load nodes tree, err: %s", err)
		return err
	}

	options := nodestree.CompareOptions{
		Nodes:       nodes,
		ProductName: fmt.Sprintf("%s%d", ProductPrefix, int(product.ID)),
	}

	resourceNode = r.GetNode(&nodes, resourceOptions.resourceKind, resourceOptions.resourceName)
	if resourceNode == nil {
		resourceNode, err = resourceOptions.operator.CreateNode(localPath, data)
		if err != nil {
			r.log.Errorf("failed to create node, err: %s", err)
			return err
		}
	} else {
		resourceNode, err = resourceOptions.operator.UpdateNode(resourceNode, data)
		if err != nil {
			r.log.Errorf("failed to update node, err: %s", err)
			return err
		}
	}

	newNodes, err := r.InsertNodes(r.nodeOperator, &nodes, resourceNode)
	if err != nil {
		r.log.Errorf("failed to insert node, err: %s", err)
		return err
	}

	if !resourceOptions.insecureSkipCheck {
		options.Nodes = *newNodes
		err = r.nodeOperator.Compare(options)
		if err != nil {
			r.log.Errorf("failed to validate, err: %s", err)
			return err
		}
	}

	err = r.WriteResource(resourceNode)
	if err != nil {
		r.log.Errorf("failed to write resource, err: %s", err)
		return err
	}

	err = r.updateKustomization(&nodes, localPath)
	if err != nil {
		r.log.Errorf("failed to saved deploy config", "err", err)
		return err
	}

	err = r.PushToGit(ctx, localPath)
	if err != nil {
		r.log.Errorf("failed to git submission, err: %s", err)
		return err
	}

	return nil
}

func (r *ResourcesUsecase) Delete(ctx context.Context, resourceOptions *resourceOptions, getResourceName getResouceName) error {
	product, project, err := r.GetGroupAndProjectByGroupID(ctx, resourceOptions.productName)
	if err != nil {
		return err
	}

	localPath, err := r.CloneCodeRepo(ctx, project.HttpUrlToRepo)
	if err != nil {
		return err
	}

	defer cleanCodeRepo(localPath)

	err = r.nodeOperator.FilterIgnoreByLayout(localPath)
	if err != nil {
		return err
	}

	nodes, err := r.nodeOperator.Load(localPath)
	if err != nil {
		return err
	}

	options := nodestree.CompareOptions{
		Nodes:       nodes,
		ProductName: fmt.Sprintf("%s%d", ProductPrefix, int(product.ID)),
	}

	resourceName, err := getResourceName(nodes)
	if err != nil {
		return err
	}

	resourceNode := r.GetNode(&nodes, resourceOptions.resourceKind, resourceName)
	if resourceNode == nil {
		return fmt.Errorf("%s resource %s not found or invalid. Please check whether the resource exists under the default project", resourceOptions.resourceKind, resourceName)
	}

	newNodes, err := r.RemoveNode(&nodes, resourceNode)
	if err != nil {
		return err
	}

	if !resourceOptions.insecureSkipCheck {
		options.Nodes = *newNodes
		err = r.nodeOperator.Compare(options)
		if err != nil {
			return err
		}
	}

	err = deleteResource(resourceNode)
	if err != nil {
		return err
	}

	err = r.updateKustomization(&nodes, localPath)
	if err != nil {
		return err
	}

	err = r.PushToGit(ctx, localPath)
	if err != nil {
		return err
	}

	return nil
}

func (r *ResourcesUsecase) loadDefaultProjectNodes(ctx context.Context, productName string) (*nodestree.Node, error) {
	_, project, err := r.GetGroupAndProjectByGroupID(ctx, productName)
	if err != nil {
		return nil, err
	}
	path, err := r.CloneCodeRepo(ctx, project.HttpUrlToRepo)
	if err != nil {
		return nil, err
	}

	err = r.nodeOperator.FilterIgnoreByLayout(path)
	if err != nil {
		return nil, err
	}

	nodes, err := r.nodeOperator.Load(path)
	if err != nil {
		return nil, err
	}

	return &nodes, nil
}

func (r *ResourcesUsecase) InsertNodes(nodeOperator nodestree.NodesTree, nodes, resource *nodestree.Node) (*nodestree.Node, error) {
	return nodeOperator.InsertNodes(nodes, resource)
}

// GetNode get specifial node according to resource kind and name
func (r *ResourcesUsecase) GetNode(nodes *nodestree.Node, kind, resourceName string) *nodestree.Node {
	return r.nodeOperator.GetNode(nodes, kind, resourceName)
}

func (r *ResourcesUsecase) GetNodes() (*nodestree.Node, error) {
	return r.nodeOperator.GetNodes()
}

// RemoveNode delete the specified node according to the path
func (r *ResourcesUsecase) RemoveNode(nodes, node *nodestree.Node) (*nodestree.Node, error) {
	nodes, err := r.nodeOperator.RemoveNode(nodes, node)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *ResourcesUsecase) GetGroupAndProjectByGroupID(ctx context.Context, gid interface{}) (*Group, *Project, error) {
	group, err := r.codeRepo.GetGroup(ctx, gid)
	if err != nil {
		return nil, nil, err
	}

	toGetCodeRepo := fmt.Sprintf("%s/%s", group.Path, r.configs.Git.DefaultProductName)
	project, err := r.codeRepo.GetCodeRepo(ctx, toGetCodeRepo)
	if err != nil {
		return nil, nil, err
	}

	return group, project, nil
}

// GetCodeRepoName The name of the codeRepo resource must be prefixed with repo-, eg: repo-1
func (r *ResourcesUsecase) GetCodeRepo(ctx context.Context, productName, codeRepoName string) (*Project, error) {
	pid := ""
	group, err := r.codeRepo.GetGroup(ctx, productName)
	if err != nil {
		return nil, err
	}

	pid = fmt.Sprintf("%s/%s", group.Path, codeRepoName)
	project, err := r.codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (r *ResourcesUsecase) CloneCodeRepo(ctx context.Context, url string) (path string, err error) {
	user, email, err := r.codeRepo.GetCurrentUser(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current user, err: %w", err)
	}

	param := &CloneRepositoryParam{
		URL:   url,
		User:  user,
		Email: email,
	}
	localCodeRepoPath, err := r.gitRepo.Clone(ctx, param)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository, the repository url: %s, err: %w", url, err)
	}

	return localCodeRepoPath, nil
}

// WriteResource Write project resource content to a file
func (r *ResourcesUsecase) WriteResource(node *nodestree.Node) error {
	jsonBytes, err := json.Marshal(node.Content)
	if err != nil {
		return fmt.Errorf("failed to convert resource to json data, err: %v", err)
	}

	jsonString, err := sjson.Delete(string(jsonBytes), "status")
	if err != nil {
		return fmt.Errorf("failed to delete status field of resource, err: %v", err)
	}

	yamlBytes, err := yaml.JSONToYAML([]byte(jsonString))
	if err != nil {
		return fmt.Errorf("failed to convert json to yaml data, err: %v", err)
	}

	subPath := filepath.Dir(node.Path)
	_, err = os.Stat(subPath)
	if !os.IsExist(err) {
		err := os.MkdirAll(subPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write resource directory, err: %v", err)
		}
	}

	err = os.WriteFile(node.Path, yamlBytes, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write resource file, err: %v", err)
	}

	return nil
}

func (r *ResourcesUsecase) updateKustomization(nodes *nodestree.Node, path string) error {
	deployDir := fmt.Sprintf("%s/%s", path, r.configs.Deploy.ArgoCD.Kustomize.DefaultPath.DefaultProject)
	kustomizationFilePath := fmt.Sprintf("%s/%s", deployDir, KustomizationFileName)
	kustomization := &kustomize.Kustomization{
		TypeMeta: kustomize.TypeMeta{
			APIVersion: kustomize.KustomizationVersion,
			Kind:       kustomize.KustomizationKind,
		},
		Resources: []string{},
	}

	addKustomizeResources(nodes, kustomization, path)

	bytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}

	err = writeKustomize(kustomizationFilePath, bytes)
	if err != nil {
		return err
	}

	return nil
}

func addKustomizeResources(nodes *nodestree.Node, kustomization *kustomize.Kustomization, path string) {
	if nodes != nil {
		for _, v := range nodes.Children {
			if !v.IsDir {
				relativePath := strings.ReplaceAll(v.Path, path, "..")
				kustomization.Resources = append(kustomization.Resources, relativePath)
			} else if v.IsDir && len(v.Children) > 0 {
				addKustomizeResources(v, kustomization, path)
			}
		}
	}
}

func writeKustomize(path string, bytes []byte) error {
	_, err := os.Stat(filepath.Dir(path))
	if err != nil {
		err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

// PushToGit Save project resource config to git platform
// If automatic merge fails will retry three times
func (r *ResourcesUsecase) PushToGit(ctx context.Context, path string) error {
	count := getCount(ctx)
	if count == nil {
		ctx = withCount(ctx, 1)
	}

	_, err := r.gitRepo.Fetch(ctx, path, "origin")
	if err != nil {
		return err
	}

	data, err := r.gitRepo.Diff(ctx, path, "main", "remotes/origin/main")
	if err != nil {
		return err
	}

	if data == "" {
		err = r.gitRepo.SaveConfig(ctx, path)
		if err != nil {
			return err
		}
	} else {
		err = r.retryAutoMerge(ctx, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ResourcesUsecase) retryAutoMerge(ctx context.Context, path string) error {
	_, err := r.gitRepo.Fetch(ctx, path)
	if err != nil {
		return fmt.Errorf("when the save configuration cannot be fetch remote branch, err: %v", err)
	}

	err = r.gitRepo.Commit(ctx, path)
	if err != nil {
		return err
	}

	_, err = r.gitRepo.Merge(ctx, path)
	if err != nil {
		return fmt.Errorf("when the save configuration cannot be merge automatically, manual approval may be required, err: %v", err)
	}

	err = r.gitRepo.Push(ctx, path)
	if err != nil {
		ok, count := isMergeExceededTimes(ctx, 3)

		if !ok {
			count++
			ctx = withCount(ctx, count)
			time.Sleep(3 * time.Second)
			return r.PushToGit(ctx, path)
		}

		err = fmt.Errorf("failed to save config, err: %v", err)
		return err
	}

	return nil
}

func isMergeExceededTimes(ctx context.Context, exceed int) (bool, int) {
	count := getCount(ctx)
	val, ok := count.(int)
	// if the count type error, immediately terminate retry.
	if !ok {
		return true, exceed
	}

	if val == exceed {
		return true, val
	}

	return false, val
}

func cleanCodeRepo(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return
	}

	_ = os.RemoveAll(filename)
}

func withCount(ctx context.Context, val interface{}) context.Context {
	return context.WithValue(ctx, RretryCount, val)
}

func getCount(ctx context.Context) interface{} {
	count := ctx.Value(RretryCount)
	return count
}

func deleteResource(node *nodestree.Node) (err error) {
	fileinfos, err := os.ReadDir(filepath.Dir(node.Path))
	if err != nil {
		if ok := os.IsNotExist(err); ok {
			return nil
		}
		return err
	}

	if len(fileinfos) == 1 {
		err = os.RemoveAll(filepath.Dir(node.Path))
		if err != nil {
			return err
		}
	} else {
		err = os.Remove(node.Path)
		if err != nil {
			return
		}
	}

	return
}
