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

package argoevent

import (
	"context"
	"fmt"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"gopkg.in/yaml.v2"

	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	networkingv1 "k8s.io/api/networking/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// init indicates will use these resources.
func init() {
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(eventsourcev1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(sensorv1alpha1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

var (
	logger = logf.Log.WithName("argoEvent")
)

const eventListenerName = "argo-events"

// CacheName indicates the name of the ConfigMap.
const CacheName = "nautes-argo-event-cache"

const (
	// ServiceAccountName indicates the name of the service account for the argo event.
	ServiceAccountName = "nautes-argo-event-sa"
	// AnnotationCodeRepoUsage defines the name of the key for event source annotation.
	AnnotationCodeRepoUsage = "code-repos"
)

type ArgoEvent struct {
	components                 *component.ComponentList
	clusterName                string
	namespace                  string
	db                         database.Snapshot
	k8sClient                  client.Client
	machineAccount             component.MachineAccount
	space                      component.Space
	entryPoint                 utils.EntryPoint
	eventSourceGenerators      map[component.EventSourceType]eventSourceGenerator
	sensorGenerator            SensorGenerator
	secMgrAuthInfo             *component.AuthInfo
	requestVarPathSearchEngine component.EventSourceSearchEngine
}

type EventManager interface {
	CreateEventSource() error
	DeleteEventSource() error
	CreateConsumer() error
	DeleteConsumer() error
}

//+kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;create;update;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;create;update;delete

func NewArgoEvent(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.EventListener, error) {
	if info.ClusterConnectInfo.ClusterKind != v1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.ClusterKind)
	}

	k8sClient, err := client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	cluster, err := info.NautesResourceSnapshot.GetCluster(info.ClusterName)
	if err != nil {
		return nil, err
	}
	var hostCluster *v1alpha1.Cluster
	if cluster.Spec.ClusterType == v1alpha1.CLUSTER_TYPE_VIRTUAL {
		hostCluster, err = info.NautesResourceSnapshot.GetCluster(cluster.Spec.HostCluster)
		if err != nil {
			return nil, err
		}
	}

	entryPoint, err := utils.GetEntryPointFromCluster(*cluster, hostCluster, "https")
	if err != nil {
		return nil, err
	}
	entryPoint.Domain = fmt.Sprintf("webhook.%s.%s", info.ClusterName, entryPoint.Domain)

	namespace := opt.Namespace

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: namespace,
		},
	}
	if err := k8sClient.Create(context.TODO(), serviceAccount); client.IgnoreAlreadyExists(err) != nil {
		return nil, fmt.Errorf("create argo event service account failed: %w", err)
	}

	account := buildArgoEventAccount(namespace)

	authInfo, err := info.Components.SecretManagement.CreateAccount(context.TODO(), account)
	if err != nil {
		return nil, fmt.Errorf("create role in secret management failed: %w", err)
	}

	ae := &ArgoEvent{
		components:                 info.Components,
		clusterName:                info.ClusterName,
		namespace:                  namespace,
		db:                         info.NautesResourceSnapshot,
		k8sClient:                  k8sClient,
		machineAccount:             account,
		space:                      buildArgoEventSpace(namespace),
		entryPoint:                 *entryPoint,
		eventSourceGenerators:      map[component.EventSourceType]eventSourceGenerator{},
		sensorGenerator:            SensorGenerator{},
		secMgrAuthInfo:             authInfo,
		requestVarPathSearchEngine: info.EventSourceSearchEngine,
	}

	ae.createEventSourceGenerators()
	ae.sensorGenerator = ae.newSensorGenerator()
	return ae, nil
}

// buildArgoEventAccount returns a machine account instance.
func buildArgoEventAccount(namespace string) component.MachineAccount {
	user := component.MachineAccount{
		Name:   ServiceAccountName,
		Spaces: []string{namespace},
	}
	return user
}

// buildArgoEventSpace returns a space that uses the Kubernetes namespace.
func buildArgoEventSpace(namespace string) component.Space {
	space := component.Space{
		ResourceMetaData: component.ResourceMetaData{
			Product: "",
			Name:    namespace,
		},
		SpaceType: component.SpaceTypeKubernetes,
		Kubernetes: &component.SpaceKubernetes{
			Namespace: namespace,
		},
	}
	return space
}

