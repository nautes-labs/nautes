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
	"reflect"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type productPipelineSyncerArgoCD struct {
	secClient               interfaces.SecretClient
	k8sClient               client.Client
	task                    interfaces.RuntimeSyncTask
	productCodeRepo         nautescrd.CodeRepo
	productCodeRepoProvider nautescrd.CodeRepoProvider
	productLabel            map[string]string
}

func NewProductPipelineSyncerArgoCD(k8sClient client.Client, secClient interfaces.SecretClient, task interfaces.RuntimeSyncTask, opts destClusterOptions) ProductPipelineSyncer {
	return &productPipelineSyncerArgoCD{
		secClient:               secClient,
		k8sClient:               k8sClient,
		task:                    task,
		productCodeRepo:         opts.product.CodeRepo,
		productCodeRepoProvider: opts.product.Provider,
		productLabel:            map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: task.Product.Name},
	}
}

func (c *productPipelineSyncerArgoCD) deploy(ctx context.Context) error {
	if err := c.syncAppProject(ctx); err != nil {
		return fmt.Errorf("sync argocd app project failed: %w", err)
	}
	if err := c.syncApp(ctx); err != nil {
		return fmt.Errorf("sync argocd app failed: %w", err)
	}

	secInfo := c.getSecretInfo()
	argoOperatorRoleName := c.task.NautesCfg.Secret.OperatorName[roleNameIndexArgoOperator]
	if err := c.secClient.GrantPermission(ctx, secInfo, argoOperatorRoleName, c.task.Cluster.Name); err != nil {
		return fmt.Errorf("grant permission to argo operator failed: %w", err)
	}

	return nil
}

func (c *productPipelineSyncerArgoCD) remove(ctx context.Context) error {
	secInfo := c.getSecretInfo()
	argoOperatorRoleName := c.task.NautesCfg.Secret.OperatorName[roleNameIndexArgoOperator]
	if err := c.secClient.RevokePermission(ctx, secInfo, argoOperatorRoleName, c.task.Cluster.Name); err != nil {
		return fmt.Errorf("revoke permission from argo operator failed: %w", err)
	}

	if err := c.deleteApp(ctx); err != nil {
		return fmt.Errorf("delete argocd app failed: %w", err)
	}

	if err := c.removeAppFromProject(ctx); err != nil {
		return fmt.Errorf("remove app from project failed: %w", err)
	}

	removable, err := c.checkAppProjectIsRemovable(ctx)
	if err != nil {
		return fmt.Errorf("check app project is removable failed: %w", err)
	}

	if removable {
		if err := c.deleteAppProject(ctx); err != nil {
			return fmt.Errorf("delete argocd app project failed: %w", err)
		}
	}

	return nil
}

func (c *productPipelineSyncerArgoCD) getSecretInfo() interfaces.SecretInfo {
	return interfaces.SecretInfo{
		Type: interfaces.SECRET_TYPE_GIT,
		CodeRepo: &interfaces.CodeRepo{
			ProviderType: c.productCodeRepoProvider.Spec.ProviderType,
			ID:           c.productCodeRepo.Name,
			User:         codeRepoUserDefault,
			Permission:   interfaces.CodeRepoPermissionReadOnly,
		},
	}
}

func (c *productPipelineSyncerArgoCD) syncAppProject(ctx context.Context) error {
	productName := c.task.Product.Name
	productNamespaceName := c.task.Product.Name
	appProject := &argocrd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
			Labels:    c.productLabel,
		},
	}

	if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	if err := utils.CheckResourceOperability(appProject, productName); err != nil {
		return err
	}

	hasDestination := false
	for _, destination := range appProject.Spec.Destinations {
		if destination.Namespace == c.task.Product.Name {
			hasDestination = true
			break
		}
	}
	if !hasDestination {
		appProject.Spec.Destinations = append(appProject.Spec.Destinations, argocrd.ApplicationDestination{
			Server:    "*",
			Namespace: productNamespaceName,
		})
	}

	hasSource := false
	for _, source := range appProject.Spec.SourceRepos {
		if source == c.productCodeRepo.Spec.URL {
			hasSource = true
			break
		}
	}
	if !hasSource {
		appProject.Spec.SourceRepos = append(appProject.Spec.SourceRepos, c.productCodeRepo.Spec.URL)
	}

	if !hasDestination || !hasSource {
		return CreateOrUpdateResource(ctx, c.k8sClient, appProject)
	}
	return nil
}

