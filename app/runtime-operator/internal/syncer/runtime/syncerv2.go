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

package deploymentruntime

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployer/argocd"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/deployer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/initinfo"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	deployerNewFuntions = map[string]deployer.NewDeployer{
		"argocd": argocd.NewDeployer,
	}
)

func newComponentInitInfo(runtime nautescrd.Runtime, db database.Database, cfg *rest.Config) (*initinfo.ComponentInitInfo, error) {
	cluster, err := db.GetClusterByRuntime(runtime)
	if err != nil {
		return nil, err
	}

	product, err := db.GetProduct(runtime.GetProduct())
	if err != nil {
		return nil, err
	}

	task := &initinfo.ComponentInitInfo{
		ClusterConnectInfo: initinfo.ClusterConnectInfo{
			ClusterType: initinfo.ClusterType(cluster.Spec.ClusterKind),
			Kubernetes: &initinfo.ClusterConnectInfoKubernetes{
				Config: cfg,
			},
		},
		ProductName: product.Name,
		ClusterName: cluster.Name,
		Labels:      map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: product.Name},
	}

	return task, nil
}

func newDeployer(component nautescrd.Component, task initinfo.ComponentInitInfo, db database.Database) (deployer.Deployer, error) {
	fn, ok := deployerNewFuntions[component.Name]
	if !ok {
		return nil, fmt.Errorf("deployer not support type %s", component.Name)
	}
	return fn(component.Namespace, task, db)
}

func newDeployAppV2(runtime nautescrd.Runtime, db database.Database) (*deployer.Application, error) {
	runtimeNamespaces := runtime.GetNamespaces()
	if len(runtimeNamespaces) == 0 {
		return nil, fmt.Errorf("get 0 namespace in runtime %s", runtime.GetName())
	}

	app := &deployer.Application{
		Name: "",
		Git:  nil,
		Destination: deployer.Destination{
			Kubernetes: &deployer.DestinationKubernetes{
				Namespace: runtimeNamespaces[0],
			},
		},
	}

	switch r := runtime.(type) {
	case *nautescrd.DeploymentRuntime:
		coderepo, err := db.GetCodeRepo(r.Spec.ManifestSource.CodeRepo)
		if err != nil {
			return nil, err
		}

		app.Git = &deployer.ApplicationGit{
			URL:      coderepo.Spec.URL,
			Revision: r.Spec.ManifestSource.TargetRevision,
			Path:     r.Spec.ManifestSource.Path,
		}

		app.Name = r.Name
	case *nautescrd.ProjectPipelineRuntime:
		if r.Spec.AdditionalResources == nil ||
			r.Spec.AdditionalResources.Git == nil {
			return nil, nil
		}

		deploySource := r.Spec.AdditionalResources.Git

		var url string
		if deploySource.URL != "" {
			url = deploySource.URL
		} else {
			coderepo, err := db.GetCodeRepo(deploySource.CodeRepo)
			if err != nil {
				return nil, err
			}
			url = coderepo.Spec.URL
		}

		app.Git = &deployer.ApplicationGit{
			URL:      url,
			Revision: deploySource.Revision,
			Path:     deploySource.Path,
		}

		app.Name = fmt.Sprintf("%s-additional", r.Name)
	default:
		return nil, fmt.Errorf("runtime type is not supported")
	}

	return app, nil
}

const (
	providerType           = "gitlab"
	repoUser               = "default"
	repoPermissionReadOnly = "readonly"
)

type RuntimeSyncer struct {
	k8sClient      client.Client
	clusterName    string
	productName    string
	resourceLabels map[string]string
	nautesCfg      nautescfg.Config
	secClient      interfaces.SecretClient
	db             database.Database
}

