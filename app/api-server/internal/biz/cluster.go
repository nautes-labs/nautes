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

	"github.com/go-kratos/kratos/v2/log"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	clustermanagement "github.com/nautes-labs/nautes/app/api-server/pkg/clusters"
	gitlab "github.com/nautes-labs/nautes/app/api-server/pkg/gitlab"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"github.com/nautes-labs/nautes/pkg/queue"
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
	clusters         clustermanagement.ClusterRegistrationOperator
	queue            queue.Queuer
}

type ClusterData struct {
	ClusterName string
	ApiServer   string
	ClusterType string
	Usage       string
	HostCluster string
}

//nolint:revive
func NewClusterUsecase(logger log.Logger, codeRepo CodeRepo, secretRepo Secretrepo, resourcesUsecase *ResourcesUsecase, configs *nautesconfigs.Config, k8sClient client.Client, clusters clustermanagement.ClusterRegistrationOperator, q queue.Queuer) *ClusterUsecase {
	var clusterUsage = &ClusterUsecase{
		log:              log.NewHelper(log.With(logger)),
		codeRepo:         codeRepo,
		secretRepo:       secretRepo,
		resourcesUsecase: resourcesUsecase,
		configs:          configs,
		client:           k8sClient,
		clusters:         clusters,
		queue:            q,
	}

	clusterUsage.queue.AddHandler(clusterUsage.RefreshHostCluster)

	return clusterUsage
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

	cluster, err := c.clusters.GetClsuter(tenantRepositoryLocalPath, clusterName)
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

	clusters, err := c.clusters.ListClusters(tenantRepositoryLocalPath)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (c *ClusterUsecase) SaveCluster(ctx context.Context, param *clustermanagement.ClusterRegistrationParams, kubeconfigData string) error {
	var cluster = param.Cluster

	err := param.Cluster.ValidateCluster(context.TODO(), param.Cluster, c.client, false)
	if err != nil {
		c.log.Errorf("failed to call 'resourceCluster.ValidateCluster', err: %s", param.Cluster.Name, err)
		return fmt.Errorf("failed to validate cluster, err: %s", err)
	}

	if clustermanagement.IsPhysical(param.Cluster) {
		err := c.SaveKubeconfig(ctx, param.Cluster.Name, param.Cluster.Spec.ApiServer, kubeconfigData)
		if err != nil {
			c.log.Errorf("failed to call 'SaveCluster', could not save kubeconfig to secret store, cluster name: %s, err: %s", cluster.Name, err)
			return fmt.Errorf("failed to save kubeconfig to secret store, err: %s", err)
		}
	}

	repositoriesInfo, err := c.getRepositoriesInfo(ctx)
	if err != nil {
		return err
	}

	defer cleanCodeRepo(repositoriesInfo.ClusterTemplateDir)
	defer cleanCodeRepo(repositoriesInfo.TenantRepoDir)

	err = c.renderClusterAndPushToGit(ctx, cluster, param.Vcluster, repositoriesInfo)
	if err != nil {
		return err
	}

	err = c.isRefreshHostCluster(ctx, cluster)
	if err != nil {
		return err
	}

	c.log.Infof("successfully register cluster, cluster name: %s", param.Cluster.Name)

	return nil
}

func (c *ClusterUsecase) renderClusterAndPushToGit(ctx context.Context, cluster *resourcev1alpha1.Cluster, vclusterInfo *clustermanagement.VclusterInfo, repositoriesInfo *clustermanagement.RepositoriesInfo) error {
	certs, err := c.getCerts(ctx, cluster)
	if err != nil {
		return err
	}

	params := &clustermanagement.ClusterRegistrationParams{
		Cluster:       cluster,
		Repo:          repositoriesInfo,
		Cert:          certs,
		NautesConfigs: c.configs.Nautes,
		OauthConfigs:  c.configs.OAuth,
		SecretConfigs: c.configs.Secret,
		GitConfigs:    c.configs.Git,
		Vcluster:      vclusterInfo,
	}

	err = c.clusters.SaveCluster(params)
	if err != nil {
		return err
	}

	err = c.resourcesUsecase.PushToGit(ctx, repositoriesInfo.TenantRepoDir)
	if err != nil {
		c.log.Errorf("failed to call 'resourcesUsecase.SaveConfig', could not save git config to git repo, cluster name: %s, err: %s", cluster.Name, err)
		return fmt.Errorf("failed to save git config to git repo, err: %s", err)
	}

	return nil
}

func (c *ClusterUsecase) getCerts(ctx context.Context, cluster *resourcev1alpha1.Cluster) (*clustermanagement.Cert, error) {
	defaultCert, err := c.GetDefaultCertificate(ctx)
	if err != nil {
		c.log.Errorf("failed to call 'GetDefaultCertificate', could not get default certificate from secret store, cluster name: %s, err: %s", cluster.Name, err)
		return nil, fmt.Errorf("failed to get certificate, err: %s", err)
	}

	gitlabCert, err := gitlab.GetCertificate(c.configs.Git.Addr)
	if err != nil {
		c.log.Errorf("failed to call 'GetCertificate', could not get gitlab certificate from secret store, cluster name: %s, err: %s", cluster.Name, err)
		return nil, fmt.Errorf("failed to get gitlab certificate, err: %s", err)
	}

	return &clustermanagement.Cert{
		Default: defaultCert,
		Gitlab:  gitlabCert,
	}, nil
}

func (c *ClusterUsecase) getRepositoriesInfo(ctx context.Context) (*clustermanagement.RepositoriesInfo, error) {
	httpURLToRepo := GetClusterTemplateHttpsURL(c.configs)
	clusterTemplateDir, err := c.CloneRepository(ctx, httpURLToRepo)
	if err != nil {
		return nil, err
	}

	repository, err := c.GetTenantRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gitlab certificate, err: %s", err)
	}
	tenantRepoDir, err := c.CloneRepository(ctx, repository.HttpUrlToRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to get gitlab certificate, err: %s", err)
	}

	return &clustermanagement.RepositoriesInfo{
		ClusterTemplateDir: clusterTemplateDir,
		TenantRepoDir:      tenantRepoDir,
		TenantRepoURL:      repository.SshUrlToRepo,
	}, nil
}

