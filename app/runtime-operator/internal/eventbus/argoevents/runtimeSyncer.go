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

package argoevents

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"
	"text/template"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/constant"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"

	externalsecretcrd "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nautescrd.AddToScheme(scheme))
	utilruntime.Must(eventsourcev1alpha1.AddToScheme(scheme))
	utilruntime.Must(sensorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(hncv1alpha2.AddToScheme(scheme))
	utilruntime.Must(externalsecretcrd.AddToScheme(scheme))
}

const (
	IsolationShared                    = "shared"
	IsolationExclusive                 = "exclusive"
	eventSourcePort              int32 = 12000
	ConfigMapTriggerTemplateName       = "nautes-runtime-trigger-templates"
)

type sensorGenerateFunction func(ctx context.Context, trigger nautescrd.PipelineTrigger) (*sensorv1alpha1.Sensor, error)
type sensorGenerateFunctions map[eventType]sensorGenerateFunction

type eventSourceGenerateFunction func(ctx context.Context) error
type eventSourceGenerateFunctions map[eventType]eventSourceGenerateFunction
type eventSourceCleanFunction func(ctx context.Context) error
type eventSourceCleanFunctions map[eventType]eventSourceCleanFunction

// runtimeSyncer is an operation unit that synchronizes eventbus to the target state.
type runtimeSyncer struct {
	// k8sClient is a kubernetes client of dest cluster
	k8sClient client.Client
	// tenantK8sClient is a kubernetes client of tenant cluster
	tenantK8sClient client.Client
	// resourceLabel is a label will add to the resources whitch doesn't has owner reference
	resouceLabel         map[string]string
	config               nautescfg.Config
	productName          string
	runtime              nautescrd.ProjectPipelineRuntime
	cluster              nautescrd.Cluster
	pipelineRepo         nautescrd.CodeRepo
	pipelineRepoProvider nautescrd.CodeRepoProvider
	secClient            interfaces.SecretClient
	triggerTemplates     map[string]string
	// vars use to create trigger template or resources names
	vars map[string]string
	// webhookURL store the domain of event source, it will be used when eventsource need callback addr, likes gitlab webhook.
	webhookURL string
	// eventSourceGeneraters store the functions to create argo eventsource. It create eventsource based on eventsources in project runtime.
	eventSourceGeneraters eventSourceGenerateFunctions
	// eventSourceCleaners store the functions to delete argo eventsource. It create eventsource based on eventsources in project runtime.
	eventSourceCleaners eventSourceCleanFunctions
	// sensorGeneraters store the funtions to create sensor's dependencies and triggers. It based on the triggers in project runtime.
	sensorGeneraters sensorGenerateFunctions
}

