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
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ClusterUsage struct {
	Products map[string]ProductUsage `yaml:"products"`
}

func (cu *ClusterUsage) AddRuntimeUsage(runtime v1alpha1.Runtime) {
	productID := runtime.GetProduct()

	_, ok := cu.Products[productID]
	if !ok {
		cu.Products[productID] = ProductUsage{
			Runtimes: StringSet{
				Set: map[string]sets.Empty{},
			},
		}
	}

	cu.Products[productID].Runtimes.Insert(runtime.GetName())
}

func (cu *ClusterUsage) DeleteRuntimeUsage(runtime v1alpha1.Runtime) {
	productID := runtime.GetProduct()
	_, ok := cu.Products[productID]
	if !ok {
		return
	}

	cu.Products[productID].Runtimes.Delete(runtime.GetName())
}

type ProductUsage struct {
	Runtimes StringSet `yaml:"runtimes"`
}

type StringSet struct {
	sets.Set[string]
}

func (ss StringSet) MarshalJson() (interface{}, error) {
	return ss.UnsortedList(), nil
}

func (ss *StringSet) UnmarshalJson(unmarshal func(interface{}) error) error {
	return ss.unmarshal(unmarshal)
}

func (ss StringSet) MarshalYAML() (interface{}, error) {
	return ss.UnsortedList(), nil
}

func (ss *StringSet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return ss.unmarshal(unmarshal)
}

func (ss *StringSet) unmarshal(unmarshal func(interface{}) error) error {
	var stringArray []string
	if err := unmarshal(&stringArray); err != nil {
		return err
	}
	ss.Set = sets.New(stringArray...)
	return nil
}

func (ss StringSet) Len() int {
	return len(ss.Set)
}

func NewClustersUsage(usage string) (*ClusterUsage, error) {
	clusterUsage := &ClusterUsage{
		Products: map[string]ProductUsage{},
	}
	if err := yaml.Unmarshal([]byte(usage), clusterUsage); err != nil {
		return nil, err
	}
	return clusterUsage, nil
}

type UsageController struct {
	nautesNamespace string
	k8sClient       client.Client
	clusterName     string
	runtime         v1alpha1.Runtime
	productID       string
}

func (cuc *UsageController) AddProductUsage(ctx context.Context) (*ProductUsage, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameClustersUsageCache,
			Namespace: cuc.nautesNamespace,
		},
	}

	var usage *ProductUsage
	_, err := controllerutil.CreateOrUpdate(ctx, cuc.k8sClient, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		clusterUsage, err := NewClustersUsage(cm.Data[cuc.clusterName])
		if err != nil {
			return err
		}

		clusterUsage.AddRuntimeUsage(cuc.runtime)

		usageStr, err := yaml.Marshal(clusterUsage)
		if err != nil {
			return err
		}
		cm.Data[cuc.clusterName] = string(usageStr)
		productUsage := clusterUsage.Products[cuc.productID]
		usage = &productUsage

		return nil
	})

	if err != nil {
		return nil, err
	}

	return usage, nil
}

func (cuc *UsageController) GetProductUsage(ctx context.Context) (*ProductUsage, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameClustersUsageCache,
			Namespace: cuc.nautesNamespace,
		},
	}

	if err := cuc.k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	clusterUsage, err := NewClustersUsage(cm.Data[cuc.clusterName])
	if err != nil {
		return nil, err
	}

	productUsage := clusterUsage.Products[cuc.productID]
	return &productUsage, nil
}

func (cuc *UsageController) DeleteProductUsage(ctx context.Context) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameClustersUsageCache,
			Namespace: cuc.nautesNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, cuc.k8sClient, cm, func() error {
		if cm.Data == nil {
			return nil
		}

		clusterUsage, err := NewClustersUsage(cm.Data[cuc.clusterName])
		if err != nil {
			return err
		}

		clusterUsage.DeleteRuntimeUsage(cuc.runtime)

		usageStr, err := yaml.Marshal(clusterUsage)
		if err != nil {
			return err
		}
		cm.Data[cuc.clusterName] = string(usageStr)
		return nil
	})

	return err
}
