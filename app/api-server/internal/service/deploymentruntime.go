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

package service

import (
	"context"
	"fmt"

	deploymentruntimev1 "github.com/nautes-labs/nautes/api/api-server/deploymentruntime/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
)

var (
	DeploymentRuntimeFilterRules = map[string]map[string]selector.FieldSelector{
		"projects_ref.in": {
			selector.EqualOperator: selector.NewStringSelector("Spec.ProjectsRef.In", selector.In),
		},
		"manifest_source.code_repo": {
			selector.EqualOperator: selector.NewStringSelector("Spec.ManifestSource.CodeRepo", selector.In),
		},
		"destination": {
			selector.EqualOperator: selector.NewStringSelector("Spec.Destination", selector.In),
		},
	}
)

type DeploymentruntimeService struct {
	deploymentruntimev1.UnimplementedDeploymentruntimeServer
	deploymentRuntime *biz.DeploymentRuntimeUsecase
	codeRepo          biz.CodeRepo
}

func NewDeploymentruntimeService(deploymentRuntime *biz.DeploymentRuntimeUsecase, codeRepo biz.CodeRepo) *DeploymentruntimeService {
	return &DeploymentruntimeService{deploymentRuntime: deploymentRuntime, codeRepo: codeRepo}
}

func (s *DeploymentruntimeService) CovertDeploymentRuntimeValueToReply(runtime *resourcev1alpha1.DeploymentRuntime) *deploymentruntimev1.GetReply {
	return &deploymentruntimev1.GetReply{
		Product: runtime.Spec.Product,
		Name:    runtime.Name,
		Destination: &deploymentruntimev1.DeploymentRuntimesDestination{
			Environment: runtime.Spec.Destination.Environment,
			Namespaces:  runtime.Spec.Destination.Namespaces,
		},
		ProjectsRef: runtime.Spec.ProjectsRef,
		ManifestSource: &deploymentruntimev1.ManifestSource{
			CodeRepo:       runtime.Spec.ManifestSource.CodeRepo,
			TargetRevision: runtime.Spec.ManifestSource.TargetRevision,
			Path:           runtime.Spec.ManifestSource.Path,
		},
		Account: runtime.Spec.Account,
	}
}

func (s *DeploymentruntimeService) GetDeploymentRuntime(ctx context.Context, req *deploymentruntimev1.GetRequest) (*deploymentruntimev1.GetReply, error) {
	runtime, err := s.deploymentRuntime.GetDeploymentRuntime(ctx, req.DeploymentruntimeName, req.ProductName)
	if err != nil {
		return nil, err
	}

	err = s.convertProductAndRepoName(ctx, runtime)
	if err != nil {
		return nil, err
	}

	return s.CovertDeploymentRuntimeValueToReply(runtime), nil
}

func (s *DeploymentruntimeService) ListDeploymentRuntimes(ctx context.Context, req *deploymentruntimev1.ListsRequest) (*deploymentruntimev1.ListsReply, error) {
	nodes, err := s.deploymentRuntime.ListDeploymentRuntimes(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	var items []*deploymentruntimev1.GetReply
	for _, node := range nodes {
		runtime, ok := node.Content.(*resourcev1alpha1.DeploymentRuntime)
		if !ok {
			continue
		}

		err := s.convertProductAndRepoName(ctx, runtime)
		if err != nil {
			return nil, err
		}
		node.Content = runtime

		passed, err := selector.Match(req.FieldSelector, node.Content, DeploymentRuntimeFilterRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		items = append(items, s.CovertDeploymentRuntimeValueToReply(runtime))
	}

	return &deploymentruntimev1.ListsReply{
		Items: items,
	}, nil
}

func (s *DeploymentruntimeService) SaveDeploymentRuntime(ctx context.Context, req *deploymentruntimev1.SaveRequest) (*deploymentruntimev1.SaveReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.SaveMethod,
		ResourceKind: nodestree.DeploymentRuntime,
		ResourceName: req.DeploymentruntimeName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	data := &biz.DeploymentRuntimeData{
		Name: req.DeploymentruntimeName,
		Spec: resourcev1alpha1.DeploymentRuntimeSpec{
			Product:     req.ProductName,
			ProjectsRef: req.Body.ProjectsRef,
			Destination: resourcev1alpha1.DeploymentRuntimesDestination{
				Environment: req.Body.Destination.Environment,
				Namespaces:  req.Body.Destination.Namespaces,
			},
			ManifestSource: resourcev1alpha1.ManifestSource{
				CodeRepo:       req.Body.ManifestSource.CodeRepo,
				TargetRevision: req.Body.ManifestSource.TargetRevision,
				Path:           req.Body.ManifestSource.Path,
			},
			Account: req.Body.Account,
		},
	}
	options := &biz.BizOptions{
		ResouceName:       req.DeploymentruntimeName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	err := s.deploymentRuntime.SaveDeploymentRuntime(ctx, options, data)
	if err != nil {
		return nil, err
	}

	return &deploymentruntimev1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %s configuration", req.DeploymentruntimeName),
	}, nil
}

func (s *DeploymentruntimeService) DeleteDeploymentRuntime(ctx context.Context, req *deploymentruntimev1.DeleteRequest) (*deploymentruntimev1.DeleteReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.DeploymentRuntime,
		ResourceName: req.DeploymentruntimeName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ResouceName:       req.DeploymentruntimeName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	err := s.deploymentRuntime.DeleteDeploymentRuntime(ctx, options)
	if err != nil {
		return nil, err
	}

	return &deploymentruntimev1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %s configuration", req.DeploymentruntimeName),
	}, nil
}

func (s *DeploymentruntimeService) convertProductAndRepoName(ctx context.Context, runtime *resourcev1alpha1.DeploymentRuntime) error {
	err := s.convertCodeRepoToRepoName(ctx, runtime)
	if err != nil {
		return err
	}

	err = s.convertProductToGroupName(ctx, runtime)
	if err != nil {
		return err
	}

	return nil
}

func (s *DeploymentruntimeService) convertCodeRepoToRepoName(ctx context.Context, runtime *resourcev1alpha1.DeploymentRuntime) error {
	if runtime.Spec.ManifestSource.CodeRepo == "" {
		return fmt.Errorf("the codeRepo field value of deploymentruntime %s should not be empty", runtime.Name)
	}

	repoName, err := biz.ConvertCodeRepoToRepoName(ctx, s.codeRepo, runtime.Spec.ManifestSource.CodeRepo)
	if err != nil {
		return err
	}
	runtime.Spec.ManifestSource.CodeRepo = repoName

	return nil
}

func (s *DeploymentruntimeService) convertProductToGroupName(ctx context.Context, runtime *resourcev1alpha1.DeploymentRuntime) error {
	if runtime.Spec.Product == "" {
		return fmt.Errorf("the product field value of deploymentruntime %s should not be empty", runtime.Name)
	}

	groupName, err := biz.ConvertProductToGroupName(ctx, s.codeRepo, runtime.Spec.Product)
	if err != nil {
		return err
	}

	runtime.Spec.Product = groupName

	return nil
}
