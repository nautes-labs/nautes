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

package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	cluster "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	gitlab "github.com/nautes-labs/nautes/app/api-server/pkg/gitlab"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kops/pkg/kubeconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	TenantLabel               = "coderepo.resource.nautes.io/tenant-management"
	NautesClusterDir          = "nautes/overlays/production/clusters"
	DefaultClusterTemplateURL = "https://github.com/nautes-labs/cluster-templates.git"
	SecretPath                = "default"
	SecretEngine              = "pki"
)

type ClusterUsecase struct {
	log              *log.Helper
	secretRepo       Secretrepo
	codeRepo         CodeRepo
	resourcesUsecase *ResourcesUsecase
	configs          *nautesconfigs.Config
	client           client.Client
	cluster          cluster.ClusterRegistrationOperator
	dex              DexRepo
}

type ClusterData struct {
	ClusterName string
	ApiServer   string
	ClusterType string
	Usage       string
	HostCluster string
}

func NewClusterUsecase(logger log.Logger, codeRepo CodeRepo, secretRepo Secretrepo, resourcesUsecase *ResourcesUsecase, configs *nautesconfigs.Config, client client.Client, cluster cluster.ClusterRegistrationOperator, dex DexRepo) *ClusterUsecase {
	return &ClusterUsecase{log: log.NewHelper(log.With(logger)), codeRepo: codeRepo, secretRepo: secretRepo, resourcesUsecase: resourcesUsecase, configs: configs, client: client, cluster: cluster, dex: dex}
}

