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

	errors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	enviromentv1 "github.com/nautes-labs/nautes/api/api-server/environment/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/validate"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EnvironmentUsecase struct {
	log              *log.Helper
	codeRepo         CodeRepo
	nodestree        nodestree.NodesTree
	config           *nautesconfigs.Config
	resourcesUsecase *ResourcesUsecase
}

type EnviromentData struct {
	Name string
	Spec resourcev1alpha1.EnvironmentSpec
}

func NewEnviromentUsecase(logger log.Logger, config *nautesconfigs.Config, codeRepo CodeRepo, nodeOperator nodestree.NodesTree, resourcesUsecase *ResourcesUsecase) *EnvironmentUsecase {
	env := &EnvironmentUsecase{log: log.NewHelper(log.With(logger)), config: config, codeRepo: codeRepo, nodestree: nodeOperator, resourcesUsecase: resourcesUsecase}
	nodeOperator.AppendOperators(env)
	return env
}

func (e *EnvironmentUsecase) ConvertProductToGroupName(ctx context.Context, env *resourcev1alpha1.Environment) error {
	if env.Spec.Product == "" {
		return fmt.Errorf("the product field value of environment %s should not be empty", env.Spec.Product)
	}

	groupName, err := ConvertProductToGroupName(ctx, e.codeRepo, env.Spec.Product)
	if err != nil {
		return err
	}

	env.Spec.Product = groupName

	return nil
}

func (e *EnvironmentUsecase) GetEnvironment(ctx context.Context, enviromentName, productName string) (*resourcev1alpha1.Environment, error) {
	node, err := e.resourcesUsecase.Get(ctx, nodestree.Environment, productName, e, func(nodes nodestree.Node) (string, error) {
		return enviromentName, nil
	})
	if err != nil {
		return nil, err
	}

	env, err := e.nodeToResource(node)
	if err != nil {
		return nil, err
	}

	err = e.ConvertProductToGroupName(ctx, env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (e *EnvironmentUsecase) ListEnvironments(ctx context.Context, productName string) ([]*nodestree.Node, error) {
	resourceNodes, err := e.resourcesUsecase.List(ctx, productName, e)
	if err != nil {
		return nil, err
	}

	nodes := nodestree.ListsResourceNodes(*resourceNodes, nodestree.Environment)

	return nodes, nil
}

func (e *EnvironmentUsecase) nodeToResource(node *nodestree.Node) (*resourcev1alpha1.Environment, error) {
	env, ok := node.Content.(*resourcev1alpha1.Environment)
	if !ok {
		return nil, errors.New(503, enviromentv1.ErrorReason_ASSERT_ERROR.String(), fmt.Sprintf("%s assert error", node.Name))
	}

	return env, nil
}

func (e *EnvironmentUsecase) SaveEnvironment(ctx context.Context, options *BizOptions, data *EnviromentData) error {
	group, err := e.codeRepo.GetGroup(ctx, options.ProductName)
	if err != nil {
		return err
	}

	data.Spec.Product = fmt.Sprintf("%s%d", ProductPrefix, int(group.ID))
	resourceOptions := &resourceOptions{
		resourceName:      options.ResouceName,
		resourceKind:      nodestree.Environment,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          e,
	}
	err = e.resourcesUsecase.Save(ctx, resourceOptions, data)
	if err != nil {
		return err
	}

	return nil
}

func (e *EnvironmentUsecase) UpdateNode(resourceNode *nodestree.Node, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*EnviromentData)
	if !ok {
		return nil, errors.New(503, enviromentv1.ErrorReason_ASSERT_ERROR.String(), fmt.Sprintf("failed to assert EnviromentData when update node, data: %v", val))
	}

	env, ok := resourceNode.Content.(*resourcev1alpha1.Environment)
	if !ok {
		return nil, errors.New(503, enviromentv1.ErrorReason_ASSERT_ERROR.String(), fmt.Sprintf("failed to assert resourcev1alpha1.Environment when update node, data: %v", val))
	}

	env.Name = val.Name
	if env.Spec == val.Spec {
		return resourceNode, nil
	}

	env.Spec = val.Spec
	resourceNode.Content = env

	return resourceNode, nil
}

func (e *EnvironmentUsecase) CheckReference(options nodestree.CompareOptions, node *nodestree.Node, k8sClient client.Client) (bool, error) {
	if node.Kind != nodestree.Environment {
		return false, nil
	}

	env, ok := node.Content.(*resourcev1alpha1.Environment)
	if !ok {
		return true, fmt.Errorf("wrong type found for %s node when checking Environment type", node.Name)
	}

	productName := env.Spec.Product
	if productName != options.ProductName {
		return true, fmt.Errorf("the product name of resource %s does not match the current product name, expected is %s, but now is %s", env.Name, options.ProductName, productName)
	}

	tenantAdminNamespace := e.config.Nautes.Namespace
	if tenantAdminNamespace == "" {
		return true, fmt.Errorf("tenant admin namspace cannot be empty")
	}

	objKey := client.ObjectKey{
		Namespace: tenantAdminNamespace,
		Name:      env.Spec.Cluster,
	}

	err := k8sClient.Get(context.TODO(), objKey, &resourcev1alpha1.Cluster{})
	if err != nil {
		return true, fmt.Errorf(_ResourceDoesNotExistOrUnavailable+"err: "+err.Error(), nodestree.Cluster, env.Spec.Cluster,
			nodestree.Environment, env.Name, EnvSubDir)
	}

	ok, err = e.compare(options.Nodes)
	if ok {
		return ok, err
	}

	return true, nil
}

func (e *EnvironmentUsecase) CreateResource(kind string) interface{} {
	if kind != nodestree.Environment {
		return nil
	}

	return &resourcev1alpha1.Environment{}
}

func (e *EnvironmentUsecase) DeleteEnvironment(ctx context.Context, options *BizOptions) error {
	resourceOptions := &resourceOptions{
		resourceKind:      nodestree.Environment,
		productName:       options.ProductName,
		insecureSkipCheck: options.InsecureSkipCheck,
		operator:          e,
	}

	err := e.resourcesUsecase.Delete(ctx, resourceOptions, func(nodes nodestree.Node) (string, error) {
		node := e.nodestree.GetNode(&nodes, nodestree.Environment, options.ResouceName)
		env, ok := node.Content.(*resourcev1alpha1.Environment)
		if !ok {
			return "", fmt.Errorf("the resource %s content type is incorrect", node.Name)
		}

		validateClient := validate.NewValidateClient(nil, e.nodestree, &nodes, e.config.Nautes.Namespace, options.ProductName)
		env.Namespace = options.ProductName
		err := env.IsDeletable(context.TODO(), validateClient)
		if err != nil {
			return "", err
		}

		return env.Name, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (e *EnvironmentUsecase) compare(nodes nodestree.Node) (bool, error) {
	resourceNodes := nodestree.ListsResourceNodes(nodes, nodestree.Environment)
	for i := 0; i < len(resourceNodes); i++ {
		for j := i + 1; j < len(resourceNodes); j++ {
			if runtime1, ok := resourceNodes[i].Content.(*resourcev1alpha1.Environment); ok {
				if runtime2, ok := resourceNodes[j].Content.(*resourcev1alpha1.Environment); ok {
					ok, err := runtime1.Compare(runtime2)
					if err != nil {
						return false, err
					}
					if ok {
						return ok, fmt.Errorf("duplicate reference cluster resources sbetween resource %s and resource %s", resourceNodes[i].Name, resourceNodes[j].Name)
					}
				}
			}
		}
	}

	return false, nil
}
