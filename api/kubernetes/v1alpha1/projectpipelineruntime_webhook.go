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

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var projectpipelineruntimelog = logf.Log.WithName("projectpipelineruntime-resource")

func (r *ProjectPipelineRuntime) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-projectpipelineruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=create;update,versions=v1alpha1,name=vprojectpipelineruntime.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ProjectPipelineRuntime{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateCreate() error {
	projectpipelineruntimelog.Info("validate create", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientFromK8s{Client: client})
	if err != nil {
		return err
	}

	if len(illegalEventSources) != 0 {
		failureReasons := []string{}
		for _, illegalEventSource := range illegalEventSources {
			failureReasons = append(failureReasons, illegalEventSource.Reason)
		}
		return fmt.Errorf("no permission code repo found %v", failureReasons)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateUpdate(old runtime.Object) error {
	projectpipelineruntimelog.Info("validate update", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	if reflect.DeepEqual(r.Spec, old.(*ProjectPipelineRuntime).Spec) {
		return nil
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientFromK8s{Client: client})
	if err != nil {
		return err
	}

	if len(illegalEventSources) != 0 {
		failureReasons := []string{}
		for _, illegalEventSource := range illegalEventSources {
			failureReasons = append(failureReasons, illegalEventSource.Reason)
		}
		return fmt.Errorf("no permission code repo found in eventsource %v", failureReasons)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateDelete() error {
	projectpipelineruntimelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

const (
	SelectFieldCodeRepoBindingProductAndRepo = "productAndRepo"
	SelectFieldMetaDataName                  = "metadata.name"
)

// Validate use to verify pipeline runtime is legal, it will check the following things
// - runtime has permission to use repo in eventsources
// if runtime has no permission to use code repo, it return them in the first var.
func (r *ProjectPipelineRuntime) Validate(ctx context.Context, validateClient ValidateClient) ([]IllegalEventSource, error) {
	productName := r.Namespace
	projectName := r.Spec.Project

	if err := r.StaticCheck(); err != nil {
		return nil, err
	}

	cluster, err := GetClusterByRuntime(ctx, validateClient, r)
	if err != nil {
		return nil, fmt.Errorf("get cluster by runtime failed: %w", err)
	}
	if cluster.Spec.Usage != CLUSTER_USAGE_WORKER || cluster.Spec.WorkerType != ClusterWorkTypePipeline {
		return nil, fmt.Errorf("cluster is not a pipeline cluster")
	}

	if err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, r.Spec.PipelineSource); err != nil {
		return nil, err
	}

	illegalEvnentSources := []IllegalEventSource{}
	for _, eventSource := range r.Spec.EventSources {

		if eventSource.Gitlab != nil &&
			eventSource.Gitlab.RepoName != "" {
			err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, eventSource.Gitlab.RepoName)
			if err != nil {
				illegalEvnentSources = append(illegalEvnentSources, IllegalEventSource{
					EventSource: eventSource,
					Reason:      err.Error(),
				})
			}

		}
	}

	return illegalEvnentSources, nil
}

func (r *ProjectPipelineRuntime) StaticCheck() error {
	eventSourceNames := make(map[string]bool, 0)
	for _, es := range r.Spec.EventSources {
		if eventSourceNames[es.Name] {
			return fmt.Errorf("event source %s is duplicate", es.Name)
		}
		eventSourceNames[es.Name] = true
	}

	pipelineNames := make(map[string]bool, 0)
	for _, pipeline := range r.Spec.Pipelines {
		if pipelineNames[pipeline.Name] {
			return fmt.Errorf("pipeline %s is duplicate", pipeline.Name)
		}
		pipelineNames[pipeline.Name] = true
	}

	triggerTags := make(map[string]bool, 0)
	for _, trigger := range r.Spec.PipelineTriggers {
		if !eventSourceNames[trigger.EventSource] {
			return fmt.Errorf("found non-existent event source %s in trigger", trigger.EventSource)
		}

		if !pipelineNames[trigger.Pipeline] {
			return fmt.Errorf("found non-existent pipeline %s in trigger", trigger.Pipeline)
		}

		tag := fmt.Sprintf("%s|%s|%s", trigger.EventSource, trigger.Pipeline, trigger.Revision)
		if triggerTags[tag] {
			return fmt.Errorf("trigger is duplicate, event source %s, pipeline %s, trigger %s", trigger.EventSource, trigger.Pipeline, trigger.Revision)
		}
		triggerTags[tag] = true
	}

	return nil
}

// Compare If true is returned, it means that the resource is duplicated
func (p *ProjectPipelineRuntime) Compare(obj client.Object) (bool, error) {
	val, ok := obj.(*ProjectPipelineRuntime)
	if !ok {
		return false, fmt.Errorf("the resource %s type is inconsistent", obj.GetName())
	}

	for i := 0; i < len(p.Spec.Pipelines); i++ {
		for j := 0; j < len(val.Spec.Pipelines); j++ {
			if p.Spec.PipelineSource == val.Spec.PipelineSource &&
				p.Spec.Destination == val.Spec.Destination &&
				p.Spec.Pipelines[i].Path == val.Spec.Pipelines[j].Path {
				return true, nil
			}
		}
	}

	return false, nil
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=get;list

func init() {
	GetClusterSubResourceFunctions = append(GetClusterSubResourceFunctions, getDependentResourcesOfClusterFromPipelineRuntime)
	GetEnvironmentSubResourceFunctions = append(GetEnvironmentSubResourceFunctions, getDependentResourcesOfEnvironmentFromPipelineRuntime)
	GetCoderepoSubResourceFunctions = append(GetCoderepoSubResourceFunctions, getDependentResourcesOfCodeRepoFromPipelineRuntime)
}

func getDependentResourcesOfClusterFromPipelineRuntime(ctx context.Context, k8sClient client.Client, clusterName string) ([]string, error) {
	runtimeList := &ProjectPipelineRuntimeList{}

	if err := k8sClient.List(ctx, runtimeList); err != nil {
		return nil, err
	}

	dependencies := []string{}
	for _, runtime := range runtimeList.Items {
		if runtime.Status.Cluster == clusterName {
			dependencies = append(dependencies, fmt.Sprintf("pipelineRuntime/%s/%s", runtime.Namespace, runtime.Name))
		}
	}
	return dependencies, nil
}

func getDependentResourcesOfEnvironmentFromPipelineRuntime(ctx context.Context, validateClient ValidateClient, productName, envName string) ([]string, error) {
	runtimes, err := validateClient.ListProjectPipelineRuntimes(ctx, productName)
	if err != nil {
		return nil, err
	}

	dependencies := []string{}
	for _, runtime := range runtimes {
		if runtime.Spec.Destination == envName {
			dependencies = append(dependencies, fmt.Sprintf("pipelineRuntime/%s", runtime.Name))
		}
	}

	return dependencies, nil
}

func getDependentResourcesOfCodeRepoFromPipelineRuntime(ctx context.Context, client ValidateClient, CodeRepoName string) ([]string, error) {
	codeRepo, err := client.GetCodeRepo(ctx, CodeRepoName)
	if err != nil {
		return nil, err
	}

	if codeRepo.Spec.Product == "" {
		return nil, fmt.Errorf("product of code repo %s is empty", getCodeRepoName(codeRepo))
	}

	runtimes, err := client.ListProjectPipelineRuntimes(ctx, codeRepo.Spec.Product)
	if err != nil {
		return nil, err
	}

	dependencies := []string{}
	for _, runtime := range runtimes {
		if runtime.Spec.PipelineSource == CodeRepoName {
			dependencies = append(dependencies, fmt.Sprintf("pipelineRuntime/%s", runtime.Name))
		}
	}

	return dependencies, nil
}