func (c *productPipelineSyncerArgoCD) removeAppFromProject(ctx context.Context) error {
	productName := c.task.Product.Name
	appProject := &argocrd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
			Labels:    c.productLabel,
		},
	}

	if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		return client.IgnoreNotFound(err)
	}

	if err := utils.CheckResourceOperability(appProject, productName); err != nil {
		return nil
	}

	needUpdate := false
	for i, destination := range appProject.Spec.Destinations {
		if destination.Namespace == c.task.Product.Name {
			appProject.Spec.Destinations = append(appProject.Spec.Destinations[:i], appProject.Spec.Destinations[i+1:]...)
			needUpdate = true
			break
		}
	}

	for i, source := range appProject.Spec.SourceRepos {
		if source == c.productCodeRepo.Spec.URL {
			appProject.Spec.SourceRepos = append(appProject.Spec.SourceRepos[:i], appProject.Spec.SourceRepos[i+1:]...)
			needUpdate = true
			break
		}
	}

	if needUpdate {
		return CreateOrUpdateResource(ctx, c.k8sClient, appProject)
	}
	return nil
}

func (c *productPipelineSyncerArgoCD) checkAppProjectIsRemovable(ctx context.Context) (bool, error) {
	productName := c.task.Product.Name
	appProject := &argocrd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
			Labels:    c.productLabel,
		},
	}

	if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if len(appProject.Spec.Destinations) != 0 ||
		len(appProject.Spec.SourceRepos) != 0 {
		return false, nil
	}

	return true, nil
}

func (c *productPipelineSyncerArgoCD) deleteAppProject(ctx context.Context) error {
	productName := c.task.Product.Name
	appProject := &argocrd.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
		},
	}

	return DeleteResource(ctx, c.k8sClient, appProject, productName)
}

func (c *productPipelineSyncerArgoCD) syncApp(ctx context.Context) error {
	productName := c.task.Product.Name
	productNamespaceName := c.task.Product.Name
	appName := fmt.Sprintf("%s-%s", productName, PipelineDeployTaskName)

	app := &argocrd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
			Labels:    c.productLabel,
		},
	}

	if err := c.k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	if err := utils.CheckResourceOperability(app, productName); err != nil {
		return err
	}

	existing := app.DeepCopyObject().(*argocrd.Application)

	app.Spec.Project = productName
	app.Spec.Destination = argocrd.ApplicationDestination{
		Server:    kubernetesAPIServerAddr,
		Namespace: productNamespaceName,
	}
	app.Spec.Source = argocrd.ApplicationSource{
		RepoURL:        c.productCodeRepo.Spec.URL,
		Path:           productPipelinePath,
		TargetRevision: productTargetRevision,
	}
	app.Spec.SyncPolicy = &argocrd.SyncPolicy{
		Automated: &argocrd.SyncPolicyAutomated{
			Prune:    true,
			SelfHeal: true,
		},
	}

	if !reflect.DeepEqual(existing.Spec, app.Spec) {
		return CreateOrUpdateResource(ctx, c.k8sClient, app)
	}
	return nil
}

func (c *productPipelineSyncerArgoCD) deleteApp(ctx context.Context) error {
	productName := c.task.Product.Name
	appName := fmt.Sprintf("%s-%s", productName, PipelineDeployTaskName)

	app := &argocrd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: c.task.NautesCfg.Deploy.ArgoCD.Namespace,
			Labels:    c.productLabel,
		},
	}

	return DeleteResource(ctx, c.k8sClient, app, productName)
}

// CreateOrUpdateResource create or update k8s resource in one function
func CreateOrUpdateResource(ctx context.Context, client client.Client, obj client.Object) error {
	if obj.GetCreationTimestamp().UTC().IsZero() {
		return client.Create(ctx, obj)
	}
	return client.Update(ctx, obj)
}

// DeleteResource check resource is belongs to product, if yes, delete resource
func DeleteResource(ctx context.Context, k8sClient client.Client, obj client.Object, productName string) error {
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	if err := utils.CheckResourceOperability(obj, productName); err != nil {
		return nil
	}

	return k8sClient.Delete(ctx, obj)
}