func newRuntimeSyncer(ctx context.Context, task interfaces.RuntimeSyncTask, tenantK8sClient client.Client) (*runtimeSyncer, error) {
	if task.AccessInfo.Type != interfaces.ACCESS_TYPE_K8S {
		return nil, fmt.Errorf("access type is not supported")
	}

	pipelineRuntime, ok := task.Runtime.(*nautescrd.ProjectPipelineRuntime)
	if !ok {
		return nil, fmt.Errorf("runtime %s is not a pipeline runtime", task.Runtime.GetName())
	}

	k8sClient, err := client.New(task.AccessInfo.Kubernetes, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return nil, fmt.Errorf("get secret client from context failed")
	}

	pipelineRepo := &nautescrd.CodeRepo{}
	key := types.NamespacedName{
		Namespace: task.Product.Name,
		Name:      pipelineRuntime.Spec.PipelineSource,
	}
	if err := tenantK8sClient.Get(ctx, key, pipelineRepo); err != nil {
		return nil, err
	}

	provider := &nautescrd.CodeRepoProvider{}
	key = types.NamespacedName{
		Namespace: task.NautesCfg.Nautes.Namespace,
		Name:      pipelineRepo.Spec.CodeRepoProvider,
	}
	if err := tenantK8sClient.Get(ctx, key, provider); err != nil {
		return nil, err
	}

	pipelineRepoURL, err := getURLFromCodeRepo(ctx, tenantK8sClient, task.NautesCfg.Nautes.Namespace, *pipelineRepo)

	useHTTPS := true
	url, err := GetURLFromCluster(task.Cluster, task.HostCluster, useHTTPS)
	if err != nil {
		return nil, fmt.Errorf("get url from cluster failed: %w", err)
	}

	templates := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapTriggerTemplateName,
			Namespace: task.NautesCfg.Nautes.Namespace,
		},
	}
	if err := tenantK8sClient.Get(ctx, client.ObjectKeyFromObject(templates), templates); err != nil {
		return nil, fmt.Errorf("load template failed: %w", err)
	}

	vars := map[string]string{
		keyProductName:              task.Product.Name,
		keyRuntimeName:              pipelineRuntime.Name,
		keyClusterName:              task.Cluster.Name,
		keyPipelineRepoProviderType: provider.Spec.ProviderType,
		keyPipelineRepoID:           pipelineRepo.Name,
		keyPipelineRepoURL:          pipelineRepoURL,
		keyServiceAccountName:       task.ServiceAccountName,
		keyRuntimeNamespaceName:     pipelineRuntime.GetNamespaces()[0],
	}

	cluster := &runtimeSyncer{
		k8sClient:             k8sClient,
		tenantK8sClient:       tenantK8sClient,
		resouceLabel:          task.GetLabel(),
		config:                task.NautesCfg,
		productName:           task.Product.Name,
		runtime:               *pipelineRuntime,
		cluster:               task.Cluster,
		pipelineRepo:          *pipelineRepo,
		pipelineRepoProvider:  *provider,
		secClient:             secClient,
		vars:                  vars,
		webhookURL:            url,
		triggerTemplates:      templates.Data,
		eventSourceGeneraters: map[eventType]eventSourceGenerateFunction{},
		eventSourceCleaners:   map[eventType]eventSourceCleanFunction{},
		sensorGeneraters:      map[eventType]sensorGenerateFunction{},
	}

	cluster.eventSourceGeneraters[eventTypeGitlab] = cluster.syncEventSourceGitlab
	cluster.eventSourceGeneraters[eventTypeCalendar] = cluster.syncEventSourceCalendar
	cluster.eventSourceCleaners[eventTypeGitlab] = cluster.deleteEventSourceGitlab
	cluster.eventSourceCleaners[eventTypeCalendar] = cluster.deleteEventSourceCalendar

	cluster.sensorGeneraters[eventTypeGitlab] = cluster.caculateSensorGitlab
	cluster.sensorGeneraters[eventTypeCalendar] = cluster.caculateSensorCalendar

	return cluster, nil
}

func getEventSourceType(event nautescrd.EventSource) []eventType {
	events := []eventType{}
	if event.Gitlab != nil {
		events = append(events, eventTypeGitlab)
	}
	if event.Calendar != nil {
		events = append(events, eventTypeCalendar)
	}
	return events
}

// SyncEventSources will loop all funtions in eventSourcesGeneraters to create eventsources.
// It will create argo event eventsources by eventsource types in project pipeline runtime. e.g. gitlab, calendar.
func (s *runtimeSyncer) SyncEventSources(ctx context.Context) error {
	for eventSourceType, syncFn := range s.eventSourceGeneraters {
		err := syncFn(ctx)
		if err != nil {
			return fmt.Errorf("generate event source failed, type %s: %w", eventSourceType, err)
		}
	}

	return nil
}

// DeleteEventSources will loop all funtions in eventSourceCleaners to clean up eventsources.
// It will delete event sources by eventsource types in project pipeline runtime.
func (s *runtimeSyncer) DeleteEventSources(ctx context.Context) error {
	for eventSourceType, delFn := range s.eventSourceCleaners {
		err := delFn(ctx)
		if err != nil {
			return fmt.Errorf("delete event source failed, type %s: %w", eventSourceType, err)
		}
	}

	return nil
}

