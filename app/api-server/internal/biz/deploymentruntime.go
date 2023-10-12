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
	"reflect"

	"github.com/go-kratos/kratos/v2/log"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/validate"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentRuntimeUsecase struct {
	log              *log.Helper
	codeRepo         CodeRepo
	nodeOperator     nodestree.NodesTree
	resourcesUsecase *ResourcesUsecase
	client           client.Client
	config           *nautesconfigs.Config
}

type DeploymentRuntimeData struct {
	Name string
	Spec resourcev1alpha1.DeploymentRuntimeSpec
}

func NewDeploymentRuntimeUsecase(logger log.Logger, codeRepo CodeRepo, nodeOperator nodestree.NodesTree, resourcesUsecase *ResourcesUsecase, k8sClient client.Client, config *nautesconfigs.Config) *DeploymentRuntimeUsecase {
	runtime := &DeploymentRuntimeUsecase{
		log:              log.NewHelper(log.With(logger)),
		codeRepo:         codeRepo,
		nodeOperator:     nodeOperator,
		resourcesUsecase: resourcesUsecase,
		client:           k8sClient,
		config:           config,
	}
	nodeOperator.AppendOperators(runtime)
	return runtime
}

func (d *DeploymentRuntimeUsecase) GetDeploymentRuntime(ctx context.Context, deploymentRuntimeName, productName string) (*resourcev1alpha1.DeploymentRuntime, error) {
	resourceNode, err := d.resourcesUsecase.Get(ctx, nodestree.DeploymentRuntime, productName, d, func(nodes nodestree.Node) (string, error) {
		return deploymentRuntimeName, nil
	})
	if err != nil {
		return nil, err
	}

	runtime, ok := resourceNode.Content.(*resourcev1alpha1.DeploymentRuntime)
	if !ok {
		return nil, fmt.Errorf("the resource type of %s is inconsistent", deploymentRuntimeName)
	}

	return runtime, nil
}

func (d *DeploymentRuntimeUsecase) ListDeploymentRuntimes(ctx context.Context, productName string) ([]*nodestree.Node, error) {
	resourceNodes, err := d.resourcesUsecase.List(ctx, productName, d)
	if err != nil {
		return nil, err
	}

	nodes := nodestree.ListsResourceNodes(*resourceNodes, nodestree.DeploymentRuntime)

	return nodes, nil
}

func (d *DeploymentRuntimeUsecase) SaveDeploymentRuntime(ctx context.Context, options *BizOptions, data *DeploymentRuntimeData) error {
	group, err := d.codeRepo.GetGroup(ctx, options.ProductName)
	if err != nil {
		return err
	}

	pid := fmt.Sprintf("%s/%s", group.Path, data.Spec.ManifestSource.CodeRepo)
	project, err := d.codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return fmt.Errorf("failed to get repository %s, err: %s", data.Spec.ManifestSource.CodeRepo, err)
	}

	data.Spec.ManifestSource.CodeRepo = fmt.Sprintf("%s%d", RepoPrefix, int(project.ID))
	data.Spec.Product = fmt.Sprintf("%s%d", ProductPrefix, int(group.ID))
	resourceOptions := &resourceOptions{
		resourceName:      options.ResouceName,
		resourceKind:      nodestree.DeploymentRuntime,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          d,
	}
	err = d.resourcesUsecase.Save(ctx, resourceOptions, data)
	if err != nil {
		return err
	}

	return nil
}

func (d *DeploymentRuntimeUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*DeploymentRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to create node, the path is %s", path)
	}

	resource := &resourcev1alpha1.DeploymentRuntime{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.DeploymentRuntime,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, RuntimesDir)
	resourceFile := fmt.Sprintf("%s/%s.yaml", resourceDirectory, val.Name)

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourceFile,
		Kind:    nodestree.DeploymentRuntime,
		Content: resource,
		Level:   3,
	}, nil
}

func (d *DeploymentRuntimeUsecase) UpdateNode(resourceNode *nodestree.Node, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*DeploymentRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to update node %s", resourceNode.Name)
	}

	deployRuntime, ok := resourceNode.Content.(*resourcev1alpha1.DeploymentRuntime)
	if !ok {
		return nil, fmt.Errorf("failed to get resource instance when update node %s", val.Name)
	}

	ok = reflect.DeepEqual(deployRuntime.Spec, val.Spec)
	if ok {
		return resourceNode, nil
	}

	deployRuntime.Spec = val.Spec
	resourceNode.Content = deployRuntime

	return resourceNode, nil
}

