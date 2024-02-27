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
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/componentutils"
	"github.com/nautes-labs/nautes/pkg/kubeconvert"
	"github.com/nautes-labs/nautes/pkg/middlewareinfo"
	"github.com/nautes-labs/nautes/pkg/middlewaresecret"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	performer "github.com/nautes-labs/nautes/app/runtime-operator/pkg/performer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("middleware-runtime-performer")

// MiddlewareRuntimePerformer is the executor for middleware runtime, used to perform deployment operations for middleware runtimes.
type MiddlewareRuntimePerformer struct {
	// components is the list of components available for deployment.
	components *component.ComponentList
	// product is the product to which the runtime belongs.
	product v1alpha1.Product
	// runtime is the runtime to be deployed.
	runtime v1alpha1.MiddlewareRuntime
	// environment is the environment information where the runtime resides.
	environment v1alpha1.Environment
	// cluster is the cluster information for deploying the runtime.
	// If the environment type is not Cluster, cluster is nil.
	cluster *v1alpha1.Cluster
	// currentStatus is the current status of the runtime.
	currentStatus *v1alpha1.MiddlewareRuntimeStatus
	// lastStatus is the status after the last deployment.
	lastStatus *v1alpha1.MiddlewareRuntimeStatus
	// tenantK8sClient is the k8s client for the Nautes tenant cluster.
	tenantK8sClient client.Client
	// providerInfo is the information about the service provider.
	providerInfo component.ProviderInfo
	nautesConfig configs.Config
}

// NewPerformer creates a new instance of the MiddlewareRuntimePerformer struct,
// which implements the performer.TaskPerformer interface. It initializes the
// performer with the provided PerformerInitInfos.
//
// Parameters:
//   - initInfo: The PerformerInitInfos containing the necessary information to
//     initialize the performer.
//
// Returns:
// - performer.TaskPerformer: The initialized performer.
// - error: An error if the initialization fails.
func NewPerformer(initInfo performer.PerformerInitInfos) (performer.TaskPerformer, error) {
	runtimePerformer := &MiddlewareRuntimePerformer{
		components:      initInfo.Components,
		tenantK8sClient: initInfo.TenantK8sClient,
		nautesConfig:    initInfo.NautesConfig,
	}

	// Get the runtime from the initInfo.
	runtime, ok := initInfo.Runtime.(*v1alpha1.MiddlewareRuntime)
	if !ok {
		return nil, fmt.Errorf("runtime is not a MiddlewareRuntime")
	}
	runtimePerformer.runtime = *runtime

	// Get the product from the initInfo.
	product, err := initInfo.ComponentInitInfo.NautesResourceSnapshot.GetProduct(runtime.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get product %s failed: %w", runtime.Namespace, err)
	}
	runtimePerformer.product = *product

	// Get the last status from the initInfo.
	runtimePerformer.lastStatus = runtime.Status.DeepCopy()
	currentStatus := runtimePerformer.lastStatus.DeepCopy()
	if currentStatus == nil {
		currentStatus = &v1alpha1.MiddlewareRuntimeStatus{}
	}

	// Get the environment and cluster from the initInfo.
	env, err := initInfo.ComponentInitInfo.NautesResourceSnapshot.GetEnvironment(runtime.Spec.Destination.Environment)
	if err != nil {
		return nil, fmt.Errorf("get environment %s failed: %w", runtime.Spec.Destination.Environment, err)
	}
	runtimePerformer.environment = *env
	currentStatus.Environment = *env
	currentStatus.Environment.Status = v1alpha1.EnvironmentStatus{}

	var cluster *v1alpha1.Cluster
	if env.Spec.EnvType == v1alpha1.EnvironmentTypeCluster {
		logger.V(1).Info("Get cluster", "cluster", env.Spec.Cluster)
		cluster, err = initInfo.ComponentInitInfo.NautesResourceSnapshot.GetCluster(env.Spec.Cluster)
		if err != nil {
			return nil, fmt.Errorf("get cluster %s failed: %w", env.Spec.Cluster, err)
		}
		runtimePerformer.cluster = cluster
		currentStatus.Cluster = cluster.DeepCopy()
		currentStatus.Cluster.Status = v1alpha1.ClusterStatus{}
	}

	// Create the provider info.
	err = runtimePerformer.CreateProviderInfo(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("create provider info failed: %w", err)
	}

	currentStatus.AccessInfoName = runtime.Spec.AccessInfoName

	runtimePerformer.currentStatus = currentStatus
	if SecretIndexDirectory == nil {
		indexes, err := middlewaresecret.NewSecretIndexDirectory(context.TODO(), types.NamespacedName{
			Name:      initInfo.NautesConfig.Nautes.ConfigMapNameMiddlewareSecretIndex,
			Namespace: initInfo.NautesConfig.Nautes.Namespace,
		},
			initInfo.TenantK8sClient)
		if err != nil {
			return nil, fmt.Errorf("new secret indexes failed: %w", err)
		}

		SecretIndexDirectory = indexes
	}

	printTaskInfo(runtimePerformer)

	return runtimePerformer, nil
}

