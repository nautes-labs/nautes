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

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/cache"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	// The mapping relationship between gitlab webhook and the gitlab event header
	// GitLab API docs:
	// https://docs.gitlab.com/ce/api/projects.html#list-project-hooks
	// GitLab Event docs:
	// https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html
	gitlabWebhookEventToGitlabEventHeaderMapping = map[string]string{
		// "confidential_issues_events": "",
		// "confidential_note_events":   "",
		"deployment_events":     "Deployment Hook",
		"issues_events":         "Issue Hook",
		"job_events":            "Job Hook",
		"merge_requests_events": "Merge Request Hook",
		"note_events":           "Note Hook",
		"pipeline_events":       "Pipeline Hook",
		"push_events":           "Push Hook",
		"releases_events":       "Release Hook",
		"tag_push_events":       "Tag Push Hook",
	}
)

// NewFunctionHandler is a method that can generate a ResourceRequestHandler.
var NewFunctionHandler NewHandler

type NewHandler func(info *component.ComponentInitInfo) (ResourceRequestHandler, error)

// ResourceRequestHandler can directly deploy specific resources in the environment based on the requested data type.
type ResourceRequestHandler interface {
	CreateResource(ctx context.Context, space component.Space, authInfo component.AuthInfo, req component.RequestResource) error
	DeleteResource(ctx context.Context, space component.Space, req component.RequestResource) error
}

// PipelineRuntimeSyncHistory records the deployment information of pipeline runtime.
type PipelineRuntimeSyncHistory struct {
	// Cluster is the name of the cluster where the runtime is located.
	Cluster string `json:"cluster,omitempty"`
	// Space is the name of the space deployed by the runtime.
	Space *component.Space `json:"space,omitempty"`
	// CodeRepo is the name of the CodeRepo where the pipeline file is located.
	CodeRepo string `json:"codeRepo,omitempty"`
	// UniqueID is the index of the event source collection.
	UniqueID string `json:"uniqueID,omitempty"`
	// Account is the information of the machine account used by the runtime.
	Account *component.MachineAccount `json:"account,omitempty"`
	// App is the information of the resources deployed by the runtime.
	App *component.Application `json:"app,omitempty"`
	// Permissions records the access permissions of the Runtime machine account for how many key information items it possesses.
	Permissions []component.SecretInfo `json:"permissions,omitempty"`
	// HookResources records the resources that need to be deployed in order for the hook to run in the space.
	HookResources []component.RequestResource `json:"HookResources,omitempty"`
}

type PipelineRuntimeDeployer struct {
	// runtime is the runtime resource to be deployed.
	runtime *v1alpha1.ProjectPipelineRuntime
	// components is the list of components used to deploy the runtime.
	components *component.ComponentList
	// snapshot is a snapshot of runtime-related resources.
	snapshot database.Snapshot
	// codeRepo is the code repo where the pipeline file is located.
	codeRepo v1alpha1.CodeRepo
	// repoProvider is the provider of the code repo.
	repoProvider v1alpha1.CodeRepoProvider
	// productID is the name of the runtime's product in nautes.
	productID string
	// productName is the name of the product that the runtime belongs to in the product provider
	productName string
	// usageController is a public cache controller used for reading and writing public caches.
	usageController UsageController
	// cache is the last deployment cache.
	cache    PipelineRuntimeSyncHistory
	newCache PipelineRuntimeSyncHistory
	// secMgrAuthInfo is the authentication information of the runtime machine account in secretManagement.
	secMgrAuthInfo component.AuthInfo
	// reqResources stores the resources required to run hooks.
	reqResources []component.RequestResource
	reqHandler   ResourceRequestHandler
}

