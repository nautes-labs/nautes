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

package v1alpha1

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var environmentlog = logf.Log.WithName("environment-resource")

func (r *Environment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-environment,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=environments,verbs=create;update;delete,versions=v1alpha1,name=venvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Environment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateCreate() error {
	environmentlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateUpdate(old runtime.Object) error {
	environmentlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Environment) ValidateDelete() error {
	environmentlog.Info("validate delete", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	return r.IsDeletable(context.TODO(), &ValidateClientFromK8s{Client: k8sClient})
}

func (r *Environment) IsDeletable(ctx context.Context, validateClient ValidateClient) error {
	dependencies, err := r.GetDependencies(ctx, validateClient)
	if err != nil {
		return err
	}

	if len(dependencies) != 0 {
		return fmt.Errorf("environment referenced by [%s], not allowed to be deleted", strings.Join(dependencies, "|"))
	}

	return nil
}

// +kubebuilder:object:generate=false
type GetEnvironmentSubResources func(ctx context.Context, validateClient ValidateClient, productName, envName string) ([]string, error)

// GetEnvironmentSubResourceFunctions stores a set of methods for obtaining a list of environment sub-resources.
// When the environment checks whether it is being referenced, it will loop through the method list here.
var GetEnvironmentSubResourceFunctions = []GetEnvironmentSubResources{}

func (r *Environment) GetDependencies(ctx context.Context, validateClient ValidateClient) ([]string, error) {
	subResources := []string{}
	for _, fn := range GetEnvironmentSubResourceFunctions {
		resources, err := fn(ctx, validateClient, r.Namespace, r.Name)
		if err != nil {
			return nil, err
		}
		subResources = append(subResources, resources...)
	}
	return subResources, nil
}
