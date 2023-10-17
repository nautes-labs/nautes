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
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/cache"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type UsageController struct {
	nautesNamespace string
	k8sClient       client.Client
	clusterName     string
}

func (cuc *UsageController) GetUsage(ctx context.Context) (*cache.ClusterUsage, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameClustersUsageCache,
			Namespace: cuc.nautesNamespace,
		},
	}

	if err := cuc.k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	return cache.NewClustersUsage(cm.Data[cuc.clusterName])
}

func (cuc *UsageController) GetProductUsage(ctx context.Context, productID string) (*cache.ProductUsage, error) {
	clusterUsage, err := cuc.GetUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cluster usage failed: %w", err)
	}
	if clusterUsage == nil {
		return nil, nil
	}

	productUsage, ok := clusterUsage.Products[productID]
	if !ok {
		productUsage = cache.NewEmptyProductUsage()
	}
	return &productUsage, nil
}

func (cuc *UsageController) UpdateProductUsage(ctx context.Context, productID string, usage cache.ProductUsage) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigMapNameClustersUsageCache,
			Namespace: cuc.nautesNamespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, cuc.k8sClient, cm, func() error {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		clusterUsage, err := cache.NewClustersUsage(cm.Data[cuc.clusterName])
		if err != nil {
			return err
		}

		if usage.Runtimes.Len() == 0 {
			delete(clusterUsage.Products, productID)
		} else {
			clusterUsage.Products[productID] = usage
		}

		usageStr, err := yaml.Marshal(clusterUsage)
		if err != nil {
			return err
		}
		cm.Data[cuc.clusterName] = string(usageStr)
		return nil
	})

	return err
}