func (s *runtimeSyncer) syncEventSource(ctx context.Context, name string, spec eventsourcev1alpha1.EventSourceSpec) error {
	key := types.NamespacedName{
		Namespace: s.config.EventBus.ArgoEvents.Namespace,
		Name:      name,
	}

	eventSource := &eventsourcev1alpha1.EventSource{}
	err := s.k8sClient.Get(ctx, key, eventSource)
	if err != nil {
		if apierrors.IsNotFound(err) {
			eventSource = &eventsourcev1alpha1.EventSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
					Labels:    s.resouceLabel,
				},
			}
		} else {
			return fmt.Errorf("get eventsource %s failed: %w", name, err)
		}
	}

	if !utils.IsBelongsToProduct(eventSource, s.productName) {
		return fmt.Errorf("event source is not belongs to product %s", s.productName)
	}

	if reflect.DeepEqual(eventSource, spec) {
		return nil
	}

	eventSource.Spec = spec

	return s.updateEventSource(ctx, eventSource)
}

func (s *runtimeSyncer) updateEventSource(ctx context.Context, eventSource *eventsourcev1alpha1.EventSource) error {
	if eventSource.CreationTimestamp.IsZero() {
		return s.k8sClient.Create(ctx, eventSource)
	}
	return s.k8sClient.Update(ctx, eventSource)
}

func (s *runtimeSyncer) deleteEventSource(ctx context.Context, eventSourceName string) error {
	err := s.k8sClient.Delete(ctx, &eventsourcev1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventSourceName,
			Namespace: s.config.EventBus.ArgoEvents.Namespace,
		},
	})
	return client.IgnoreNotFound(err)
}

// SyncSensors used to create argo event sensor by triggers in project pipeline runtime.
// It will create one sensor by one project pipeline runtime.
func (s *runtimeSyncer) SyncSensors(ctx context.Context) error {
	sensorName, err := getStringFromTemplate(tmplSensorName, s.vars)
	if err != nil {
		return err
	}

	spec, err := s.calculateSensor(ctx)
	if err != nil {
		return fmt.Errorf("calculate sensor failed: %w", err)
	}
	if spec == nil {
		err := s.k8sClient.Delete(ctx, &sensorv1alpha1.Sensor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorName,
				Namespace: s.config.EventBus.ArgoEvents.Namespace,
			},
		})
		return client.IgnoreNotFound(err)
	}

	err = s.syncSensor(ctx, sensorName, *spec)
	if err != nil {
		return fmt.Errorf("sync sensor failed: %w", err)
	}

	return nil
}

// DeleteSensors will delete sensor belongs to project pipeline runtime.
func (s *runtimeSyncer) DeleteSensors(ctx context.Context) error {
	sensorName, err := getStringFromTemplate(tmplSensorName, s.vars)
	if err != nil {
		return err
	}

	return s.deleteSensor(ctx, sensorName)
}

// calculateSensor will return a sensor spec based on project pipeline runtime
// It will loop trigger in runtime, create many sensors and append into one sensor.
func (s *runtimeSyncer) calculateSensor(ctx context.Context) (*sensorv1alpha1.SensorSpec, error) {
	spec := &sensorv1alpha1.SensorSpec{
		Dependencies: []sensorv1alpha1.EventDependency{},
		Triggers:     []sensorv1alpha1.Trigger{},
		Template: &sensorv1alpha1.Template{
			ServiceAccountName: s.config.EventBus.ArgoEvents.TemplateServiceAccount,
		},
	}

	for _, trigger := range s.runtime.Spec.PipelineTriggers {
		eventSource, err := s.runtime.GetEventSource(trigger.EventSource)
		if err != nil {
			return nil, err
		}
		eventSourceTypes := getEventSourceType(*eventSource)

		for _, eventSourceType := range eventSourceTypes {
			sensor, err := s.sensorGeneraters[eventSourceType](ctx, trigger)
			if err != nil {
				return nil, fmt.Errorf("generate sensor failed, name %s, type %s: %w", eventSource.Name, eventSourceType, err)
			}
			spec.Dependencies = append(spec.Dependencies, sensor.Spec.Dependencies...)
			spec.Triggers = append(spec.Triggers, sensor.Spec.Triggers...)
		}
	}

	if len(spec.Triggers) == 0 {
		return nil, nil
	}

	return spec, nil
}

