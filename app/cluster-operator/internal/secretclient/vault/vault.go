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

// +kubebuilder:skip

// Package vault is a secret client to save cluster info to vault
package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kratos "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	kubernetesauth "github.com/hashicorp/vault/api/auth/kubernetes"
	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	kubeclient "github.com/nautes-labs/nautes/app/cluster-operator/internal/pkg/kubeclient"
	operatorerror "github.com/nautes-labs/nautes/app/cluster-operator/pkg/error"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
	vaultproxyv1 "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	nautesctx "github.com/nautes-labs/nautes/pkg/context"
	kubeconvert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	loadcert "github.com/nautes-labs/nautes/pkg/util/loadcerts"
	ctrl "sigs.k8s.io/controller-runtime"
)

var requeueAfterEnvIsNotReady = time.Second * 60

type KubernetesClient interface {
	GetCluster(ctx context.Context, name, namespace string) (*nautescrd.Cluster, error)
	GetServiceAccount(ctx context.Context, name, namespace string) (*v1.ServiceAccount, error)
	GetSecret(ctx context.Context, name, namespace string) (*v1.Secret, error)
	ListStatefulSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.StatefulSetList, error)
}

type kubeClientFactory func(kubeconfig string) (KubernetesClient, error)

type VaultStatus struct {
	SecretName    string `json:"secret_name" yaml:"secretName"`
	SecretPath    string `json:"secret_path" yaml:"secretPath"`
	SecretVersion int    `json:"secret_version" yaml:"secretVersion"`
}

const (
	CONTEXT_KEY_CFG nautesctx.ContextKey = "vault.client.config"
)

func NewK8SClient(kubeconfig string) (KubernetesClient, error) {
	var client *kubernetes.Clientset
	var dclient dynamic.Interface
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = ctrl.GetConfig()
	} else {
		config, err = kubeconvert.ConvertStringToRestConfig(kubeconfig)
	}
	if err != nil {
		return nil, err
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dclient, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &kubeclient.K8SClient{
		Clientset: *client,
		Dynamic:   dclient,
	}, nil
}

type VaultClient struct {
	log            logr.Logger
	ctx            context.Context
	Vault          *vault.Client
	VaultProxy     vaultproxyv1.SecretHTTPClient
	VaultAuth      vaultproxyv1.AuthHTTPClient
	VaultAuthGrant vaultproxyv1.AuthGrantHTTPClient

	TenantAuthName string
	Configs        *configs.Config
	Client         client.Client
	GetKubeClient  kubeClientFactory
}

type InitOpt func(vs *VaultClient) error

func NewVaultClient(ctx context.Context, configs *configs.Config, k8sClient client.Client) (secretclient.SecretClient, error) {
	return NewVaultClientWithOpts(ctx, configs, func(vs *VaultClient) error {
		vs.Client = k8sClient
		return nil
	})
}

func NewVaultClientWithOpts(ctx context.Context, configs *configs.Config, opts ...InitOpt) (*VaultClient, error) {
	vs := &VaultClient{}

	for _, init := range opts {
		if err := init(vs); err != nil {
			return nil, err
		}

	}

	vs.ctx = ctx
	vs.log = log.FromContext(ctx)

	if vs.Vault == nil {
		vaultClient, err := newVaultClient(configs.Secret.Vault, configs.Secret.OperatorName["Cluster"])
		if err != nil {
			return nil, err
		}
		vs.Vault = vaultClient
	}

	if vs.VaultProxy == nil && vs.VaultAuth == nil && vs.VaultAuthGrant == nil {
		secProxy, authProxy, authGrantProxy, err := newVProxyClient(configs.Secret.Vault.ProxyAddr, loadcert.NautesDefaultCertsPath)
		if err != nil {
			return nil, err
		}
		vs.VaultProxy = secProxy
		vs.VaultAuth = authProxy
		vs.VaultAuthGrant = authGrantProxy
	}

	if vs.GetKubeClient == nil {
		vs.GetKubeClient = NewK8SClient
	}

	vs.TenantAuthName = configs.Secret.Vault.MountPath
	vs.Configs = configs

	return vs, nil
}

