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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	callerhttp "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/caller/http"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/transformer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	runtimeconfig "github.com/nautes-labs/nautes/app/runtime-operator/pkg/config"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
	"github.com/nautes-labs/nautes/pkg/middlewaresecret"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	component.AddFunctionNewCaller(component.CallerTypeHTTP, callerhttp.NewCaller)
	if err := transformer.LoadMiddlewareTransformRules(); err != nil {
		panic(fmt.Errorf("failed to load middleware transform rules: %w", err))
	}
	if err := transformer.LoadResourceTransformers(); err != nil {
		panic(fmt.Errorf("failed to load resource transformers: %w", err))
	}
}

var (
	SecretIndexDirectory middlewaresecret.SecretIndexDirectoryInterface
)

// MiddlewareDeploymentInfo contains the information needed to deploy middleware.
type MiddlewareDeploymentInfo struct {
	providerInfo   component.ProviderInfo
	callerType     string
	implementation string
}

type RuntimeInfo struct {
	Name            string
	Product         string
	Environment     string
	Cluster         string
	NautesNamespace string
}

// CommonMiddlewareStatus represents the status of a middleware deployment.
type CommonMiddlewareStatus struct {
	ResourceStatus []resources.Resource `json:"resourceStatus" yaml:"resourceStatus"`
}

func (cms *CommonMiddlewareStatus) RemoveSensitiveData() {
	for i, resource := range cms.ResourceStatus {
		resource.ClearSensitiveAttributes()
		if resource.IsSensitiveResource() {
			resource.ClearSensitiveStatus()
		}
		cms.ResourceStatus[i] = resource
	}
}

// unmarshalTmpCommonMiddlewareStatus is a struct used for unmarshaling the temporary common middleware status.
type unmarshalTmpCommonMiddlewareStatus struct {
	ResourceStatus []json.RawMessage `json:"resourceStatus" yaml:"resourceStatus"`
}

func (cms *CommonMiddlewareStatus) UnmarshalJSON(data []byte) error {
	tmp := unmarshalTmpCommonMiddlewareStatus{
		ResourceStatus: []json.RawMessage{},
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("failed to unmarshal common middleware status: %w", err)
	}

	resArray := make([]resources.Resource, len(tmp.ResourceStatus))
	for i, resourceByte := range tmp.ResourceStatus {
		resource := resources.CommonResource{}
		if err := json.Unmarshal(resourceByte, &resource); err != nil {
			return fmt.Errorf("failed to unmarshal resource: %w", err)
		}
		resArray[i] = &resource
	}

	cms.ResourceStatus = resArray

	return nil
}

// NewMiddlewareDeploymentInfo creates a new instance of MiddlewareDeploymentInfo.
// It takes a providerInfo of type component.ProviderInfo and a middleware of type v1alpha1.Middleware as input.
// It returns a pointer to MiddlewareDeploymentInfo and an error.
// The function initializes the MiddlewareDeploymentInfo with the provided providerInfo.
// It then retrieves the configuration using runtimeconfig.NewConfig() and checks if the caller type is available for the given provider.
// If the middleware implementation is provided, it sets the implementation field in the MiddlewareDeploymentInfo.
// Finally, it sets the callerType field in the MiddlewareDeploymentInfo and returns the initialized instance.
func NewMiddlewareDeploymentInfo(providerInfo component.ProviderInfo, middleware v1alpha1.Middleware) (*MiddlewareDeploymentInfo, error) {
	info := &MiddlewareDeploymentInfo{
		providerInfo: providerInfo,
	}

	cfg, err := runtimeconfig.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	callerType, ok := cfg.ProviderCallerMapping[providerInfo.Type]
	if !ok {
		return nil, fmt.Errorf("unable to find caller corresponding to provider %s", providerInfo.Type)
	}

	info.callerType = callerType
	info.implementation = middleware.Implementation

	return info, nil
}

