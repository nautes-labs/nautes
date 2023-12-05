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
	"strconv"
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/common"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SensorGenerator struct {
	namespace                  string
	k8sClient                  client.Client
	requestVarPathSearchEngine component.EventSourceSearchEngine
}

type ConsumerCache struct {
	component.ResourceMetaData
	Hash uint32 `json:"hash"`
}

func NewConsumerCache(consumerSet component.ConsumerSet) ConsumerCache {
	return ConsumerCache{
		ResourceMetaData: consumerSet.ResourceMetaData,
		Hash:             utils.GetStructHash(consumerSet),
	}
}

func (c *ConsumerCache) isSame(consumerSet component.ConsumerSet) bool {
	hash := utils.GetStructHash(consumerSet)
	return c.Hash == hash
}

const LabelConsumer = "nautes.argoevents.consumer"
const ConfigMapNameConsumerCache = "nautes-consumer-cache"
const ServiceAccountArgoEvents = "argo-events-sa"

// CreateSensor creates the consumer to trigger the init pipeline.
// The consumers listen to different types of event sources that generate events.
// 1. It checks the cache that finding if the consumer is changed then deleted.
// 2. It creates or updates all the Sensors by the consumer.
// 3. Tt updates the consumer to the cache.
func (sg *SensorGenerator) CreateSensor(ctx context.Context, consumer component.ConsumerSet) error {
	cache, err := sg.getConsumerCache(ctx, buildConsumerLabel(consumer.Product, consumer.Name))
	if err != nil {
		return fmt.Errorf("load consumer cache failed: %w", err)
	}

	if cache == nil || !cache.isSame(consumer) {
		if err := sg.deleteSensor(ctx, consumer.Product, consumer.Name); err != nil {
			return fmt.Errorf("clean sensor failed: %w", err)
		}
	}

	for i := range consumer.Consumers {
		sensor := sg.buildSensor(consumer.Product, consumer.Name, i)

		dependencies := []sensorv1alpha1.EventDependency{buildDependencies(consumer.Consumers[i])}

		trigger, err := sg.buildTrigger(consumer.Consumers[i])
		if err != nil {
			return fmt.Errorf("build trigger failed: %w", err)
		}
		triggers := []sensorv1alpha1.Trigger{*trigger}

		_, err = controllerutil.CreateOrUpdate(ctx, sg.k8sClient, sensor, func() error {
			sensor.Spec.Dependencies = dependencies
			sensor.Spec.Triggers = triggers
			sensor.Spec.Template = &sensorv1alpha1.Template{
				ServiceAccountName: ServiceAccountArgoEvents,
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("create sensor failed: %w", err)
		}
	}

	if err := sg.updateConsumerCache(ctx, consumer); err != nil {
		return fmt.Errorf("update consumer cache failed: %w", err)
	}

	return nil
}

// DeleteSensor deletes all consumers by product name and name of consumer collection.
func (sg *SensorGenerator) DeleteSensor(ctx context.Context, productName, name string) error {
	if err := sg.deleteSensor(ctx, productName, name); err != nil {
		return fmt.Errorf("delete sensor failed: %w", err)
	}

	if err := sg.deleteConsumerCache(ctx, buildConsumerLabel(productName, name)); err != nil {
		return fmt.Errorf("delete consumer cache failed: %w", err)
	}
	return nil
}

// deleteSensor deletes the sensor by product name and consumer name.
func (sg *SensorGenerator) deleteSensor(ctx context.Context, productName, name string) error {
	logger.V(1).Info("delete sensor set", "name", name)
	deleteOpts := []client.DeleteAllOfOption{
		client.InNamespace(sg.namespace),
		client.MatchingLabels{LabelConsumer: buildConsumerLabel(productName, name)},
	}
	if err := sg.k8sClient.DeleteAllOf(ctx, &sensorv1alpha1.Sensor{}, deleteOpts...); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("delete sensors from consumer %s failed: %w", buildConsumerLabel(productName, name), err)
	}
	return nil
}

// getConsumerCache gets the consumers by product name and consumer name.
func (sg *SensorGenerator) getConsumerCache(ctx context.Context, consumerLabel string) (*ConsumerCache, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.namespace,
		},
	}

	err := sg.k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	consumerStr, ok := cm.Data[consumerLabel]
	if !ok {
		return nil, nil
	}

	num, err := strconv.Atoi(consumerStr)
	if err != nil {
		logger.Error(err, "parse consumer cache failed", "name", consumerLabel)
		num = 0
	}

	consumer := &ConsumerCache{
		Hash: uint32(num),
	}

	return consumer, nil
}

// updateConsumerCache updates the consumer by product name and consumer name.
func (sg *SensorGenerator) updateConsumerCache(ctx context.Context, consumer component.ConsumerSet) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.namespace,
		},
	}

	cache := NewConsumerCache(consumer)
	_, err := controllerutil.CreateOrPatch(ctx, sg.k8sClient, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[buildConsumerLabel(consumer.Product, consumer.Name)] = fmt.Sprintf("%d", cache.Hash)
		return nil
	})

	return err
}

