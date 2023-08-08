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

	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"

	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// PipelineDeployTaskName is a name for ProductPipelineSyncer if it need a name to record this deploy,
	// such as argocd app name for argocd
	PipelineDeployTaskName    = "tekton-pipeline"
	kubernetesAPIServerAddr   = "https://kubernetes.default.svc"
	productPipelinePath       = "templates/pipelines"
	productTargetRevision     = "HEAD"
	codeRepoUserDefault       = "default"
	roleNameIndexArgoOperator = "Argo"
	rolebindingName           = "tekton-user"
	rolebindingRole           = "tekton-user-namespaced"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nautescrd.AddToScheme(scheme))
	utilruntime.Must(hncv1alpha2.AddToScheme(scheme))
	utilruntime.Must(argocrd.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
}

var (
	scheme                                                                            = runtime.NewScheme()
	productPipelineSyncerFactory map[nautescfg.DeployAppType]NewProductPipelineSyncer = map[nautescfg.DeployAppType]NewProductPipelineSyncer{
		nautescfg.DEPLOY_APP_TYPE_ARGOCD: NewProductPipelineSyncerArgoCD,
	}
)

// ProductPipelineSyncer is a object use to sync tekton pipeline from git repo to k8s
type ProductPipelineSyncer interface {
	deploy(ctx context.Context) error
	remove(ctx context.Context) error
}

// NewProductPipelineSyncer is used to return a productPipelineSyncer object
type NewProductPipelineSyncer func(client.Client, interfaces.SecretClient, interfaces.RuntimeSyncTask, destClusterOptions) ProductPipelineSyncer

type codeRepoInfo struct {
	CodeRepo nautescrd.CodeRepo
	Provider nautescrd.CodeRepoProvider
}

type destClusterOptions struct {
	product codeRepoInfo
	runtime codeRepoInfo
}

type destCluster struct {
	k8sClient                client.Client
	secClient                interfaces.SecretClient
	task                     interfaces.RuntimeSyncTask
	productCodeRepo          nautescrd.CodeRepo
	pipelineCodeRepo         nautescrd.CodeRepo
	productPipelineSyncer    ProductPipelineSyncer
	runtime                  *nautescrd.ProjectPipelineRuntime
	pipelineCodeRepoProvider nautescrd.CodeRepoProvider
}

func newDestCluster(ctx context.Context, task interfaces.RuntimeSyncTask, opts destClusterOptions) (*destCluster, error) {
	if task.AccessInfo.Type != interfaces.ACCESS_TYPE_K8S {
		return nil, fmt.Errorf("access type is not supported")
	}

	k8sClient, err := client.New(task.AccessInfo.Kubernetes, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return nil, fmt.Errorf("get secret client from context failed")
	}
	if task.RuntimeType != interfaces.RUNTIME_TYPE_PIPELINE {
		return nil, fmt.Errorf("runtime type %s is not support", task.RuntimeType)
	}

	deployApp, ok := task.NautesCfg.Deploy.DefaultApp[string(task.Cluster.Spec.ClusterKind)]
	if !ok {
		return nil, fmt.Errorf("can not find deploy app for cluster kind %s", task.Cluster.Spec.ClusterKind)
	}
	fnGetDeployer, ok := productPipelineSyncerFactory[deployApp]
	if !ok {
		return nil, fmt.Errorf("deployment app %s is unsupported", deployApp)
	}
	cluster := &destCluster{
		k8sClient:                k8sClient,
		secClient:                secClient,
		task:                     task,
		productCodeRepo:          opts.product.CodeRepo,
		pipelineCodeRepo:         opts.runtime.CodeRepo,
		productPipelineSyncer:    fnGetDeployer(k8sClient, secClient, task, opts),
		runtime:                  task.Runtime.(*nautescrd.ProjectPipelineRuntime),
		pipelineCodeRepoProvider: opts.runtime.Provider,
	}

	return cluster, nil
}

