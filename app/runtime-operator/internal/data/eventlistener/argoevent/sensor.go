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
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/common"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SensorGenerator struct {
	namespace                  string
	k8sClient                  client.Client
	requestVarPathSearchEngine component.EventSourceSearchEngine
}

const LabelConsumer = "nautes.argoevents.consumer"
const ConfigMapNameConsumerCache = "nautes-consumer-cache"
const ServiceAccountArgoEvents = "argo-events-sa"

// CreateSensor creates the consumer to trigger the init pipeline.
// The consumers listen to different types of event sources that generate events.
// 1. It creates or updates all the Sensors by the consumer.
// 2. Remove out date sensors.
func (sg *SensorGenerator) CreateSensor(ctx context.Context, consumer component.ConsumerSet) error {
	listOpts := []client.ListOption{
		client.InNamespace(sg.namespace),
		client.MatchingLabels{LabelConsumer: buildConsumerLabel(consumer.Product, consumer.Name)},
	}
	sensorList := &sensorv1alpha1.SensorList{}
	if err := sg.k8sClient.List(ctx, sensorList, listOpts...); err != nil {
		return fmt.Errorf("list existed sensor failed, product %s, name %s: %w", consumer.Product, consumer.Name, err)
	}

	for i := range consumer.Consumers {
		sensor := sg.buildSensor(consumer.Product, consumer.Name, i)

		dependencies := []sensorv1alpha1.EventDependency{buildDependencies(consumer.Consumers[i])}

		trigger, err := sg.buildTrigger(consumer.Consumers[i])
		if err != nil {
			return fmt.Errorf("build trigger failed: %w", err)
		}
		triggers := []sensorv1alpha1.Trigger{*trigger}

		operation, err := controllerutil.CreateOrUpdate(ctx, sg.k8sClient, sensor, func() error {
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
		if operation != controllerutil.OperationResultNone {
			logger.V(1).Info("sensor has been modified", "name", sensor.Name, "operation", operation)
		}
	}

	for i := len(consumer.Consumers); i < len(sensorList.Items); i++ {
		sensor := &sensorv1alpha1.Sensor{ObjectMeta: metav1.ObjectMeta{
			Name:      buildSensorName(consumer.Product, consumer.Name, i),
			Namespace: sg.namespace,
		}}
		if err := sg.k8sClient.Delete(ctx, sensor); err != nil {
			return fmt.Errorf("delete sensor failed: %w", err)
		}
		logger.V(1).Info("sensor has been deleted", "name", sensor.Name)
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
				Value:      filter.Value,
				Comparator: comparatorMap[filter.Comparator],
			})
		}
	}
	eventDependency.Filters.Data = dataFilters

	if len(matchRules) != 0 {
		eventDependency.Filters.Script = buildMatchScript(matchRules)
	}

	if component.CodeRepoEventSourceList.Has(consumer.EventSourceType) {
		eventDependency.Transform = buildTransformer(consumer.EventSourceType)
	}

	return eventDependency
}

func buildTransformer(eventSourceType component.EventSourceType) *sensorv1alpha1.EventDependencyTransformer {
	switch eventSourceType {
	case component.EventSourceTypeGitlab:
		return &sensorv1alpha1.EventDependencyTransformer{
			Script: `event.default = {}
event.default.ref = ""

eventType = ""
for k, v in pairs(event.headers['X-Gitlab-Event']) do
    eventType = v
end
if eventType == "Push Hook" or eventType == "Tag Push Hook"
then
    event.default.ref = event.body.ref
end

return event
`,
		}
	}
	logger.Info("unknown event source type", "TypeName", string(eventSourceType))
	return nil
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
		eventType := ""
		if len(consumer.EventTypes) != 0 {
			eventType = consumer.EventTypes[0]
		}

		eventSourcePath, err := sg.requestVarPathSearchEngine.GetTargetPathInEventSource(component.RequestDataConditions{
			EventType:         eventType,
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