func (s *runtimeSyncer) syncSensor(ctx context.Context, name string, spec sensorv1alpha1.SensorSpec) error {
	sensor := &sensorv1alpha1.Sensor{}
	key := types.NamespacedName{
		Namespace: s.config.EventBus.ArgoEvents.Namespace,
		Name:      name,
	}

	err := s.k8sClient.Get(ctx, key, sensor)
	if err != nil {
		if apierrors.IsNotFound(err) {
			sensor = &sensorv1alpha1.Sensor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
					Labels:    s.resouceLabel,
				},
			}
		} else {
			return err
		}
	}

	if !utils.IsBelongsToProduct(sensor, s.productName) {
		return fmt.Errorf("sensor %s is not belongs to product %s", sensor.Name, s.productName)
	}

	if reflect.DeepEqual(sensor, spec) {
		return nil
	}

	sensor.Spec = spec

	return s.updateSensor(ctx, sensor)
}

func (s *runtimeSyncer) updateSensor(ctx context.Context, sensor *sensorv1alpha1.Sensor) error {
	if sensor == nil {
		return fmt.Errorf("sensor is nil")
	}
	if sensor.CreationTimestamp.IsZero() {
		return s.k8sClient.Create(ctx, sensor)
	}
	return s.k8sClient.Update(ctx, sensor)
}

func (s *runtimeSyncer) deleteSensor(ctx context.Context, name string) error {
	sensor := &sensorv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.config.EventBus.ArgoEvents.Namespace,
		},
	}
	err := s.k8sClient.Delete(ctx, sensor)
	return client.IgnoreNotFound(err)
}

func (s *runtimeSyncer) updateService(ctx context.Context, service *corev1.Service) error {
	if service == nil {
		return fmt.Errorf("service is nil")
	}
	if service.CreationTimestamp.IsZero() {
		return s.k8sClient.Create(ctx, service)
	}
	return s.k8sClient.Update(ctx, service)
}

func (s *runtimeSyncer) updateIngress(ctx context.Context, ingress *networkv1.Ingress) error {
	if ingress == nil {
		return fmt.Errorf("ingress is nil")
	}
	if ingress.CreationTimestamp.IsZero() {
		return s.k8sClient.Create(ctx, ingress)
	}
	return s.k8sClient.Update(ctx, ingress)
}

func (s *runtimeSyncer) InitSecretRepo(ctx context.Context) error {
	switch s.config.Secret.RepoType {
	case nautescfg.SECRET_STORE_VAULT:
		return s.initSecretRepoVault(ctx)
	default:
		return fmt.Errorf("secret repo type %s is not supported", s.config.Secret.RepoType)
	}
}

const (
	vaultArgoEventRole = "argoevent"
)

func (s *runtimeSyncer) initSecretRepoVault(ctx context.Context) error {
	role := interfaces.Role{
		Name:   vaultArgoEventRole,
		Users:  []string{constant.ServiceAccountDefault},
		Groups: []string{s.config.EventBus.ArgoEvents.Namespace},
	}
	vaultRole, err := s.secClient.GetRole(ctx, s.cluster.Name, interfaces.Role{
		Name: vaultArgoEventRole,
	})
	if err != nil {
		return err
	}

	if vaultRole != nil && equality.Semantic.DeepEqual(*vaultRole, role) {
		return nil
	}

	return s.secClient.CreateRole(ctx, s.cluster.Name, role)
}