func (c *ClusterUsecase) GetCluster(ctx context.Context, clusterName string) (*resourcev1alpha1.Cluster, error) {
	repository, err := c.GetTenantRepository(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepositoryLocalPath, err := c.CloneRepository(ctx, repository.HttpUrlToRepo)
	if err != nil {
		c.log.Errorf("failed to clone tenant repository, the url %s may be invalid or does not exist", repository.HttpUrlToRepo)
		return nil, err
	}

	defer cleanCodeRepo(tenantRepositoryLocalPath)

	cluster, err := c.cluster.GetClsuter(tenantRepositoryLocalPath, clusterName)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (c *ClusterUsecase) ListClusters(ctx context.Context) ([]*resourcev1alpha1.Cluster, error) {
	repository, err := c.GetTenantRepository(ctx)
	if err != nil {
		return nil, err
	}

	tenantRepositoryLocalPath, err := c.CloneRepository(ctx, repository.HttpUrlToRepo)
	if err != nil {
		c.log.Errorf("failed to clone tenant repository, the url %s may be invalid or does not exist", repository.HttpUrlToRepo)
		return nil, err
	}

	defer cleanCodeRepo(tenantRepositoryLocalPath)

	clusters, err := c.cluster.GetClsuters(tenantRepositoryLocalPath)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (c *ClusterUsecase) SaveCluster(ctx context.Context, param *cluster.ClusterRegistrationParam, kubeconfig string) error {
	err := param.Cluster.ValidateCluster(context.TODO(), param.Cluster, c.client, false)
	if err != nil {
		c.log.Errorf("failed to call 'resourceCluster.ValidateCluster', err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to validate cluster, err: %s", err)
	}

	if cluster.IsPhysical(param.Cluster) {
		err := c.SaveKubeconfig(ctx, param.Cluster.Name, param.Cluster.Spec.ApiServer, kubeconfig)
		if err != nil {
			c.log.Errorf("failed to call 'SaveCluster', could not save kubeconfig to secret store, cluster name: %s, err: %s", param.Cluster.Name, err)
			return fmt.Errorf("failed to save kubeconfig to secret store, err: %s", err)
		}
	}

	httpURLToRepo := GetClusterTemplateHttpsURL(c.configs)
	clusterTemplateLocalPath, err := c.CloneRepository(ctx, httpURLToRepo)
	if err != nil {
		c.log.Errorf("failed to call 'CloneRepository', could not clone cluster template repository, the url %s may be invalid or does not exist, err: %s", httpURLToRepo, err)
		return fmt.Errorf("failed to clone cluster template repository, the url %s may be invalid or does not exist, err: %s", httpURLToRepo, err)
	}
	defer cleanCodeRepo(clusterTemplateLocalPath)

	repository, err := c.GetTenantRepository(ctx)
	if err != nil {
		c.log.Errorf("failed to call 'GetTenantRepository', could not get tenant repository, cluster name: %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to get tenant repository, err: %s", err)
	}
	tenantRepositoryLocalPath, err := c.CloneRepository(ctx, repository.HttpUrlToRepo)
	if err != nil {
		c.log.Errorf("failed to call 'CloneRepository', could not clone tenant repository, the url %s may be invalid or does not exist, err: %s", repository.HttpUrlToRepo, err)
		return fmt.Errorf("failed to clone tenant repository, the url %s may be invalid or does not exist, err: %s", repository.HttpUrlToRepo, err)
	}
	defer cleanCodeRepo(tenantRepositoryLocalPath)

	defaultCert, err := c.GetDefaultCertificate(ctx)
	if err != nil {
		c.log.Errorf("failed to call 'GetDefaultCertificate', could not get default certificate from secret store, cluster name: %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to get default certificate, err: %s", err)
	}

	gitlabCert, err := gitlab.GetCertificate(c.configs.Git.Addr)
	if err != nil {
		c.log.Errorf("failed to call 'GetCertificate', could not get gitlab certificate from secret store, cluster name: %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to get gitlab certificate, err: %s", err)
	}

	param.ClusterTemplateRepoLocalPath = clusterTemplateLocalPath
	param.CaBundleList = cluster.CaBundleList{
		Default: defaultCert,
		Gitlab:  gitlabCert,
	}

	param.TenantConfigRepoLocalPath = tenantRepositoryLocalPath
	param.RepoURL = repository.SshUrlToRepo
	param.Configs = c.configs
	err = c.cluster.InitializeClusterConfig(param)
	if err != nil {
		c.log.Errorf("failed to call 'cluster.InitializeClusterConfig', could not initial cluster %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to initial cluster, err: %s", err)
	}

	err = c.cluster.Save()
	if err != nil {
		c.log.Errorf("failed to call 'cluster.Save', could not save cluster %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to save cluster, err: %s", err)
	}

	err = c.resourcesUsecase.SaveConfig(ctx, tenantRepositoryLocalPath)
	if err != nil {
		c.log.Errorf("failed to call 'resourcesUsecase.SaveConfig', could not save git config to git repo, cluster name: %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to save git config to git repo, err: %s", err)
	}

	err = c.SaveDexConfig(param, tenantRepositoryLocalPath)
	if err != nil {
		c.log.Errorf("failed to call 'SaveDexConfig', could not save dex config, cluster name: %s, err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to save dex config, err: %s", err)
	}

	c.log.Infof("successfully register cluster, cluster name: %s", param.Cluster.Name)

	return nil
}

func (c *ClusterUsecase) DeleteCluster(ctx context.Context, clusterName string) error {
	url := GetClusterTemplateHttpsURL(c.configs)
	clusterTemplateLocalPath, err := c.CloneRepository(ctx, url)
	if err != nil {
		c.log.Errorf("failed to clone cluster template repository, cluster name: %s, url: %s", clusterName, url)
		return err
	}
	defer cleanCodeRepo(clusterTemplateLocalPath)

	project, err := c.GetTenantRepository(ctx)
	if err != nil {
		c.log.Errorf("failed to get tenant repository, cluster name: %s", clusterName)
		return err
	}
	tenantRepositoryLocalPath, err := c.CloneRepository(ctx, project.HttpUrlToRepo)
	if err != nil {
		c.log.Errorf("failed to get tenant repository local path, cluster name: %s", clusterName)
		return err
	}
	defer cleanCodeRepo(tenantRepositoryLocalPath)

	resourceCluster, err := GetClusterFromTenantRepository(tenantRepositoryLocalPath, clusterName)
	if err != nil {
		c.log.Errorf("cluster %s does not exist or is invalid", clusterName)
		return fmt.Errorf("cluster %s does not exist or is invalid", clusterName)
	}

	err = resourceCluster.ValidateCluster(context.TODO(), resourceCluster, c.client, true)
	if err != nil {
		c.log.Errorf("failed to call 'resourceCluster.ValidateCluster', err: %s", clusterName, err)
		return fmt.Errorf("failed to validate cluster, err: %s", err)
	}

	param := &cluster.ClusterRegistrationParam{
		Cluster:                      resourceCluster,
		RepoURL:                      project.SshUrlToRepo,
		Configs:                      c.configs,
		ClusterTemplateRepoLocalPath: clusterTemplateLocalPath,
		TenantConfigRepoLocalPath:    tenantRepositoryLocalPath,
	}
	err = c.cluster.InitializeClusterConfig(param)
	if err != nil {
		c.log.Errorf("failed to call 'cluster.InitializeClusterConfig', could not initial cluster %s, err: %s", clusterName, err)
		return fmt.Errorf("failed to initial cluster, err: %s", err)
	}

	err = c.DeleteDexConfig(param)
	if err != nil {
		c.log.Errorf("failed to call 'DeleteDexConfig', could not delete dex config, cluster name: %s, err: %s", clusterName, err)
		return fmt.Errorf("failed to delete dex config, err: %s", err)
	}

	err = c.cluster.Remove()
	if err != nil {
		c.log.Errorf("failed to call 'cluster.Remove', could not remove cluster %s, err: %s", clusterName, err)
		return fmt.Errorf("failed to remove cluster, err: %s", err)
	}

	err = c.resourcesUsecase.SaveConfig(ctx, tenantRepositoryLocalPath)
	if err != nil {
		c.log.Errorf("failed to call 'resourcesUsecase.SaveConfig', could not save git config to git repo, cluster name: %s, err: %s", clusterName, err)
		return fmt.Errorf("failed to save git config to git repo, err: %s", err)
	}

	c.log.Infof("successfully remove cluster, cluster name: %s", clusterName)

	return nil
}

func (c *ClusterUsecase) CloneRepository(ctx context.Context, url string) (string, error) {
	path, err := c.resourcesUsecase.CloneCodeRepo(ctx, url)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (c *ClusterUsecase) SaveKubeconfig(ctx context.Context, id, server, config string) error {
	if config == "" {
		return fmt.Errorf("register physical cluster, kubeconfig is not empty")
	}

	config, err := c.ConvertKubeconfig(config, server)
	if err != nil {
		return err
	}
	err = c.secretRepo.SaveClusterConfig(ctx, id, config)
	if err != nil {
		return err
	}

	return nil
}

func (r *ClusterUsecase) ConvertKubeconfig(config, server string) (string, error) {
	kubeconfig := &kubeconfig.KubectlConfig{}
	jsonData, err := yaml.YAMLToJSONStrict([]byte(config))
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(jsonData), kubeconfig)
	if err != nil {
		return "", err
	}

	if len(kubeconfig.Clusters) < 1 {
		return "", fmt.Errorf("invalid kubeconfig file: must have at least one cluster")
	}

	if len(kubeconfig.Users) < 1 {
		return "", fmt.Errorf("invalid kubeconfig file: must have at least one user")
	}

	kubeconfig.Clusters[0].Cluster.Server = server

	bytes, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *ClusterUsecase) GetDefaultCertificate(ctx context.Context) (string, error) {
	secretOptions := &SecretOptions{
		SecretPath:   SecretPath,
		SecretEngine: SecretEngine,
		SecretKey:    "cacert",
	}
	cert, err := c.secretRepo.GetSecret(ctx, secretOptions)
	if err != nil {
		return "", err
	}

	return cert, nil
}

func (c *ClusterUsecase) GetTenantRepository(ctx context.Context) (*Project, error) {
	codeRepos := &resourcev1alpha1.CodeRepoList{}
	labelSelector := labels.SelectorFromSet(map[string]string{TenantLabel: c.configs.Nautes.TenantName})
	err := c.client.List(context.Background(), codeRepos, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	if len(codeRepos.Items) == 0 {
		return nil, fmt.Errorf("tenant repository is not found")
	}

	pid, _ := utilstrings.ExtractNumber(RepoPrefix, codeRepos.Items[0].Name)
	repository, err := c.codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return nil, err
	}

	return repository, nil
}

func (c *ClusterUsecase) SaveDexConfig(param *cluster.ClusterRegistrationParam, teantLocalPath string) error {
	if !cluster.IsHostCluser(param.Cluster) {
		argocdOauthURL, err := c.cluster.GetArgocdURL()
		if err != nil {
			return err
		}
		if argocdOauthURL != "" {
			err = c.dex.UpdateRedirectURIs(argocdOauthURL)
			if err != nil {
				return err
			}
		}
	}

	if cluster.IsPhysical(param.Cluster) {
		tektonOauthURL, err := c.cluster.GetTektonOAuthURL()
		if err != nil {
			return err
		}

		if tektonOauthURL != "" {
			err = c.dex.UpdateRedirectURIs(tektonOauthURL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ClusterUsecase) DeleteDexConfig(param *cluster.ClusterRegistrationParam) error {
	if !cluster.IsHostCluser(param.Cluster) {
		argocdOauthURL, err := c.cluster.GetArgocdURL()
		if err != nil {
			return err
		}

		if argocdOauthURL != "" {
			err = c.dex.RemoveRedirectURIs(argocdOauthURL)
			if err != nil {
				return err
			}
		}
	}

	if cluster.IsPhysical(param.Cluster) {
		tektonOauthURL, err := c.cluster.GetTektonOAuthURL()
		if err != nil {
			return err
		}

		if tektonOauthURL != "" {
			err = c.dex.RemoveRedirectURIs(tektonOauthURL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetClusterTemplateHttpsURL(configs *nautesconfigs.Config) string {
	if configs.Nautes.RuntimeTemplateSource != "" {
		return configs.Nautes.RuntimeTemplateSource
	}

	return DefaultClusterTemplateURL
}

func GetClusterFromTenantRepository(tenantRepositoryLocalPath, clusterName string) (*resourcev1alpha1.Cluster, error) {
	filePath := fmt.Sprintf("%s/%s/%s.yaml", tenantRepositoryLocalPath, NautesClusterDir, clusterName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cluster resourcev1alpha1.Cluster
	err = yaml.Unmarshal(content, &cluster)
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}