// DeployMiddleware synchronizes resources to the remote side according to the resource declaration format.
// DeployMiddleware is a function that deploys a middleware based on the provided information.
// It takes a context.Context, a v1alpha1.Middleware, a *v1alpha1.MiddlewareStatus, and a MiddlewareDeploymentInfo as input parameters.
// It returns a *v1alpha1.MiddlewareStatus and an error.
// The function converts the middleware to resources using the transformer.ConvertMiddlewareToResources function.
// It then creates a caller based on the deploymentInfo.callerType and deploymentInfo.providerInfo using the component.GetCaller function.
// The function deploys the middleware based on the caller's implementation type using different deployment methods.
// Finally, it updates the middleware status with the new deployment state and returns the updated status.
func DeployMiddleware(ctx context.Context,
	middleware v1alpha1.Middleware,
	state *v1alpha1.MiddlewareStatus,
	deploymentInfo MiddlewareDeploymentInfo,
	runtimeInfo RuntimeInfo,
) (newState *v1alpha1.MiddlewareStatus, err error) {
	// Get the resources from the middleware.
	res, err := transformer.ConvertMiddlewareToResources(deploymentInfo.providerInfo.Type, middleware)
	if err != nil {
		return nil, fmt.Errorf("failed to convert middleware to resources: %w", err)
	}

	// Create a caller based on the caller type and provider information.
	caller, err := component.GetCaller(deploymentInfo.callerType, deploymentInfo.providerInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to create caller: %w", err)
	}

	var newDeployState interface{}
	var previousDeployState []byte
	if state != nil && state.Status != nil {
		previousDeployState = state.Status.Raw
	}

	// Deploy the middleware based on the caller's implementation type.
	switch caller.GetImplementationType() {
	case component.CallerImplBasic:
		caller, ok := caller.(component.BasicCaller)
		if !ok {
			return nil, fmt.Errorf("caller is not a basic caller")
		}
		newDeployState, err = deployMiddlewareByBasicCaller(ctx, caller, res, previousDeployState, deploymentInfo.providerInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy by basic caller: %w", err)
		}
	case component.CallerImplAdvanced:
		caller, ok := caller.(component.AdvancedCaller)
		if !ok {
			return nil, fmt.Errorf("caller is not a advanced caller")
		}
		newDeployState, err = deployMiddlewareByAdvancedCaller(ctx, middleware.Type, caller, res, previousDeployState)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy by advanced caller: %w", err)
		}
	default:
		return nil, fmt.Errorf("caller type %s not supported", caller.GetImplementationType())
	}

	// Update the middleware status with the new deployment state.
	newState = &v1alpha1.MiddlewareStatus{
		Middleware: middleware,
		Status:     nil,
	}

	if middleware.InitAccessInfo != nil && needNewSecret(middleware, state) {
		metadata := middlewaresecret.MiddlewareMetadata{
			ProductName:     runtimeInfo.Product,
			EnvironmentName: runtimeInfo.Environment,
			ClusterName:     runtimeInfo.Cluster,
			RuntimeName:     runtimeInfo.Name,
			MiddlewareName:  middleware.Name,
		}
		secretID, err := SecretIndexDirectory.AddSecret(ctx, metadata, *middleware.InitAccessInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to add secret: %w", err)
		}
		newState.SecretID = secretID
	} else if state != nil {
		newState.SecretID = state.SecretID
	}

	if newDeployState != nil {
		jsonState, err := json.Marshal(newDeployState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal state: %w", err)
		}
		newState.Status = &runtime.RawExtension{Raw: jsonState}
	}

	return newState, nil
}

func needNewSecret(middleware v1alpha1.Middleware, status *v1alpha1.MiddlewareStatus) bool {
	return !(middleware.InitAccessInfo != nil && status != nil && status.SecretID != "")
}