func newPipelineRuntimeDeployer(initInfo performerInitInfos) (taskPerformer, error) {
	deployRuntime := initInfo.runtime.(*v1alpha1.ProjectPipelineRuntime)

	productID := deployRuntime.GetProduct()
	product, err := initInfo.NautesResourceSnapshot.GetProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("get product %s failed: %w", productID, err)
	}

	codeRepoName := deployRuntime.Spec.PipelineSource
	codeRepo, err := initInfo.NautesResourceSnapshot.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, fmt.Errorf("get code repo %s failed: %w", codeRepoName, err)
	}

	provider, err := initInfo.NautesResourceSnapshot.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	history := &PipelineRuntimeSyncHistory{}
	newHistory := &PipelineRuntimeSyncHistory{}
	if initInfo.cache != nil {
		if err := json.Unmarshal(initInfo.cache.Raw, history); err != nil {
			return nil, fmt.Errorf("unmarshal history failed: %w", err)
		}
		_ = json.Unmarshal(initInfo.cache.Raw, newHistory)
	}

	usageController := UsageController{
		nautesNamespace: initInfo.NautesConfig.Nautes.Namespace,
		k8sClient:       initInfo.tenantK8sClient,
		clusterName:     initInfo.ClusterName,
	}

	handler, err := NewFunctionHandler(initInfo.ComponentInitInfo)
	if err != nil {
		return nil, fmt.Errorf("create deployer failed: %w", err)
	}

	return &PipelineRuntimeDeployer{
		runtime:         deployRuntime,
		components:      initInfo.Components,
		snapshot:        initInfo.NautesResourceSnapshot,
		codeRepo:        *codeRepo,
		repoProvider:    *provider,
		productID:       productID,
		productName:     product.Spec.Name,
		usageController: usageController,
		cache:           *history,
		newCache:        *newHistory,
		reqHandler:      handler,
	}, nil
}

func (prd *PipelineRuntimeDeployer) Deploy(ctx context.Context) (interface{}, error) {
	err := prd.deploy(ctx)
	return &prd.newCache, err
}

