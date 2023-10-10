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

	"github.com/go-kratos/kratos/v2/log"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	projectpipelineruntimev1 "github.com/nautes-labs/nautes/api/api-server/projectpipelineruntime/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nautes-labs/nautes/app/api-server/pkg/validate"
)

type ProjectPipelineRuntimeUsecase struct {
	log              *log.Helper
	codeRepo         CodeRepo
	nodeOperator     nodestree.NodesTree
	resourcesUsecase *ResourcesUsecase
	client           client.Client
	config           *nautesconfigs.Config
}

type ProjectPipelineRuntimeData struct {
	Name string
	Spec resourcev1alpha1.ProjectPipelineRuntimeSpec
}

func NewProjectPipelineRuntimeUsecase(logger log.Logger, codeRepo CodeRepo, nodeOperator nodestree.NodesTree, resourcesUsecase *ResourcesUsecase, client client.Client, config *nautesconfigs.Config) *ProjectPipelineRuntimeUsecase {
	runtime := &ProjectPipelineRuntimeUsecase{
		log:              log.NewHelper(log.With(logger)),
		codeRepo:         codeRepo,
		nodeOperator:     nodeOperator,
		resourcesUsecase: resourcesUsecase,
		client:           client,
		config:           config,
	}
	nodeOperator.AppendOperators(runtime)
	return runtime
}

