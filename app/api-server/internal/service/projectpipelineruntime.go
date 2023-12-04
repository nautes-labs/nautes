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

	projectpipelineruntimev1 "github.com/nautes-labs/nautes/api/api-server/projectpipelineruntime/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
)

var (
	ProjectPipelineRuntimeFilterRules = map[string]map[string]selector.FieldSelector{
		"project_name": {
			selector.EqualOperator: selector.NewStringSelector("Name", selector.In),
		},
		"pipeline_triggers.pipeline": {
			selector.EqualOperator: selector.NewStringSelector("Spec.PipelineTriggers.Pipeline", selector.In),
		},
		"destination": {
			selector.EqualOperator: selector.NewStringSelector("Spec.Destination", selector.In),
		},
		"pipeline_source": {
			selector.EqualOperator: selector.NewStringSelector("Spec.PipelineSource", selector.In),
		},
		"project": {
			selector.EqualOperator: selector.NewStringSelector("Spec.Project", selector.In),
		},
	}
)

type ProjectPipelineRuntimeService struct {
	projectpipelineruntimev1.UnimplementedProjectPipelineRuntimeServer
	projectPipelineRuntime *biz.ProjectPipelineRuntimeUsecase
	codeRepo               biz.CodeRepo
}

func NewProjectPipelineRuntimeService(projectPipelineRuntime *biz.ProjectPipelineRuntimeUsecase, codeRepo biz.CodeRepo) *ProjectPipelineRuntimeService {
	return &ProjectPipelineRuntimeService{
		projectPipelineRuntime: projectPipelineRuntime,
		codeRepo:               codeRepo,
	}
}

func (s *ProjectPipelineRuntimeService) GetProjectPipelineRuntime(ctx context.Context, req *projectpipelineruntimev1.GetRequest) (*projectpipelineruntimev1.GetReply, error) {
	node, err := s.projectPipelineRuntime.GetProjectPipelineRuntime(ctx, req.ProjectPipelineRuntimeName, req.ProductName)
	if err != nil {
		return nil, err
	}

	runtime, ok := node.Content.(*resourcev1alpha1.ProjectPipelineRuntime)
	if !ok {
		return nil, fmt.Errorf("unexpected content type, resource: %s", node.Name)
	}

	err = s.convertCodeRepoNameToRepoName(ctx, runtime)
	if err != nil {
		return nil, err
	}

	return covertProjectPipelineRuntime(runtime)
}

func (s *ProjectPipelineRuntimeService) ListProjectPipelineRuntimes(ctx context.Context, req *projectpipelineruntimev1.ListsRequest) (*projectpipelineruntimev1.ListsReply, error) {
	var items []*projectpipelineruntimev1.GetReply

	nodes, err := s.projectPipelineRuntime.ListProjectPipelineRuntimes(ctx, req.ProductName)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		runtime, ok := node.Content.(*resourcev1alpha1.ProjectPipelineRuntime)
		if !ok {
			return nil, fmt.Errorf("unexpected content type, resource: %s", node.Name)
		}

		err = s.convertCodeRepoNameToRepoName(ctx, runtime)
		if err != nil {
			return nil, err
		}

		node.Content = runtime

		passed, err := selector.Match(req.FieldSelector, node.Content, ProjectPipelineRuntimeFilterRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		item, err := covertProjectPipelineRuntime(runtime)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return &projectpipelineruntimev1.ListsReply{
		Items: items,
	}, nil
}

func (s *ProjectPipelineRuntimeService) SaveProjectPipelineRuntime(ctx context.Context, req *projectpipelineruntimev1.SaveRequest) (*projectpipelineruntimev1.SaveReply, error) {
	err := s.validate(req)
	if err != nil {
		return nil, err
	}

	data := s.constructData(req)

	options := &biz.BizOptions{
		ResouceName:       req.ProjectPipelineRuntimeName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}

	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.SaveMethod,
		ResourceKind: nodestree.ProjectPipelineRuntime,
		ResourceName: req.ProjectPipelineRuntimeName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)
	err = s.projectPipelineRuntime.SaveProjectPipelineRuntime(ctx, options, data)
	if err != nil {
		return nil, err
	}

	return &projectpipelineruntimev1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %s configuration", req.ProjectPipelineRuntimeName),
	}, nil
}

func (s *ProjectPipelineRuntimeService) DeleteProjectPipelineRuntime(ctx context.Context, req *projectpipelineruntimev1.DeleteRequest) (*projectpipelineruntimev1.DeleteReply, error) {
	options := &biz.BizOptions{
		ResouceName:       req.ProjectPipelineRuntimeName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	rescourceInfo := &biz.RescourceInformation{
		Method:       biz.DeleteMethod,
		ResourceKind: nodestree.ProjectPipelineRuntime,
		ResourceName: req.ProjectPipelineRuntimeName,
		ProductName:  req.ProductName,
	}
	ctx = biz.SetResourceContext(ctx, rescourceInfo)
	err := s.projectPipelineRuntime.DeleteProjectPipelineRuntime(ctx, options)
	if err != nil {
		return nil, err
	}

	return &projectpipelineruntimev1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %s configuration", req.ProjectPipelineRuntimeName),
	}, nil
}

