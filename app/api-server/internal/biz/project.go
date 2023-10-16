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
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectUsecase struct {
	log              *log.Helper
	codeRepo         CodeRepo
	secretRepo       Secretrepo
	nodestree        nodestree.NodesTree
	configs          *nautesconfigs.Config
	resourcesUsecase *ResourcesUsecase
}

type ProjectData struct {
	ProjectName string
	ProductName string
	Language    string
}

func NewProjectUsecase(logger log.Logger, codeRepo CodeRepo, secretRepo Secretrepo, nodeOperator nodestree.NodesTree, configs *nautesconfigs.Config, resourcesUsecase *ResourcesUsecase) *ProjectUsecase {
	project := &ProjectUsecase{log: log.NewHelper(log.With(logger)), codeRepo: codeRepo, secretRepo: secretRepo, nodestree: nodeOperator, configs: configs, resourcesUsecase: resourcesUsecase}
	nodeOperator.AppendOperators(project)
	return project
}

func (p *ProjectUsecase) GetProject(ctx context.Context, projectName, productName string) (*resourcev1alpha1.Project, error) {
	resourceNode, err := p.resourcesUsecase.Get(ctx, nodestree.Project, productName, p, func(nodes nodestree.Node) (string, error) {
		return projectName, nil
	})
	if err != nil {
		return nil, err
	}

	project, err := p.nodeToProject(resourceNode)
	if err != nil {
		return nil, err
	}

	project.Spec.Product = productName

	return project, nil
}

func (p *ProjectUsecase) nodeToProject(node *nodestree.Node) (*resourcev1alpha1.Project, error) {
	if project, ok := node.Content.(*resourcev1alpha1.Project); ok {
		return project, nil
	}

	return nil, fmt.Errorf("failed to get %s project", node.Name)
}

func (p *ProjectUsecase) ListProjects(ctx context.Context, productName string) ([]*resourcev1alpha1.Project, error) {
	nodes, err := p.resourcesUsecase.List(ctx, productName, p)
	if err != nil {
		return nil, err
	}

	projects, err := p.listProjects(*nodes, productName)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (p *ProjectUsecase) SaveProject(ctx context.Context, options *BizOptions, data *ProjectData) error {
	group, err := p.codeRepo.GetGroup(ctx, options.ProductName)
	if err != nil {
		return err
	}

	data.ProductName = fmt.Sprintf("%s%d", ProductPrefix, int(group.ID))
	resourceOptions := &resourceOptions{
		resourceKind:      nodestree.Project,
		resourceName:      options.ResouceName,
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

func (p *ProjectUsecase) DeleteProject(ctx context.Context, options *BizOptions) error {
	resourceOptions := &resourceOptions{
		resourceKind:      nodestree.Project,
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

func (p *ProjectUsecase) listProjects(nodes nodestree.Node, productName string) ([]*resourcev1alpha1.Project, error) {
	var projects []*resourcev1alpha1.Project

	projectNodes := nodestree.ListsResourceNodes(nodes, nodestree.Project)
	for _, node := range projectNodes {
		project, err := p.nodeToProject(node)
		if err != nil {
			return nil, err
		}

		project.Spec.Product = productName

		projects = append(projects, project)
	}

	return projects, nil
}

func (p *ProjectUsecase) UpdateNode(resourceNode *nodestree.Node, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*ProjectData)
	if !ok {
		return nil, fmt.Errorf("failed to get project data when update %s node", resourceNode.Name)
	}

	project, ok := resourceNode.Content.(*resourcev1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("failed to get project insatnce when update %s node", resourceNode.Name)
	}

	if project.Spec.Language == val.Language {
		return resourceNode, nil
	}

	project.Spec.Language = val.Language
	resourceNode.Content = project

	return resourceNode, nil
}

func (p *ProjectUsecase) CheckReference(options nodestree.CompareOptions, node *nodestree.Node, _ client.Client) (bool, error) {
	if node.Kind != nodestree.Project {
		return false, nil
	}

	err := nodestree.CheckResourceSubdirectory(&options.Nodes, node)
	if err != nil {
		return true, err
	}

	project, ok := node.Content.(*resourcev1alpha1.Project)
	if !ok {
		return true, fmt.Errorf("wrong type found for %s node when checking Project type", node.Name)
	}

	productName := project.Spec.Product
	if productName != options.ProductName {
		return true, fmt.Errorf("the product name of resource %s does not match the current product name, expected %s, but now is %s ", project.Name, options.ProductName, productName)
	}

	return true, nil
}

func (p *ProjectUsecase) CreateResource(kind string) interface{} {
	if kind != nodestree.Project {
		return nil
	}

	return &resourcev1alpha1.Project{}
}