// CleanUp represents when the component generates cache information, implement this method to clean data.
// This method will be automatically called by the syncer after each tuning is completed.
func (ae *ArgoEvent) CleanUp() error {
	return nil
}

func (ae *ArgoEvent) GetComponentMachineAccount() *component.MachineAccount {
	return &ae.machineAccount
}

// CreateEventSource creates the event source to listen to events, that support to create multi-event source.
// 1. It creates a cache by event from event sources are in requests event sources.
// 2. It deletes the event sources that are not in the requested event sources.
// 3. It adds the request event sources to the cache.
func (ae *ArgoEvent) CreateEventSource(ctx context.Context, eventSource component.EventSourceSet) error {
	newCache := ae.createCacheFromEventSource(eventSource)
	cache, err := ae.getCache(ctx, eventSource.UniqueID)
	if err != nil {
		return fmt.Errorf("create cache from event source failed: %w", err)
	}

	for _, eventType := range newCache.GetEventType() {
		err := ae.eventSourceGenerators[eventType].CreateEventSource(ctx, eventSource)
		if err != nil {
			return fmt.Errorf("create event source failed, type %s: %w", eventType, err)
		}
	}

	for _, eventType := range cache.GetMissingEventTypes(newCache) {
		err := ae.eventSourceGenerators[eventType].DeleteEventSource(ctx, eventSource.UniqueID)
		if err != nil {
			return fmt.Errorf("delete event source failed, type %s: %w", eventType, err)
		}
	}

	if err := ae.updateCache(ctx, eventSource.UniqueID, newCache); err != nil {
		return fmt.Errorf("update argo event cache failed: %w", err)
	}

	return nil
}

// DeleteEventSource deletes an event source by event source collection unique ID.
// 1. It checks the old event source in the cache by unique ID.
// 2. It deletes the old event source.
// 3. It deletes the old event source in the cache by unique ID.
func (ae *ArgoEvent) DeleteEventSource(ctx context.Context, uniqueID string) error {
	cache, err := ae.getCache(ctx, uniqueID)
	if err != nil {
		return fmt.Errorf("get cache failed: %w", err)
	}

	for _, eventType := range cache.GetEventType() {
		err := ae.eventSourceGenerators[eventType].DeleteEventSource(ctx, uniqueID)
		if err != nil {
			return fmt.Errorf("delete event source failed, type %s: %w", eventType, err)
		}
	}

	if err := ae.deleteCache(ctx, uniqueID); err != nil {
		return fmt.Errorf("delete argo event cache failed: %w", err)
	}

	return nil
}

// CreateConsumer calls the implementation of creating consumers.
func (ae *ArgoEvent) CreateConsumer(ctx context.Context, consumers component.ConsumerSet) error {
	return ae.sensorGenerator.CreateSensor(ctx, consumers)
}

// DeleteConsumer calls the implementation of deleting consumers.
func (ae *ArgoEvent) DeleteConsumer(ctx context.Context, productName, name string) error {
	return ae.sensorGenerator.DeleteSensor(ctx, productName, name)
}

type Cache struct {
	Gitlab   utils.StringSet `yaml:"gitlab"`
	Calendar utils.StringSet `yaml:"calendar"`
}

// GetMissingEventTypes returns the difference between the current saving cache and the request cache.
func (c *Cache) GetMissingEventTypes(pairCache Cache) []component.EventSourceType {
	currentEventTypes := sets.New[component.EventSourceType](c.GetEventType()...)
	pairEventTypes := sets.New[component.EventSourceType](pairCache.GetEventType()...)

	return currentEventTypes.Difference(pairEventTypes).UnsortedList()
}

// GetEventType gets the event source type of current cache.
func (c *Cache) GetEventType() []component.EventSourceType {
	var evTypes []component.EventSourceType
	if c.Gitlab.Len() != 0 {
		evTypes = append(evTypes, component.EventTypeGitlab)
	}
	if c.Calendar.Len() != 0 {
		evTypes = append(evTypes, component.EventTypeCalendar)
	}
	return evTypes
}