func (c *destCluster) syncProduct(ctx context.Context) error {
	if err := c.productPipelineSyncer.deploy(ctx); err != nil {
		return fmt.Errorf("sync pipeline task failed: %w", err)
	}
	if err := c.syncRoleBinding(ctx); err != nil {
		return fmt.Errorf("sync role binding failed: %w", err)
	}

	if err := c.syncHNCConfig(ctx); err != nil {
		return fmt.Errorf("sync hnc config failed: %w", err)
	}
	return nil
}

func (c *destCluster) cleanUpProduct(ctx context.Context) error {
	if err := c.productPipelineSyncer.remove(ctx); err != nil {
		return fmt.Errorf("delete pipeline task failed: %w", err)
	}

	if err := c.k8sClient.Delete(ctx, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{
		Name:      rolebindingName,
		Namespace: c.task.Product.Name,
	}}); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("delete rolebinding failed: %w", err)
	}
	return nil
}

func (c *destCluster) grantPermissionToPipelineRun(ctx context.Context) error {
	permissons := []interfaces.CodeRepoPermission{
		interfaces.CodeRepoPermissionReadOnly,
		interfaces.CodeRepoPermissionReadWrite,
	}

	clusterName := c.task.Cluster.Name
	roleName := c.runtime.Name

	for _, permission := range permissons {
		secInfo := c.getSecretInfo(permission)
		err := c.secClient.GrantPermission(ctx, secInfo, roleName, clusterName)
		if err != nil {
			return fmt.Errorf("grant permision %s-%s to role %s failed: %w", secInfo.CodeRepo.ID, secInfo.CodeRepo.Permission, roleName, err)
		}
	}
	return nil
}

func (c *destCluster) revokePermissionFromPipelineRun(ctx context.Context) error {
	permissons := []interfaces.CodeRepoPermission{
		interfaces.CodeRepoPermissionReadOnly,
		interfaces.CodeRepoPermissionReadWrite,
	}

	clusterName := c.task.Cluster.Name
	roleName := c.runtime.Name

	for _, permission := range permissons {
		secInfo := c.getSecretInfo(permission)
		err := c.secClient.RevokePermission(ctx, secInfo, roleName, clusterName)
		if err != nil {
			return fmt.Errorf("rovoke permision %s-%s from role %s failed: %w", secInfo.CodeRepo.ID, secInfo.CodeRepo.Permission, roleName, err)
		}
	}
	return nil
}

func (c *destCluster) getSecretInfo(permission interfaces.CodeRepoPermission) interfaces.SecretInfo {
	return interfaces.SecretInfo{
		Type: interfaces.SECRET_TYPE_GIT,
		CodeRepo: &interfaces.CodeRepo{
			ProviderType: c.pipelineCodeRepoProvider.Spec.ProviderType,
			ID:           c.runtime.Spec.PipelineSource,
			User:         codeRepoUserDefault,
			Permission:   permission,
		},
	}
}

const (
	clusterRoleKind = "ClusterRole"
	userRoleKind    = "ServiceAccount"
)

func (c *destCluster) syncRoleBinding(ctx context.Context) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rolebindingName,
			Namespace: c.task.Product.Name,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c.k8sClient, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     clusterRoleKind,
			APIGroup: rbacv1.GroupName,
			Name:     rolebindingRole,
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:     rbacv1.GroupKind,
				APIGroup: rbacv1.GroupName,
				Name:     c.task.Product.Spec.Name,
			},
			{
				Kind: userRoleKind,
				Name: c.task.ServiceAccountName,
			},
		}
		return nil
	})
	return err
}

const (
	hncConfigName        = "config"
	pipelineResourceName = "pipeline"
)

func (c *destCluster) syncHNCConfig(ctx context.Context) error {
	hncConfig := &hncv1alpha2.HNCConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: hncConfigName,
		},
		Spec: hncv1alpha2.HNCConfigurationSpec{
			Resources: []hncv1alpha2.ResourceSpec{},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c.k8sClient, hncConfig, func() error {
		hasResource := false
		for _, resource := range hncConfig.Spec.Resources {
			if resource.Resource == pipelineResourceName {
				hasResource = true
			}
		}
		if !hasResource {
			hncConfig.Spec.Resources = append(hncConfig.Spec.Resources, hncv1alpha2.ResourceSpec{
				Resource: pipelineResourceName,
			})
		}
		return nil
	})
	return err
}