// deployMiddlewareByBasicCaller deploys middleware based on the basic caller.
// This function takes the context, basic caller, resource list, state, and provider information as parameters, and returns the new state and an error.
// The function performs the following steps:
// 1. Converts the state to a resource list.
// 2. Compares the resource list with the state.
// 3. Compares the state with the differences in the remote environment.
// 4. Merges the results of the cache and remote comparison to generate a list of resources to create, update, or delete.
// 5. Performs the appropriate actions for create or update operations and sets the status of the resources.
// 6. Performs the appropriate actions for delete operations.
// 7. Combines the newly added resources, the updated resource statuses, and the unchanged resource statuses into a new state.
// 8. Returns the new state.
func deployMiddlewareByBasicCaller(ctx context.Context,
	caller component.BasicCaller,
	res []resources.Resource,
	state []byte,
	providerInfo component.ProviderInfo) (*CommonMiddlewareStatus, error) {
	var err error
	newStatus := &CommonMiddlewareStatus{
		ResourceStatus: []resources.Resource{},
	}

	// Convert json state to resources.
	lastStatus, err := ConvertResourcesJsonToResources(state)
	if err != nil {
		return nil, fmt.Errorf("failed to convert resources json to resources: %w", err)
	}

	// compare resources with last state.
	comparisonResultWithDeclaration := compareResources(res, lastStatus)

	// compare resources with peer.
	comparisonResultWithPeer, err := compareResourcesWithPeer(ctx, lastStatus, providerInfo.Type, caller)
	if err != nil {
		return nil, fmt.Errorf("compare resources with peer failed: %w", err)
	}

	// merge compare results.
	compareResult := mergeCompareResults(comparisonResultWithDeclaration, comparisonResultWithPeer)

	// build action list.
	createOrUpdateList := NewActionListCreateOrUpdate(compareResult.New, compareResult.Diff)
	deleteList := NewActionListDelete(compareResult.Expire)

	for _, action := range createOrUpdateList {
		var state *resources.ResourceStatus
		resTransformer, err := transformer.GetResourceTransformer(providerInfo.Type, caller.GetType(), action.Resource.GetType())
		if err != nil {
			return nil, fmt.Errorf("failed to get resource transformer: %w", err)
		}

		if action.Action == ResourceActionCreate {
			state, err = createResource(ctx, action.Resource, *resTransformer, caller)
		} else if action.Action == ResourceActionUpdate {
			state, err = updateResource(ctx, action.Resource, *resTransformer, caller)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create or update resource %s: %w", action.Resource.GetUniqueID(), err)
		}
		newRes := action.Resource
		newRes.SetStatus(*state)
		newStatus.ResourceStatus = append(newStatus.ResourceStatus, newRes)
	}

	for _, action := range deleteList {
		resTransformer, err := transformer.GetResourceTransformer(providerInfo.Type, caller.GetType(), action.Resource.GetType())
		if err != nil {
			return nil, fmt.Errorf("failed to get resource transformer: %w", err)
		}

		if err := deleteResource(ctx, action.Resource, *resTransformer, caller); err != nil {
			if !runtimeerr.IsResourceNotFoundError(err) {
				return nil, fmt.Errorf("failed to delete resource %s: %w", action.Resource.GetUniqueID(), err)
			}
		}
	}

	// combine new resources, updated resource statuses, and unchanged resource statuses into a new state.
	for _, resource := range compareResult.Unchanged {
		newStatus.ResourceStatus = append(newStatus.ResourceStatus, resource)
	}

	if len(newStatus.ResourceStatus) == 0 {
		return nil, nil
	}

	newStatus.RemoveSensitiveData()
	return newStatus, nil
}

// mergeCompareResults merges the results of the cache and remote comparison to generate a list of resources to create, update, or delete.
func mergeCompareResults(rstLocal, rstPeer resourceComparisonResult) (newResult resourceComparisonResult) {
	newResult = resourceComparisonResult{
		New:       make(map[string]resources.Resource),
		Diff:      make(map[string]resources.Resource),
		Expire:    make(map[string]resources.Resource),
		Unchanged: make(map[string]resources.Resource),
	}

	for id, r := range rstLocal.New {
		newResult.New[id] = r
	}

	for id, r := range rstLocal.Diff {
		newResult.Diff[id] = r
	}

	for id, r := range rstLocal.Expire {
		newResult.Expire[id] = r
	}

	// If a resource is missing on the remote side, it needs to be regenerated with a new definition.
	for id := range rstPeer.New {
		res, ok := rstLocal.Unchanged[id]
		if ok {
			newResult.New[id] = res
			delete(rstLocal.Unchanged, id)
		}
	}

	for id := range rstPeer.Diff {
		res, ok := rstLocal.Unchanged[id]
		if ok {
			newResult.Diff[id] = res
			delete(rstLocal.Unchanged, id)
		}
	}

	for id := range rstPeer.Expire {
		res, ok := rstLocal.Unchanged[id]
		if ok {
			newResult.Expire[id] = res
			delete(rstLocal.Unchanged, id)
		}
	}

	for id, r := range rstLocal.Unchanged {
		newResult.Unchanged[id] = r
	}

	// If the resource declaration and the remote state have not changed, the resource is considered unchanged.
	for id, r := range rstPeer.Unchanged {
		if _, ok := newResult.New[id]; ok {
			continue
		}
		if _, ok := newResult.Diff[id]; ok {
			continue
		}
		if _, ok := newResult.Expire[id]; ok {
			continue
		}

		newResult.Unchanged[id] = r
	}

	return newResult
}