func (s *ProjectPipelineRuntimeService) constructData(req *projectpipelineruntimev1.SaveRequest) *biz.ProjectPipelineRuntimeData {
	eventSources := convertEventsToEventSource(req.Body.EventSources)
	pipelines := convertPipelines(req.Body.Pipelines)
	pipelineTriggers := convertTriggersToPipelineTriggers(req.Body.PipelineTriggers)
	additionalResources := convertAdditionalResources(req.Body.AdditionalResources).(*resourcev1alpha1.ProjectPipelineRuntimeAdditionalResources)

	var preHooks, postHooks []resourcev1alpha1.Hook
	if req.Body.Hooks != nil && req.Body.Hooks.PreHooks != nil {
		preHooks = convertPipelineHooks(req.GetBody().Hooks.PreHooks).([]resourcev1alpha1.Hook)
	}

	if req.Body.Hooks != nil && req.Body.Hooks.PostHooks != nil {
		postHooks = convertPipelineHooks(req.GetBody().Hooks.PostHooks).([]resourcev1alpha1.Hook)
	}

	return &biz.ProjectPipelineRuntimeData{
		Name: req.ProjectPipelineRuntimeName,
		Spec: resourcev1alpha1.ProjectPipelineRuntimeSpec{
			Project:          req.Body.Project,
			PipelineSource:   req.Body.PipelineSource,
			EventSources:     eventSources,
			Pipelines:        pipelines,
			PipelineTriggers: pipelineTriggers,
			Destination: resourcev1alpha1.ProjectPipelineDestination{
				Environment: req.Body.Destination.Environment,
				Namespace:   req.Body.Destination.Namespace,
			},
			Hooks: &resourcev1alpha1.Hooks{
				PreHooks:  preHooks,
				PostHooks: postHooks,
			},
			Isolation:           req.Body.Isolation,
			AdditionalResources: additionalResources,
			Account:             req.Body.Account,
		},
	}
}

// convertPipelineHooks Convert the type of hooks.
// Supports mutual conversion between API hooks and resource hooks.
func convertPipelineHooks(hooks interface{}) interface{} {
	switch val := hooks.(type) {
	// Change the hooks type of api to the hooks type of the resource.
	case []*projectpipelineruntimev1.Hook:
		var hooks []resourcev1alpha1.Hook
		for _, hook := range val {
			h := resourcev1alpha1.Hook{
				Name: hook.Name,
				Vars: hook.Vars,
			}
			if hook.Alias != "" {
				h.Alias = &hook.Alias
			}
			hooks = append(hooks, h)
		}

		return hooks

	// Change the hooks type of a resource to the hooks type of api.
	case []resourcev1alpha1.Hook:
		var hooks []*projectpipelineruntimev1.Hook
		for _, hook := range val {
			h := projectpipelineruntimev1.Hook{
				Name: hook.Name,
				Vars: hook.Vars,
			}
			if hook.Alias != nil {
				h.Alias = *hook.Alias
			}
			hooks = append(hooks, &h)
		}
		return hooks
	}

	return nil
}

func convertAdditionalResources(additionalResources interface{}) interface{} {
	switch val := additionalResources.(type) {
	case *projectpipelineruntimev1.ProjectPipelineRuntimeAdditionalResources:
		var additional *resourcev1alpha1.ProjectPipelineRuntimeAdditionalResources
		if additionalResources != nil {
			additional = &resourcev1alpha1.ProjectPipelineRuntimeAdditionalResources{}
			if val != nil && val.Git != nil {
				additional.Git = &resourcev1alpha1.ProjectPipelineRuntimeAdditionalResourcesGit{
					CodeRepo: val.Git.Coderepo,
					URL:      val.Git.Url,
					Path:     val.Git.Path,
					Revision: val.Git.Revision,
				}
			}
		}

		return additional
	case *resourcev1alpha1.ProjectPipelineRuntimeAdditionalResources:
		var additional *projectpipelineruntimev1.ProjectPipelineRuntimeAdditionalResources
		if additionalResources != nil {
			additional = &projectpipelineruntimev1.ProjectPipelineRuntimeAdditionalResources{}
			if val.Git != nil {
				additional.Git = &projectpipelineruntimev1.ProjectPipelineRuntimeAdditionalResourcesGit{
					Coderepo: val.Git.CodeRepo,
					Url:      val.Git.URL,
					Path:     val.Git.Path,
					Revision: val.Git.Revision,
				}
			}
		}
		return additional
	}

	return nil
}