func printTaskInfo(m *MiddlewareRuntimePerformer) {
	logger := logger.WithValues("product", m.product.Name, "runtime", m.runtime.Name, "environment", m.environment.Name)
	clusterName := ""
	if m.cluster != nil {
		clusterName = m.cluster.Name
	}
	logger.V(1).Info("TaskInfo", "environment", m.environment.Name, "cluster", clusterName)
}

func (m *MiddlewareRuntimePerformer) Deploy(ctx context.Context) (interface{}, error) {
	err := m.deploy(ctx)
	if err != nil {
		return m.currentStatus, fmt.Errorf("deploy middleware runtime failed: %w", err)
	}
	return m.currentStatus, nil
}

// deploy is a method of the MiddlewareRuntimePerformer struct that deploys the middleware runtime.
// It performs the following steps:
// 1. If the environment points to Cluster resources, it updates the cluster resource usage and creates or updates the product namespaces.
// 2. It loops through the middleware and deploys each middleware.
// 3. It deletes expired namespaces.
// 4. It updates the current status of spaces and entry points.
// 5. It updates the cluster resource usage.
// The method returns an error if any of the steps fail.
func (m *MiddlewareRuntimePerformer) deploy(ctx context.Context) error {
	var newClusterResourceUsage *v1alpha1.ResourceUsage
	var expiredNamespaces []string

	// 1. If the environment points to Cluster resources, it updates the cluster resource usage and creates or updates the product namespaces.
	if m.environment.Spec.EnvType == v1alpha1.EnvironmentTypeCluster {
		var err error

		// If the cluster resource usage is nil, it creates a new one.
		if m.cluster.Status.ResourceUsage != nil {
			newClusterResourceUsage = m.cluster.Status.ResourceUsage.DeepCopy()
		} else {
			newClusterResourceUsage = &v1alpha1.ResourceUsage{}
		}

		// Adds or updates the runtime usage in the cluster resource usage.
		_ = newClusterResourceUsage.AddOrUpdateRuntimeUsage(m.product.Name, v1alpha1.RuntimeUsage{
			Name:        m.runtime.Name,
			AccountName: m.runtime.Spec.Account,
			Namespaces:  m.runtime.GetNamespaces(),
		})

		newProductNameSpaces := newClusterResourceUsage.GetNamespaces(
			v1alpha1.WithProductNameForResourceUsage(m.product.Name),
		)

		// Creates or updates the product and namespaces in cluster.
		expiredNamespaces, err = componentutils.CreateOrUpdateProduct(ctx, m.components.MultiTenant, m.product.Name, newProductNameSpaces)
		if err != nil {
			return fmt.Errorf("create or update product %s failed: %w", m.product.Name, err)
		}

		// Create network entry points for the services based on the entrypoint information.
		// TODO
	}

	// 2. It loops through the middleware and deploys each middleware.
	middlewares, err := FillMissingFieldsInMiddleware(m.runtime.GetMiddlewares(), m.GetDefaultVars())
	if err != nil {
		return fmt.Errorf("fill runtime middlewares failed: %w", err)
	}

	if err := m.SyncMiddlewares(ctx, middlewares, m.lastStatus.Middlewares); err != nil {
		return fmt.Errorf("sync middlewares failed: %w", err)
	}

	// 3. It deletes expired namespaces.
	if err := componentutils.DeleteNamespaces(ctx, m.components.MultiTenant, m.product.Name, expiredNamespaces); err != nil {
		return fmt.Errorf("delete namespaces failed: %w", err)
	}

	// 4. It updates the current status of spaces and entry points.
	m.currentStatus.Spaces = m.runtime.GetNamespaces()
	m.currentStatus.EntryPoints = m.runtime.GetEntrypoints()

	// 5. It updates the cluster resource usage.
	if err := m.UpdateClusterResourceUsage(ctx, newClusterResourceUsage); err != nil {
		return fmt.Errorf("update cluster resource usage failed: %w", err)
	}

	return nil
}