// deploy will deploy the pipeline runtime in the cluster.
// 1.Create a product in the component.
// 2.If the machine account name changes, update the information in the multi tenant and secret management of the account.
// 3.Create a space and machine account in the cluster.
// 4.Get account information.
// 5.Create event sources and consumers.
// 6.Create a machine account in secret management and update the account's permissions.
// 7.If additional resources need to be deployed, deploy apps.
func (prd *PipelineRuntimeDeployer) deploy(ctx context.Context) error {
	productUsage, err := prd.usageController.GetProductUsage(ctx, prd.productID)
	if err != nil {
		return fmt.Errorf("add product usage failed")
	}
	if productUsage == nil {
		tmp := cache.NewEmptyProductUsage()
		productUsage = &tmp
	}
	defer func() {
		err := prd.usageController.UpdateProductUsage(context.TODO(), prd.productID, *productUsage)
		if err != nil {
			logger.Error(err, "update product usage failed")
		}
	}()

	productUsage.Runtimes.Insert(prd.runtime.Name)
	if err := CreateProduct(ctx, prd.productID, prd.productName, *prd.components); err != nil {
		return fmt.Errorf("create product in components failed: %w", err)
	}

	accountName := prd.runtime.GetAccount()
	if prd.cache.Account != nil && accountName != prd.cache.Account.Name {
		oldAccountName := prd.cache.Account.Name
		if AccountIsRemovable(productUsage.Account, oldAccountName, prd.runtime.Name) {
			err := DeleteProductAccount(ctx, *prd.cache.Account, *prd.components)
			if err != nil {
				return fmt.Errorf("delete machine account failed: %w", err)
			}
		} else {
			if err := prd.revokeAccountPermissionInSecretManagement(ctx, &productUsage.Account); err != nil {
				return fmt.Errorf("clean up old account failed: %w", err)
			}
			req := component.PermissionRequest{
				Resource: component.ResourceMetaData{
					Product: prd.productID,
					Name:    prd.cache.Space.Name,
				},
				User:         oldAccountName,
				RequestScope: component.RequestScopeAccount,
			}

			if err := prd.components.MultiTenant.DeleteSpaceUser(ctx, req); err != nil {
				return fmt.Errorf("remove user %s from space %s failed :%w", oldAccountName, req.Resource.Name, err)
			}
		}
		productUsage.Account.DeleteRuntime(oldAccountName, prd.runtime.Name)
	}

	ns := prd.runtime.GetNamespaces()[0]
	if err := prd.components.MultiTenant.CreateSpace(ctx, prd.productID, ns); err != nil {
		return err
	}

	space, _ := prd.components.MultiTenant.GetSpace(ctx, prd.productID, ns)
	prd.newCache.Space = &space.Space

	if err := prd.components.MultiTenant.CreateAccount(ctx, prd.productID, accountName); err != nil {
		return fmt.Errorf("create account failed: %w", err)
	}

	if err := prd.syncNamespacePermission(ctx, &productUsage.Account); err != nil {
		return fmt.Errorf("sync namespace permission failed: %w", err)
	}

	account, err := prd.components.MultiTenant.GetAccount(ctx, prd.productID, accountName)
	if err != nil {
		return fmt.Errorf("get account info failed: %w", err)
	}

	authInfo, err := prd.components.SecretManagement.CreateAccount(ctx, *account)
	if err != nil {
		return fmt.Errorf("create account in secret store failed: %w", err)
	}
	prd.secMgrAuthInfo = *authInfo

	newPermissions := prd.GetPermissionsFromRuntime()
	err = syncAccountPermissionsInSecretManagement(ctx, *account, newPermissions, prd.cache.Permissions, *prd.components, prd.runtime.Name, &productUsage.Account)
	if err != nil {
		return fmt.Errorf("sync account in secret store failed: %w", err)
	}
	prd.newCache.Permissions = newPermissions
	prd.newCache.Account = account

	if err := prd.syncListenEvents(ctx, *account); err != nil {
		return fmt.Errorf("sync listen events failed: %w", err)
	}

	if err := prd.deployOrDeleteAdditionalResource(ctx); err != nil {
		return fmt.Errorf("deploy app failed: %w", err)
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) Delete(ctx context.Context) (interface{}, error) {
	err := prd.delete(ctx)
	return prd.newCache, err
}

// delete will clean up the resources deployed in the cluster runtime, which includes the following steps:
// 1.Get account information.
// 2.Delete the event source and consumer
// 3.Delete the additional deployed resources.
// 4.Update the information of the account in secret management.
// 5.Update product information.
func (prd *PipelineRuntimeDeployer) delete(ctx context.Context) error {
	productUsage, err := prd.usageController.GetProductUsage(ctx, prd.productID)
	if err != nil {
		return err
	}
	if productUsage == nil {
		return nil
	}
	defer func() {
		err := prd.usageController.UpdateProductUsage(context.TODO(), prd.productID, *productUsage)
		if err != nil {
			logger.Error(err, "update product usage failed")
		}
	}()

	productUsage.Runtimes.Delete(prd.runtime.Name)
	if err := prd.deleteListenEvents(ctx); err != nil {
		return err
	}

	if err := prd.deployOrDeleteAdditionalResource(ctx); err != nil {
		return err
	}

	if prd.cache.Account != nil {
		err := revokeAccountPermissionsInSecretManagement(ctx, *prd.cache.Account, prd.cache.Permissions, *prd.components, prd.runtime.Name, productUsage.Account)
		if err != nil {
			return fmt.Errorf("revoke permission in secret management failed: %w", err)
		}

		if AccountIsRemovable(productUsage.Account, prd.cache.Account.Name, prd.runtime.Name) {
			if err := DeleteProductAccount(ctx, *prd.cache.Account, *prd.components); err != nil {
				return fmt.Errorf("delete account failed: %w", err)
			}
		}
		productUsage.Account.DeleteRuntime(prd.cache.Account.Name, prd.runtime.Name)
		prd.newCache.Account = nil
	}

	if prd.cache.Space != nil {
		if err := prd.components.MultiTenant.DeleteSpace(ctx, prd.productID, prd.cache.Space.Name); err != nil {
			return fmt.Errorf("delete space %s failed: %w", prd.cache.Space.Name, err)
		}
		prd.newCache.Space = nil
	}

	if productUsage.Runtimes.Len() == 0 {
		if err := DeleteProduct(ctx, prd.productID, prd.productName, *prd.components); err != nil {
			return fmt.Errorf("delete product in components failed: %w", err)
		}
	}

	return nil
}

// deployOrDeleteAdditionalResource will deploy or delete resources based on the 'additionalResources' in the runtime.
func (prd *PipelineRuntimeDeployer) deployOrDeleteAdditionalResource(ctx context.Context) error {
	app, err := prd.buildApp()
	if err != nil {
		return fmt.Errorf("get additional app failed: %w", err)
	}
	if app != nil {
		prd.newCache.App = app
		return prd.components.Deployment.CreateApp(ctx, *app)
	}
	if app == nil && prd.cache.App != nil {
		return prd.components.Deployment.DeleteApp(ctx, *prd.cache.App)
	}
	return nil
}

func (prd *PipelineRuntimeDeployer) syncListenEvents(ctx context.Context, account component.MachineAccount) error {
	eventSourceSet, err := prd.convertRuntimeToEventSource(*prd.runtime)
	if err != nil {
		return fmt.Errorf("get event source failed: %w", err)
	}
	prd.newCache.UniqueID = eventSourceSet.UniqueID
	if err := prd.components.EventListener.CreateEventSource(ctx, eventSourceSet); err != nil {
		return fmt.Errorf("create event source failed: %w", err)
	}

	consumers, err := prd.convertRuntimeToConsumers(eventSourceSet.UniqueID, account, *prd.runtime, eventSourceSet)
	if err != nil {
		return fmt.Errorf("get consumers failed: %w", err)
	}

	space, err := prd.components.MultiTenant.GetSpace(ctx, prd.productID, prd.runtime.GetNamespaces()[0])
	if err != nil {
		return fmt.Errorf("get space failed: %w", err)
	}

	hookSpace := component.HookSpace{
		BaseSpace:       space.Space,
		DeployResources: removeDuplicateRequests(prd.reqResources),
	}

	if err := prd.createHookResource(ctx, prd.secMgrAuthInfo, hookSpace.BaseSpace, hookSpace.DeployResources); err != nil {
		return fmt.Errorf("create hook space failed: %w", err)
	}

	var oldHookSpace *component.HookSpace
	if prd.cache.HookResources != nil {
		oldHookSpace = &component.HookSpace{
			BaseSpace:       *prd.cache.Space,
			DeployResources: prd.cache.HookResources,
		}
	}
	if err := prd.removeUnusedHookResource(ctx, oldHookSpace, &hookSpace); err != nil {
		return fmt.Errorf("remove unused hook resources failed: %w", err)
	}
	prd.newCache.HookResources = hookSpace.DeployResources

	if err := prd.components.EventListener.CreateConsumer(ctx, *consumers); err != nil {
		return fmt.Errorf("create consumer failed: %w", err)
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) deleteListenEvents(ctx context.Context) error {
	if err := prd.components.EventListener.DeleteEventSource(ctx, prd.cache.UniqueID); err != nil {
		return fmt.Errorf("delete event source failed: %w", err)
	}

	if err := prd.components.EventListener.DeleteConsumer(ctx, prd.productID, prd.runtime.Name); err != nil {
		return fmt.Errorf("delete consumer failed: %w", err)
	}

	var oldHookSpace *component.HookSpace
	if prd.cache.Space != nil && prd.cache.HookResources != nil {
		oldHookSpace = &component.HookSpace{
			BaseSpace:       *prd.cache.Space,
			DeployResources: prd.cache.HookResources,
		}
	}
	if err := prd.removeUnusedHookResource(ctx, oldHookSpace, nil); err != nil {
		return fmt.Errorf("remove unused hook resources failed: %w", err)
	}
	prd.newCache.HookResources = nil

	return nil
}

func (prd *PipelineRuntimeDeployer) revokeAccountPermissionInSecretManagement(ctx context.Context, usage *cache.AccountUsage) error {
	account := prd.cache.Account
	accountPermissions := usage.Accounts[account.Name].ListPermissions(cache.ExcludedRuntimes([]string{prd.runtime.Name}))

	_, apIndex := utils.GetHashMap(accountPermissions)
	oldPermissionMap, oldPermissionIndex := utils.GetHashMap(prd.cache.Permissions)
	revokePermissionIndexes := oldPermissionIndex.Difference(apIndex)
	for _, index := range revokePermissionIndexes.UnsortedList() {
		if err := prd.components.SecretManagement.RevokePermission(ctx, oldPermissionMap[index], *account); err != nil {
			return fmt.Errorf("revoke permission to account %s failed: %w", account.Name, err)
		}
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) syncNamespacePermission(ctx context.Context, usage *cache.AccountUsage) error {
	accountName := prd.runtime.GetAccount()
	nameSpaces := prd.runtime.GetNamespaces()
	for i := range nameSpaces {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: prd.productID,
				Name:    nameSpaces[i],
			},
			User:         accountName,
			RequestScope: component.RequestScopeAccount,
		}
		if err := prd.components.MultiTenant.AddSpaceUser(ctx, req); err != nil {
			return fmt.Errorf("add user %s to space %s failed: %w", req.User, req.Resource.Name, err)
		}
	}

	usage.UpdateSpaces(accountName, prd.runtime.Name, utils.NewStringSet(nameSpaces...))
	return nil
}

func buildAdditionalAppName(runtimeName string) string {
	return fmt.Sprintf("%s-additional", runtimeName)
}

func (prd *PipelineRuntimeDeployer) buildApp() (*component.Application, error) {
	if prd.runtime.Spec.AdditionalResources == nil ||
		prd.runtime.Spec.AdditionalResources.Git == nil {
		return nil, nil
	}
	addResources := prd.runtime.Spec.AdditionalResources.Git

	ns := prd.runtime.GetNamespaces()[0]
	app := &component.Application{
		ResourceMetaData: component.ResourceMetaData{
			Product: prd.runtime.GetProduct(),
			Name:    buildAdditionalAppName(prd.runtime.Name),
		},
		Git: &component.ApplicationGit{
			URL:      "",
			Revision: addResources.Revision,
			Path:     addResources.Path,
			CodeRepo: "",
		},
		Destinations: []component.Space{{
			ResourceMetaData: component.ResourceMetaData{
				Product: prd.productID,
				Name:    ns,
			},
			SpaceType: component.SpaceTypeKubernetes,
			Kubernetes: &component.SpaceKubernetes{
				Namespace: ns,
			},
		}},
	}

	if addResources.URL != "" {
		app.Git.URL = addResources.URL
	} else {
		codeRepo, err := prd.snapshot.GetCodeRepo(addResources.CodeRepo)
		if err != nil {
			return nil, err
		}
		app.Git.CodeRepo = codeRepo.Name
		app.Git.URL = codeRepo.Spec.URL
	}

	return app, nil
}

// convertRuntimeToEventSource converts the runtime to an EventSourceSet.
// The same type of event sources will try to merge.(Like events from a code library)
func (prd *PipelineRuntimeDeployer) convertRuntimeToEventSource(runtime v1alpha1.ProjectPipelineRuntime) (component.EventSourceSet, error) {
	es := component.EventSourceSet{
		ResourceMetaData: component.ResourceMetaData{
			Product: runtime.GetProduct(),
			Name:    runtime.GetName(),
		},
		UniqueID:     buildUniqueID(runtime.GetProduct(), runtime.GetName()),
		EventSources: []component.EventSource{},
	}

	gitlabEvents, err := prd.getGitlabEventSourcesFromEventSource(runtime.Spec.EventSources)
	if err != nil {
		return es, err
	}
	es.EventSources = append(es.EventSources, gitlabEvents...)
	es.EventSources = append(es.EventSources, prd.getEventCalendarFromEventSource(runtime.Spec.EventSources)...)

	return es, nil
}

// getGitlabEventSourcesFromEventSource Gets gitlab-type data sources from the runtime.
// If the code repos are the same, merge the event sources.
func (prd *PipelineRuntimeDeployer) getGitlabEventSourcesFromEventSource(runtimeEventSources []v1alpha1.EventSource) ([]component.EventSource, error) {
	eventMap := map[string]component.EventSource{}
	for _, runtimeEventSource := range runtimeEventSources {
		if runtimeEventSource.Gitlab == nil {
			continue
		}

		envSrc := runtimeEventSource.Gitlab

		_, ok := eventMap[envSrc.RepoName]
		if ok {
			continue
		}

		apiServer, err := prd.getCodeRepoApiServerAddr(envSrc.RepoName)
		if err != nil {
			return nil, fmt.Errorf("get gitlab api server addr failed: %w", err)
		}

		codeRepo, err := prd.snapshot.GetCodeRepo(envSrc.RepoName)
		if err != nil {
			return nil, fmt.Errorf("get code repo failed: %w", err)
		}

		eventMap[envSrc.RepoName] = component.EventSource{
			Name: envSrc.RepoName,
			Gitlab: &component.EventSourceGitlab{
				APIServer: apiServer,
				Events:    codeRepo.Spec.Webhook.Events,
				CodeRepo:  envSrc.RepoName,
				RepoID:    getIDFromCodeRepo(envSrc.RepoName),
			},
		}
	}

	var events []component.EventSource
	for _, event := range eventMap {
		events = append(events, event)
	}

	return events, nil
}

func (prd *PipelineRuntimeDeployer) getEventCalendarFromEventSource(runtimeEventSources []v1alpha1.EventSource) []component.EventSource {
	var events []component.EventSource
	for _, runtimeEventSource := range runtimeEventSources {
		if runtimeEventSource.Calendar == nil {
			continue
		}

		calendarEvent := runtimeEventSource.Calendar

		events = append(events, component.EventSource{
			Name: buildCalendarEventName(runtimeEventSource.Name),
			Calendar: &component.EventSourceCalendar{
				Schedule:       calendarEvent.Schedule,
				Interval:       calendarEvent.Interval,
				ExclusionDates: calendarEvent.ExclusionDates,
				Timezone:       calendarEvent.Timezone,
			},
		})
	}
	return events
}

// convertRuntimeToConsumers converts the runtime to event consumers.
// A trigger + event type in the runtime is converted into a consumer.
func (prd *PipelineRuntimeDeployer) convertRuntimeToConsumers(
	uniqueID string,
	account component.MachineAccount,
	runtime v1alpha1.ProjectPipelineRuntime,
	eventSourceSet component.EventSourceSet,
) (*component.ConsumerSet, error) {
	consumers := &component.ConsumerSet{
		ResourceMetaData: component.ResourceMetaData{
			Product: prd.productID,
			Name:    prd.runtime.Name,
		},
		Account: account,
	}
	for i := range runtime.Spec.PipelineTriggers {
		trigger := runtime.Spec.PipelineTriggers[i]
		runtimePipeline, err := getPipelineFromPipelines(trigger.Pipeline, runtime.Spec.Pipelines)
		if err != nil {
			return nil, err
		}
		runtimeEventSource, err := getEventSourceFromEventSources(trigger.EventSource, runtime.Spec.EventSources)
		if err != nil {
			return nil, err
		}

		builtInVars, err := prd.buildBuiltinVars(trigger.Revision, runtime.Spec.PipelineSource, account, runtimePipeline)
		if err != nil {
			return nil, err
		}

		if runtimeEventSource.Gitlab != nil {
			if err := prd.addEventSourceToBuiltinVars(builtInVars, runtimeEventSource.Gitlab.RepoName); err != nil {
				return nil, err
			}

			filters, err := getGitlabFiltersFromEventSource(*runtimeEventSource.Gitlab)
			if err != nil {
				return nil, err
			}

			eventSourceName := runtimeEventSource.Gitlab.RepoName

			eventSource, _ := GetEventSourceFromEventSourceSet(eventSourceSet, eventSourceName)
			evTask, err := prd.getEventTaskFromEventSource(*eventSource, component.EventSourceTypeGitlab, builtInVars, trigger.Inputs, runtimeEventSource.Gitlab.Events)
			if err != nil {
				return nil, err
			}

			consumer := component.Consumer{
				UniqueID:        uniqueID,
				EventSourceName: eventSourceName,
				EventSourceType: component.EventSourceTypeGitlab,
				EventTypes:      runtimeEventSource.Gitlab.Events,
				Filters:         filters,
				Task:            evTask,
			}
			consumers.Consumers = append(consumers.Consumers, consumer)
		}

		if runtimeEventSource.Calendar != nil {
			eventSourceName := buildCalendarEventName(runtimeEventSource.Name)

			eventSource, _ := GetEventSourceFromEventSourceSet(eventSourceSet, eventSourceName)
			evTask, err := prd.getEventTaskFromEventSource(*eventSource, component.EventSourceTypeCalendar, builtInVars, trigger.Inputs, nil)
			if err != nil {
				return nil, err
			}

			consumer := component.Consumer{
				UniqueID:        uniqueID,
				EventSourceName: eventSourceName,
				EventSourceType: component.EventSourceTypeCalendar,
				Task:            evTask,
			}
			consumers.Consumers = append(consumers.Consumers, consumer)
		}
	}
	return consumers, nil
}

func GetEventSourceFromEventSourceSet(esSet component.EventSourceSet, name string) (*component.EventSource, error) {
	for i, eventSource := range esSet.EventSources {
		if eventSource.Name == name {
			return &esSet.EventSources[i], nil
		}
	}
	return nil, fmt.Errorf("event source %s not found", name)
}

func (prd *PipelineRuntimeDeployer) getEventTaskFromEventSource(
	eventSource component.EventSource,
	eventSourceType component.EventSourceType,
	builtInVars map[component.BuiltinVar]string,
	userVars []v1alpha1.UserPipelineInput,
	eventTypes []string) (component.EventTask, error) {
	task := component.EventTask{
		Type: component.EventTaskTypeRaw,
		Vars: nil,
		Raw:  nil,
	}

	userDefHooks := v1alpha1.Hooks{}
	if prd.runtime.Spec.Hooks != nil {
		userDefHooks = *prd.runtime.Spec.Hooks
	}

	info := component.HooksInitInfo{
		BuiltinVars:       builtInVars,
		UserRequestInputs: userVars,
		Hooks:             userDefHooks,
		EventSource:       eventSource,
		EventSourceType:   eventSourceType,
		EventTypes:        eventTypes,
	}

	hooks, requestResources, err := prd.components.Pipeline.GetHooks(info)
	if err != nil {
		return task, fmt.Errorf("get hooks failed: %w", err)
	}
	task.Vars = hooks.RequestVars
	task.Raw = hooks.Resource

	prd.reqResources = append(prd.reqResources, requestResources...)
	return task, nil
}

// func getEventsFromEventSource(eventSource component.EventSource, eventSourceType component.EventSourceType) []string {
// switch eventSourceType {
// case component.EventSourceTypeGitlab:
// return eventSource.Gitlab.Events
// }
// return nil
// }

const (
	GitlabWebhookIndexEventType = "headers.X-Gitlab-Event"
	GitlabWebhookIndexRef       = "event.body.ref"
)

func getGitlabFiltersFromEventSource(eventSource v1alpha1.Gitlab) ([]component.Filter, error) {
	var filters []component.Filter
	var events []string
	for i := range eventSource.Events {
		eventHead, ok := gitlabWebhookEventToGitlabEventHeaderMapping[eventSource.Events[i]]
		if !ok {
			return nil, fmt.Errorf("un support event type %s", eventSource.Events[i])
		}
		events = append(events, eventHead)

	}
	filters = append(filters, component.Filter{
		Key:        GitlabWebhookIndexEventType,
		Value:      events,
		Comparator: component.EqualTo,
	})

	if eventSource.Revision != "" {
		filters = append(filters, component.Filter{
			Key:        GitlabWebhookIndexRef,
			Value:      []string{eventSource.Revision},
			Comparator: component.Match,
		})
	}
	return filters, nil
}

func (prd *PipelineRuntimeDeployer) getCodeRepoApiServerAddr(name string) (string, error) {
	codeRepo, err := prd.snapshot.GetCodeRepo(name)
	if err != nil {
		return "", err
	}
	provider, err := prd.snapshot.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return "", err
	}
	return provider.Spec.ApiServer, nil
}

