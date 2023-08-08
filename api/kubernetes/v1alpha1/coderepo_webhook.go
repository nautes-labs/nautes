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

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes;products;coderepoproviders,verbs=get;list

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var coderepolog = logf.Log.WithName("coderepo-resource")
var manager ctrl.Manager

func (r *CodeRepo) SetupWebhookWithManager(mgr ctrl.Manager) error {
	manager = mgr
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &CodeRepo{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CodeRepo) Default() {

	coderepolog.Info("default", "name", r.Name, "url", r.Spec.URL)
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-coderepo,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=coderepoes,verbs=create;update;delete,versions=v1alpha1,name=vcoderepo.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CodeRepo{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateCreate() error {
	coderepolog.Info("validate create", "name", r.Name)

	err := r.Validate()
	if err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateUpdate(old runtime.Object) error {
	coderepolog.Info("validate update", "name", r.Name)

	err := r.Validate()
	if err != nil {
		return err
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CodeRepo) ValidateDelete() error {
	coderepolog.Info("validate delete", "name", r.Name)
	k8sClient, err := getClient()
	if err != nil {
		return err
	}
	return r.IsDeletable(context.TODO(), NewValidateClientFromK8s(k8sClient))
}

// GetURL Get the address of the repository address corresponding to CodeRepo
// If the CodeRepo spec URL is not empty, return it directly,
// Otherwise, the repository address will be spliced through the information of CodeRepoProvider and Product resources.
func (r *CodeRepo) GetURL(spec CodeRepoSpec) (string, error) {
	if spec.URL != "" {
		return spec.URL, nil
	} else if spec.URL == "" && spec.Product != "" && spec.CodeRepoProvider != "" {
		client, err := getClient()
		if err != nil {
			return "", err
		}

		configs, err := nautesconfigs.NewNautesConfigFromFile()
		if err != nil {
			return "", err
		}

		product := &Product{}
		err = client.Get(context.Background(), types.NamespacedName{
			Namespace: configs.Nautes.Namespace,
			Name:      r.Spec.Product,
		}, product)
		if err != nil {
			return "", err
		}

		provider := &CodeRepoProvider{}
		err = client.Get(context.Background(), types.NamespacedName{
			Namespace: configs.Nautes.Namespace,
			Name:      r.Spec.CodeRepoProvider,
		}, provider)
		if err != nil {
			return "", err
		}

		if provider.Spec.SSHAddress == "" ||
			product.Spec.Name == "" {
			return "", fmt.Errorf("ssh base address of the codeRepoProvider %s or name of the product %s is empty", r.Spec.CodeRepoProvider, product.Spec.Name)
		}

		url := fmt.Sprintf("%s/%s/%s.git", provider.Spec.SSHAddress, product.Spec.Name, spec.RepoName)

		return url, nil
	}

	return "", errors.New("the resource is not avaiable. if the url does not exist, it should contain coderepo provider and product resources")
}

func (r *CodeRepo) Validate() error {
	url, err := r.GetURL(r.Spec)
	if err != nil {
		return err
	}
	if url == "" {
		return fmt.Errorf("failed to get repository address the CodeRepo %s, the url value is empty", r.Name)
	}

	matched, err := regexp.MatchString(`^(ssh:\/\/)?git@(?:((?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}|[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(?:\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+):(?:(\d{1,5})\/)?([\w-]+)\/([\w.-]+)\.git$`, url)
	if err != nil {
		return err
	}

	if !matched {
		return fmt.Errorf("the url %s is illegal", url)
	}

	return nil
}

func (r *CodeRepo) IsDeletable(ctx context.Context, client ValidateClient) error {
	dependencies, err := r.GetDependencies(ctx, client)
	if err != nil {
		return err
	}
	if len(dependencies) != 0 {
		return fmt.Errorf("coderepo referenced by [%s], not allowed to be deleted", strings.Join(dependencies, "|"))
	}

	return nil
}

// +kubebuilder:object:generate=false
type GetCodeRepoSubResources func(ctx context.Context, client ValidateClient, CodeRepoName string) ([]string, error)

var GetCoderepoSubResourceFunctions = []GetCodeRepoSubResources{}

func (r *CodeRepo) GetDependencies(ctx context.Context, client ValidateClient) ([]string, error) {
	subResources := []string{}
	for _, fn := range GetCoderepoSubResourceFunctions {
		resources, err := fn(ctx, client, r.Name)
		if err != nil {
			return nil, fmt.Errorf("get dependent resources failed: %w", err)
		}
		subResources = append(subResources, resources...)
	}

	return subResources, nil
}

func getCodeRepoName(codeRepo *CodeRepo) string {
	_, exist := os.LookupEnv(EnvUseRealName)
	if exist {
		return codeRepo.Spec.RepoName
	}

	return codeRepo.Name
}