func (m *MiddlewareRuntimePerformer) Delete(ctx context.Context) (interface{}, error) {
	err := m.delete(ctx)
	if err != nil {
		return m.currentStatus, fmt.Errorf("delete middleware runtime failed: %w", err)
	}
	return nil, nil
}

// delete deletes the middleware runtime.
// It synchronizes the middlewares, updates or deletes the product, and updates the cluster resource usage.
// If the environment type is cluster, it removes the runtime usage from the cluster resource usage.
// It returns an error if any of the operations fail.
func (m *MiddlewareRuntimePerformer) delete(ctx context.Context) error {

	// Synchronize the middlewares.
	if err := m.SyncMiddlewares(ctx, nil, m.lastStatus.Middlewares); err != nil {
		return fmt.Errorf("sync middlewares failed: %w", err)
	}

	// Update or delete the product.
	if m.environment.Spec.EnvType == v1alpha1.EnvironmentTypeCluster {
		var clusterResourcesUsage *v1alpha1.ResourceUsage
		var productNamespaces []string

		// Remove the runtime usage from the cluster resource usage.
		clusterResourcesUsage = m.cluster.Status.ResourceUsage
		if clusterResourcesUsage != nil {
			clusterResourcesUsage.RemoveRuntimeUsage(m.product.Name, m.runtime.Name)
			productNamespaces = clusterResourcesUsage.GetNamespaces(
				v1alpha1.WithProductNameForResourceUsage(m.product.Name),
			)
		}

		// Update or delete the product in cluster.
		if err := componentutils.DeleteOrUpdateProduct(ctx, m.components.MultiTenant, m.product.Name, productNamespaces); err != nil {
			return fmt.Errorf("update or delete product %s failed: %w", m.product.Name, err)
		}

		// Update the cluster resource usage.
		if err := m.UpdateClusterResourceUsage(ctx, clusterResourcesUsage); err != nil {
			return fmt.Errorf("update cluster resource usage failed: %w", err)
		}
	}

	return nil
}

// CreateProviderInfo creates the provider information based on the environment type.
// It returns an error if there is any failure during the creation process.
func (m *MiddlewareRuntimePerformer) CreateProviderInfo(ctx context.Context) error {
	switch m.environment.Spec.EnvType {
	case v1alpha1.EnvironmentTypeCluster:
		info, err := createProviderInfoFromCluster(ctx, m.components.SecretManagement, m.cluster.Spec.MiddlewareProvider.Type)
		if err != nil {
			return fmt.Errorf("create cluster provider info failed: %w", err)
		}
		m.providerInfo = *info
		return nil
	default:
		m.providerInfo = component.ProviderInfo{}
		return nil
	}
}

