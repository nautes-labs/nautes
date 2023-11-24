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

package task

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (prd *PipelineRuntimeDeployer) removeUnusedHookResource(ctx context.Context, oldSpace, newSpace *component.HookSpace) error {
	if oldSpace == nil {
		return nil
	}

	if newSpace == nil || !reflect.DeepEqual(oldSpace.BaseSpace, newSpace.BaseSpace) {
		return prd.deleteHookResources(ctx, oldSpace.BaseSpace, oldSpace.DeployResources)
	}

	removeRes := getUnusedResources(newSpace.DeployResources, oldSpace.DeployResources)
	err := prd.deleteHookResources(ctx, newSpace.BaseSpace, removeRes)
	if err != nil {
		return fmt.Errorf("remove resources failed: %w", err)
	}

	return nil
}

// createHookResource will create resources in the corresponding space based on the incoming resource request list.
// If the resource needs to access the secret management system, use authInfo for authentication.
//
// Clean up logic:
// - If the runtime space changes, all resources on the old space will be cleared.
// - If the requested resource changes, the resources that are no longer used will be cleared.
func (prd *PipelineRuntimeDeployer) createHookResource(ctx context.Context, authInfo component.AuthInfo, space component.Space, res []component.RequestResource) error {
	for i := range res {
		err := prd.reqHandler.CreateResource(ctx, space, authInfo, res[i])
		if err != nil {
			return fmt.Errorf("create resource failed: %w", err)
		}
	}

	return nil
}

// deleteHookResources will clear all the resources required for the runtime deployed on the space based on the currently processed runtime.
func (prd *PipelineRuntimeDeployer) deleteHookResources(ctx context.Context, space component.Space, res []component.RequestResource) error {
	for i := range res {
		err := prd.reqHandler.DeleteResource(ctx, space, res[i])
		if err != nil {
			return fmt.Errorf("delete resource failed: %w", err)
		}
	}
	return nil
}

// removeDuplicateRequests will remove duplicate resource requests in reqs.
func removeDuplicateRequests(reqs []component.RequestResource) []component.RequestResource {
	resMap := []component.RequestResource{}
	resMapHash := sets.New[uint32]()
	for i := range reqs {
		hash := utils.GetStructHash(reqs[i])
		if resMapHash.Has(hash) {
			continue
		}

		resMap = append(resMap, reqs[i])
		resMapHash.Insert(hash)
	}
	return resMap
}

// getUnusedResources compares newRes and oldRes to find 'ResourceRequest' that are only in oldRes.
func getUnusedResources(newRes []component.RequestResource, oldRes []component.RequestResource) []component.RequestResource {
	var removeRes []component.RequestResource
	newHash := sets.New[uint32]()
	for i := range newRes {
		newHash.Insert(utils.GetStructHash(newRes[i]))
	}
	for i := range oldRes {
		if newHash.Has(utils.GetStructHash(oldRes[i])) {
			continue
		}
		removeRes = append(removeRes, oldRes[i])
	}
	return removeRes
}
