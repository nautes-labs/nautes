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

package tekton

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Syncer struct {
	k8sClient client.Client
}

func NewSyncer(client client.Client) interfaces.Pipeline {
	return Syncer{
		k8sClient: client,
	}
}

func (s Syncer) DeployPipelineRuntime(ctx context.Context, task interfaces.RuntimeSyncTask) error {
	opts, err := s.getDestClusterOptions(ctx, task)
	if err != nil {
		return err
	}
	destCluster, err := newDestCluster(ctx, task, *opts)
	if err != nil {
		return err
	}

	if err := destCluster.syncProduct(ctx); err != nil {
		return fmt.Errorf("sync product pipeline failed: %w", err)
	}

	if err := destCluster.grantPermissionToPipelineRun(ctx); err != nil {
		return fmt.Errorf("grant permission for run pipeline failed: %w", err)
	}

	return nil
}

func (s Syncer) UnDeployPipelineRuntime(ctx context.Context, task interfaces.RuntimeSyncTask) error {
	opts, err := s.getDestClusterOptions(ctx, task)
	if err != nil {
		return err
	}
	destCluster, err := newDestCluster(ctx, task, *opts)
	if err != nil {
		return err
	}

	removeRuntime, ok := task.Runtime.(*nautescrd.ProjectPipelineRuntime)
	if !ok {
		return fmt.Errorf("convert runtime to pipeline runtime failed")
	}
	removable, err := s.checkProductIsCleanable(ctx, task.Product.Name, removeRuntime)
	if err != nil {
		return fmt.Errorf("check product pipeline is removable failed: %w", err)
	}

	if removable {
		if err := destCluster.cleanUpProduct(ctx); err != nil {
			return fmt.Errorf("delete product pipeline failed: %w", err)
		}
	}

	if err := destCluster.revokePermissionFromPipelineRun(ctx); err != nil {
		return fmt.Errorf("grant permission for run pipeline failed: %w", err)
	}

	return nil
}

func (s *Syncer) checkProductIsCleanable(ctx context.Context, productName string, removeRuntime *nautescrd.ProjectPipelineRuntime) (bool, error) {
	runtimeList := &nautescrd.ProjectPipelineRuntimeList{}
	listOpts := []client.ListOption{
		client.InNamespace(productName),
	}

	if err := s.k8sClient.List(ctx, runtimeList, listOpts...); err != nil {
		return false, err
	}

	for _, runtime := range runtimeList.Items {
		if runtime.DeletionTimestamp.IsZero() &&
			runtime.Spec.Destination == removeRuntime.Spec.Destination {
			return false, nil
		}
	}
	return true, nil
}

func (s *Syncer) getDestClusterOptions(ctx context.Context, task interfaces.RuntimeSyncTask) (*destClusterOptions, error) {
	productCodeRepoList := &nautescrd.CodeRepoList{}
	listOpts := []client.ListOption{
		client.InNamespace(task.NautesCfg.Nautes.Namespace),
		client.MatchingLabels(map[string]string{nautescrd.LABEL_FROM_PRODUCT: task.Product.Name}),
	}
	if err := s.k8sClient.List(ctx, productCodeRepoList, listOpts...); err != nil {
		return nil, fmt.Errorf("list product coderepo failed: %w", err)
	}
	if len(productCodeRepoList.Items) != 1 {
		return nil, fmt.Errorf("product coderepo is not unique")
	}

	productCodeRepoProvider := &nautescrd.CodeRepoProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productCodeRepoList.Items[0].Spec.CodeRepoProvider,
			Namespace: task.NautesCfg.Nautes.Namespace,
		},
	}
	if err := s.k8sClient.Get(ctx, client.ObjectKeyFromObject(productCodeRepoProvider), productCodeRepoProvider); err != nil {
		return nil, fmt.Errorf("get product coderepo provider failed: %w", err)
	}

	if task.RuntimeType != interfaces.RUNTIME_TYPE_PIPELINE {
		return nil, fmt.Errorf("runtime type %s is not support", task.RuntimeType)
	}
	runtime := task.Runtime.(*nautescrd.ProjectPipelineRuntime)
	runtimeKey := types.NamespacedName{
		Name:      runtime.Spec.PipelineSource,
		Namespace: task.Product.Name,
	}
	runtimeCodeRepoProvider, runtimeCodeRepo, err := utils.GetCodeRepoProviderAndCodeRepoWithURL(ctx, s.k8sClient, runtimeKey, task.NautesCfg.Nautes.Namespace)
	if err != nil {
		return nil, err
	}

	return &destClusterOptions{
		product: codeRepoInfo{
			CodeRepo: productCodeRepoList.Items[0],
			Provider: *productCodeRepoProvider,
		},
		runtime: codeRepoInfo{
			CodeRepo: *runtimeCodeRepo,
			Provider: *runtimeCodeRepoProvider,
		},
	}, nil
}
