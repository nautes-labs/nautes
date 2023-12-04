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
	"encoding/json"
	"errors"
	"fmt"

	productv1 "github.com/nautes-labs/nautes/api/api-server/product/v1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

var (
	// Data rules for filtering list cluster api.
	ProductFilterRules = map[string]map[string]selector.FieldSelector{
		"product_name": {
			selector.EqualOperator: selector.NewStringSelector("Name", selector.In),
		},
	}
)

type ProductService struct {
	productv1.UnimplementedProductServer
	product *biz.ProductUsecase
	configs *nautesconfigs.Config
}

func NewProductService(product *biz.ProductUsecase, configs *nautesconfigs.Config) *ProductService {
	return &ProductService{
		product: product,
		configs: configs,
	}
}

func (s *ProductService) covertCodeRepoValueToReply(group *biz.Group) *productv1.GetProductReply {
	var git *productv1.GitGroup
	if s.configs.Git.GitType == nautesconfigs.GIT_TYPE_GITLAB {
		git = &productv1.GitGroup{
			Gitlab: &productv1.GitlabGroup{
				Visibility:  group.Visibility,
				Description: group.Description,
				Path:        group.Path,
			},
		}
	} else {
		git = &productv1.GitGroup{
			Github: &productv1.GithubGroup{
				Visibility:  group.Visibility,
				Description: group.Description,
				Path:        group.Path,
			},
		}
	}
	return &productv1.GetProductReply{
		Name: group.Name,
		Git:  git,
	}
}

func (s *ProductService) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductReply, error) {
	product, err := s.product.GetProduct(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	return s.covertCodeRepoValueToReply(product.Group), nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsReply, error) {
	products, err := s.product.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	var items []*productv1.GetProductReply
	for _, product := range products {
		passed, err := selector.Match(req.FieldSelector, product.Group, ProductFilterRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		item := s.covertCodeRepoValueToReply(product.Group)

		items = append(items, item)
	}

	return &productv1.ListProductsReply{
		Items: items,
	}, nil
}

func (s *ProductService) SaveProduct(ctx context.Context, req *productv1.SaveProductRequest) (*productv1.SaveProductReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.SaveMethod,
		ResourceKind: nodestree.Product,
		ResourceName: req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	git := &biz.GitGroupOptions{}
	if req.Git == nil {
		return nil, fmt.Errorf("the git request parameter cannot be empty, request: %v", req)
	}

	if s.configs.Git.GitType == nautesconfigs.GIT_TYPE_GITLAB {
		gitlab := &biz.GroupOptions{}

		bytes, err := json.Marshal(req.Git.Gitlab)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(bytes, gitlab)
		if err != nil {
			return nil, err
		}

		git.Gitlab = gitlab

		if git.Gitlab == nil {
			return nil, fmt.Errorf("failed to get gitlab parameter")
		}

		if git.Gitlab.Path != "" && git.Gitlab.Path != req.ProductName {
			return nil, fmt.Errorf(`when creating a GitLab group, the path must be consistent with the product name. the expected name is %s, but it is currently %s`, req.ProductName, git.Gitlab.Path)
		}

		if git.Gitlab.Name == "" {
			git.Gitlab.Name = req.ProductName
		}

		if git.Gitlab.Path == "" {
			git.Gitlab.Path = git.Gitlab.Name
		}
	} else {
		if req.Git.Github != nil {
			return nil, errors.New("coming soon to support github")
		}
	}

	_, _, err := s.product.SaveProduct(ctx, req.ProductName, git)
	if err != nil {
		return nil, err
	}

	return &productv1.SaveProductReply{
		Msg: "Successfully saved",
	}, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.Product,
		ResourceName: req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	err := s.product.DeleteProduct(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	return &productv1.DeleteProductReply{
		Msg: "Successfully deleted",
	}, nil
}