// createProviderInfoFromCluster creates the provider information from the cluster.
func createProviderInfoFromCluster(ctx context.Context, secMgr component.SecretManagement, providerType string) (*component.ProviderInfo, error) {
	info := &component.ProviderInfo{
		Type: providerType,
		URL:  "",
		TLS:  &component.TLSInfo{},
		Auth: &component.ProviderAuthInfo{},
	}

	connectInfo, err := secMgr.GetAccessInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connect info failed: %w", err)
	}

	restConfig, err := kubeconvert.ConvertStringToRestConfig(connectInfo)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	keypair := component.AuthInfoKeypair{
		Key:  restConfig.KeyData,
		Cert: restConfig.CertData,
	}

	authInfo, err := component.NewAuthInfo(keypair)
	if err != nil {
		return nil, fmt.Errorf("create auth info failed: %w", err)
	}

	info.URL = restConfig.Host
	info.TLS = &component.TLSInfo{
		CABundle: string(restConfig.CAData),
	}
	info.Auth = authInfo

	return info, nil
}

func (m *MiddlewareRuntimePerformer) UpdateClusterResourceUsage(ctx context.Context, newUsage *v1alpha1.ResourceUsage) error {
	if m.cluster == nil {
		return nil
	}

	cluster := m.cluster.DeepCopy()
	cluster.Status.ResourceUsage = newUsage
	err := m.tenantK8sClient.Status().Update(ctx, cluster)
	if err != nil {
		return err
	}

	return nil
}

// CompareMiddlewares compares two slices of v1alpha1.Middleware and returns the added, updated, and deleted middlewares.
func CompareMiddlewares(old, new []v1alpha1.Middleware) (added, updated, deleted []v1alpha1.Middleware) {
	// Create a map to store the old middlewares
	oldMap := make(map[string]v1alpha1.Middleware)
	for _, middleware := range old {
		oldMap[middleware.GetUniqueID()] = middleware
	}

	// Create a map to store the new middlewares
	newMap := make(map[string]v1alpha1.Middleware)
	for _, middleware := range new {
		newMap[middleware.GetUniqueID()] = middleware
	}

	// Compare the new middlewares with the old middlewares
	for _, middleware := range new {
		if _, ok := oldMap[middleware.GetUniqueID()]; ok {
			updated = append(updated, middleware)
		} else {
			added = append(added, middleware)
		}
	}

	// Check for deleted middlewares
	for _, middleware := range old {
		if _, ok := newMap[middleware.GetUniqueID()]; !ok {
			deleted = append(deleted, middleware)
		}
	}

	return
}

