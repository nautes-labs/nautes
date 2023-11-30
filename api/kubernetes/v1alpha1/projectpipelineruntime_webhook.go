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
	"encoding/json"
	"fmt"
	"reflect"

	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	nautesconst "github.com/nautes-labs/nautes/pkg/nautesconst"
	resource "github.com/nautes-labs/nautes/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var projectPipelineRuntimeLogger = logf.Log.WithName("projectPipelineRuntime")

func (r *ProjectPipelineRuntime) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-projectpipelineruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=create;update,versions=v1alpha1,name=vprojectpipelineruntime.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ProjectPipelineRuntime{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateCreate() error {
	projectPipelineRuntimeLogger.Info("validate create", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientFromK8s{Client: client}, client)
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
	projectPipelineRuntimeLogger.Info("validate update", "name", r.Name)
	client, err := getClient()
	if err != nil {
		return err
	}

	if reflect.DeepEqual(r.Spec, old.(*ProjectPipelineRuntime).Spec) {
		return nil
	}

	illegalEventSources, err := r.Validate(context.TODO(), &ValidateClientFromK8s{Client: client}, client)
	if err != nil {
		return err
	}

	if len(illegalEventSources) != 0 {
		failureReasons := []string{}
		for _, illegalEventSource := range illegalEventSources {
			failureReasons = append(failureReasons, illegalEventSource.Reason)
		}
		return fmt.Errorf("no permission code repo found in event source %v", failureReasons)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectPipelineRuntime) ValidateDelete() error {
	projectPipelineRuntimeLogger.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

const (
	SelectFieldCodeRepoBindingProductAndRepo = "productAndRepo"
	SelectFieldMetaDataName                  = "metadata.name"
)

// Validate use to verify pipeline runtime is legal, it will check the following things
// - runtime has permission to use repo in event sources
// if runtime has no permission to use code repo, it return them in the first var.
func (r *ProjectPipelineRuntime) Validate(ctx context.Context, validateClient ValidateClient, k8sClient client.Client) ([]IllegalEventSource, error) {
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

	if r.Spec.AdditionalResources != nil &&
		r.Spec.AdditionalResources.Git != nil &&
		r.Spec.AdditionalResources.Git.CodeRepo != "" {
		if err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, r.Spec.AdditionalResources.Git.CodeRepo); err != nil {
			return nil, fmt.Errorf("pipeline can not use code repo %s in additional resource: %s", r.Spec.AdditionalResources.Git.CodeRepo, err)
		}
	}

	namespaceUsage, err := GetUsedNamespaces(ctx, validateClient,
		GetUsedNamespaceWithOutRuntimes([]Runtime{r}),
		GetUsedNamespacesInCluster(cluster.Name))
	if err != nil {
		return nil, fmt.Errorf("get used namespaces in cluster %s failed: %w", cluster.Name, err)
	}

	if namespaces, ok := namespaceUsage[cluster.Name]; ok {
		nsSet := sets.New(namespaces...)
		if nsSet.Has(r.GetNamespaces()[0]) {
			return nil, fmt.Errorf("namespace %s is used by other runtime", r.GetNamespaces()[0])
		}
	}

	illegalEventSources := []IllegalEventSource{}
	for _, eventSource := range r.Spec.EventSources {
		if eventSource.Gitlab != nil &&
			eventSource.Gitlab.RepoName != "" {
			err := hasCodeRepoPermission(ctx, validateClient, productName, projectName, eventSource.Gitlab.RepoName)
			if err != nil {
				illegalEventSources = append(illegalEventSources, IllegalEventSource{
					EventSource: eventSource,
					Reason:      err.Error(),
				})
			}

		}
	}

	if cluster.Spec.ComponentsList.Pipeline == nil {
		return nil, fmt.Errorf("cluster component pipeline is empty")
	}

	metadataArr, err := GetPipelineHooksMetaData(ctx, k8sClient, cluster.Spec.ComponentsList.Pipeline.Name)
	if err != nil {
		return nil, fmt.Errorf("get hooks metadata failed: %w", err)
	}
	eventSourceTypes := getEventSourceTypesFromPipelineRuntime(*r)

	if err := validateHooks(r.Spec.Hooks, eventSourceTypes, *cluster, metadataArr); err != nil {
		return nil, fmt.Errorf("hook is illegal: %w", err)
	}

	return illegalEventSources, nil
}

// StaticCheck check resource is legal without connecting kubernetes
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

	if r.Spec.AdditionalResources != nil {
		res := r.Spec.AdditionalResources
		if res.Git != nil {
			if res.Git.CodeRepo == "" && res.Git.URL == "" {
				return fmt.Errorf("code repo and url can not be empty at the same time")
			}
		}
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

// Compare will check whether the pipeline already exists in the runtime.
func (r *ProjectPipelineRuntime) Compare(obj client.Object) (bool, error) {
	val, ok := obj.(*ProjectPipelineRuntime)
	if !ok {
		return false, fmt.Errorf("the resource %s type is inconsistent", obj.GetName())
	}

	for i := 0; i < len(r.Spec.Pipelines); i++ {
		for j := 0; j < len(val.Spec.Pipelines); j++ {
			if r.Spec.PipelineSource == val.Spec.PipelineSource &&
				r.Spec.Destination == val.Spec.Destination &&
				r.Spec.Pipelines[i].Path == val.Spec.Pipelines[j].Path {
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
		if runtime.Spec.Destination.Environment == envName {
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

const (
	EventSourceTypeGitlab   = "gitlab"
	EventSourceTypeCalendar = "calendar"
)

func getEventSourceTypesFromPipelineRuntime(runtime ProjectPipelineRuntime) []string {
	evSet := sets.New[string]()
	for _, es := range runtime.Spec.EventSources {
		if es.Gitlab != nil {
			evSet.Insert(EventSourceTypeGitlab)
		}
		if es.Calendar != nil {
			evSet.Insert(EventSourceTypeCalendar)
		}
	}
	return evSet.UnsortedList()
}

func validateHooks(hooks *Hooks, eventSourceType []string, cluster Cluster, rules []resource.HookMetadata) error {
	if hooks == nil {
		return nil
	}
	clusterEventListenerName := cluster.Spec.ComponentsList.EventListener.Name

	ruleMap := convertMetadataArrayToMap(rules)
	hookTypes := sets.New[string]()
	hookNames := sets.New[string]()
	for _, hook := range hooks.PreHooks {
		rule, ok := ruleMap[hook.Name]
		if !ok {
			return fmt.Errorf("unknown hook %s", hook.Name)
		}

		if !rule.IsPreHook {
			return fmt.Errorf("hook %s can not be a pre hook", hook.Name)
		}

		if err := validateHookGeneral(hook, rule, eventSourceType, clusterEventListenerName, hookTypes, hookNames); err != nil {
			return err
		}
		hookTypes.Insert(hook.Name)
		hookNames.Insert(getHookName(hook))

	}

	hookTypes = sets.New[string]()
	for _, hook := range hooks.PostHooks {
		rule, ok := ruleMap[hook.Name]
		if !ok {
			return fmt.Errorf("unknown hook %s", hook.Name)
		}

		if !rule.IsPostHook {
			return fmt.Errorf("hook %s can not be a post hook", hook.Name)
		}

		if err := validateHookGeneral(hook, rule, eventSourceType, clusterEventListenerName, hookTypes, hookNames); err != nil {
			return err
		}
		hookTypes.Insert(hook.Name)
		hookNames.Insert(getHookName(hook))
	}
	return nil
}

func validateHookGeneral(hook Hook, rule resource.HookMetadata, eventSourceTypes []string, eventListenerName string, existHookName, existAlias sets.Set[string]) error {
	if existHookName.Has(hook.Name) {
		return fmt.Errorf("a duplicate hook %s was found", hook.Name)
	}

	hookName := getHookName(hook)
	if existAlias.Has(hookName) {
		return fmt.Errorf("hook alias %s is used by other hook", hookName)
	}

	if !isHookSupportEventListener(rule.SupportEventListeners, eventListenerName) {
		return fmt.Errorf("hook %s doesn't support event listener %s",
			hook.Name,
			eventListenerName,
		)
	}

	if err := isHookSupportEventSourceTypes(rule.SupportEventSourceTypes, eventSourceTypes); err != nil {
		return fmt.Errorf("hook %s verify failed: %w", hook.Name, err)
	}

	return nil
}

func getHookName(hook Hook) string {
	hookName := hook.Name
	if hook.Alias != nil {
		hookName = *hook.Alias
	}

	return hookName
}

func isHookSupportEventListener(supportEventListeners []string, clusterEventListener string) bool {
	if len(supportEventListeners) == 0 {
		return true
	}
	supportEventListenerSet := sets.New(supportEventListeners...)
	ok := supportEventListenerSet.Has(clusterEventListener)

	return ok
}

func isHookSupportEventSourceTypes(supportEventSourceTypes, runtimeEventSourceTypes []string) error {
	if len(supportEventSourceTypes) == 0 {
		return nil
	}

	supportEventSourceTypeSet := sets.New(supportEventSourceTypes...)
	unsupportedEventSources := sets.New(runtimeEventSourceTypes...).Difference(supportEventSourceTypeSet)
	if len(unsupportedEventSources) != 0 {
		return fmt.Errorf("hook doesn't support event source %v",
			unsupportedEventSources.UnsortedList(),
		)
	}
	return nil
}

func convertMetadataArrayToMap(metadataArr []resource.HookMetadata) map[string]resource.HookMetadata {
	metadataMap := map[string]resource.HookMetadata{}
	for i := range metadataArr {
		metadataMap[metadataArr[i].Name] = metadataArr[i]
	}
	return metadataMap
}

func GetPipelineHooksMetaData(ctx context.Context, k8sClient client.Client, pipelineName string) ([]resource.HookMetadata, error) {
	nautesCfg, err := nautesconfigs.NewNautesConfigFromFile()
	if err != nil {
		return nil, fmt.Errorf("load nautes config failed: %w", err)
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nautesconst.ConfigMapNameHooksMetadata,
			Namespace: nautesCfg.Nautes.Namespace,
		},
	}

	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get hooks metadata failed: %w", err)
	}

	metadataJson, ok := cm.Data[pipelineName]
	if !ok {
		return nil, nil
	}

	metadataArr := &[]resource.HookMetadata{}
	err = json.Unmarshal([]byte(metadataJson), metadataArr)
	if err != nil {
		return nil, fmt.Errorf("unmarshal meta data failed: %w", err)
	}

	return *metadataArr, nil
}