func buildUniqueID(product, name string) string {
	return fmt.Sprintf("%s-%s", product, name)
}

func getIDFromCodeRepo(name string) string {
	repoParts := strings.SplitN(name, "-", 2)
	if len(repoParts) != 2 {
		return ""
	}
	return repoParts[1]
}

func buildCalendarEventName(eventName string) string {
	return fmt.Sprintf("%s-%s", eventName, component.EventSourceTypeCalendar)
}

func getPipelineFromPipelines(name string, pipelines []v1alpha1.Pipeline) (v1alpha1.Pipeline, error) {
	for i := range pipelines {
		if pipelines[i].Name == name {
			return pipelines[i], nil
		}
	}
	return v1alpha1.Pipeline{}, fmt.Errorf("can not find pipeline %s in pipelines", name)
}

func getEventSourceFromEventSources(name string, eventSources []v1alpha1.EventSource) (v1alpha1.EventSource, error) {
	for i := range eventSources {
		if eventSources[i].Name == name {
			return eventSources[i], nil
		}
	}
	return v1alpha1.EventSource{}, fmt.Errorf("can not find event source %s in event sources", name)
}

// buildBuiltinVars returns the builtin vars corresponding to the consumer.
func (prd *PipelineRuntimeDeployer) buildBuiltinVars(pipelineRevision, codeRepoName string, account component.MachineAccount, pipeline v1alpha1.Pipeline) (map[component.BuiltinVar]string, error) {
	codeRepo, err := prd.snapshot.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, err
	}
	provider, err := prd.snapshot.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, err
	}

	builtinVars := map[component.BuiltinVar]string{
		component.VarServiceAccount:       account.Name,
		component.VarNamespace:            prd.runtime.GetNamespaces()[0],
		component.VarPipelineCodeRepoName: codeRepo.Name,
		component.VarPipelineCodeRepoURL:  codeRepo.Spec.URL,
		component.VarPipelineFilePath:     pipeline.Path,
		component.VarCodeRepoProviderType: provider.Spec.ProviderType,
		component.VarCodeRepoProviderURL:  provider.Spec.ApiServer,
		component.VarPipelineLabel:        pipeline.Label,
		component.VarPipelineDashBoardURL: prd.components.Pipeline.GetPipelineDashBoardURL(),
	}
	if pipelineRevision != "" {
		builtinVars[component.VarPipelineRevision] = pipelineRevision
	}
	return builtinVars, nil
}