func deployMiddlewareByAdvancedCaller(ctx context.Context, middlewareType string, caller component.AdvancedCaller, res []resources.Resource, state []byte,
) (newState interface{}, err error) {
	// TODO not implemented
	_ = ctx
	_ = middlewareType
	_ = caller
	_ = res
	_ = state
	panic("not implemented")
}

// DeleteMiddleware deletes a middleware based on the provided context, middleware status, and middleware deployment information.
// It returns an error if there was a problem creating the caller or if the caller type is not supported.
func DeleteMiddleware(ctx context.Context, state *v1alpha1.MiddlewareStatus, taskInfo MiddlewareDeploymentInfo) error {
	if state == nil || state.Status == nil {
		return nil
	}

	caller, err := component.GetCaller(taskInfo.callerType, taskInfo.providerInfo)
	if err != nil {
		return fmt.Errorf("unable to create caller: %w", err)
	}

	switch caller.GetImplementationType() {
	case component.CallerImplBasic:
		caller := caller.(component.BasicCaller)
		return deleteMiddlewareByBasicCaller(ctx, caller, state.Status.Raw, taskInfo)
	case component.CallerImplAdvanced:
		caller := caller.(component.AdvancedCaller)
		return deleteMiddlewareByAdvancedCaller(ctx, caller, state)
	default:
		return fmt.Errorf("caller type %s not supported", caller.GetImplementationType())
	}
}

// deleteMiddlewareByBasicCaller deletes middleware based on the basic caller.
// This function takes a context object, a basic caller object, a state byte slice, and middleware deployment information as parameters.
// If the state is empty, it returns nil directly.
// It converts the state byte slice into an array of resource objects.
// It iterates over the resource array and performs a delete operation on each resource.
// If an error occurs while deleting a resource due to resource not found, it returns nil directly.
// If an error occurs while deleting a resource due to other reasons, it returns the error message.
// It returns nil if the deletion is successful, otherwise it indicates deletion failure.
func deleteMiddlewareByBasicCaller(ctx context.Context, caller component.BasicCaller, state []byte, taskInfo MiddlewareDeploymentInfo) error {
	if len(state) == 0 {
		return nil
	}

	resArray, err := ConvertResourcesJsonToResources(state)
	if err != nil {
		return fmt.Errorf("failed to convert resources json to resources: %w", err)
	}

	resMap := map[string]resources.Resource{}
	for _, res := range resArray {
		resMap[res.GetUniqueID()] = res
	}
	deleteList := NewActionListDelete(resMap)

	for _, action := range deleteList {
		resourceTransformer, err := transformer.GetResourceTransformer(taskInfo.providerInfo.Type, caller.GetType(), action.Resource.GetType())
		if err != nil {
			return fmt.Errorf("failed to get resource transformer: %w", err)
		}
		if err := deleteResource(ctx, action.Resource, *resourceTransformer, caller); err != nil {
			if !runtimeerr.IsResourceNotFoundError(err) {
				return fmt.Errorf("failed to delete resource %s: %w", action.Resource.GetUniqueID(), err)
			}
		}
	}
	return nil
}

func deleteMiddlewareByAdvancedCaller(ctx context.Context, caller component.AdvancedCaller, state interface{}) error {
	// TODO not implemented
	_ = ctx
	_ = caller
	_ = state
	panic("not implemented")
}

// compareResources compares two slices of resources and returns the differences between them.
// It takes in the current resources (res) and the last resources (lastRes) as input parameters.
// The function returns a DiffResult struct that contains the new resources, different resources,
// expired resources, and unchanged resources.
func compareResources(res []resources.Resource, lastRes []resources.Resource) (rst resourceComparisonResult) {
	rst = resourceComparisonResult{
		New:       make(map[string]resources.Resource),
		Diff:      make(map[string]resources.Resource),
		Expire:    make(map[string]resources.Resource),
		Unchanged: make(map[string]resources.Resource),
	}

	lastResMap := make(map[string]resources.Resource)
	for _, r := range lastRes {
		lastResMap[r.GetUniqueID()] = r
	}

	for _, r := range res {
		if lastR, ok := lastResMap[r.GetUniqueID()]; ok {
			if isResourceEqual(r, lastR) {
				rst.Unchanged[r.GetUniqueID()] = r
			} else {
				rst.Diff[r.GetUniqueID()] = r
			}
			delete(lastResMap, r.GetUniqueID())
		} else {
			rst.New[r.GetUniqueID()] = r
		}
	}

	for id, r := range lastResMap {
		rst.Expire[id] = r
	}

	return rst
}

