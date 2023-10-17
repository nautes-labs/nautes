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

	coderepov1 "github.com/nautes-labs/nautes/api/api-server/coderepo/v1"
	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

var (
	codeRepoFilterFieldRules = map[string]map[string]selector.FieldSelector{
		FieldPilelineRuntime: {
			selector.EqualOperator: selector.NewBoolSelector(_PilelineRuntime, selector.Eq),
		},
		FieldDeploymentRuntime: {
			selector.EqualOperator: selector.NewBoolSelector(_DeploymentRuntime, selector.Eq),
		},
		FieldProject: {
			selector.EqualOperator: selector.NewStringSelector(_Project, selector.In),
		},
	}
)

type CodeRepoService struct {
	coderepov1.UnimplementedCodeRepoServer
	codeRepo *biz.CodeRepoUsecase
	configs  *nautesconfigs.Config
}

func NewCodeRepoService(codeRepo *biz.CodeRepoUsecase, configs *nautesconfigs.Config) *CodeRepoService {
	return &CodeRepoService{
		codeRepo: codeRepo,
		configs:  configs,
	}
}

func (s *CodeRepoService) GetCodeRepo(ctx context.Context, req *coderepov1.GetRequest) (*coderepov1.GetReply, error) {
	projectCodeRepo, err := s.codeRepo.GetCodeRepo(ctx, req.CoderepoName, req.ProductName)
	if err != nil {
		return nil, err
	}

	reply := s.covertCodeRepoValueToReply(ctx, projectCodeRepo.CodeRepo, projectCodeRepo.Project)

	return reply, nil
}

func (s *CodeRepoService) ListCodeRepos(ctx context.Context, req *coderepov1.ListsRequest) (*coderepov1.ListsReply, error) {
	var items []*coderepov1.GetReply

	projectCodeRepos, err := s.codeRepo.ListCodeRepos(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	for _, projectCodeRepo := range projectCodeRepos {
		passed, err := selector.Match(req.FieldSelector, projectCodeRepo.CodeRepo, codeRepoFilterFieldRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		item := s.covertCodeRepoValueToReply(ctx, projectCodeRepo.CodeRepo, projectCodeRepo.Project)
		items = append(items, item)
	}

	return &coderepov1.ListsReply{
		Items: items,
	}, nil
}

func (s *CodeRepoService) SaveCodeRepo(ctx context.Context, req *coderepov1.SaveRequest) (*coderepov1.SaveReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.SaveMethod,
		ResourceKind: nodestree.CodeRepo,
		ResourceName: req.CoderepoName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	gitOptions := &biz.GitCodeRepoOptions{
		Gitlab: &biz.GitlabCodeRepoOptions{},
	}

	// TODO
	// Coming soon to support github
	if s.configs.Git.GitType == nautesconfigs.GIT_TYPE_GITLAB {
		bytes, err := json.Marshal(req.Body.Git.Gitlab)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(bytes, gitOptions.Gitlab)
		if err != nil {
			return nil, err
		}

		if gitOptions.Gitlab.Name == "" {
			gitOptions.Gitlab.Name = req.CoderepoName
		}

		if gitOptions.Gitlab.Path == "" {
			gitOptions.Gitlab.Path = gitOptions.Gitlab.Name
		}

		if gitOptions.Gitlab.Path != req.CoderepoName {
			return nil, fmt.Errorf("the name of the codeRepo resource must be the same as the repository path")
		}
	} else {
		if gitOptions.Github != "" {
			return nil, errors.New("coming soon to support github")
		}
	}

	data := &biz.CodeRepoData{
		Spec: resourcev1alpha1.CodeRepoSpec{
			Project:           req.Body.Project,
			RepoName:          req.CoderepoName,
			DeploymentRuntime: req.Body.DeploymentRuntime,
			PipelineRuntime:   req.Body.PipelineRuntime,
		},
	}

	if req.Body.Webhook != nil && req.Body.Webhook.Events != nil {
		data.Spec.Webhook = &resourcev1alpha1.Webhook{
			Events: req.Body.Webhook.Events,
		}
	}

	options := &biz.BizOptions{
		ResouceName:       req.CoderepoName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}

	err := s.codeRepo.SaveCodeRepo(ctx, options, data, gitOptions)
	if err != nil {
		if commonv1.IsRefreshPermissionsAccessDenied(err) {
			return &coderepov1.SaveReply{
				Msg: fmt.Sprintf("[warning] the CodeRepo %s was deleted, but refresh permission failed, please check the current user permission, err: %s", req.CoderepoName, err.Error()),
			}, nil
		}

		return nil, err
	}

	return &coderepov1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %v configuration", req.CoderepoName),
	}, nil
}

func (s *CodeRepoService) DeleteCodeRepo(ctx context.Context, req *coderepov1.DeleteRequest) (*coderepov1.DeleteReply, error) {
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.Cluster,
		ResourceName: req.CoderepoName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)

	options := &biz.BizOptions{
		ResouceName:       req.CoderepoName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	err := s.codeRepo.DeleteCodeRepo(ctx, options)
	if err != nil {
		return nil, err
	}

	return &coderepov1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %v configuration", req.CoderepoName),
	}, nil
}

func (s *CodeRepoService) covertCodeRepoValueToReply(_ context.Context, codeRepo *resourcev1alpha1.CodeRepo, project *biz.Project) *coderepov1.GetReply {
	git := s.contructGit(project)

	webhook := s.setWebhook(codeRepo)

	return &coderepov1.GetReply{
		Product:           codeRepo.Spec.Product,
		Name:              codeRepo.Spec.RepoName,
		Project:           codeRepo.Spec.Project,
		Webhook:           webhook,
		PipelineRuntime:   codeRepo.Spec.PipelineRuntime,
		DeploymentRuntime: codeRepo.Spec.DeploymentRuntime,
		Git:               git,
	}
}

func (*CodeRepoService) setWebhook(codeRepo *resourcev1alpha1.CodeRepo) *coderepov1.Webhook {
	webhook := &coderepov1.Webhook{}
	if codeRepo.Spec.Webhook != nil {
		webhook.Events = codeRepo.Spec.Webhook.Events
	}
	return webhook
}

func (s *CodeRepoService) contructGit(project *biz.Project) *coderepov1.GitProject {
	var git *coderepov1.GitProject

	if s.configs.Git.GitType == nautesconfigs.GIT_TYPE_GITLAB {
		git = &coderepov1.GitProject{
			Gitlab: &coderepov1.GitlabProject{
				Name:          project.Name,
				Description:   project.Description,
				Path:          project.Path,
				Visibility:    project.Visibility,
				HttpUrlToRepo: project.HttpUrlToRepo,
				SshUrlToRepo:  project.SshUrlToRepo,
			},
		}
	} else {
		git = &coderepov1.GitProject{
			Github: &coderepov1.GithubProject{
				Name:          project.Name,
				Description:   project.Description,
				Path:          project.Path,
				Visibility:    project.Visibility,
				HttpUrlToRepo: project.HttpUrlToRepo,
				SshUrlToRepo:  project.SshUrlToRepo,
			},
		}
	}

	return git
}