// SyncMiddlewares synchronizes the state of middlewares in the system.
// It takes a context, a list of middlewares, and a list of middleware statuses as input.
// The function compares the old middlewares with the new middlewares and performs the necessary actions (add, update, delete) to synchronize them.
// If any errors occur during the synchronization process, the function returns an error.
func (m *MiddlewareRuntimePerformer) SyncMiddlewares(ctx context.Context, middlewares []v1alpha1.Middleware, statues []v1alpha1.MiddlewareStatus) error {
	errs := []error{}
	oldMiddlewares := make([]v1alpha1.Middleware, len(statues))
	statusMap := map[string]v1alpha1.MiddlewareStatus{}
	oldStatusMap := map[string]v1alpha1.MiddlewareStatus{}
	for i, status := range statues {
		statusMap[status.Middleware.GetUniqueID()] = status
		oldStatusMap[status.Middleware.GetUniqueID()] = status
		oldMiddlewares[i] = status.Middleware
	}

	// Compare the old middlewares with the new middlewares
	added, updated, deleted := CompareMiddlewares(oldMiddlewares, middlewares)

	for _, middleware := range added {
		uniqueID := middleware.GetUniqueID()
		deploymentInfo, err := NewMiddlewareDeploymentInfo(m.providerInfo, middleware)
		if err != nil {
			errs = append(errs, fmt.Errorf("create middleware %s deployment info failed: %w", uniqueID, err))
			continue
		}
		runtimeInfo := m.GetRuntimeInfo()

		logger.V(1).Info("Deploy middleware", "uniqueID", uniqueID)
		status, err := DeployMiddleware(ctx, middleware, nil, *deploymentInfo, runtimeInfo)
		if err != nil {
			errs = append(errs, fmt.Errorf("deploy middleware %s failed: %w", uniqueID, err))
			continue
		}
		statusMap[uniqueID] = *status
	}

	for _, middleware := range updated {
		uniqueID := middleware.GetUniqueID()
		deploymentInfo, err := NewMiddlewareDeploymentInfo(m.providerInfo, middleware)
		if err != nil {
			errs = append(errs, fmt.Errorf("create middleware %s deployment info failed: %w", uniqueID, err))
			continue
		}
		runtimeInfo := m.GetRuntimeInfo()

		lastStatus := statusMap[middleware.GetUniqueID()]
		logger.V(1).Info("Update middleware", "uniqueID", uniqueID)
		status, err := DeployMiddleware(ctx, middleware, &lastStatus, *deploymentInfo, runtimeInfo)
		if err != nil {
			errs = append(errs, fmt.Errorf("update middleware %s failed: %w", uniqueID, err))
			continue
		}
		statusMap[uniqueID] = *status
	}

	for _, middleware := range deleted {
		uniqueID := middleware.GetUniqueID()
		deploymentInfo, err := NewMiddlewareDeploymentInfo(m.providerInfo, middleware)
		if err != nil {
			errs = append(errs, fmt.Errorf("create middleware %s deployment info failed: %w", uniqueID, err))
			continue
		}
		status := oldStatusMap[middleware.GetUniqueID()]
		logger.V(1).Info("Delete middleware", "uniqueID", uniqueID)
		err = DeleteMiddleware(ctx, &status, *deploymentInfo)
		if err != nil {
			errs = append(errs, fmt.Errorf("delete middleware %s failed: %w", uniqueID, err))
			continue
		}
		delete(statusMap, uniqueID)
	}

	// Update the status
	newStatuses := make([]v1alpha1.MiddlewareStatus, 0, len(statusMap))
	for _, status := range statusMap {
		newStatuses = append(newStatuses, status)
	}

	m.currentStatus.Middlewares = removeSensitiveDataInMiddleware(newStatuses)

	if len(errs) != 0 {
		return fmt.Errorf("sync middlewares failed: %v", errs)
	}

	return nil
}

func (m *MiddlewareRuntimePerformer) GetRuntimeInfo() RuntimeInfo {
	var clusterName string
	if m.cluster != nil {
		clusterName = m.cluster.Name
	}
	return RuntimeInfo{
		Name:            m.runtime.GetName(),
		Product:         m.product.Name,
		Environment:     m.environment.Name,
		Cluster:         clusterName,
		NautesNamespace: m.nautesConfig.Nautes.Namespace,
	}
}

func (m *MiddlewareRuntimePerformer) GetDefaultVars() map[string]string {
	defaultVars := make(map[string]string)
	defaultVars[defaultVarKeySpace] = m.runtime.GetName()
	if m.runtime.Spec.Destination.Space != "" {
		defaultVars[defaultVarKeySpace] = m.runtime.Spec.Destination.Space
	}

	if m.cluster != nil {
		defaultVars[defaultVarKeyDomain] = m.cluster.Spec.PrimaryDomain
	}

	return defaultVars
}

const defaultImplementation = "default"
const (
	defaultVarKeySpace  = "space"
	defaultVarKeyDomain = "domain"
	defaultVarKeyPort   = "port"
)

// FillMissingFieldsInMiddleware fills the missing key in the middleware.
func FillMissingFieldsInMiddleware(middlewares []v1alpha1.Middleware, defaultVars map[string]string) ([]v1alpha1.Middleware, error) {
	for i, middleware := range middlewares {
		if middleware.Space == "" {
			middlewares[i].Space = defaultVars[defaultVarKeySpace]
		}

		if middleware.Implementation == "" {
			middlewares[i].Implementation = defaultImplementation
		}

		if middleware.Entrypoint != nil {
			if middleware.Entrypoint.Domain == "" {
				middlewares[i].Entrypoint.Domain = defaultVars[defaultVarKeyDomain]
			}
		}

		needAuthInfo, authType, err := checkMiddlewareNeedNewAuthInfo(middleware)
		if err != nil {
			return nil, fmt.Errorf("check middleware %s need new auth info failed: %w", middleware.Type, err)
		}

		if needAuthInfo {
			accessInfo, err := createNewSecret(middleware, authType)
			if err != nil {
				return nil, fmt.Errorf("create new secret for middleware %s failed: %w", middleware.Type, err)
			}
			middlewares[i].InitAccessInfo = accessInfo
		}
	}

	return middlewares, nil
}