func (prd *PipelineRuntimeDeployer) GetPermissionsFromRuntime() []component.SecretInfo {
	var permission []component.SecretInfo

	permission = append(permission, component.SecretInfo{
		Type: component.SecretTypeCodeRepo,
		CodeRepo: &component.CodeRepo{
			ProviderType: prd.repoProvider.Spec.ProviderType,
			ID:           prd.codeRepo.Name,
			User:         "default",
			Permission:   component.CodeRepoPermissionReadOnly,
		},
	})

	permission = append(permission, component.SecretInfo{
		Type: component.SecretTypeCodeRepo,
		CodeRepo: &component.CodeRepo{
			ProviderType: prd.repoProvider.Spec.ProviderType,
			ID:           prd.codeRepo.Name,
			User:         "default",
			Permission:   component.CodeRepoPermissionReadWrite,
		},
	})

	eventSourceSet := sets.New[string]()
	for i, eventSrc := range prd.runtime.Spec.EventSources {
		if eventSrc.Gitlab != nil {
			eventSourceSet.Insert(prd.runtime.Spec.EventSources[i].Gitlab.RepoName)
		}
	}

	for _, repoName := range eventSourceSet.UnsortedList() {
		permission = append(permission, component.SecretInfo{
			Type: component.SecretTypeCodeRepo,
			CodeRepo: &component.CodeRepo{
				ProviderType: prd.repoProvider.Spec.ProviderType,
				ID:           repoName,
				User:         "default",
				Permission:   component.CodeRepoPermissionAccessToken,
			},
		})
	}

	return permission
}

func (prd *PipelineRuntimeDeployer) addEventSourceToBuiltinVars(vars map[component.BuiltinVar]string, repoName string) error {
	codeRepo, err := prd.snapshot.GetCodeRepo(repoName)
	if err != nil {
		return fmt.Errorf("get code repo %s failed: %w", repoName, err)
	}

	vars[component.VarEventSourceCodeRepoName] = codeRepo.Name
	vars[component.VarEventSourceCodeRepoURL] = codeRepo.Spec.URL
	return nil
}