func (d *DeploymentRuntimeUsecase) CheckReference(options nodestree.CompareOptions, node *nodestree.Node, _ client.Client) (bool, error) {
	if node.Kind != nodestree.DeploymentRuntime {
		return false, nil
	}

	deploymentRuntime, ok := node.Content.(*resourcev1alpha1.DeploymentRuntime)
	if !ok {
		return true, fmt.Errorf("wrong type found for %s node when checking DeploymentRuntime type", node.Name)
	}

	productName := deploymentRuntime.Spec.Product
	if productName != options.ProductName {
		return true, fmt.Errorf("the product name of resource %s does not match the current product name, expected %s, but now is %s ", deploymentRuntime.Name, options.ProductName, productName)
	}

	projectsRef := deploymentRuntime.Spec.ProjectsRef
	for _, projectName := range projectsRef {
		if projectName == "" {
			continue
		}

		ok := nodestree.IsResourceExist(options, projectName, nodestree.Project)
		if !ok {
			return true, fmt.Errorf("the referenced project %s by the deployment runtime %s does not exist while verifying the validity of the global template", projectName, deploymentRuntime.Name)
		}
	}

	envName := deploymentRuntime.Spec.Destination.Environment
	if envName != "" {
		ok = nodestree.IsResourceExist(options, envName, nodestree.Environment)
		if !ok {
			resourcePath := fmt.Sprintf("%s/%s", RuntimesDir, deploymentRuntime.Name)
			err := ResourceDoesNotExistOrUnavailable(envName, nodestree.Environment, resourcePath)
			return true, err
		}
	}

	codeRepoName := deploymentRuntime.Spec.ManifestSource.CodeRepo
	if codeRepoName != "" {
		ok = nodestree.IsResourceExist(options, codeRepoName, nodestree.CodeRepo)
		if !ok {
			err := fmt.Errorf("failed to get codeRepo %s", codeRepoName)
			resourcePath := fmt.Sprintf("%s/%s", RuntimesDir, deploymentRuntime.Name)
			return true, ResourceDoesNotExistOrUnavailable(codeRepoName, nodestree.CodeRepo, resourcePath, err.Error())
		}
	}

	ok, err := d.checkDuplicateResource(options.Nodes)
	if ok {
		return true, err
	}

	validateClient := validate.NewValidateClient(d.client, d.nodeOperator, &options.Nodes, d.config.Nautes.Namespace, options.ProductName)
	deploymentRuntime.Namespace = options.ProductName
	illegalProjectRefs, err := deploymentRuntime.Validate(context.TODO(), validateClient)
	if err != nil {
		return true, fmt.Errorf("verify deployment runtime failed, err: %w", err)
	}
	if len(illegalProjectRefs) != 0 {
		errMsg := ""
		for i, project := range illegalProjectRefs {
			if i > 0 {
				errMsg += fmt.Sprintf("; %s", project.Reason)
			} else {
				errMsg += project.Reason
			}
		}
		return true, fmt.Errorf("verify deployment runtime failed, err: %s", errMsg)
	}

	return true, nil
}

func (d *DeploymentRuntimeUsecase) checkDuplicateResource(nodes nodestree.Node) (bool, error) {
	resourceNodes := nodestree.ListsResourceNodes(nodes, nodestree.DeploymentRuntime)
	for i := 0; i < len(resourceNodes); i++ {
		for j := i + 1; j < len(resourceNodes); j++ {
			if runtime1, ok := resourceNodes[i].Content.(*resourcev1alpha1.DeploymentRuntime); ok {
				if runtime2, ok := resourceNodes[j].Content.(*resourcev1alpha1.DeploymentRuntime); ok {
					ok, err := runtime1.Compare(runtime2)
					if err != nil {
						return false, err
					}
					if ok {
						return ok, fmt.Errorf("duplicate reference found between resource %s and resource %s", resourceNodes[i].Name, resourceNodes[j].Name)
					}
				}
			}
		}
	}

	return false, nil
}

func (d *DeploymentRuntimeUsecase) CreateResource(kind string) interface{} {
	if kind != nodestree.DeploymentRuntime {
		return nil
	}

	return &resourcev1alpha1.DeploymentRuntime{}
}

func (d *DeploymentRuntimeUsecase) DeleteDeploymentRuntime(ctx context.Context, options *BizOptions) error {
	resourceOptions := &resourceOptions{
		resourceKind:      nodestree.DeploymentRuntime,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          d,
	}
	err := d.resourcesUsecase.Delete(ctx, resourceOptions, func(nodes nodestree.Node) (string, error) {
		return options.ResouceName, nil
	})

	if err != nil {
		return err
	}

	return nil
}
