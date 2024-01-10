// Copyright 2024 Nautes Authors
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

package middlewareruntime

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"k8s.io/apimachinery/pkg/runtime"
)

var ProviderCallerMapping map[string]string

type MiddlewareDeployer struct {
	providerInfo   component.ProviderInfo
	caller         component.Caller
	implementation string
}

func NewMiddlewareDeployer(providerInfo component.ProviderInfo) (*MiddlewareDeployer, error) {
	deployer := &MiddlewareDeployer{
		providerInfo: providerInfo,
		caller:       nil,
	}
	callerName, ok := ProviderCallerMapping[providerInfo.Name]
	if !ok {
		return nil, fmt.Errorf("unable to find caller corresponding to provider %s", providerInfo.Name)
	}

	caller, err := component.CallerFactory.GetCaller(callerName, providerInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to create caller: %s", err.Error())
	}
	deployer.caller = caller

	return deployer, nil
}

// Deploy synchronizes resources to the remote side according to the resource declaration format.
// Steps:
// 1. Convert the middleware declaration into a resource declaration.
// 2. Call the corresponding deployment method based on the type of Caller.
func (md *MiddlewareDeployer) Deploy(ctx context.Context, middleware v1alpha1.Middleware, state *v1alpha1.MiddlewareStatus) (newState *v1alpha1.MiddlewareStatus, err error) {
	res, err := ConvertMiddlewareToResources(md.providerInfo.Name, md.implementation, middleware)
	if err != nil {
		return nil, err
	}

	var newDeployState interface{}
	var oldDeployState []byte
	if state != nil && state.Status != nil {
		oldDeployState = state.Status.Raw
	}

	switch md.caller.(type) {
	case component.BasicCaller:
		newDeployState, err = md.deployByBasicCaller(ctx, res, oldDeployState)
		if err != nil {
			return nil, err
		}
	case component.AdvancedCaller:
		newDeployState, err = md.deployByAdvancedCaller(ctx, middleware.Type, res, oldDeployState)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("caller type %s not supported", md.caller.GetMetaData().Type)
	}

	return &v1alpha1.MiddlewareStatus{
		Resource: middleware,
		Status:   &runtime.RawExtension{Raw: newDeployState.([]byte)},
	}, nil
}

// deployByBasicCaller deploys using the basic caller.
func (md *MiddlewareDeployer) deployByBasicCaller(ctx context.Context, res []resources.Resource, state []byte) (newState interface{}, err error) {
	// 1. Compare the resource declaration with the cache to determine the resources to add, update, and delete.
	// 2. Remove the resources that need to be modified and compare with the actual environment on the remote side to determine the resources to add, update, and delete.
	// 3. Perform operations using the corresponding caller based on the resources to add, update, and delete.
	return nil, nil
}

// deployByAdvancedCaller 会使用 advanced caller 进行部署。
func (md *MiddlewareDeployer) deployByAdvancedCaller(ctx context.Context, middlewareType string, res []resources.Resource, state []byte,
) (newState interface{}, err error) {
	callerRes, opts, err := ConvertResourcesToAdvanceCallerResources(md.providerInfo.Name, md.caller.GetMetaData().Type, md.implementation, middlewareType, res)
	if err != nil {
		return nil, err
	}

	return md.caller.(component.AdvancedCaller).Deploy(ctx, callerRes, state, opts)
}

// Delete clears the deployed resources on the remote side based on the provided cache information.
//
// Parameters:
// - state is the cache returned from the previous deployment.
//
// Returns:
// - err is the error message when the deletion fails.
func (md *MiddlewareDeployer) Delete(ctx context.Context, state *v1alpha1.MiddlewareStatus) error {
	switch md.caller.(type) {
	case component.BasicCaller:
		return md.deleteByBasicCaller(ctx, state)
	case component.AdvancedCaller:
		return md.deleteByAdvancedCaller(ctx, state)
	default:
		return fmt.Errorf("caller type %s not supported", md.caller.GetMetaData().Type)
	}
}

func (md *MiddlewareDeployer) deleteByBasicCaller(ctx context.Context, state interface{}) error {
	return nil
}

func (md *MiddlewareDeployer) deleteByAdvancedCaller(ctx context.Context, state interface{}) error {
	return nil
}

// compareWithCache compares the given resource declaration with the cache and returns the resources that need to be added, updated, and deleted.
//
// Parameters:
// - resources is the collection of resources to be deployed.
// - state is the cache returned from the previous deployment, used for comparison with the declaration.
//
// Returns:
// - rst is the comparison result.
// - err is the error message during the comparison process.
func (md *MiddlewareDeployer) compareWithCache(ctx context.Context, res []resources.Resource, state map[string]string) (rst DiffResult, err error) {
	return
}

// compareWithPeer compares the given resource declaration with the remote environment and returns the resources that need to be added, updated, and deleted.
//
// Parameters:
// - resources is the collection of resources to be deployed.
// - state is the cache returned from the previous deployment, used to retrieve resources from the remote environment.
//
// Returns:
// - rst is the comparison result.
// - err is the error message during the comparison process.
func (md *MiddlewareDeployer) compareWithPeer(ctx context.Context, res []resources.Resource, state map[string]string) (rst DiffResult, err error) {
	return
}

// DiffResult represents the comparison result between the resource declaration and the cache or the actual remote environment.
type DiffResult struct {
	// New represents the resources that need to be added.
	New []resources.Resource
	// Diff represents the resources that need to be updated.
	Diff []resources.Resource
	// Expire represents the resources that need to be deleted.
	Expire []resources.Resource
}

// SortResources sorts the resources based on their dependencies.
func SortResources(res []resources.Resource) []resources.Resource {
	return nil
}