// deleteConsumerCache deletes the consumer by product name and consumer name.
func (sg *SensorGenerator) deleteConsumerCache(ctx context.Context, consumerLabel string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, sg.k8sClient, cm, func() error {
		delete(cm.Data, consumerLabel)
		return nil
	})
	return err
}

// buildSensor return an instance of the sensor which is custom resource in Kubernetes.
func (sg *SensorGenerator) buildSensor(productName, name string, num int) *sensorv1alpha1.Sensor {
	return &sensorv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildSensorName(productName, name, num),
			Namespace: sg.namespace,
			Labels: map[string]string{
				LabelConsumer: buildConsumerLabel(productName, name),
			},
		},
		Spec: sensorv1alpha1.SensorSpec{
			Dependencies: []sensorv1alpha1.EventDependency{},
			Triggers:     []sensorv1alpha1.Trigger{},
		},
	}
}

// Transform custom comparable symbol to sensor comparable symbol.
var (
	comparatorMap = map[component.Comparator]sensorv1alpha1.Comparator{
		component.GreaterThanOrEqualTo: sensorv1alpha1.GreaterThanOrEqualTo,
		component.GreaterThan:          sensorv1alpha1.GreaterThan,
		component.EqualTo:              sensorv1alpha1.EqualTo,
		component.NotEqualTo:           sensorv1alpha1.NotEqualTo,
		component.LessThan:             sensorv1alpha1.LessThan,
		component.LessThanOrEqualTo:    sensorv1alpha1.LessThanOrEqualTo,
	}
)

// buildDependencies return an instance of event dependency as sub of the sensor which is custom resource in Kubernetes.
// Transform consumer event trigger condition to the event dependency.
func buildDependencies(consumer component.Consumer) sensorv1alpha1.EventDependency {
	eventDependency := sensorv1alpha1.EventDependency{
		Name:            buildDependencyName(consumer.UniqueID, consumer.EventSourceName, consumer.EventSourceType),
		EventSourceName: buildEventSourceName(consumer.UniqueID, consumer.EventSourceType),
		EventName:       consumer.EventSourceName,
		Filters:         &sensorv1alpha1.EventDependencyFilter{},
	}

	var dataFilters []sensorv1alpha1.DataFilter
	var matchRules []component.Filter

	for i := range consumer.Filters {
		filter := consumer.Filters[i]
		if filter.Comparator == component.Match {
			matchRules = append(matchRules, filter)
		} else {
			dataFilters = append(dataFilters, sensorv1alpha1.DataFilter{
				Path:       filter.Key,
				Type:       sensorv1alpha1.JSONTypeString,
				Value:      []string{filter.Value},
				Comparator: comparatorMap[filter.Comparator],
			})
		}
	}
	eventDependency.Filters.Data = dataFilters

	if len(matchRules) != 0 {
		eventDependency.Filters.Script = buildMatchScript(matchRules)
	}

	return eventDependency
}

var scriptTemplate = "if not string.match(%s, \"%s\") then return false end"

// buildMatchScript builds a filter of the Lua script from webhook events.
func buildMatchScript(filters []component.Filter) string {
	scripts := make([]string, len(filters)+1)
	for i := range filters {
		scripts[i] = fmt.Sprintf(scriptTemplate, filters[i].Key, filters[i].Value)
	}

	scripts[len(filters)] = "return true"
	return strings.Join(scripts, "\n")
}

// buildTrigger builds a trigger instance which is custom resource in Kubernetes.
// Combines parameters to trigger a stander Kubernetes trigger that is init pipeline.
func (sg *SensorGenerator) buildTrigger(consumer component.Consumer) (*sensorv1alpha1.Trigger, error) {
	comRes := common.NewResource(consumer.Task.Raw)

	parameters := make([]sensorv1alpha1.TriggerParameter, len(consumer.Task.Vars))
	for i, inputOverWrite := range consumer.Task.Vars {
		eventSourcePath, err := sg.requestVarPathSearchEngine.GetTargetPathInEventSource(component.RequestDataConditions{
			EventType:         "",
			EventSourceType:   string(consumer.EventSourceType),
			EventListenerType: eventListenerName,
			RequestVar:        inputOverWrite.RequestVar,
		})
		if err != nil {
			return nil, fmt.Errorf("get target date path in event source failed: %w", err)
		}
		paras := sensorv1alpha1.TriggerParameter{
			Src: &sensorv1alpha1.TriggerParameterSource{
				DependencyName: buildDependencyName(consumer.UniqueID, consumer.EventSourceName, consumer.EventSourceType),
				DataKey:        eventSourcePath,
			},
			Dest: consumer.Task.Vars[i].Dest,
		}
		parameters[i] = paras
	}

	return &sensorv1alpha1.Trigger{
		Template: &sensorv1alpha1.TriggerTemplate{
			Name:       buildDependencyName(consumer.UniqueID, consumer.EventSourceName, consumer.EventSourceType),
			Conditions: buildDependencyName(consumer.UniqueID, consumer.EventSourceName, consumer.EventSourceType),
			K8s: &sensorv1alpha1.StandardK8STrigger{
				Source: &sensorv1alpha1.ArtifactLocation{
					Resource: &comRes,
				},
				Operation:  sensorv1alpha1.Create,
				Parameters: parameters,
			},
		},
	}, nil
}