func (p *ProjectPipelineRuntimeUsecase) GetProjectPipelineRuntime(ctx context.Context, projectPipelineName, productName string) (*nodestree.Node, error) {
	node, err := p.resourcesUsecase.Get(ctx, nodestree.ProjectPipelineRuntime, productName, p, func(nodes nodestree.Node) (string, error) {
		return projectPipelineName, nil
	})
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (p *ProjectPipelineRuntimeUsecase) ListProjectPipelineRuntimes(ctx context.Context, productName string) ([]*nodestree.Node, error) {
	nodes, err := p.resourcesUsecase.List(ctx, productName, p)
	if err != nil {
		return nil, err
	}

	projectPipelineRuntimeNodes := nodestree.ListsResourceNodes(*nodes, nodestree.ProjectPipelineRuntime)

	return projectPipelineRuntimeNodes, nil
}

func (p *ProjectPipelineRuntimeUsecase) SaveProjectPipelineRuntime(ctx context.Context, options *BizOptions, data *ProjectPipelineRuntimeData) error {
	codeRepoName, err := p.resourcesUsecase.ConvertRepoNameToCodeRepoName(ctx, options.ProductName, data.Spec.PipelineSource)
	if err != nil {
		return fmt.Errorf("failed to get codeRepo name when converting pipeline repository name, err: %v", err)
	}
	data.Spec.PipelineSource = codeRepoName

	// AdditionalResources is optional
	if data.Spec.AdditionalResources != nil && data.Spec.AdditionalResources.Git != nil {
		codeRepoName, err = p.resourcesUsecase.ConvertRepoNameToCodeRepoName(ctx, options.ProductName, data.Spec.AdditionalResources.Git.CodeRepo)
		if err != nil {
			return fmt.Errorf("failed to get codeRepo name when converting git repository name for additional resources, err: %v", err)
		}
		data.Spec.AdditionalResources.Git.CodeRepo = codeRepoName
	}

	for idx, eventSource := range data.Spec.EventSources {
		if eventSource.Gitlab != nil && eventSource.Gitlab.RepoName != "" {
			codeRepoName, err = p.resourcesUsecase.ConvertRepoNameToCodeRepoName(ctx, options.ProductName, eventSource.Gitlab.RepoName)
			if err != nil {
				return fmt.Errorf("failed to get codeRepo name when converting git repository name for event source, err: %v", err)
			}
			eventSource.Gitlab.RepoName = codeRepoName
			data.Spec.EventSources[idx] = eventSource
		}
	}

	resourceOptions := &resourceOptions{
		resourceName:      options.ResouceName,
		resourceKind:      nodestree.ProjectPipelineRuntime,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          p,
	}
	err = p.resourcesUsecase.Save(ctx, resourceOptions, data)
	if err != nil {
		return err
	}

	return nil
}

func (p *ProjectPipelineRuntimeUsecase) DeleteProjectPipelineRuntime(ctx context.Context, options *BizOptions) error {
	resourceOptions := &resourceOptions{
		resourceKind:      nodestree.ProjectPipelineRuntime,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          p,
	}
	err := p.resourcesUsecase.Delete(ctx, resourceOptions, func(nodes nodestree.Node) (string, error) {
		return options.ResouceName, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *ProjectPipelineRuntimeUsecase) IsRepositoryExist(ctx context.Context, productName, repoName string) (*Project, error) {
	project, err := p.resourcesUsecase.GetCodeRepo(ctx, productName, repoName)
	if err != nil {
		if ok := commonv1.IsProjectNotFound(err); ok {
			return nil, projectpipelineruntimev1.ErrorPipelineResourceNotFound("failed to get repository %s in product %s", repoName, productName)
		}

		return nil, err
	}

	return project, nil
}

func (p *ProjectPipelineRuntimeUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	var resourceNode *nodestree.Node

	val, ok := data.(*ProjectPipelineRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to save project when create specify node path: %s", path)
	}

	runtime := &resourcev1alpha1.ProjectPipelineRuntime{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.ProjectPipelineRuntime,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	storageResourceDirectory := fmt.Sprintf("%s/%s", path, ProjectsDir)
	resourceParentDir := fmt.Sprintf("%s/%s", storageResourceDirectory, val.Spec.Project)
	resourceFile := fmt.Sprintf("%s/%s.yaml", resourceParentDir, val.Name)
	resourceNode = &nodestree.Node{
		Name:    val.Name,
		Path:    resourceFile,
		Content: runtime,
		Kind:    nodestree.ProjectPipelineRuntime,
		Level:   4,
	}

	return resourceNode, nil
}
func (p *ProjectPipelineRuntimeUsecase) UpdateNode(node *nodestree.Node, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*ProjectPipelineRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to get project data when update %s node", node.Name)
	}

	runtime, ok := node.Content.(*resourcev1alpha1.ProjectPipelineRuntime)
	if !ok {
		return nil, fmt.Errorf("failed to get project insatnce when update %s node", node.Name)
	}

	if val.Spec.Project != runtime.Spec.Project {
		return nil, fmt.Errorf("existing pipeline runtime is not allow modifying the project field")
	}

	runtime.Spec = val.Spec
	node.Content = runtime

	return node, nil
}

func (p *ProjectPipelineRuntimeUsecase) CheckReference(options nodestree.CompareOptions, node *nodestree.Node, _ client.Client) (bool, error) {
	if node.Kind != nodestree.ProjectPipelineRuntime {
		return false, nil
	}

	projectPipelineRuntime, ok := node.Content.(*resourcev1alpha1.ProjectPipelineRuntime)
	if !ok {
		return true, fmt.Errorf("wrong type found for %s node", node.Name)
	}

	ok, err := p.isRepeatPipelinePath(projectPipelineRuntime)
	if ok {
		return true, err
	}

	projectName := projectPipelineRuntime.Spec.Project
	resourceDirectory := fmt.Sprintf("%s/%s", ProjectsDir, projectPipelineRuntime.Spec.Project)
	ok = nodestree.IsResourceExist(options, projectName, nodestree.Project)
	if !ok {
		err := fmt.Errorf(_ResourceDoesNotExistOrUnavailable,
			nodestree.Project,
			projectName,
			nodestree.ProjectPipelineRuntime,
			projectPipelineRuntime.Name,
			resourceDirectory,
		)
		return true, err
	}

	targetEnvironment := projectPipelineRuntime.Spec.Destination.Environment
	ok = nodestree.IsResourceExist(options, targetEnvironment, nodestree.Environment)
	if !ok {
		err := fmt.Errorf(_ResourceDoesNotExistOrUnavailable,
			nodestree.Environment,
			targetEnvironment,
			nodestree.ProjectPipelineRuntime,
			projectPipelineRuntime.Name, resourceDirectory,
		)
		return true, err
	}

	pipelineRepository := projectPipelineRuntime.Spec.PipelineSource
	ok = nodestree.IsResourceExist(options, pipelineRepository, nodestree.CodeRepo)
	if !ok {
		err := fmt.Errorf(_ResourceDoesNotExistOrUnavailable, nodestree.CodeRepo, pipelineRepository, nodestree.ProjectPipelineRuntime,
			projectPipelineRuntime.Name, resourceDirectory)
		return true, err
	}

	if len(projectPipelineRuntime.Spec.EventSources) > 0 {
		for _, event := range projectPipelineRuntime.Spec.EventSources {
			if event.Gitlab != nil {
				// TODO
				// In the future, cross product query codeRepo will be supported.
				if ok := nodestree.IsResourceExist(options, event.Gitlab.RepoName, nodestree.CodeRepo); !ok {
					err := fmt.Errorf(_ResourceDoesNotExistOrUnavailable, nodestree.CodeRepo, event.Gitlab.RepoName, nodestree.ProjectPipelineRuntime,
						projectPipelineRuntime.Name, resourceDirectory)
					return true, err
				}
			}
		}
	}

	ok, err = p.compare(options.Nodes)
	if ok && err != nil {
		return true, err
	}

	validateClient := validate.NewValidateClient(p.client, p.nodeOperator, &options.Nodes, p.config.Nautes.Namespace, options.ProductName)
	projectPipelineRuntime.Namespace = options.ProductName
	illegalEventSources, err := projectPipelineRuntime.Validate(context.TODO(), validateClient)
	if err != nil {
		return true, fmt.Errorf("verify project pipeline runtime failed, err: %w", err)
	}
	if len(illegalEventSources) != 0 {
		errMsg := ""
		for i, source := range illegalEventSources {
			if i > 0 {
				errMsg += fmt.Sprintf("; %s", source.Reason)
			} else {
				errMsg += source.Reason
			}
		}
		return true, fmt.Errorf("verify project pipeline runtime failed, err: %s", errMsg)
	}

	return true, nil
}

func (p *ProjectPipelineRuntimeUsecase) isRepeatPipelinePath(runtime *resourcev1alpha1.ProjectPipelineRuntime) (bool, error) {
	pipelines := runtime.Spec.Pipelines
	length := len(pipelines)

	for i := 0; i < length-1; i++ {
		for j := i + 1; j < length; j++ {
			if pipelines[i].Path == pipelines[j].Path {
				return true, fmt.Errorf("ProjectPipelineRuntime %s uses the same code repository for both codeSource and pipelineSource under %s directory, as found in the global validation", runtime.Name, runtime.Spec.Project)
			}
		}
	}

	return false, nil
}

func (p *ProjectPipelineRuntimeUsecase) compare(nodes nodestree.Node) (bool, error) {
	resourceNodes := nodestree.ListsResourceNodes(nodes, nodestree.ProjectPipelineRuntime)

	for i := 0; i < len(resourceNodes); i++ {
		runtime1, ok := resourceNodes[i].Content.(*resourcev1alpha1.ProjectPipelineRuntime)
		if !ok {
			continue
		}

		for j := i + 1; j < len(resourceNodes); j++ {
			runtime2, ok := resourceNodes[j].Content.(*resourcev1alpha1.ProjectPipelineRuntime)
			if !ok {
				continue
			}

			ok, err := runtime1.Compare(runtime2)
			if err != nil {
				return true, err
			}

			if ok {
				n1 := resourceNodes[i].Name
				n2 := resourceNodes[j].Name
				p1 := nodestree.GetResourceValue(resourceNodes[i].Content, "Spec", "Project")
				p2 := nodestree.GetResourceValue(resourceNodes[j].Content, "Spec", "Project")
				d1 := fmt.Sprintf("%s/%s", p1, n1)
				d2 := fmt.Sprintf("%s/%s", p2, n2)
				return true, fmt.Errorf("duplicate pipeline found in verify the validity of the global template, respectively %s and %s", d1, d2)
			}
		}
	}

	return false, nil
}

func (p *ProjectPipelineRuntimeUsecase) CreateResource(kind string) interface{} {
	if kind != nodestree.ProjectPipelineRuntime {
		return nil
	}

	return &resourcev1alpha1.ProjectPipelineRuntime{}
}

func SpliceCodeRepoResourceName(id int) string {
	return fmt.Sprintf("%s%d", RepoPrefix, id)
}

type PipelineRuntimeValidateClient struct {
	nodes nodestree.Node
}

func (p *PipelineRuntimeValidateClient) GetCodeRepoList(repoName string) (*resourcev1alpha1.CodeRepoList, error) {
	resourceNodes := nodestree.ListsResourceNodes(p.nodes, nodestree.CodeRepo)
	list := &resourcev1alpha1.CodeRepoList{}
	for _, node := range resourceNodes {
		val, ok := node.Content.(*resourcev1alpha1.CodeRepo)
		if !ok {
			return nil, fmt.Errorf("wrong type found for %s node", node.Name)
		}
		if val.Name == repoName {
			list.Items = append(list.Items, *val)
		}
	}

	return list, nil
}

func (p *PipelineRuntimeValidateClient) GetCodeRepoBindingList(productName, repoName string) (*resourcev1alpha1.CodeRepoBindingList, error) {
	resourceNodes := nodestree.ListsResourceNodes(p.nodes, nodestree.CodeRepoBinding)
	list := &resourcev1alpha1.CodeRepoBindingList{}
	for _, node := range resourceNodes {
		val, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
		if !ok {
			return nil, fmt.Errorf("wrong type found for %s node", node.Name)
		}
		if val.Spec.Product == productName && val.Spec.CodeRepo == repoName {
			list.Items = append(list.Items, *val)
		}
	}

	return list, nil
}
