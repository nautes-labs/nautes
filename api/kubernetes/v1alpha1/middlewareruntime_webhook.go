/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/nautes-labs/nautes/pkg/middlewareinfo"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var middlewareruntimelog = logf.Log.WithName("middlewareruntime-resource")

func (r *MiddlewareRuntime) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-middlewareruntime,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=middlewareruntimes,verbs=create;update,versions=v1alpha1,name=vmiddlewareruntime.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MiddlewareRuntime{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MiddlewareRuntime) ValidateCreate() error {
	middlewareruntimelog.Info("validate create", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	validateClient := NewValidateClientFromK8s(k8sClient)
	return r.Validate(context.TODO(), validateClient, k8sClient)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MiddlewareRuntime) ValidateUpdate(old runtime.Object) error {
	middlewareruntimelog.Info("validate update", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	validateClient := NewValidateClientFromK8s(k8sClient)
	return r.Validate(context.TODO(), validateClient, k8sClient)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MiddlewareRuntime) ValidateDelete() error {
	middlewareruntimelog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *MiddlewareRuntime) Validate(ctx context.Context, validateClient ValidateClient, k8sClient client.Client) error {
	if err := r.StaticCheck(); err != nil {
		return fmt.Errorf("static check failed: %w", err)
	}

	nautesCfg, err := nautesconfigs.NewNautesConfigFromFile()
	if err != nil {
		return fmt.Errorf("failed to get nautes config: %w", err)
	}
	nautesNamespace := nautesCfg.Nautes.Namespace

	env, err := validateClient.GetEnvironment(ctx, r.Namespace, r.Spec.Destination.Environment)
	if err != nil {
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Get the cluster if the environment type is cluster
	var cluster *Cluster
	if env.Spec.EnvType == EnvironmentTypeCluster {
		cluster = &Cluster{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Name: env.Spec.Cluster, Namespace: nautesNamespace}, cluster); err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}
		if cluster.Spec.WorkerType != ClusterWorkTypeMiddleware {
			return fmt.Errorf("cluster %s is not a middleware cluster", cluster.Name)
		}
	}

	// Check if the namespace is used by other runtime
	usedSpaces, _ := getSpacesIsUsedByOtherRuntime(r.Namespace, r.GetNamespaces(), env.Spec.EnvType, cluster)
	if len(usedSpaces) > 0 {
		return fmt.Errorf("namespaces %v are used by other runtime", usedSpaces)
	}

	// Check if the middleware can be deployed to the cluster
	for _, middleware := range r.GetMiddlewares() {
		var providerType string
		if env.Spec.EnvType == EnvironmentTypeCluster {
			if cluster.Spec.MiddlewareProvider == nil {
				return fmt.Errorf("cluster %s does not have a middleware provider", cluster.Name)
			}
			providerType = cluster.Spec.MiddlewareProvider.Type
		} else {
			providerType = env.Spec.EnvType
		}

		if err := checkImplementationCanBeUsed(providerType, middleware, middlewareinfo.MiddlewareMetadata, cluster); err != nil {
			return fmt.Errorf("failed to validate middleware %s: %w", middleware.Name, err)
		}
	}

	return nil
}

func checkImplementationCanBeUsed(providerType string, middleware Middleware, metadataList middlewareinfo.Middlewares, cluster *Cluster) error {
	// Check whether the middleware is valid
	middlewareMetadata, ok := metadataList[middleware.Type]
	if !ok {
		return fmt.Errorf("middleware %s does not found", middleware.Type)
	}
	providerInfo, ok := middlewareMetadata.Providers[providerType]
	if !ok {
		return fmt.Errorf("provider %s does not support middleware type %s", providerType, middleware.Type)
	}

	if middleware.Implementation == "" {
		middleware.Implementation = providerInfo.DefaultImplementation
	}

	implementationInfo, ok := providerInfo.Implementations[middleware.Implementation]
	if !ok {
		return fmt.Errorf("provider %s does not support implementation %s for middleware type %s", providerType, middleware.Implementation, middleware.Type)
	}
	availableVars := sets.New(implementationInfo.AvailableVars...)
	for k := range middleware.CommonMiddlewareInfo {
		if !availableVars.Has(k) {
			return fmt.Errorf("implementation %s does not support var %s for middleware type %s", middleware.Implementation, k, middleware.Type)
		}
	}

	// Check if the middleware is supported by the cluster
	if cluster != nil {
		if cluster.Spec.MiddlewareProvider == nil {
			return fmt.Errorf("cluster %s does not have a middleware provider", cluster.Name)
		}
		if cluster.Spec.MiddlewareProvider.Type != providerType {
			return fmt.Errorf("cluster %s does not support provider %s", cluster.Name, providerType)
		}
		middlewareImplementations, ok := cluster.Spec.MiddlewareProvider.GetMiddlewares()[middleware.Type]
		if !ok {
			return fmt.Errorf("cluster %s can not deploy middleware %s", cluster.Name, middleware.Type)
		}
		hasImplementation := false
		for _, impl := range middlewareImplementations {
			if impl.Type == middleware.Implementation {
				hasImplementation = true
				break
			}
		}
		if !hasImplementation {
			return fmt.Errorf("cluster %s can not deploy middleware %s with implementation %s", cluster.Name, middleware.Type, middleware.Implementation)
		}
	}

	return nil
}

func getSpacesIsUsedByOtherRuntime(product string, spacesRequired []string, envType string, cluster *Cluster) ([]string, error) {
	usedSpaces := []string{}
	if envType == EnvironmentTypeCluster {
		if cluster == nil {
			return usedSpaces, fmt.Errorf("cluster is nil")
		}

		if cluster.Status.ResourceUsage == nil {
			return usedSpaces, nil
		}

		namespacesInCluster := sets.New(cluster.Status.ResourceUsage.GetNamespaces()...)
		productUsedNamespaces := sets.New(cluster.Status.ResourceUsage.GetNamespaces(WithProductNameForResourceUsage(product))...)
		for _, space := range spacesRequired {
			if namespacesInCluster.Has(space) && !productUsedNamespaces.Has(space) {
				usedSpaces = append(usedSpaces, space)
			}
		}
	}
	return usedSpaces, nil
}

// +kubebuilder:object:generate=false
type MiddlewareStaticCheckFunc func(Middleware) error

// +kubebuilder:object:generate=false
var MiddlewareStaticCheckFunctions = map[string]MiddlewareStaticCheckFunc{}

func (r *MiddlewareRuntime) StaticCheck() error {
	for _, middleware := range r.GetMiddlewares() {
		if middleware.InitAccessInfo != nil {
			if middleware.InitAccessInfo.GetType() == MiddlewareAccessInfoTypeNotSpecified {
				return fmt.Errorf("unknown middleware access info type for middleware %s", middleware.Name)
			}
		}

		if fn, ok := MiddlewareStaticCheckFunctions[middleware.Type]; ok {
			if err := fn(middleware); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	MiddlewareStaticCheckFunctions["redis"] = RedisStaticCheck
}

func RedisStaticCheck(m Middleware) error {
	if m.Redis == nil {
		return fmt.Errorf("empty redis deployment for middleware %s", m.Name)
	}

	if m.Redis.NodeNumber < 1 {
		return fmt.Errorf("invalid node number for middleware %s", m.Name)
	}

	if ValidateSizeFormat(m.Redis.Storage.Size) {
		return fmt.Errorf("invalid storage format for middleware %s", m.Name)
	}

	return nil
}

var supportedSizeUnits = sets.New[string]("B", "KB", "MB", "GB", "TB")

func ValidateSizeFormat(size string) bool {
	// Define the regular expression pattern for the size format
	pattern := `^\d+\s*(B|KB|MB|GB|TB)?$`

	// Create a regular expression object
	regex := regexp.MustCompile(pattern)

	// Check if the size matches the pattern
	if !regex.MatchString(size) {
		return false
	}

	// Extract the size and unit from the string
	match := regex.FindStringSubmatch(size)
	sizeValue, _ := strconv.Atoi(match[1])
	if sizeValue <= 0 {
		return false
	}

	unit := match[2]
	if supportedSizeUnits.Has(unit) {
		return false
	}

	return true
}