func (s *runtimeSyncer) getTriggerFromTemplate(templateName string, vars interface{}) (map[string]interface{}, error) {
	// Currently unable to specify which template to select, the first template is obtained by default.
	if len(s.triggerTemplates) == 0 {
		return nil, fmt.Errorf("trigger template is empty")
	}
	var tmplName string
	for name, _ := range s.triggerTemplates {
		tmplName = name
		break
	}

	tmpl, err := template.New(tmplName).Parse(s.triggerTemplates[tmplName])
	if err != nil {
		return nil, err
	}

	var triggerByte bytes.Buffer
	err = tmpl.Execute(&triggerByte, vars)
	if err != nil {
		return nil, err
	}

	var trigger map[string]interface{}
	if err := yaml.Unmarshal(triggerByte.Bytes(), &trigger); err != nil {
		return nil, err
	}

	return trigger, nil
}

func getProviderTypeFromCodeRepo(ctx context.Context, client client.Client, nautesNamespace string, repo nautescrd.CodeRepo) (string, error) {
	provider := &nautescrd.CodeRepoProvider{}
	key := types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      repo.Spec.CodeRepoProvider,
	}
	if err := client.Get(ctx, key, provider); err != nil {
		return "", err
	}

	return provider.Spec.ProviderType, nil
}

func getURLFromCodeRepo(ctx context.Context, client client.Client, nautesNamespace string, repo nautescrd.CodeRepo) (string, error) {
	provider := &nautescrd.CodeRepoProvider{}
	key := types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      repo.Spec.CodeRepoProvider,
	}
	if err := client.Get(ctx, key, provider); err != nil {
		return "", err
	}

	product := &nautescrd.Product{}
	key = types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      repo.Spec.Product,
	}
	if err := client.Get(ctx, key, product); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s.git", provider.Spec.SSHAddress, product.Spec.Name, repo.Spec.RepoName), nil
}

func deepCopyStringMap(oldMap map[string]string) map[string]string {
	newMap := make(map[string]string)
	for key, value := range oldMap {
		newMap[key] = value
	}
	return newMap
}

func getIDFromCodeRepo(repoName string) string {
	repoParts := strings.SplitN(repoName, "-", 2)
	if len(repoParts) != 2 {
		return ""
	}
	return repoParts[1]
}

func GetURLFromCluster(cluster nautescrd.Cluster, hostCluster *nautescrd.Cluster, isSecret bool) (string, error) {
	if cluster.Status.EntryPoints == nil {
		return "", fmt.Errorf("cluster %s does not has entrypoint", cluster.Name)
	}

	var serviceName string
	for k := range cluster.Status.EntryPoints {
		serviceName = k
		break
	}

	point := cluster.Status.EntryPoints[serviceName]

	var port int32
	if isSecret && point.HTTPSPort != 0 {
		port = point.HTTPSPort
	} else if !isSecret && point.HTTPPort != 0 {
		port = point.HTTPPort
	} else {
		return "", fmt.Errorf("can not find dest port in service %s", serviceName)
	}

	var protocol string
	if isSecret {
		protocol = "https"
	} else {
		protocol = "http"
	}

	var domain string
	if cluster.Spec.PrimaryDomain != "" {
		domain = cluster.Spec.PrimaryDomain
	} else if hostCluster != nil && hostCluster.Spec.PrimaryDomain != "" {
		domain = hostCluster.Spec.PrimaryDomain
	} else {
		var err error
		domain, err = getDomainFromURL(cluster.Spec.ApiServer)
		if err != nil {
			return "", err
		}
	}

	switch point.Type {
	case nautescrd.ServiceTypeNodePort:
		return fmt.Sprintf("%s://webhook.%s.%s:%d", protocol, cluster.Name, domain, port), nil
	default:
		return "", fmt.Errorf("unknow ingress service type")
	}
}

func getDomainFromURL(serviceURL string) (string, error) {
	var domain string
	clusterURL, err := url.Parse(serviceURL)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(clusterURL.Host)
	if err != nil {
		return "", err
	}
	ip := net.ParseIP(host)
	if ip != nil {
		domain = fmt.Sprintf("%s.nip.io", host)
	} else {
		domain = host
	}

	return domain, nil
}
