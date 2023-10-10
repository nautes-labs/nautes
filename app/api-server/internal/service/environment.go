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

	environmentv1 "github.com/nautes-labs/nautes/api/api-server/environment/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
)

const (
	_Cluster = "Spec.Cluster"
	_EnvType = "Spec.EnvType"
)

var (
	environmentFilterFieldRules = map[string]map[string]selector.FieldSelector{
		FieldCluster: {
			selector.EqualOperator: selector.NewStringSelector(_Cluster, selector.In),
		},
		FieldEnvType: {
			selector.EqualOperator: selector.NewStringSelector(_EnvType, selector.In),
		},
	}
)

type EnvironmentService struct {
	environmentv1.UnimplementedEnvironmentServer
	environment *biz.EnvironmentUsecase
}

func NewEnvironmentService(environment *biz.EnvironmentUsecase) *EnvironmentService {
	return &EnvironmentService{environment: environment}
}

func (s *EnvironmentService) CovertCodeRepoValueToReply(env *resourcev1alpha1.Environment) *environmentv1.GetReply {
	return &environmentv1.GetReply{
		Product: env.Spec.Product,
		Name:    env.Name,
		Cluster: env.Spec.Cluster,
		EnvType: env.Spec.EnvType,
	}
}

func (s *EnvironmentService) GetEnvironment(ctx context.Context, req *environmentv1.GetRequest) (*environmentv1.GetReply, error) {
	env, err := s.environment.GetEnvironment(ctx, req.EnvironmentName, req.ProductName)
	if err != nil {
		return nil, err
	}

	return s.CovertCodeRepoValueToReply(env), nil
}

func (s *EnvironmentService) ListEnvironments(ctx context.Context, req *environmentv1.ListsRequest) (*environmentv1.ListsReply, error) {
	nodes, err := s.environment.ListEnvironments(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	var items []*environmentv1.GetReply
	for _, node := range nodes {
		env, ok := node.Content.(*resourcev1alpha1.Environment)
		if !ok {
			continue
		}

		err := s.environment.ConvertProductToGroupName(ctx, env)
		if err != nil {
			return nil, err
		}
		node.Content = env

		passed, err := selector.Match(req.FieldSelector, node.Content, environmentFilterFieldRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		items = append(items, s.CovertCodeRepoValueToReply(env))
	}

	return &environmentv1.ListsReply{Items: items}, nil
}

func (s *EnvironmentService) SaveEnvironment(ctx context.Context, req *environmentv1.SaveRequest) (*environmentv1.SaveReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.SaveMethod,
		ResourceKind: nodestree.Environment,
		ResourceName: req.EnvironmentName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ResouceName:       req.EnvironmentName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	err := s.environment.SaveEnvironment(ctx, options, &biz.EnviromentData{
		Name: req.EnvironmentName,
		Spec: resourcev1alpha1.EnvironmentSpec{
			Cluster: req.Body.Cluster,
			EnvType: req.Body.EnvType,
		},
	})
	if err != nil {
		return nil, err
	}

	return &environmentv1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %v configuration", req.EnvironmentName),
	}, nil
}

func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, req *environmentv1.DeleteRequest) (*environmentv1.DeleteReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.Environment,
		ResourceName: req.EnvironmentName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ResouceName:       req.EnvironmentName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	err := s.environment.DeleteEnvironment(ctx, options)
	if err != nil {
		return nil, err
	}

	return &environmentv1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %v configuration", req.EnvironmentName),
	}, nil
}