func init() {
	metaData, err := middlewareinfo.NewMiddlewares()
	if err != nil {
		panic(fmt.Errorf("failed to load middleware metadata: %w", err))
	}
	MiddlewareMetaData = *metaData
}

var MiddlewareMetaData middlewareinfo.Middlewares

func checkMiddlewareNeedNewAuthInfo(middleware v1alpha1.Middleware) (needAuthInfo bool, authType string, err error) {
	metadata, ok := MiddlewareMetaData[middleware.Type]
	if !ok {
		return false, "", fmt.Errorf("middleware %s not found", middleware.Type)
	}

	if middleware.InitAccessInfo == nil {
		if metadata.DefaultAuthType == "" {
			return false, "", nil
		}
		return true, metadata.DefaultAuthType, nil
	}

	switch middleware.InitAccessInfo.GetType() {
	case v1alpha1.MiddlewareAccessInfoTypeNotSpecified:
		return true, metadata.DefaultAuthType, nil
	case v1alpha1.MiddlewareAccessInfoTypeUserPassword:
		if middleware.InitAccessInfo.UserPassword.Username == "" ||
			middleware.InitAccessInfo.UserPassword.Password == "" {
			return true, v1alpha1.MiddlewareAccessInfoTypeUserPassword, nil
		}
	default:
		return false, "", fmt.Errorf("unknown access info type %s", middleware.InitAccessInfo.GetType())
	}
	return
}

func createNewSecret(middleware v1alpha1.Middleware, authType string) (*v1alpha1.MiddlewareInitAccessInfo, error) {
	accessInfo := &v1alpha1.MiddlewareInitAccessInfo{}
	switch authType {
	case v1alpha1.MiddlewareAccessInfoTypeUserPassword:
		password, err := generatePassword(8)
		if err != nil {
			return nil, fmt.Errorf("generate password failed: %w", err)
		}
		userName := ""
		if middleware.InitAccessInfo != nil && middleware.InitAccessInfo.UserPassword != nil {
			userName = middleware.InitAccessInfo.UserPassword.Username
		}
		accessInfo.UserPassword = &v1alpha1.MiddlewareInitAccessInfoUserPassword{
			Username: userName,
			Password: password,
		}
	}

	return accessInfo, nil
}

func generatePassword(length int) (string, error) {
	const (
		lowerLetters = "abcdefghijklmnopqrstuvwxyz"
		upperLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits       = "0123456789"
		specials     = "!@#$%^&*()-_=+,.?/:;{}[]`~"
		allChars     = lowerLetters + upperLetters + digits + specials
	)

	if length < 4 {
		return "", fmt.Errorf("password length must be at least 4")
	}

	charGroups := []string{lowerLetters, upperLetters, digits, specials}
	result := make([]byte, length)

	// Ensure the password contains at least one character from each group
	for i, group := range charGroups {
		val, err := rand.Int(rand.Reader, big.NewInt(int64(len(group))))
		if err != nil {
			return "", err
		}
		result[i] = group[val.Int64()]
	}

	// Fill the rest of the password with random characters from all groups
	for i := len(charGroups); i < length; i++ {
		val, err := rand.Int(rand.Reader, big.NewInt(int64(len(allChars))))
		if err != nil {
			return "", err
		}
		result[i] = allChars[val.Int64()]
	}

	return string(result), nil
}

func removeSensitiveDataInMiddleware(status []v1alpha1.MiddlewareStatus) []v1alpha1.MiddlewareStatus {
	if status == nil {
		return nil
	}

	for i := range status {
		status[i].Middleware.InitAccessInfo = nil
	}

	return status
}