func covertProjectPipelineRuntime(projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) (*projectpipelineruntimev1.GetReply, error) {
	pipelines := convertProjectRuntimeToPipelines(projectPipelineRuntime)
	eventSources := convertEventSourceToEvent(projectPipelineRuntime)
	pipelineTriggers := convertPipelineTriggersToTriggers(projectPipelineRuntime)
	additionalResources := convertAdditionalResources(projectPipelineRuntime.Spec.AdditionalResources)

	var preHooks, postHooks []*projectpipelineruntimev1.Hook
	if projectPipelineRuntime.Spec.Hooks != nil && projectPipelineRuntime.Spec.Hooks.PreHooks != nil {
		preHooks = convertPipelineHooks(projectPipelineRuntime.Spec.Hooks.PreHooks).([]*projectpipelineruntimev1.Hook)
	}

	if projectPipelineRuntime.Spec.Hooks != nil && projectPipelineRuntime.Spec.Hooks.PreHooks != nil {
		postHooks = convertPipelineHooks(projectPipelineRuntime.Spec.Hooks.PostHooks).([]*projectpipelineruntimev1.Hook)
	}

	return &projectpipelineruntimev1.GetReply{
		Name:             projectPipelineRuntime.Name,
		Project:          projectPipelineRuntime.Spec.Project,
		PipelineSource:   projectPipelineRuntime.Spec.PipelineSource,
		EventSources:     eventSources,
		Pipelines:        pipelines,
		PipelineTriggers: pipelineTriggers,
		Destination: &projectpipelineruntimev1.ProjectPipelineDestination{
			Environment: projectPipelineRuntime.Spec.Destination.Environment,
			Namespace:   projectPipelineRuntime.Spec.Destination.Namespace,
		},
		Isolation:           projectPipelineRuntime.Spec.Isolation,
		AdditionalResources: additionalResources.(*projectpipelineruntimev1.ProjectPipelineRuntimeAdditionalResources),
		Account:             projectPipelineRuntime.Spec.Account,
		Hooks: &projectpipelineruntimev1.Hooks{
			PreHooks:  preHooks,
			PostHooks: postHooks,
		},
	}, nil
}

func (s *ProjectPipelineRuntimeService) convertCodeRepoNameToRepoName(ctx context.Context, projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) error {
	if projectPipelineRuntime.Spec.PipelineSource == "" {
		return fmt.Errorf("the pipelineSource field value of projectPipelineRuntime %s should not be empty", projectPipelineRuntime.Name)
	}

	if projectPipelineRuntime.Spec.PipelineSource != "" {
		repoName, err := biz.ConvertCodeRepoToRepoName(ctx, s.codeRepo, projectPipelineRuntime.Spec.PipelineSource)
		if err != nil {
			return err
		}
		projectPipelineRuntime.Spec.PipelineSource = repoName
	}

	if projectPipelineRuntime.Spec.AdditionalResources != nil && projectPipelineRuntime.Spec.AdditionalResources.Git != nil && projectPipelineRuntime.Spec.AdditionalResources.Git.CodeRepo != "" {
		repoName, err := biz.ConvertCodeRepoToRepoName(ctx, s.codeRepo, projectPipelineRuntime.Spec.AdditionalResources.Git.CodeRepo)
		if err != nil {
			return err
		}
		projectPipelineRuntime.Spec.AdditionalResources.Git.CodeRepo = repoName
	}

	for _, event := range projectPipelineRuntime.Spec.EventSources {
		if event.Gitlab != nil {
			repoName, err := biz.ConvertCodeRepoToRepoName(ctx, s.codeRepo, event.Gitlab.RepoName)
			if err != nil {
				return err
			}
			event.Gitlab.RepoName = repoName
		}
	}

	return nil
}

func (s *ProjectPipelineRuntimeService) validate(req *projectpipelineruntimev1.SaveRequest) error {
	err := checkPipelineTriggers(req)
	if err != nil {
		return err
	}

	err = checkDestination(req)
	if err != nil {
		return err
	}

	return nil
}

func checkDestination(req *projectpipelineruntimev1.SaveRequest) error {
	if req.Body.Destination == nil {
		return fmt.Errorf("pipeline deployment target configuration not submitted, please checked '.Destination' field ")
	}

	if req.Body.Destination.Environment == "" {
		return fmt.Errorf("pipeline deployment environment cannot be empty, please checked '.Destination.Environment' field ")
	}

	return nil
}

