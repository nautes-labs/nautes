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
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
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
)

func init() {
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(eventsourcev1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(sensorv1alpha1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

const CacheName = "nautes-argo-event-cache"

const (
	ServiceAccountName      = "nautes-argo-event-sa"
	AnnotationCodeRepoUsage = "code-repos"
)

type ArgoEvent struct {
	components            *syncer.ComponentList
	clusterName           string
	namespace             string
	db                    database.Database
	k8sClient             client.Client
	user                  syncer.User
	space                 syncer.Space
	entryPoint            utils.EntryPoint
	eventSourceGenerators map[syncer.EventType]eventSourceGenerator
	sensorGenerator       SensorGenerator
}

type EventManager interface {
	CreateEventSource() error
	DeleteEventSource() error
	CreateConsumer() error
	DeleteConsumer() error
}

//+kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;create;update;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;create;update;delete

func NewArgoEvent(opt v1alpha1.Component, info *syncer.ComponentInitInfo) (syncer.EventListener, error) {
	if info.ClusterConnectInfo.Type != v1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.Type)
	}

	k8sClient, err := client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	cluster, err := info.NautesDB.GetCluster(info.ClusterName)
	if err != nil {
		return nil, err
	}
	var hostCluster *v1alpha1.Cluster
	if cluster.Spec.ClusterType == v1alpha1.CLUSTER_TYPE_VIRTUAL {
		hostCluster, err = info.NautesDB.GetCluster(cluster.Spec.HostCluster)
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
		return nil, fmt.Errorf("create argo event aservice account failed: %w", err)
	}

	ae := &ArgoEvent{
		components:  info.Components,
		clusterName: info.ClusterName,
		namespace:   namespace,
		db:          info.NautesDB,
		k8sClient:   k8sClient,
		user:        buildArgoEventUser(namespace),
		space:       buildArgoEventSpace(namespace),
		entryPoint:  *entryPoint,
	}

	ae.createEventSourceGenerators()
	ae.sensorGenerator = ae.newSensorGenerator()
	return ae, nil
}

func buildArgoEventUser(namespace string) syncer.User {
	user := syncer.User{
		Resource: syncer.Resource{
			Product: "",
			Name:    ServiceAccountName,
		},
		UserType: syncer.UserTypeMachine,
		AuthInfo: &syncer.Auth{
			Kubernetes: []syncer.AuthKubernetes{
				{
					ServiceAccount: ServiceAccountName,
					Namespace:      namespace,
				},
			},
		},
	}
	return user
}

func buildArgoEventSpace(namespace string) syncer.Space {
	space := syncer.Space{
		Resource: syncer.Resource{
			Product: "",
			Name:    namespace,
		},
		SpaceType: syncer.SpaceTypeKubernetes,
		Kubernetes: syncer.SpaceKubernetes{
			Namespace: namespace,
		},
	}
	return space
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (ae *ArgoEvent) CleanUp() error {
	return nil
}

func (ae *ArgoEvent) CreateEventSource(ctx context.Context, eventSource syncer.EventSource) error {
	if err := ae.components.SecretManagement.CreateUser(ctx, ae.user); err != nil {
		return fmt.Errorf("create role in secret database failed: %w", err)
	}

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

func (ae *ArgoEvent) CreateConsumer(ctx context.Context, consumers syncer.Consumers) error {
	return ae.sensorGenerator.CreateSensor(ctx, consumers)
}

func (ae *ArgoEvent) DeleteConsumer(ctx context.Context, productName, name string) error {
	return ae.sensorGenerator.DeleteSensor(ctx, productName, name)
}

type Cache struct {
	Gitlab   syncer.StringSet `yaml:"gitlab"`
	Calendar syncer.StringSet `yaml:"calendar"`
}

func (c *Cache) GetMissingEventTypes(pairCache Cache) []syncer.EventType {
	currentEventTypes := sets.New[syncer.EventType](c.GetEventType()...)
	pairEventTypes := sets.New[syncer.EventType](pairCache.GetEventType()...)

	return currentEventTypes.Difference(pairEventTypes).UnsortedList()
}

func (c *Cache) GetEventType() []syncer.EventType {
	var evTypes []syncer.EventType
	if c.Gitlab.Len() != 0 {
		evTypes = append(evTypes, syncer.EventTypeGitlab)
	}
	if c.Calendar.Len() != 0 {
		evTypes = append(evTypes, syncer.EventTypeCalendar)
	}
	return evTypes
}

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

func (ae *ArgoEvent) createCacheFromEventSource(es syncer.EventSource) Cache {
	cache := newCache()
	for i := range es.Events {
		if es.Events[i].Gitlab != nil {
			cache.Gitlab.Insert(es.Events[i].Name)
		}
		if es.Events[i].Calendar != nil {
			cache.Calendar.Insert(es.Events[i].Name)
		}
	}

	return cache
}

func newCache() Cache {
	return Cache{
		Gitlab: syncer.StringSet{
			Set: sets.New[string](),
		},
		Calendar: syncer.StringSet{
			Set: sets.New[string](),
		},
	}
}

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
	CreateEventSource(context.Context, syncer.EventSource) error
	DeleteEventSource(ctx context.Context, uniqueID string) error
}

func (ae *ArgoEvent) createEventSourceGenerators() {
	generators := map[syncer.EventType]eventSourceGenerator{}
	generators[syncer.EventTypeGitlab] = ae.newGitlabEventSourceGenerator()
	generators[syncer.EventTypeCalendar] = ae.newCalendarEventSourceGenerator()
	ae.eventSourceGenerators = generators
}

func (ae *ArgoEvent) newGitlabEventSourceGenerator() *GitlabEventSourceGenerator {
	esGenerator := &GitlabEventSourceGenerator{
		Components:     ae.components,
		HostEntrypoint: ae.entryPoint,
		Namespace:      ae.namespace,
		K8sClient:      ae.k8sClient,
		DB:             ae.db,
		User:           ae.user,
		Space:          ae.space,
	}
	return esGenerator
}

func (ae *ArgoEvent) newCalendarEventSourceGenerator() *CalendarEventSourceGenerator {
	return &CalendarEventSourceGenerator{
		Components:     ae.components,
		HostEntrypoint: ae.entryPoint,
		Namespace:      ae.namespace,
		K8sClient:      ae.k8sClient,
		DB:             ae.db,
		User:           ae.user,
		Space:          ae.space,
	}
}

func (ae *ArgoEvent) newSensorGenerator() SensorGenerator {
	return SensorGenerator{
		Namespace: ae.namespace,
		k8sClient: ae.k8sClient,
	}
}