// updateCache updates the cache to the ConfigMap in Kubernetes.
func (ae *ArgoEvent) updateCache(ctx context.Context, uuid string, cache Cache) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CacheName,
			Namespace: ae.namespace,
		},
	}

	cacheString, err := yaml.Marshal(cache)
	if err != nil {
		return fmt.Errorf("convert cache to string failed: %w", err)
	}

	_, err = controllerutil.CreateOrPatch(ctx, ae.k8sClient, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[uuid] = string(cacheString)
		return nil
	})
	return err
}

// deleteCache deletes the cache form the ConfigMap in Kubernetes.
func (ae *ArgoEvent) deleteCache(ctx context.Context, uuid string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CacheName,
			Namespace: ae.namespace,
		},
		Data: map[string]string{},
	}

	_, err := controllerutil.CreateOrPatch(ctx, ae.k8sClient, cm, func() error {
		delete(cm.Data, uuid)
		return nil
	})
	return err
}

// createCacheFromEventSource returns a cache instance built by the string set.
func (ae *ArgoEvent) createCacheFromEventSource(es component.EventSourceSet) Cache {
	cache := newCache()
	for i := range es.EventSources {
		if es.EventSources[i].Gitlab != nil {
			cache.Gitlab.Insert(es.EventSources[i].Name)
		}
		if es.EventSources[i].Calendar != nil {
			cache.Calendar.Insert(es.EventSources[i].Name)
		}
	}

	return cache
}

func newCache() Cache {
	return Cache{
		Gitlab:   utils.NewStringSet(),
		Calendar: utils.NewStringSet(),
	}
}

// getCache gets cache from the ConfigMap in Kubernetes.
func (ae *ArgoEvent) getCache(ctx context.Context, uuid string) (Cache, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CacheName,
			Namespace: ae.namespace,
		},
	}
	if err := ae.k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm); client.IgnoreNotFound(err) != nil {
		return Cache{}, err
	}

	if cm.Data == nil || cm.Data[uuid] == "" {
		return Cache{}, nil
	}

	cache := &Cache{}
	if err := yaml.Unmarshal([]byte(cm.Data[uuid]), cache); err != nil {
		return Cache{}, err
	}

	return *cache, nil
}

type eventSourceGenerator interface {
	CreateEventSource(context.Context, component.EventSourceSet) error
	DeleteEventSource(ctx context.Context, uniqueID string) error
}

// createEventSourceGenerators creates a map to store implementation instance.
func (ae *ArgoEvent) createEventSourceGenerators() {
	generators := map[component.EventSourceType]eventSourceGenerator{}
	generators[component.EventTypeGitlab] = ae.newGitlabEventSourceGenerator()
	generators[component.EventTypeCalendar] = ae.newCalendarEventSourceGenerator()
	ae.eventSourceGenerators = generators
}

// newGitlabEventSourceGenerator returns an instance of GitLab event source generator.
func (ae *ArgoEvent) newGitlabEventSourceGenerator() *GitlabEventSourceGenerator {
	esGenerator := &GitlabEventSourceGenerator{
		Components:     ae.components,
		HostEntrypoint: ae.entryPoint,
		Namespace:      ae.namespace,
		K8sClient:      ae.k8sClient,
		DB:             ae.db,
		User:           ae.machineAccount,
		Space:          ae.space,
		secMgrAuthInfo: ae.secMgrAuthInfo,
	}
	return esGenerator
}

// newCalendarEventSourceGenerator returns an instance of calendar event source generator.
func (ae *ArgoEvent) newCalendarEventSourceGenerator() *CalendarEventSourceGenerator {
	return &CalendarEventSourceGenerator{
		Components:     ae.components,
		HostEntrypoint: ae.entryPoint,
		Namespace:      ae.namespace,
		K8sClient:      ae.k8sClient,
		DB:             ae.db,
		User:           ae.machineAccount,
		Space:          ae.space,
	}
}

// newSensorGenerator returns an instance of sensor generator.
func (ae *ArgoEvent) newSensorGenerator() SensorGenerator {
	return SensorGenerator{
		namespace:                  ae.namespace,
		k8sClient:                  ae.k8sClient,
		requestVarPathSearchEngine: ae.requestVarPathSearchEngine,
	}
}
