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

	coderepobindingv1 "github.com/nautes-labs/nautes/api/api-server/coderepobinding/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
)

var (
	codeRepoBindingFilterFieldRules = map[string]map[string]selector.FieldSelector{
		"coderepo": {
			selector.EqualOperator: selector.NewStringSelector("Spec.CodeRepo", selector.In),
		},
		"product": {
			selector.EqualOperator: selector.NewStringSelector("Spec.Product", selector.In),
		},
		"projects.in": {
			selector.EqualOperator: selector.NewStringSelector("Spec.Projects.In", selector.In),
		},
	}
)

type CodeRepoBindingService struct {
	coderepobindingv1.UnimplementedCodeRepoBindingServer
	codeRepoBindingUsecase *biz.CodeRepoBindingUsecase
	codeRepo               biz.CodeRepo
}

func NewCodeRepoBindingService(codeRepoBindingUsecase *biz.CodeRepoBindingUsecase, codeRepo biz.CodeRepo) *CodeRepoBindingService {
	return &CodeRepoBindingService{codeRepoBindingUsecase: codeRepoBindingUsecase, codeRepo: codeRepo}
}

func (s *CodeRepoBindingService) CovertCodeRepoBindingValueToReply(codeRepoBinding *resourcev1alpha1.CodeRepoBinding) *coderepobindingv1.GetReply {
	return &coderepobindingv1.GetReply{
		Name:        codeRepoBinding.Name,
		Coderepo:    codeRepoBinding.Spec.CodeRepo,
		Product:     codeRepoBinding.Spec.Product,
		Projects:    codeRepoBinding.Spec.Projects,
		Permissions: codeRepoBinding.Spec.Permissions,
	}
}

func (s *CodeRepoBindingService) GetCodeRepoBinding(ctx context.Context, req *coderepobindingv1.GetRequest) (*coderepobindingv1.GetReply, error) {
	options := &biz.BizOptions{
		ProductName: req.ProductName,
		ResouceName: req.CoderepoBindingName,
	}

	codeRepoBinding, err := s.codeRepoBindingUsecase.GetCodeRepoBinding(ctx, options)
	if err != nil {
		return nil, err
	}

	err = s.ConvertProductAndRepoName(ctx, codeRepoBinding)
	if err != nil {
		return nil, err
	}

	return s.CovertCodeRepoBindingValueToReply(codeRepoBinding), nil
}

func (s *CodeRepoBindingService) ListCodeRepoBindings(ctx context.Context, req *coderepobindingv1.ListsRequest) (*coderepobindingv1.ListsReply, error) {
	options := &biz.BizOptions{
		ProductName: req.ProductName,
	}

	nodes, err := s.codeRepoBindingUsecase.ListCodeRepoBindings(ctx, options)
	if err != nil {
		return nil, err
	}

	var items []*coderepobindingv1.GetReply

	for _, node := range nodes {
		codeRepoBinding, ok := node.Content.(*resourcev1alpha1.CodeRepoBinding)
		if !ok {
			continue
		}

		err = s.ConvertProductAndRepoName(ctx, codeRepoBinding)
		if err != nil {
			return nil, err
		}
		node.Content = codeRepoBinding

		passed, err := selector.Match(req.FieldSelector, node.Content, codeRepoBindingFilterFieldRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		item := s.CovertCodeRepoBindingValueToReply(codeRepoBinding)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return &coderepobindingv1.ListsReply{Items: items}, nil
}

func (s *CodeRepoBindingService) SaveCodeRepoBinding(ctx context.Context, req *coderepobindingv1.SaveRequest) (*coderepobindingv1.SaveReply, error) {
	productResourceName, err := biz.ConvertGroupToProductName(ctx, s.codeRepo, req.ProductName)
	if err != nil {
		return nil, err
	}

	codeRepoResourceName, err := biz.ConvertRepoNameToCodeRepoName(ctx, s.codeRepo, req.ProductName, req.Body.Coderepo)
	if err != nil {
		return nil, err
	}

	rescourceInfo := &biz.RescourceInformation{
		Method:            biz.SaveMethod,
		ResourceKind:      nodestree.CodeRepoBinding,
		ResourceName:      req.CoderepoBindingName,
		ProductName:       req.ProductName,
		ParentResouceKind: nodestree.CodeRepo,
		ParentResouceName: codeRepoResourceName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ProductName:       req.ProductName,
		ResouceName:       req.CoderepoBindingName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}

	data := &biz.CodeRepoBindingData{
		Name: req.CoderepoBindingName,
		Spec: resourcev1alpha1.CodeRepoBindingSpec{
			CodeRepo:    codeRepoResourceName,
			Product:     productResourceName,
			Projects:    req.Body.Projects,
			Permissions: req.Body.Permissions,
		},
	}

	if err := s.codeRepoBindingUsecase.SaveCodeRepoBinding(ctx, options, data); err != nil {
		return nil, err
	}

	return &coderepobindingv1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %v configuration", req.CoderepoBindingName),
	}, nil
}

func (s *CodeRepoBindingService) DeleteCodeRepoBinding(ctx context.Context, req *coderepobindingv1.DeleteRequest) (*coderepobindingv1.DeleteReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.CodeRepoBinding,
		ResourceName: req.CoderepoBindingName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ProductName:       req.ProductName,
		ResouceName:       req.CoderepoBindingName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}

	if err := s.codeRepoBindingUsecase.DeleteCodeRepoBinding(ctx, options); err != nil {
		return nil, err
	}

	return &coderepobindingv1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %v configuration", req.CoderepoBindingName),
	}, nil
}

func (s *CodeRepoBindingService) ConvertProductAndRepoName(ctx context.Context, resource *resourcev1alpha1.CodeRepoBinding) error {
	repoName, err := biz.ConvertCodeRepoToRepoName(ctx, s.codeRepo, resource.Spec.CodeRepo)
	if err != nil {
		return err
	}
	resource.Spec.CodeRepo = repoName

	groupName, err := biz.ConvertProductToGroupName(ctx, s.codeRepo, resource.Spec.Product)
	if err != nil {
		return err
	}
	resource.Spec.Product = groupName

	return nil
}