// compareResourcesWithPeer compares the given lastResources with its peer resources and returns the differences.
func compareResourcesWithPeer(ctx context.Context, lastResources []resources.Resource, providerType string, caller component.BasicCaller) (rst resourceComparisonResult, err error) {
	// Initialize the DiffResult struct with empty maps for New, Diff, Expire, and Unchanged.
	rst = resourceComparisonResult{
		New:       make(map[string]resources.Resource),
		Diff:      make(map[string]resources.Resource),
		Expire:    make(map[string]resources.Resource),
		Unchanged: make(map[string]resources.Resource),
	}

	// Iterate over each resource in lastResources.
	for _, resource := range lastResources {
		// Get the appropriate resource transformer based on the providerType, caller type, and resource type.
		resourceTransformer, err := transformer.GetResourceTransformer(providerType, caller.GetType(), resource.GetType())
		if err != nil {
			return rst, err
		}

		currentStatus, err := getResource(ctx, resource, *resourceTransformer, caller)
		if err != nil {
			// If the resource is not found, resource need to recreate.
			if runtimeerr.IsResourceNotFoundError(err) {
				rst.New[resource.GetUniqueID()] = resource
				continue
			}
			return rst, fmt.Errorf("failed to get resource: %w", err)
		}

		statusCache := resource.GetStatus()
		if reflect.DeepEqual(currentStatus.Properties, statusCache.Properties) {
			rst.Unchanged[resource.GetUniqueID()] = resource
		} else {
			rst.Diff[resource.GetUniqueID()] = resource
		}
	}

	return rst, nil
}

// resourceComparisonResult represents the comparison result between the resource declaration and the cache or the actual remote environment.
type resourceComparisonResult struct {
	// New represents the resources that need to be added.
	// The key is the unique identifier of the resource.
	New map[string]resources.Resource
	// Diff represents the resources that need to be updated.
	// The key is the unique identifier of the resource.
	Diff map[string]resources.Resource
	// Expire represents the resources that need to be deleted.
	// The key is the unique identifier of the resource.
	Expire map[string]resources.Resource
	// Unchanged represents a list of resources that have not been modified.
	// The key is the unique identifier of the resource.
	Unchanged map[string]resources.Resource
}

// isResourceEqual checks if two resources are equal by comparing their name, type, and resource attributes.
func isResourceEqual(resourceA, resourceB resources.Resource) bool {
	return resourceA.GetName() == resourceB.GetName() &&
		resourceA.GetType() == resourceB.GetType() &&
		reflect.DeepEqual(resourceA.GetAttributes(), resourceB.GetAttributes())
}

// getResource gets the resource from the remote environment.
func getResource(ctx context.Context,
	resource resources.Resource,
	resTransformer transformer.ResourceTransformer,
	caller component.BasicCaller,
) (*resources.ResourceStatus, error) {
	logger.V(1).Info("Get resource", "resource", resource.GetUniqueID())
	request, err := resTransformer.Get.GenerateRequest(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to generate request: %w", err)
	}

	response, err := caller.Post(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("unable to post request: %w", err)
	}

	return resTransformer.Get.ParseResponse(response)
}

// createResource creates the resource in the remote environment.
func createResource(ctx context.Context,
	resource resources.Resource,
	resTransformer transformer.ResourceTransformer,
	caller component.BasicCaller,
) (*resources.ResourceStatus, error) {
	logger.V(1).Info("Create resource", "resource", resource.GetUniqueID())
	request, err := resTransformer.Create.GenerateRequest(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to generate request: %w", err)
	}

	response, err := caller.Post(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("unable to post request: %w", err)
	}

	return resTransformer.Create.ParseResponse(response)
}