type Body struct {
	Token string
}

func (c *ClusterUsecase) RefreshHostCluster(clusterName string, bytes []byte) error {
	body := Body{}
	err := json.Unmarshal(bytes, &body)
	if err != nil {
		return err
	}

	key := "token"
	ctx := context.WithValue(context.Background(), key, body.Token)

	repositoriesInfo, err := c.getRepositoriesInfo(ctx)
	if err != nil {
		return err
	}

	cluster, err := c.clusters.GetClsuter(repositoriesInfo.TenantRepoDir, clusterName)
	if err != nil {
		return err
	}

	if clustermanagement.IsHostCluser(cluster) {
		err = c.renderClusterAndPushToGit(ctx, cluster, nil, repositoriesInfo)
		if err != nil {
			c.log.Error(err)
			return err
		}
	}

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

	repository, err := c.GetTenantRepository(ctx)
	if err != nil {
		c.log.Errorf("failed to get tenant repository, cluster name: %s", clusterName)
		return err
	}
	tenantRepositoryLocalPath, err := c.CloneRepository(ctx, repository.HttpUrlToRepo)
	if err != nil {
		c.log.Errorf("failed to get tenant repository local path, cluster name: %s", clusterName)
		return err
	}
	defer cleanCodeRepo(tenantRepositoryLocalPath)

	cluster, err := c.clusters.GetClsuter(tenantRepositoryLocalPath, clusterName)
	if err != nil {
		return err
	}

	params := &clustermanagement.ClusterRegistrationParams{
		Cluster: cluster,
		Repo: &clustermanagement.RepositoriesInfo{
			ClusterTemplateDir: clusterTemplateLocalPath,
			TenantRepoDir:      tenantRepositoryLocalPath,
			TenantRepoURL:      repository.SshUrlToRepo,
		},
		NautesConfigs: c.configs.Nautes,
		OauthConfigs:  c.configs.OAuth,
		SecretConfigs: c.configs.Secret,
		GitConfigs:    c.configs.Git,
	}

	if err = c.clusters.RemoveCluster(params); err != nil {
		return err
	}

	err = c.resourcesUsecase.PushToGit(ctx, tenantRepositoryLocalPath)
	if err != nil {
		c.log.Errorf("failed to call 'resourcesUsecase.SaveConfig', could not save git config to git repo, cluster name: %s, err: %s", clusterName, err)
		return fmt.Errorf("failed to save git config to git repo, err: %s", err)
	}

	err = c.isRefreshHostCluster(ctx, cluster)
	if err != nil {
		return err
	}

	c.log.Infof("successfully remove cluster, cluster name: %s", clusterName)

	return nil
}

func (c *ClusterUsecase) isRefreshHostCluster(ctx context.Context, cluster *resourcev1alpha1.Cluster) error {
	if clustermanagement.IsVirtual(cluster) {
		token := ctx.Value("token").(string)
		body := &Body{
			Token: token,
		}
		bytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		c.queue.Send(cluster.Spec.HostCluster, bytes)
	}
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

func (c *ClusterUsecase) ConvertKubeconfig(config, server string) (string, error) {
	cfg := &kubeconfig.KubectlConfig{}
	jsonData, err := yaml.YAMLToJSONStrict([]byte(config))
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(jsonData, cfg)
	if err != nil {
		return "", err
	}

	if len(cfg.Clusters) < 1 {
		return "", fmt.Errorf("invalid kubeconfig file: must have at least one cluster")
	}

	if len(cfg.Users) < 1 {
		return "", fmt.Errorf("invalid kubeconfig file: must have at least one user")
	}

	cfg.Clusters[0].Cluster.Server = server

	bytes, err := yaml.Marshal(cfg)
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
	codeRepoList := &resourcev1alpha1.CodeRepoList{}
	labelSelector := labels.SelectorFromSet(map[string]string{TenantLabel: c.configs.Nautes.TenantName})
	err := c.client.List(context.Background(), codeRepoList, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	if len(codeRepoList.Items) == 0 {
		return nil, fmt.Errorf("tenant repository is not found")
	}

	pid, _ := utilstrings.ExtractNumber(RepoPrefix, codeRepoList.Items[0].Name)
	repository, err := c.codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return nil, err
	}

	return repository, nil
}

func GetClusterTemplateHttpsURL(configs *nautesconfigs.Config) string {
	if configs.Nautes.RuntimeTemplateSource != "" {
		return configs.Nautes.RuntimeTemplateSource
	}

	return DefaultClusterTemplateURL
}