func (vc *VaultClient) createSecret(cluster *nautescrd.Cluster) error {
	if cluster == nil {
		return nil
	}

	kubeconfig, err := vc.getVClusterKubeConfig(*cluster)
	if err != nil {
		return err
	}
	clusterRequest := getClusterRequest(*cluster, &clusterAccountKubeconfig{
		kubeconfig: kubeconfig,
	})
	clusterMeta, err := clusterRequest.ConvertRequest()
	if err != nil {
		return err
	}
	vc.log.V(1).Info("creating secret in vault", "secret path", clusterMeta.SecretPath)
	sec, err := vc.Vault.KVv2("cluster").Get(context.Background(), clusterMeta.SecretPath)
	if sec != nil {
		if _, ok := sec.Data["kubeconfig"]; ok {
			if sec.Data["kubeconfig"] == kubeconfig {
				return nil
			}
		}
	}
	_, err = vc.VaultProxy.CreateCluster(context.Background(), clusterRequest)
	if err != nil {
		return err
	}

	return nil
}

func (vc *VaultClient) deleteSecret(cluster *nautescrd.Cluster) error {
	if cluster == nil {
		return nil
	}

	clusterRequest := getClusterRequest(*cluster)
	// kratos api input verify force to write something in Account key
	clusterRequest.Account = &vaultproxyv1.ClusterAccount{
		Kubeconfig: "avoid input check",
	}
	clusterMeta, err := clusterRequest.ConvertRequest()
	if err != nil {
		return err
	}
	vc.log.V(1).Info("removing secret in vault", "secret path", clusterMeta.SecretPath)
	_, err = vc.VaultProxy.DeleteCluster(context.Background(), clusterRequest)
	if err != nil {
		return err
	}
	return nil
}

