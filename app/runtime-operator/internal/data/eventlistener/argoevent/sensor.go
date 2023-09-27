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
	"reflect"
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/common"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SensorGenerator struct {
	Namespace string
	k8sClient client.Client
}

type ConsumerCache struct {
	syncer.Consumers
}

const LabelConsumer = "nautes.argoevents.consumer"
const ConfigMapNameConsumerCache = "nautes-consumer-cache"
const ServiceAccountArgoEvents = "argo-events-sa"

func (sg *SensorGenerator) CreateSensor(ctx context.Context, consumer syncer.Consumers) error {
	if consumer.User.AuthInfo == nil || len(consumer.User.AuthInfo.Kubernetes) != 1 {
		return fmt.Errorf("can not find service account in user info")
	}
	cache, err := sg.getConsumerCache(ctx, buildConsumerLabel(consumer.Product, consumer.Name))
	if err != nil {
		return fmt.Errorf("load consumer cache failed: %w", err)
	}
	isSame := reflect.DeepEqual(cache, &ConsumerCache{consumer})

	if !isSame {
		if err := sg.deleteSensor(ctx, consumer.Product, consumer.Name); err != nil {
			return fmt.Errorf("clean sensor failed: %w", err)
		}
	}

	for i := range consumer.Consumers {
		sensor := sg.buildSensor(consumer.Product, consumer.Name, i)

		dependencies := []sensorv1alpha1.EventDependency{buildDependencies(consumer.Consumers[i])}

		trigger, err := buildTrigger(consumer.Consumers[i])
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

func (sg *SensorGenerator) DeleteSensor(ctx context.Context, productName, name string) error {
	if err := sg.deleteSensor(ctx, productName, name); err != nil {
		return fmt.Errorf("delete sensor failed: %w", err)
	}

	if err := sg.deleteConsumerCache(ctx, buildConsumerLabel(productName, name)); err != nil {
		return fmt.Errorf("delete consumer cache failed: %w", err)
	}
	return nil
}

func (sg *SensorGenerator) deleteSensor(ctx context.Context, productName, name string) error {
	deleteOpts := []client.DeleteAllOfOption{
		client.InNamespace(sg.Namespace),
		client.MatchingLabels{LabelConsumer: buildConsumerLabel(productName, name)},
	}
	if err := sg.k8sClient.DeleteAllOf(ctx, &sensorv1alpha1.Sensor{}, deleteOpts...); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("delete sensors from consumer %s failed: %w", buildConsumerLabel(productName, name), err)
	}
	return nil
}

func (sg *SensorGenerator) getConsumerCache(ctx context.Context, consumerLabel string) (*ConsumerCache, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.Namespace,
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

	var consumer *ConsumerCache
	if err := yaml.Unmarshal([]byte(consumerStr), &consumer); err != nil {
		return nil, fmt.Errorf("umarshal consumer cache failed: %w", err)
	}

	return consumer, nil
}

func (sg *SensorGenerator) updateConsumerCache(ctx context.Context, consumer syncer.Consumers) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.Namespace,
		},
	}

	cache, err := yaml.Marshal(ConsumerCache{
		Consumers: consumer,
	})
	if err != nil {
		return fmt.Errorf("marshal cache failed: %w", err)
	}
	_, err = controllerutil.CreateOrPatch(ctx, sg.k8sClient, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[buildConsumerLabel(consumer.Product, consumer.Name)] = string(cache)
		return nil
	})

	return err
}

func (sg *SensorGenerator) deleteConsumerCache(ctx context.Context, consumerLabel string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameConsumerCache,
			Namespace: sg.Namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, sg.k8sClient, cm, func() error {
		delete(cm.Data, consumerLabel)
		return nil
	})
	return err
}

func (sg *SensorGenerator) buildSensor(productName, name string, num int) *sensorv1alpha1.Sensor {
	return &sensorv1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildSensorName(productName, name, num),
			Namespace: sg.Namespace,
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

var (
	comparatorMap = map[syncer.Comparator]sensorv1alpha1.Comparator{
		syncer.GreaterThanOrEqualTo: sensorv1alpha1.GreaterThanOrEqualTo,
		syncer.GreaterThan:          sensorv1alpha1.GreaterThan,
		syncer.EqualTo:              sensorv1alpha1.EqualTo,
		syncer.NotEqualTo:           sensorv1alpha1.NotEqualTo,
		syncer.LessThan:             sensorv1alpha1.LessThan,
		syncer.LessThanOrEqualTo:    sensorv1alpha1.LessThanOrEqualTo,
	}
)

func buildDependencies(consumer syncer.Consumer) sensorv1alpha1.EventDependency {
	eventDependency := sensorv1alpha1.EventDependency{
		Name:            buildDependencyName(consumer.UniqueID, consumer.EventName, consumer.EventType),
		EventSourceName: buildEventSourceName(consumer.UniqueID, consumer.EventType),
		EventName:       consumer.EventName,
		Filters:         &sensorv1alpha1.EventDependencyFilter{},
	}

	var dataFilters []sensorv1alpha1.DataFilter
	var matchRuls []syncer.Filter

	for i := range consumer.Filters {
		filter := consumer.Filters[i]
		if filter.Comparator == syncer.Match {
			matchRuls = append(matchRuls, filter)
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

	if len(matchRuls) != 0 {
		eventDependency.Filters.Script = buildMatchScript(matchRuls)
	}

	return eventDependency
}

var scriptTemplate = "if not string.match(%s, \"%s\") then return false end"

func buildMatchScript(filters []syncer.Filter) string {
	scripts := make([]string, len(filters)+1)
	for i := range filters {
		scripts[i] = fmt.Sprintf(scriptTemplate, filters[i].Key, filters[i].Value)
	}

	scripts[len(filters)] = "return true"
	return strings.Join(scripts, "\n")
}

func buildTrigger(consumer syncer.Consumer) (*sensorv1alpha1.Trigger, error) {
	var resource interface{}
	if err := yaml.Unmarshal([]byte(consumer.Task.Raw), &resource); err != nil {
		return nil, fmt.Errorf("convert raw template to resource failed: %w", err)
	}
	comRes := common.NewResource(convert(resource))

	parameters := make([]sensorv1alpha1.TriggerParameter, len(consumer.Task.Vars))
	for i := range consumer.Task.Vars {
		paras := sensorv1alpha1.TriggerParameter{
			Src: &sensorv1alpha1.TriggerParameterSource{
				DependencyName: buildDependencyName(consumer.UniqueID, consumer.EventName, consumer.EventType),
			},
			Dest: consumer.Task.Vars[i].Destination,
		}

		if consumer.Task.Vars[i].Value != "" {
			paras.Src.Value = &consumer.Task.Vars[i].Value
		}
		if consumer.Task.Vars[i].Source != "" {
			paras.Src.DataKey = consumer.Task.Vars[i].Source
		}
		parameters[i] = paras
	}

	return &sensorv1alpha1.Trigger{
		Template: &sensorv1alpha1.TriggerTemplate{
			Name:       buildDependencyName(consumer.UniqueID, consumer.EventName, consumer.EventType),
			Conditions: buildDependencyName(consumer.UniqueID, consumer.EventName, consumer.EventType),
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

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