// updateResource updates the resource in the remote environment.
func updateResource(ctx context.Context,
	resource resources.Resource,
	resTransformer transformer.ResourceTransformer,
	caller component.BasicCaller,
) (*resources.ResourceStatus, error) {
	// Get resource status to update resource.
	// e.g. update k8s deployment, need resource's resourceVersion.
	currentStatus, err := getResource(ctx, resource, resTransformer, caller)
	if err != nil {
		return nil, err
	}

	resource.SetStatus(*currentStatus)

	logger.V(1).Info("Update resource", "resource", resource.GetUniqueID())
	request, err := resTransformer.Update.GenerateRequest(resource)
	if err != nil {
		return nil, fmt.Errorf("unable to generate request: %w", err)
	}

	response, err := caller.Post(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("unable to post request: %w", err)
	}

	return resTransformer.Update.ParseResponse(response)
}

// deleteResource deletes the resource in the remote environment.
func deleteResource(ctx context.Context, resource resources.Resource, resTransformer transformer.ResourceTransformer, caller component.BasicCaller) (err error) {
	logger.V(1).Info("Delete resource", "resource", resource.GetUniqueID())
	request, err := resTransformer.Delete.GenerateRequest(resource)
	if err != nil {
		return fmt.Errorf("unable to generate request: %w", err)
	}

	response, err := caller.Post(ctx, request)
	if err != nil {
		return fmt.Errorf("unable to post request: %w", err)
	}

	_, err = resTransformer.Delete.ParseResponse(response)
	return err
}

// ResourceActionType represents the action to be performed on the resource.
type ResourceActionType string

const (
	ResourceActionCreate ResourceActionType = "create"
	ResourceActionUpdate ResourceActionType = "update"
	ResourceActionDelete ResourceActionType = "delete"
)

// ResourceAction represents the resource and the action to be performed on it.
type ResourceAction struct {
	Resource resources.Resource
	Action   ResourceActionType
}

type ResourceActions []ResourceAction

// NewActionListCreateOrUpdate creates a new ActionList for creating or updating resources.
func NewActionListCreateOrUpdate(newResources, updateResources map[string]resources.Resource) ResourceActions {
	actionList := ResourceActions{}
	allResources := make(map[string]resources.Resource)

	for id, res := range newResources {
		allResources[id] = res
	}
	for id, res := range updateResources {
		allResources[id] = res
	}

	sortedResources := SortResources(allResources)

	for _, res := range sortedResources {
		if _, ok := newResources[res.GetUniqueID()]; ok {
			actionList = append(actionList, ResourceAction{Resource: res, Action: ResourceActionCreate})
		} else {
			actionList = append(actionList, ResourceAction{Resource: res, Action: ResourceActionUpdate})
		}
	}

	for i, j := 0, len(actionList)-1; i < j; i, j = i+1, j-1 {
		actionList[i], actionList[j] = actionList[j], actionList[i]
	}

	return actionList
}

// NewActionListDelete creates a new ActionList for deleting resources.
func NewActionListDelete(expireResource map[string]resources.Resource) ResourceActions {
	actionList := ResourceActions{}

	sortedResources := SortResources(expireResource)

	for _, res := range sortedResources {
		actionList = append(actionList, ResourceAction{Resource: res, Action: ResourceActionDelete})
	}

	return actionList
}

// SortResources sorts the resources based on their dependencies.
// Perform topological sorting based on the resource dependencies in resources.
// If A depends on B, and C depends on A and B, the sorted result will be B, A, C.
func SortResources(res map[string]resources.Resource) []resources.Resource {
	var result []resources.Resource
	visited := make(map[string]bool)

	var dfs func(resource resources.Resource)
	dfs = func(resource resources.Resource) {
		visited[resource.GetUniqueID()] = true

		for _, dep := range resource.GetDependencies() {
			depResource, exists := res[dep.GetUniqueID()]
			if exists && !visited[dep.GetUniqueID()] {
				dfs(depResource)
			}
		}

		result = append([]resources.Resource{resource}, result...)
	}

	for _, resource := range res {
		if !visited[resource.GetUniqueID()] {
			dfs(resource)
		}
	}

	return result
}

// ConvertResourcesJsonToResources converts the resources in JSON format to a slice of resources.
// TODO: Choose the corresponding structure based on the resource type
func ConvertResourcesJsonToResources(resourcesJson []byte) ([]resources.Resource, error) {
	if len(resourcesJson) == 0 {
		return nil, nil
	}

	resourcesData := &CommonMiddlewareStatus{}

	if err := json.Unmarshal(resourcesJson, resourcesData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	return resourcesData.ResourceStatus, nil
}