func (vc *VaultClient) getSecretMeta(cluster *nautescrd.Cluster) (*vault.KVMetadata, error) {
	clusterRequest := getClusterRequest(*cluster)
	clusterMeta, err := clusterRequest.ConvertRequest()
	if err != nil {
		return nil, err
	}
	meta, err := vc.Vault.KVv2("cluster").GetMetadata(context.Background(), clusterMeta.SecretPath)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (vc *VaultClient) createAuth(cluster *nautescrd.Cluster) error {
	if cluster == nil {
		return nil
	}

	newAuth, err := getAuthRequest(cluster, func(authReq *vaultproxyv1.AuthRequest) error {
		kubeconfig, err := vc.getKubeConfig(*cluster)
		if err != nil {
			return err
		}
		token, err := vc.getVaultToken(kubeconfig)
		if err != nil {
			return operatorerror.NewError(err, requeueAfterEnvIsNotReady)
		}
		rawConfig, err := kubeconvert.ConvertStringToRawConfig(kubeconfig)
		if err != nil {
			return err
		}

		clusterName := rawConfig.Contexts[rawConfig.CurrentContext].Cluster
		cluster := rawConfig.Clusters[clusterName]
		authReq.Kubernetes = &vaultproxyv1.Kubernetes{
			Url:      cluster.Server,
			Cabundle: string(cluster.CertificateAuthorityData),
			Token:    token,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("make request failed: %w", err)
	}

	authMeta, err := newAuth.ConvertRequest()
	if err != nil {
		return err
	}
	vc.log.V(1).Info("creating auth", "mount path", authMeta.FullPath)
	_, err = vc.VaultAuth.CreateAuth(context.Background(), newAuth)
	if err != nil {
		return fmt.Errorf("create auth failed: %w", err)
	}

	return nil
}

func (vc *VaultClient) disableAuth(cluster *nautescrd.Cluster) error {
	if cluster == nil {
		return nil
	}

	oldAuth, err := getAuthRequest(cluster)
	if err != nil {
		return err
	}

	authMeta, err := oldAuth.ConvertRequest()
	if err != nil {
		return err
	}

	vc.log.V(1).Info("removing auth in vault", "mount path", authMeta.FullPath)
	_, err = vc.VaultAuth.DeleteAuth(context.Background(), oldAuth)
	if err != nil {
		return err
	}
	return nil
}

func (vc *VaultClient) createRole(cluster *nautescrd.Cluster) error {
	if cluster == nil {
		return nil
	}

	operatorRoleNameList := vc.Configs.Secret.OperatorName
	for role, roleName := range operatorRoleNameList {
		serviceAccountName := vc.Configs.Nautes.ServiceAccount[role]
		roleRequest := &vaultproxyv1.AuthroleRequest{
			ClusterName: cluster.Name,
			DestUser:    roleName,
			Role: &vaultproxyv1.AuthroleRequest_Kubernetes{
				Kubernetes: &vaultproxyv1.KubernetesAuthRoleMeta{
					Namespaces:      []string{vc.Configs.Nautes.Namespace},
					ServiceAccounts: []string{serviceAccountName},
				},
			},
		}

		roleMeta, err := roleRequest.ConvertRequest()
		if err != nil {
			return err
		}
		vc.log.V(1).Info("creating role", "rolePath", roleMeta.FullPath)
		_, err = vc.VaultAuth.CreateAuthrole(context.Background(), roleRequest)
		if err != nil {
			return err
		}
	}
	return nil
}

type clusterAccount interface {
	GetAccount() *vaultproxyv1.ClusterAccount
}

type clusterAccountKubeconfig struct {
	kubeconfig string
}

func (ca *clusterAccountKubeconfig) GetAccount() *vaultproxyv1.ClusterAccount {
	return &vaultproxyv1.ClusterAccount{
		Kubeconfig: ca.kubeconfig,
	}
}

func getClusterRequest(cluster nautescrd.Cluster, accounts ...clusterAccount) *vaultproxyv1.ClusterRequest {
	var clusterAccount *vaultproxyv1.ClusterAccount
	for _, account := range accounts {
		clusterAccount = account.GetAccount()
	}
	clusterRequest := &vaultproxyv1.ClusterRequest{
		Meta: &vaultproxyv1.ClusterMeta{
			Type:       "kubernetes",
			Id:         cluster.Name,
			Username:   "default",
			Permission: "admin",
		},
		Account: clusterAccount,
	}

	return clusterRequest
}

type vaultAuthOptions func(authReq *vaultproxyv1.AuthRequest) error

func getAuthRequest(cluster *nautescrd.Cluster, auths ...vaultAuthOptions) (*vaultproxyv1.AuthRequest, error) {
	authReq := &vaultproxyv1.AuthRequest{
		ClusterName: cluster.Name,
		AuthType:    "kubernetes",
		Kubernetes:  nil,
	}

	for _, authOpt := range auths {
		err := authOpt(authReq)
		if err != nil {
			return nil, err
		}
	}
	return authReq, nil
}

func getGrantRequest(ctx context.Context, cluster nautescrd.Cluster, authName string) (*vaultproxyv1.AuthroleClusterPolicyRequest, error) {
	cfg, err := nautesctx.FromConfigContext(ctx, CONTEXT_KEY_CFG)
	if err != nil {
		return nil, err
	}

	roleName, ok := cfg.Secret.OperatorName[ROLE_NAME_KEY_RUNTIME]
	if !ok {
		return nil, ErrorRoleNameNotFound(ROLE_NAME_KEY_RUNTIME)
	}
	return &vaultproxyv1.AuthroleClusterPolicyRequest{
		ClusterName: authName,
		DestUser:    roleName,
		Secret: &vaultproxyv1.ClusterMeta{
			Type:       "kubernetes",
			Id:         cluster.Name,
			Username:   "default",
			Permission: "admin",
		},
	}, nil
}

func (vc *VaultClient) grantClusterPermission(ctx context.Context, cluster *nautescrd.Cluster, authName string) error {
	if cluster == nil {
		return nil
	}

	grantReq, err := getGrantRequest(ctx, *cluster, authName)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	policyMeta, secretMeta, err := grantReq.ConvertToAuthPolicyReqeuest()
	if err != nil {
		return err
	}
	vc.log.V(1).Info("grant permission to user", "user", policyMeta.Name, "permission", secretMeta.PolicyName)
	_, err = vc.VaultAuthGrant.GrantAuthroleClusterPolicy(context.Background(), grantReq)
	if err != nil {
		return err
	}
	return nil
}

func (vc *VaultClient) revokeClusterPermission(ctx context.Context, cluster *nautescrd.Cluster, authName string) error {
	if cluster == nil {
		return nil
	}

	grantReq, err := getGrantRequest(ctx, *cluster, authName)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	policyMeta, secretMeta, err := grantReq.ConvertToAuthPolicyReqeuest()
	if err != nil {
		return err
	}
	vc.log.V(1).Info("revoke permission from user", "user", policyMeta.Name, "permission", secretMeta.PolicyName)
	_, err = vc.VaultAuthGrant.RevokeAuthroleClusterPolicy(context.Background(), grantReq)
	if vaultproxyv1.IsResourceNotFound(err) {
		vc.log.V(1).Info("can not find role in vault")
	} else if err != nil {
		return err
	}
	return nil
}

func (vc *VaultClient) grantTenantPermission(ctx context.Context, cluster *clustercrd.Cluster, authName string) error {
	if cluster == nil {
		return nil
	}

	cfg, err := nautesctx.FromConfigContext(ctx, CONTEXT_KEY_CFG)
	if err != nil {
		return err
	}
	roleName, ok := cfg.Secret.OperatorName[ROLE_NAME_KEY_ARGO]
	if !ok {
		return ErrorRoleNameNotFound(ROLE_NAME_KEY_ARGO)
	}
	tenantCodeRepoID, err := vc.getTenantRepoID(ctx)
	if err != nil {
		return fmt.Errorf("get tenant id failed: %w", err)
	}
	_, err = vc.VaultAuthGrant.GrantAuthroleGitPolicy(ctx, &vaultproxyv1.AuthroleGitPolicyRequest{
		ClusterName: authName,
		DestUser:    roleName,
		Secret: &vaultproxyv1.GitMeta{
			ProviderType: string(cfg.Git.GitType),
			Id:           tenantCodeRepoID,
			Username:     "default",
			Permission:   "readonly",
		},
	})

	return err
}

func (vc *VaultClient) GetKubeConfig(ctx context.Context, cluster *nautescrd.Cluster) (string, error) {
	return vc.getKubeConfig(*cluster)
}

func (vc *VaultClient) getKubeConfig(cluster nautescrd.Cluster) (string, error) {
	clusterRequest := getClusterRequest(cluster)
	clusterMeta, err := clusterRequest.ConvertRequest()
	if err != nil {
		return "", err
	}
	kubeconfig, err := vc.Vault.KVv2(clusterMeta.SecretType).Get(context.Background(), clusterMeta.SecretPath)
	if err != nil {
		return "", err
	}

	return kubeconfig.Data["kubeconfig"].(string), nil
}

// Steps:
// *. Get Host cluster from current cluster
// *. Create client of host cluster
// *. Get vcluster secret from host cluster
// *. Update vcluster host path
// *. Return vcluster's kubeconfig string
func (vc *VaultClient) getVClusterKubeConfig(cluster nautescrd.Cluster) (string, error) {
	currentClient, err := vc.GetKubeClient("")
	if err != nil {
		return "", err
	}

	vc.log.V(1).Info("get host cluster")
	hostCluster, err := currentClient.GetCluster(context.Background(), cluster.Spec.HostCluster, cluster.Namespace)
	if err != nil {
		return "", err
	}

	kubeconfig, err := vc.getKubeConfig(*hostCluster)
	if err != nil {
		return "", err
	}

	hostClient, err := vc.GetKubeClient(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("init host client failed: %w", err)
	}

	sts, err := hostClient.ListStatefulSets(context.Background(), cluster.Name, metav1.ListOptions{
		LabelSelector: "app=vcluster",
	})
	if err != nil {
		return "", fmt.Errorf("get vcluster name failed: %w", err)
	} else if len(sts.Items) != 1 {
		err := errors.New("get vcluster name failed, vcluster is none or not only")
		return "", operatorerror.NewError(err, requeueAfterEnvIsNotReady)
	}

	vc.log.V(1).Info("get vcluster")
	configName := fmt.Sprintf("vc-%s", sts.Items[0].Name)
	secret, err := hostClient.GetSecret(context.Background(), configName, cluster.Name)
	if err != nil {
		return "", operatorerror.NewError(err, requeueAfterEnvIsNotReady)
	}
	vclusterRawConfig, err := kubeconvert.ConvertStringToRawConfig(string(secret.Data["config"]))
	if err != nil {
		return "", err
	}

	clusterName := vclusterRawConfig.Contexts[vclusterRawConfig.CurrentContext].Cluster
	vclusterRawConfig.Clusters[clusterName].Server = cluster.Spec.ApiServer
	newKubeconfig, err := clientcmd.Write(vclusterRawConfig)
	if err != nil {
		return "", err
	}
	return string(newKubeconfig), nil

}

func (vc *VaultClient) getTenantRepoID(ctx context.Context) (string, error) {
	cfg, err := nautesctx.FromConfigContext(ctx, CONTEXT_KEY_CFG)
	if err != nil {
		return "", err
	}

	lables := map[string]string{nautescrd.LABEL_TENANT_MANAGEMENT: cfg.Nautes.TenantName}
	coderepos := &nautescrd.CodeRepoList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(lables),
		client.InNamespace(cfg.Nautes.Namespace),
	}
	if err = vc.Client.List(ctx, coderepos, listOpts...); err != nil {
		return "", fmt.Errorf("list code repo failed: %w", err)
	}

	switch len(coderepos.Items) {
	case 1:
		return coderepos.Items[0].Name, nil
	default:
		return "", fmt.Errorf("code repo is not unique")
	}
}

func (vc *VaultClient) getVaultToken(kubeconfig string) (string, error) {

	vaultNamespace := vc.Configs.Secret.Vault.Namesapce
	vaultServiceAccount := vc.Configs.Secret.Vault.ServiceAccount
	k8sClient, err := vc.GetKubeClient(kubeconfig)
	if err != nil {
		return "", err
	}

	serviceAccount, err := k8sClient.GetServiceAccount(context.Background(), vaultServiceAccount, vaultNamespace)
	if err != nil {
		return "", err
	}

	token, err := k8sClient.GetSecret(context.Background(), serviceAccount.Secrets[0].Name, vaultNamespace)
	if err != nil {
		return "", err
	}
	return string(token.Data["token"]), nil
}

func newVaultClient(vaultConfig configs.Vault, roleName string) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = vaultConfig.Addr

	if strings.HasPrefix(config.Address, "https://") {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(vaultConfig.CABundle))
		config.HttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	if vaultConfig.Token != "" {
		client.SetToken(vaultConfig.Token)
		return client, nil
	}

	var vaultOpts []kubernetesauth.LoginOption
	vaultOpts = append(vaultOpts, kubernetesauth.WithMountPath(vaultConfig.MountPath))

	k8sAuth, err := kubernetesauth.NewKubernetesAuth(
		roleName,
		vaultOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Kubernetes auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to log in with Kubernetes auth: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}
	return client, nil
}

func (vc *VaultClient) Logout() {
	err := vc.Vault.Auth().Token().RevokeSelf("")
	if err != nil {
		vc.log.Error(err, "clean up token failed")
	}
}

const (
	vaultProxyClientKeypairPath    = "/opt/nautes/keypair/"
	EnvVaultProxyClientKeypairPath = "VAULT_PROXY_CLIENT_KEYPAIR_PATH"
)

func newVProxyClient(url, PKIPath string) (vaultproxyv1.SecretHTTPClient, vaultproxyv1.AuthHTTPClient, vaultproxyv1.AuthGrantHTTPClient, error) {
	caCertPool, err := loadcert.GetCertPool(loadcert.CertPath(PKIPath))
	if err != nil {
		return nil, nil, nil, err
	}

	keypairPath := os.Getenv(EnvVaultProxyClientKeypairPath)
	if keypairPath == "" {
		keypairPath = vaultProxyClientKeypairPath
	}

	cert, err := tls.LoadX509KeyPair(
		filepath.Join(keypairPath, "client.crt"),
		filepath.Join(keypairPath, "client.key"))
	if err != nil {
		return nil, nil, nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	httpClient, err := kratos.NewClient(context.Background(),
		kratos.WithEndpoint(url),
		kratos.WithTLSConfig(tlsConfig))
	if err != nil {
		return nil, nil, nil, err
	}

	return vaultproxyv1.NewSecretHTTPClient(httpClient),
		vaultproxyv1.NewAuthHTTPClient(httpClient),
		vaultproxyv1.NewAuthGrantHTTPClient(httpClient),
		nil
}

func ErrorRoleNameNotFound(name string) error {
	return fmt.Errorf("can not find the name of %s in global config", name)
}