func NewRuntimeSyncer(ctx context.Context, initInfo initinfo.ComponentInitInfo, db database.Database, nautescfg nautescfg.Config) (*RuntimeSyncer, error) {
	cfg := initInfo.ClusterConnectInfo.Kubernetes.Config
	if initInfo.ClusterConnectInfo.Kubernetes.Config == nil {
		return nil, fmt.Errorf("rest config is null")
	}

	if err := corev1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}
	if err := nautescrd.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return nil, fmt.Errorf("get secret client from context failed")
	}

	return &RuntimeSyncer{
		k8sClient:      k8sClient,
		nautesCfg:      nautescfg,
		productName:    initInfo.ProductName,
		clusterName:    initInfo.ClusterName,
		resourceLabels: initInfo.Labels,
		db:             db,
		secClient:      secClient,
	}, nil

}

func newSecretInfo(repo nautescrd.CodeRepo) interfaces.SecretInfo {
	return interfaces.SecretInfo{
		Type: interfaces.SECRET_TYPE_GIT,
		CodeRepo: &interfaces.CodeRepo{
			ProviderType: providerType,
			ID:           repo.Name,
			User:         repoUser,
			Permission:   repoPermissionReadOnly,
		},
		AritifaceRepo: nil,
	}
}

func (rs *RuntimeSyncer) SyncCodeRepos(ctx context.Context) error {
	errs := []error{}

	tenantCodeRepos, err := rs.db.ListUsedCodeRepos(
		database.InCluster(rs.clusterName),
		database.WithOutDeletedRuntimes(),
	)
	if err != nil {
		return fmt.Errorf("list code repos failed: %w", err)
	}

	if len(tenantCodeRepos) != 1 {
		for _, repo := range tenantCodeRepos {
			tmp := repo.DeepCopy()
			tmp.ObjectMeta.Namespace = rs.nautesCfg.Nautes.Namespace

			operate, err := controllerutil.CreateOrUpdate(ctx, rs.k8sClient, tmp, func() error {
				tmp.ObjectMeta.Labels = rs.resourceLabels
				return nil
			})
			if err != nil {
				errs = append(errs, err)
			}

			if operate == controllerutil.OperationResultCreated {
				repo := newSecretInfo(repo)
				if err := rs.secClient.GrantPermission(ctx, repo, rs.nautesCfg.Secret.OperatorName["Argo"], rs.clusterName); err != nil {
					return fmt.Errorf("grant permission failed: %w", err)
				}
			}
		}

	} else {
		productCodeRepo := tenantCodeRepos[0]
		productCodeRepo.Namespace = rs.nautesCfg.Nautes.Namespace

		repo := newSecretInfo(productCodeRepo)
		if err := rs.secClient.RevokePermission(ctx, repo, rs.nautesCfg.Secret.OperatorName["Argo"], rs.clusterName); err != nil {
			return fmt.Errorf("grant permission failed: %w", err)
		}

		if err := rs.k8sClient.Delete(ctx, &productCodeRepo); client.IgnoreNotFound(err) != nil {
			errs = append(errs, err)
		}
	}

	codeRepoSet := map[string]bool{}
	for _, codeRepo := range tenantCodeRepos {
		codeRepoSet[codeRepo.Name] = true
	}

	runtimeCodeRepoList := &nautescrd.CodeRepoList{}
	belongsToProduct := client.MatchingLabels(rs.resourceLabels)
	if err := rs.k8sClient.List(ctx, runtimeCodeRepoList, belongsToProduct); err != nil {
		return fmt.Errorf("list code repos in runtime failed: %w", err)
	}

	for _, codeRepo := range runtimeCodeRepoList.Items {
		_, ok := codeRepoSet[codeRepo.Name]
		if ok {
			continue
		}

		repo := newSecretInfo(codeRepo)
		if err := rs.secClient.RevokePermission(ctx, repo, rs.nautesCfg.Secret.OperatorName["Argo"], rs.clusterName); err != nil {
			return fmt.Errorf("grant permission failed: %w", err)
		}

		if err := rs.k8sClient.Delete(ctx, &codeRepo); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("sync coderepos failed: %v", errs)
	}
	return nil
}
