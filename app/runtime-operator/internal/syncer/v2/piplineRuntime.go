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

package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	nautesconst "github.com/nautes-labs/nautes/pkg/const"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
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

type PipelineRuntimeSyncHistory struct {
	Cluster  string       `json:"cluster,omitempty"`
	Space    string       `json:"space,omitempty"`
	CodeRepo string       `json:"codeRepo,omitempty"`
	UniqueID string       `json:"uniqueID,omitempty"`
	User     User         `json:"user,omitempty"`
	App      *Application `json:"app,omitempty"`
}

type PipelineRuntimeDeployer struct {
	runtime         *v1alpha1.ProjectPipelineRuntime
	deployer        Deployment
	productMgr      MultiTenant
	secMgr          SecretManagement
	evListener      EventListener
	db              database.Database
	codeRepo        v1alpha1.CodeRepo
	repoProvider    v1alpha1.CodeRepoProvider
	productID       string
	productName     string
	usageController UsageController
	rawCache        *pkgruntime.RawExtension
	cache           PipelineRuntimeSyncHistory
	newCache        PipelineRuntimeSyncHistory
}

func newPipelineRuntimeDeployer(initInfo runnerInitInfos) (taskRunner, error) {
	multiTenant := initInfo.Components.MultiTenant
	deployer := initInfo.Components.Deployment
	secMgr := initInfo.Components.SecretManagement
	deployRuntime := initInfo.runtime.(*v1alpha1.ProjectPipelineRuntime)

	productID := deployRuntime.GetProduct()
	product, err := initInfo.NautesDB.GetProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("get product %s failed: %w", productID, err)
	}

	codeRepoName := deployRuntime.Spec.PipelineSource
	codeRepo, err := initInfo.NautesDB.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, fmt.Errorf("get code repo %s failed: %w", codeRepoName, err)
	}

	provider, err := initInfo.NautesDB.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	history := &PipelineRuntimeSyncHistory{}
	if initInfo.cache != nil {
		if err := json.Unmarshal(initInfo.cache.Raw, history); err != nil {
			return nil, fmt.Errorf("unmarshal history failed: %w", err)
		}
	}

	usageController := UsageController{
		nautesNamespace: initInfo.NautesConfig.Nautes.Namespace,
		k8sClient:       initInfo.tenantK8sClient,
		clusterName:     initInfo.ClusterName,
		runtime:         initInfo.runtime,
		productID:       productID,
	}

	return &PipelineRuntimeDeployer{
		runtime:         deployRuntime,
		deployer:        deployer,
		productMgr:      multiTenant,
		secMgr:          secMgr,
		evListener:      initInfo.Components.EventListener,
		db:              initInfo.NautesDB,
		codeRepo:        *codeRepo,
		repoProvider:    *provider,
		productID:       productID,
		productName:     product.Spec.Name,
		usageController: usageController,

		rawCache: initInfo.cache,
		cache:    *history,
		newCache: *history,
	}, nil
}

func (prd *PipelineRuntimeDeployer) Deploy(ctx context.Context) (*pkgruntime.RawExtension, error) {
	err := prd.deploy(ctx)
	cache, convertErr := json.Marshal(prd.newCache)
	if convertErr != nil {
		return prd.rawCache, convertErr
	}

	return &pkgruntime.RawExtension{
		Raw: cache,
	}, err
}

