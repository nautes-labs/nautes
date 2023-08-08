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
	projectPipelineRuntimeFilterFieldRules = map[string]map[string]selector.FieldSelector{
		FieldPipelineTriggersPipeline: {
			selector.EqualOperator: selector.NewStringSelector(_PipelineTriggerPipeline, selector.In),
		},
		FieldDestination: {
			selector.EqualOperator: selector.NewStringSelector(_Destination, selector.In),
		},
		FieldPipelineSource: {
			selector.EqualOperator: selector.NewStringSelector(_PipelineSource, selector.In),
		},
		FieldProject: {
			selector.EqualOperator: selector.NewStringSelector(_Project, selector.In),
		},
	}
)

type ProjectPipelineRuntimeService struct {
	projectpipelineruntimev1.UnimplementedProjectPipelineRuntimeServer
	projectPipelineRuntime *biz.ProjectPipelineRuntimeUsecase
	resourcesUsecase       *biz.ResourcesUsecase
}

func NewProjectPipelineRuntimeService(projectPipelineRuntime *biz.ProjectPipelineRuntimeUsecase, resourcesUsecase *biz.ResourcesUsecase) *ProjectPipelineRuntimeService {
	return &ProjectPipelineRuntimeService{
		projectPipelineRuntime: projectPipelineRuntime,
		resourcesUsecase:       resourcesUsecase,
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

	err = s.ConvertCodeRepoToRepoName(ctx, runtime)
	if err != nil {
		return nil, err
	}

	return covertProjectPipelineRuntime(runtime, req.ProductName)
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

		err = s.ConvertCodeRepoToRepoName(ctx, runtime)
		if err != nil {
			return nil, err
		}

		node.Content = runtime

		passed, err := selector.Match(req.FieldSelector, node.Content, projectPipelineRuntimeFilterFieldRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		item, err := covertProjectPipelineRuntime(runtime, req.ProductName)
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
	eventSources := s.convertEventSources(req.Body.EventSources)
	pipelines := s.convertPipelines(req.Body.Pipelines)
	pipelineTriggers := s.convertPipelineTriggers(req.Body.PipelineTriggers)

	data := &biz.ProjectPipelineRuntimeData{
		Name: req.ProjectPipelineRuntimeName,
		Spec: resourcev1alpha1.ProjectPipelineRuntimeSpec{
			Project:          req.Body.Project,
			PipelineSource:   req.Body.PipelineSource,
			EventSources:     eventSources,
			Pipelines:        pipelines,
			PipelineTriggers: pipelineTriggers,
			Destination:      req.Body.Destination,
			Isolation:        req.Body.Isolation,
		},
	}

	err := s.Validate(data.Spec)
	if err != nil {
		return nil, err
	}

	options := &biz.BizOptions{
		ResouceName:       req.ProjectPipelineRuntimeName,
		ProductName:       req.ProductName,
		InsecureSkipCheck: req.InsecureSkipCheck,
	}
	ctx = biz.SetResourceContext(ctx, req.ProductName, biz.SaveMethod, nodestree.Project, req.Body.Project, nodestree.ProjectPipelineRuntime, req.ProjectPipelineRuntimeName)
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
	ctx = biz.SetResourceContext(ctx, req.ProductName, biz.DeleteMethod, "", "", nodestree.ProjectPipelineRuntime, req.ProjectPipelineRuntimeName)
	err := s.projectPipelineRuntime.DeleteProjectPipelineRuntime(ctx, options)
	if err != nil {
		return nil, err
	}

	return &projectpipelineruntimev1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %s configuration", req.ProjectPipelineRuntimeName),
	}, nil
}

func covertProjectPipelineRuntime(projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime, productName string) (*projectpipelineruntimev1.GetReply, error) {
	var pipelines []*projectpipelineruntimev1.Pipeline
	for _, pipeline := range projectPipelineRuntime.Spec.Pipelines {
		pipelines = append(pipelines, &projectpipelineruntimev1.Pipeline{
			Name:  pipeline.Name,
			Label: pipeline.Label,
			Path:  pipeline.Path,
		})
	}

	var eventSources []*projectpipelineruntimev1.EventSource
	for _, source := range projectPipelineRuntime.Spec.EventSources {
		event := &projectpipelineruntimev1.EventSource{}
		if source.Name != "" {
			event.Name = source.Name
		}

		if source.Gitlab != nil {
			event.Gitlab = &projectpipelineruntimev1.Gitlab{
				RepoName: source.Gitlab.RepoName,
				Revision: source.Gitlab.Revision,
				Events:   source.Gitlab.Events,
			}
		}

		if event.Calendar != nil {
			event.Calendar = &projectpipelineruntimev1.Calendar{
				Schedule:       source.Calendar.Schedule,
				Interval:       source.Calendar.Interval,
				ExclusionDates: source.Calendar.ExclusionDates,
				Timezone:       source.Calendar.Timezone,
			}
		}

		eventSources = append(eventSources, event)
	}

	var pipelineTriggers []*projectpipelineruntimev1.PipelineTriggers
	for _, trigger := range projectPipelineRuntime.Spec.PipelineTriggers {
		pipelineTriggers = append(pipelineTriggers, &projectpipelineruntimev1.PipelineTriggers{
			EventSource: trigger.EventSource,
			Pipeline:    trigger.Pipeline,
			Revision:    trigger.Revision,
		})
	}

	return &projectpipelineruntimev1.GetReply{
		Name:             projectPipelineRuntime.Name,
		Project:          projectPipelineRuntime.Spec.Project,
		PipelineSource:   projectPipelineRuntime.Spec.PipelineSource,
		EventSources:     eventSources,
		Pipelines:        pipelines,
		PipelineTriggers: pipelineTriggers,
		Destination:      projectPipelineRuntime.Spec.Destination,
		Isolation:        projectPipelineRuntime.Spec.Isolation,
	}, nil
}

func (p *ProjectPipelineRuntimeService) ConvertCodeRepoToRepoName(ctx context.Context, projectPipelineRuntime *resourcev1alpha1.ProjectPipelineRuntime) error {
	if projectPipelineRuntime.Spec.PipelineSource == "" {
		return fmt.Errorf("the pipelineSource field value of projectPipelineRuntime %s should not be empty", projectPipelineRuntime.Name)
	}

	if projectPipelineRuntime.Spec.PipelineSource != "" {
		repoName, err := p.resourcesUsecase.ConvertCodeRepoToRepoName(ctx, projectPipelineRuntime.Spec.PipelineSource)
		if err != nil {
			return err
		}
		projectPipelineRuntime.Spec.PipelineSource = repoName
	}

	for _, event := range projectPipelineRuntime.Spec.EventSources {
		if event.Gitlab != nil {
			repoName, err := p.resourcesUsecase.ConvertCodeRepoToRepoName(ctx, event.Gitlab.RepoName)
			if err != nil {
				return err
			}
			event.Gitlab.RepoName = repoName
		}
	}

	return nil
}

func (s *ProjectPipelineRuntimeService) Validate(spec resourcev1alpha1.ProjectPipelineRuntimeSpec) error {
	eventSourcesMap := make(map[string]resourcev1alpha1.EventSource)
	for _, event := range spec.EventSources {
		eventSourcesMap[event.Name] = event
	}

	pipelinesMap := make(map[string]resourcev1alpha1.Pipeline)
	for _, pipeline := range spec.Pipelines {
		pipelinesMap[pipeline.Name] = pipeline
	}

	for _, trigger := range spec.PipelineTriggers {
		eventName := trigger.EventSource
		pipelineName := trigger.Pipeline

		eventExists := false
		if _, ok := eventSourcesMap[eventName]; ok {
			eventExists = true
		}

		pipelineExists := false
		if _, ok := pipelinesMap[pipelineName]; ok {
			pipelineExists = true
		}

		if !eventExists {
			return fmt.Errorf("event source %s does not exist, please check if the parameters filled in the 'pipeline_triggers.event_source' field are correct", eventName)
		}

		if !pipelineExists {
			return fmt.Errorf("pipeline %s does not exist, please check if the parameters filled in the 'pipeline_triggers.pipeline' field are correct", pipelineName)
		}
	}

	return nil
}

func (s *ProjectPipelineRuntimeService) convertPipelineTriggers(triggers []*projectpipelineruntimev1.PipelineTriggers) (pipelineTriggers []resourcev1alpha1.PipelineTrigger) {
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

func (s *ProjectPipelineRuntimeService) convertPipelines(pipelines []*projectpipelineruntimev1.Pipeline) (resourcePipelines []resourcev1alpha1.Pipeline) {

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

func (s *ProjectPipelineRuntimeService) convertEventSources(events []*projectpipelineruntimev1.EventSource) (eventSources []resourcev1alpha1.EventSource) {
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