func checkPipelineTriggers(req *projectpipelineruntimev1.SaveRequest) error {
	for _, trigger := range req.Body.PipelineTriggers {
		isExist := false

		for _, event := range req.Body.EventSources {
			if trigger.EventSource == event.Name {
				isExist = true
			}
		}

		if !isExist {
			return fmt.Errorf("the event source %s in the trigger does not exist", trigger.EventSource)
		}

		for _, pipeline := range req.Body.Pipelines {
			if trigger.Pipeline == pipeline.Name {
				isExist = true
			}
		}

		if !isExist {
			return fmt.Errorf("the pipeline %s in the trigger does not exist", trigger.Pipeline)
		}
	}

	return nil
}

func convertTriggersToPipelineTriggers(triggers []*projectpipelineruntimev1.PipelineTriggers) (pipelineTriggers []resourcev1alpha1.PipelineTrigger) {
	for _, trigger := range triggers {
		resourcePipelineTrigger := resourcev1alpha1.PipelineTrigger{
			EventSource: trigger.EventSource,
			Pipeline:    trigger.Pipeline,
			Revision:    trigger.Revision,
		}

		pipelineTriggers = append(pipelineTriggers, resourcePipelineTrigger)
	}

	return pipelineTriggers
}

func convertPipelineTriggersToTriggers(projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) []*projectpipelineruntimev1.PipelineTriggers {
	var pipelineTriggers []*projectpipelineruntimev1.PipelineTriggers
	for _, trigger := range projectPipelineRuntime.Spec.PipelineTriggers {
		pipelineTriggers = append(pipelineTriggers, &projectpipelineruntimev1.PipelineTriggers{
			EventSource: trigger.EventSource,
			Pipeline:    trigger.Pipeline,
			Revision:    trigger.Revision,
		})
	}
	return pipelineTriggers
}

func convertProjectRuntimeToPipelines(projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) []*projectpipelineruntimev1.Pipeline {
	var pipelines []*projectpipelineruntimev1.Pipeline
	for _, pipeline := range projectPipelineRuntime.Spec.Pipelines {
		pipelines = append(pipelines, &projectpipelineruntimev1.Pipeline{
			Name:  pipeline.Name,
			Label: pipeline.Label,
			Path:  pipeline.Path,
		})
	}
	return pipelines
}

func convertPipelines(pipelines []*projectpipelineruntimev1.Pipeline) (resourcePipelines []resourcev1alpha1.Pipeline) {
	for _, pipeline := range pipelines {
		resourcePipeline := resourcev1alpha1.Pipeline{
			Name:  pipeline.Name,
			Label: pipeline.Label,
			Path:  pipeline.Path,
		}

		resourcePipelines = append(resourcePipelines, resourcePipeline)
	}

	return resourcePipelines
}

func convertEventsToEventSource(events []*projectpipelineruntimev1.EventSource) (eventSources []resourcev1alpha1.EventSource) {
	for _, eventSource := range events {
		var gitlab *resourcev1alpha1.Gitlab
		var calendar *resourcev1alpha1.Calendar

		if eventSource.Gitlab != nil {
			gitlab = &resourcev1alpha1.Gitlab{
				RepoName: eventSource.Gitlab.RepoName,
				Revision: eventSource.Gitlab.Revision,
				Events:   eventSource.Gitlab.Events,
			}
		}

		if eventSource.Calendar != nil {
			calendar = &resourcev1alpha1.Calendar{
				Schedule:       eventSource.Calendar.Schedule,
				Interval:       eventSource.Calendar.Interval,
				ExclusionDates: eventSource.Calendar.ExclusionDates,
				Timezone:       eventSource.Calendar.Timezone,
			}
		}

		resourceEventSource := resourcev1alpha1.EventSource{
			Name:     eventSource.Name,
			Gitlab:   gitlab,
			Calendar: calendar,
		}

		eventSources = append(eventSources, resourceEventSource)
	}

	return eventSources
}

func convertEventSourceToEvent(projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) []*projectpipelineruntimev1.EventSource {
	var eventSources []*projectpipelineruntimev1.EventSource
	for _, eventSource := range projectPipelineRuntime.Spec.EventSources {
		event := &projectpipelineruntimev1.EventSource{}
		if eventSource.Name != "" {
			event.Name = eventSource.Name
		}

		if eventSource.Gitlab != nil {
			event.Gitlab = &projectpipelineruntimev1.Gitlab{
				RepoName: eventSource.Gitlab.RepoName,
				Revision: eventSource.Gitlab.Revision,
				Events:   eventSource.Gitlab.Events,
			}
		}

		if eventSource.Calendar != nil {
			event.Calendar = &projectpipelineruntimev1.Calendar{
				Schedule:       eventSource.Calendar.Schedule,
				Interval:       eventSource.Calendar.Interval,
				ExclusionDates: eventSource.Calendar.ExclusionDates,
				Timezone:       eventSource.Calendar.Timezone,
			}
		}

		eventSources = append(eventSources, event)
	}

	return eventSources
}