func (prd *PipelineRuntimeDeployer) deploy(ctx context.Context) error {
	_, err := prd.usageController.AddProductUsage(ctx)
	if err != nil {
		return fmt.Errorf("add product usage failed")
	}

	if err := prd.initEnvironment(ctx); err != nil {
		return fmt.Errorf("init environment failed: %w", err)
	}

	user, err := prd.productMgr.GetUser(ctx, prd.productID, prd.runtime.Name)
	if err != nil {
		return fmt.Errorf("get user info failed: %w", err)
	}

	prd.newCache.User = *user

	if err := prd.syncListenEvents(ctx, *user); err != nil {
		return fmt.Errorf("sync listen events failed: %w", err)
	}

	if err := prd.syncUserInSecretStore(ctx, *user); err != nil {
		return fmt.Errorf("sync user in secret store failed: %w", err)
	}

	if err := prd.deployApp(ctx); err != nil {
		return fmt.Errorf("deploy app failed: %w", err)
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) deployApp(ctx context.Context) error {
	app, err := prd.buildApp()
	if err != nil {
		return fmt.Errorf("get additional app failed: %w", err)
	}
	if app == nil && prd.cache.App != nil {
		return prd.deployer.DeleteApp(ctx, *prd.cache.App)
	}

	if err := prd.deployer.CreateProduct(ctx, prd.productID); err != nil {
		return fmt.Errorf("create deployment product failed: %w", err)
	}

	if err := prd.deployer.AddProductUser(ctx, PermissionRequest{
		RequestScope: RequestScopeProduct,
		Resource: Resource{
			Product: prd.productID,
			Name:    prd.productID,
		},
		User:       prd.productName,
		Permission: Permission{},
	}); err != nil {
		return fmt.Errorf("grant deployment product %s admin permission to group %s failed: %w",
			prd.productID, prd.productName, err)
	}

	prd.newCache.App = app
	return prd.deployer.CreateApp(ctx, *app)
}

func (prd *PipelineRuntimeDeployer) syncListenEvents(ctx context.Context, user User) error {
	eventSource, err := prd.convertRuntimeToEventSource(*prd.runtime)
	if err != nil {
		return fmt.Errorf("get event source failed: %w", err)
	}
	prd.newCache.UniqueID = eventSource.UniqueID
	if err := prd.evListener.CreateEventSource(ctx, eventSource); err != nil {
		return fmt.Errorf("create event source failed: %w", err)
	}

	consumers, err := prd.convertRuntimeToConsumers(eventSource.UniqueID, user, *prd.runtime)
	if err != nil {
		return fmt.Errorf("get consumers failed: %w", err)
	}

	if err := prd.evListener.CreateConsumer(ctx, *consumers); err != nil {
		return fmt.Errorf("create consumer failed: %w", err)
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) deleteListenEvents(ctx context.Context) error {
	if err := prd.evListener.DeleteEventSource(ctx, prd.cache.UniqueID); err != nil {
		return fmt.Errorf("delete event source failed: %w", err)
	}

	if err := prd.evListener.DeleteConsumer(ctx, prd.productID, prd.runtime.Name); err != nil {
		return fmt.Errorf("delete consumer failed: %w", err)
	}

	return nil
}

func (prd *PipelineRuntimeDeployer) syncUserInSecretStore(ctx context.Context, user User) error {
	if err := prd.secMgr.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user in secret store failed: %w", err)
	}
	prd.newCache.CodeRepo = prd.codeRepo.Name

	repo := SecretInfo{
		Type: SecretTypeCodeRepo,
		CodeRepo: &CodeRepo{
			ProviderType: prd.repoProvider.Spec.ProviderType,
			ID:           prd.codeRepo.Name,
			User:         "default",
			Permission:   CodeRepoPermissionReadWrite,
		},
	}
	if err := prd.secMgr.GrantPermission(ctx, repo, user); err != nil {
		return fmt.Errorf("grant code repo readwrite permission to user failed: %w", err)
	}

	repo.CodeRepo.Permission = CodeRepoPermissionReadOnly
	if err := prd.secMgr.GrantPermission(ctx, repo, user); err != nil {
		return fmt.Errorf("grant code repo readonly permission to user failed: %w", err)
	}
	return nil
}

func (prd *PipelineRuntimeDeployer) initEnvironment(ctx context.Context) error {
	if err := prd.productMgr.CreateProduct(ctx, prd.productID); err != nil {
		return err
	}

	ns := prd.runtime.GetNamespaces()[0]
	if err := prd.productMgr.CreateSpace(ctx, prd.productID, ns); err != nil {
		return err
	}
	prd.newCache.Space = ns

	userName := prd.runtime.Name
	if err := prd.productMgr.CreateUser(ctx, prd.productID, userName); err != nil {
		return err
	}

	err := prd.productMgr.AddSpaceUser(ctx, PermissionRequest{
		RequestScope: RequestScopeUser,
		Resource: Resource{
			Product: prd.productID,
			Name:    ns,
		},
		User:       userName,
		Permission: Permission{},
	})
	return err
}

func (prd *PipelineRuntimeDeployer) Delete(ctx context.Context) (*pkgruntime.RawExtension, error) {
	err := prd.delete(ctx)
	cache, convertErr := json.Marshal(prd.newCache)
	if convertErr != nil {
		return prd.rawCache, convertErr
	}

	return &pkgruntime.RawExtension{
		Raw: cache,
	}, err
}

func (prd *PipelineRuntimeDeployer) delete(ctx context.Context) error {
	usage, err := prd.usageController.GetProductUsage(ctx)
	if err != nil {
		return err
	}

	usage.Runtimes.Delete(prd.runtime.Name)

	user, err := prd.productMgr.GetUser(ctx, prd.productID, prd.runtime.GetName())
	if err != nil {
		if !IsUserNotFound(err) {
			return fmt.Errorf("get user %s's info failed: %w", prd.runtime.GetName(), err)
		}
	}

	if err := prd.deleteListenEvents(ctx); err != nil {
		return err
	}

	if err := prd.deleteDeploymentApps(ctx, *usage); err != nil {
		return err
	}

	if user != nil {
		if err := prd.deleteUserInSecretDatabase(ctx, *user); err != nil {
			return err
		}
	}

	if err := prd.cleanUpProduct(ctx, user, *usage); err != nil {
		return err
	}
	return prd.usageController.DeleteProductUsage(ctx)
}

func (prd *PipelineRuntimeDeployer) deleteUserInSecretDatabase(ctx context.Context, user User) error {
	if prd.cache.CodeRepo != "" {
		if err := prd.secMgr.RevokePermission(ctx, buildSecretInfoCodeRepo(prd.repoProvider.Spec.ProviderType, prd.cache.CodeRepo), user); err != nil {
			return fmt.Errorf("revoke code repo %s readonly permission from user %s failed: %w", prd.cache.CodeRepo, user.Name, err)
		}
		prd.newCache.CodeRepo = ""
	}

	if err := prd.secMgr.DeleteUser(ctx, user); err != nil {
		return fmt.Errorf("delete user %s in secret database failed: %w", user.Name, err)
	}
	return nil
}

func (prd *PipelineRuntimeDeployer) deleteDeploymentApps(ctx context.Context, usage ProductUsage) error {
	if prd.cache.App != nil {
		if err := prd.deployer.DeleteApp(ctx, *prd.cache.App); err != nil {
			return err
		}
	}

	if usage.Runtimes.Len() == 0 {
		err := prd.deployer.DeleteProductUser(ctx, PermissionRequest{
			RequestScope: RequestScopeProduct,
			Resource: Resource{
				Product: prd.productID,
				Name:    prd.productID,
			},
			User:       prd.productName,
			Permission: Permission{},
		})
		if err != nil {
			return fmt.Errorf("revoke permission from deployment product failed: %w", err)
		}

		err = prd.deployer.DeleteProduct(ctx, prd.productID)
		if err != nil {
			return fmt.Errorf("delete deployment product failed: %w", err)
		}
	}
	return nil
}

func (prd *PipelineRuntimeDeployer) cleanUpProduct(ctx context.Context, user *User, usage ProductUsage) error {
	if user != nil {
		if err := prd.productMgr.DeleteUser(ctx, prd.productID, user.Name); err != nil {
			return fmt.Errorf("delete user %s in product %s failed: %w", user.Name, prd.productID, err)
		}
	}

	if prd.cache.Space != "" {
		ns := prd.cache.Space
		err := prd.productMgr.DeleteSpace(ctx, prd.productID, ns)
		if err != nil {
			return fmt.Errorf("delete space %s failed: %w", ns, err)
		}
		prd.newCache.Space = ""
	}

	if usage.Runtimes.Len() == 0 {
		if err := prd.productMgr.DeleteProduct(ctx, prd.productID); err != nil {
			return fmt.Errorf("delete product %s failed: %w", prd.productID, err)
		}
	}
	return nil
}

func buildAdditionalAppName(runtimeName string) string {
	return fmt.Sprintf("%s-additional", runtimeName)
}

func (prd *PipelineRuntimeDeployer) buildApp() (*Application, error) {
	if prd.runtime.Spec.AdditionalResources == nil ||
		prd.runtime.Spec.AdditionalResources.Git == nil {
		return nil, nil
	}
	addResources := prd.runtime.Spec.AdditionalResources.Git

	ns := prd.runtime.GetNamespaces()[0]
	app := &Application{
		Resource: Resource{
			Product: prd.runtime.GetProduct(),
			Name:    buildAdditionalAppName(prd.runtime.Name),
		},
		Git: &ApplicationGit{
			URL:      "",
			Revision: addResources.Revision,
			Path:     addResources.Path,
			CodeRepo: "",
		},
		Destinations: []Space{{
			Resource: Resource{
				Product: prd.productID,
				Name:    ns,
			},
			SpaceType: SpaceTypeKubernetes,
			Kubernetes: SpaceKubernetes{
				Namespace: ns,
			},
		}},
	}

	if addResources.URL != "" {
		app.Git.URL = addResources.URL
	} else {
		codeRepo, err := prd.db.GetCodeRepo(addResources.CodeRepo)
		if err != nil {
			return nil, err
		}
		app.Git.CodeRepo = codeRepo.Name
		app.Git.URL = codeRepo.Spec.URL
	}

	return app, nil
}

func (prd *PipelineRuntimeDeployer) convertRuntimeToEventSource(runtime v1alpha1.ProjectPipelineRuntime) (EventSource, error) {
	es := EventSource{
		Resource: Resource{
			Product: runtime.GetProduct(),
			Name:    runtime.GetName(),
		},
		UniqueID: buildUniqueID(runtime.GetProduct(), runtime.GetName()),
		Events:   []Event{},
	}

	gitlabEvents, err := prd.getEventGitlabsFromEventSource(runtime.Spec.EventSources)
	if err != nil {
		return es, err
	}
	es.Events = append(es.Events, gitlabEvents...)
	es.Events = append(es.Events, prd.getEventCalendarFromEventSource(runtime.Spec.EventSources)...)

	return es, nil
}

func (prd *PipelineRuntimeDeployer) getEventGitlabsFromEventSource(runtimeEventSources []v1alpha1.EventSource) ([]Event, error) {
	eventMap := map[string]Event{}
	for _, runtimeEventSource := range runtimeEventSources {
		if runtimeEventSource.Gitlab == nil {
			continue
		}

		gitEvent := runtimeEventSource.Gitlab

		_, ok := eventMap[gitEvent.RepoName]
		if !ok {
			apiServer, err := prd.getCodeRepoApiServerAddr(gitEvent.RepoName)
			if err != nil {
				return nil, fmt.Errorf("get gitlab api server addr failed: %w", err)
			}

			eventMap[gitEvent.RepoName] = Event{
				Name: gitEvent.RepoName,
				Gitlab: &EventGitlab{
					APIServer: apiServer,
					Events:    gitEvent.Events,
					CodeRepo:  gitEvent.RepoName,
					RepoID:    getIDFromCodeRepo(gitEvent.RepoName),
				},
			}
		} else {
			eventMap[gitEvent.RepoName].Gitlab.Events = append(eventMap[gitEvent.RepoName].Gitlab.Events, gitEvent.Events...)
		}
	}

	var events []Event
	for _, event := range eventMap {
		event.Gitlab.Events = sets.New(event.Gitlab.Events...).UnsortedList()
		events = append(events, event)
	}

	return events, nil
}

func (prd *PipelineRuntimeDeployer) getEventCalendarFromEventSource(runtimeEventSources []v1alpha1.EventSource) []Event {
	var events []Event
	for _, runtimeEventSource := range runtimeEventSources {
		if runtimeEventSource.Calendar == nil {
			continue
		}

		calendarEvent := runtimeEventSource.Calendar

		events = append(events, Event{
			Name: buildCalendarEventName(runtimeEventSource.Name),
			Calendar: &EventCalendar{
				Schedule:       calendarEvent.Schedule,
				Interval:       calendarEvent.Interval,
				ExclusionDates: calendarEvent.ExclusionDates,
				Timezone:       calendarEvent.Timezone,
			},
		})
	}
	return events
}

func (prd *PipelineRuntimeDeployer) convertRuntimeToConsumers(uniqueID string, user User, runtime v1alpha1.ProjectPipelineRuntime) (*Consumers, error) {
	consumers := &Consumers{
		Resource: Resource{
			Product: prd.productID,
			Name:    prd.runtime.Name,
		},
		User: user,
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

		vars, err := prd.getVars(trigger.Revision, runtime.Spec.PipelineSource, user, runtimePipeline)
		if err != nil {
			return nil, err
		}

		evTask, err := getEventTaskFromEventSource(trigger.Revision, vars)
		if err != nil {
			return nil, err
		}

		if runtimeEventSource.Gitlab != nil {
			filters, err := getGitlabFiltersFromEventSource(*runtimeEventSource.Gitlab)
			if err != nil {
				return nil, err
			}

			consumer := Consumer{
				UniqueID:  uniqueID,
				EventName: runtimeEventSource.Gitlab.RepoName,
				EventType: EventTypeGitlab,
				Filters:   filters,
				Task:      evTask,
			}
			consumers.Consumers = append(consumers.Consumers, consumer)
		}

		if runtimeEventSource.Calendar != nil {
			consumer := Consumer{
				UniqueID:  uniqueID,
				EventName: buildCalendarEventName(runtimeEventSource.Name),
				EventType: EventTypeCalendar,
				Task:      evTask,
			}
			consumers.Consumers = append(consumers.Consumers, consumer)
		}
	}
	return consumers, nil
}

func getEventTaskFromEventSource(pipelineRevision string, vars map[string]interface{}) (EventTask, error) {
	rawTask, err := getRawTask(vars)
	if err != nil {
		return EventTask{}, fmt.Errorf("get raw task failed: %w", err)
	}

	task := EventTask{
		Type: EventTaskTypeRaw,
		Vars: nil,
		Raw:  rawTask,
	}

	revison := "main"
	if pipelineRevision != "" {
		task.Vars = []VariableTransmission{
			{
				Value:       revison,
				Destination: "spec.params.0.value",
			},
			{
				Source:      "body.ref",
				Destination: "spec.params.1.value",
			},
		}
	} else {
		task.Vars = []VariableTransmission{
			{
				Source:      "body.ref",
				Destination: "spec.params.0.value",
			},
		}
	}

	return task, nil
}

const (
	GitlabWebhookIndexEventType = "headers.X-Gitlab-Event"
	GitlabWebhookIndexRef       = "event.body.ref"
)

func getGitlabFiltersFromEventSource(eventSource v1alpha1.Gitlab) ([]Filter, error) {
	var filters []Filter
	for i := range eventSource.Events {
		eventHead, ok := gitlabWebhookEventToGitlabEventHeaderMapping[eventSource.Events[i]]
		if !ok {
			return nil, fmt.Errorf("un support event type %s", eventSource.Events[i])
		}

		filters = append(filters, Filter{
			Key:        GitlabWebhookIndexEventType,
			Value:      eventHead,
			Comparator: EqualTo,
		})
	}
	if eventSource.Revision != "" {
		filters = append(filters, Filter{
			Key:        GitlabWebhookIndexRef,
			Value:      eventSource.Revision,
			Comparator: Match,
		})
	}
	return filters, nil
}

func (prd *PipelineRuntimeDeployer) getCodeRepoApiServerAddr(name string) (string, error) {
	codeRepo, err := prd.db.GetCodeRepo(name)
	if err != nil {
		return "", err
	}
	provider, err := prd.db.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
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
	return fmt.Sprintf("%s-%s", eventName, EventTypeCalendar)
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

const (
	rawTaskFilePath = "./config/base"
)

func getRawTask(vars map[string]interface{}) (string, error) {
	nautesHomePath := os.Getenv(nautesconst.EnvNautesHome)
	taskFilePath := path.Join(nautesHomePath, rawTaskFilePath)
	tmpl, err := template.ParseFiles(taskFilePath)
	if err != nil {
		return "", err
	}

	var triggerByte bytes.Buffer
	err = tmpl.Execute(&triggerByte, vars)
	if err != nil {
		return "", err
	}

	return triggerByte.String(), nil
}

func (prd *PipelineRuntimeDeployer) getVars(pipelineRevision, codeRepoName string, user User, pipeline v1alpha1.Pipeline) (map[string]interface{}, error) {
	codeRepo, err := prd.db.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, err
	}
	provider, err := prd.db.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, err
	}

	isCodeRepoTrigger := "false"
	if pipelineRevision != "" {
		isCodeRepoTrigger = "true"
	}

	return map[string]interface{}{
		"isCodeRepoTrigger":        isCodeRepoTrigger,
		"runtimeNamespace":         prd.runtime.GetNamespaces()[0],
		"pipelinePath":             pipeline.Path,
		"serviceAccountName":       user.AuthInfo.Kubernetes[0].ServiceAccount,
		"runtimeName":              user.AuthInfo.Kubernetes[0].ServiceAccount,
		"pipelineRepoProviderType": provider.Spec.ProviderType,
		"pipelineRepoID":           codeRepo.Name,
		"pipelineRepoURL":          codeRepo.Spec.URL,
		"pipelineLabel":            pipeline.Label,
	}, nil
}
