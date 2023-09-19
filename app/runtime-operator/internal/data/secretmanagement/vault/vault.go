package vault

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	kratos "github.com/go-kratos/kratos/v2/transport/http"
	vaultapi "github.com/hashicorp/vault/api"
	vaultauthkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	vaultproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	loadcert "github.com/nautes-labs/nautes/pkg/util/loadcerts"
	"k8s.io/apimachinery/pkg/util/sets"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type secretManager interface {
	GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error
	RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error
}

type vault struct {
	client *vaultapi.Client
	vaultproxy.SecretHTTPClient
	vaultproxy.AuthHTTPClient
	vaultproxy.AuthGrantHTTPClient
	clusterName string
	secMgrMap   map[syncer.SecretType]secretManager
}

func NewSecretManagement(opt v1alpha1.Component, info *syncer.ComponentInitInfo) (syncer.SecretManagement, error) {
	return NewVaultClient(opt, info)
}

type NewVaultProxy func(url string) (vaultproxy.SecretHTTPClient, vaultproxy.AuthHTTPClient, vaultproxy.AuthGrantHTTPClient, error)
type newOptions struct {
	NewVaultProxyClient NewVaultProxy
}
type newOption func(*newOptions)

func SetNewVaultProxyClientFunction(fn NewVaultProxy) newOption {
	return func(no *newOptions) { no.NewVaultProxyClient = fn }
}

func NewVaultClient(opt v1alpha1.Component, info *syncer.ComponentInitInfo, opts ...newOption) (*vault, error) {
	options := &newOptions{
		NewVaultProxyClient: newVaultProxyClient,
	}

	for _, fn := range opts {
		fn(options)
	}

	vaultRawClient, err := newVaultRawClient(info.NautesConfig.Secret)
	if err != nil {
		return nil, err
	}

	secClient, authClient, grantClient, err := options.NewVaultProxyClient(info.NautesConfig.Secret.Vault.ProxyAddr)
	if err != nil {
		return nil, err
	}

	vaultClient := &vault{
		client:              vaultRawClient,
		SecretHTTPClient:    secClient,
		AuthHTTPClient:      authClient,
		AuthGrantHTTPClient: grantClient,
		clusterName:         info.ClusterName,
		secMgrMap:           map[syncer.SecretType]secretManager{},
	}

	vaultClient.secMgrMap[syncer.SecretTypeCodeRepo] = &codeRepoManager{
		AuthGrantHTTPClient: vaultClient.AuthGrantHTTPClient,
		clusterName:         vaultClient.clusterName,
	}
	vaultClient.secMgrMap[syncer.SecretTypeArtifactRepo] = &artifactRepoManager{
		AuthGrantHTTPClient: vaultClient.AuthGrantHTTPClient,
		clusterName:         vaultClient.clusterName,
	}

	return vaultClient, nil
}

const (
	clusterEngineName    = "cluster"
	clusterPathTemplate  = "kubernetes/%s/default/admin"
	clusterKubeconfigKey = "kubeconfig"
)

func (v *vault) CleanUp() error {
	err := v.client.Auth().Token().RevokeSelf("")
	if err != nil {
		logf.Log.Error(err, "revoke token failed")
		return err
	}
	return nil
}

// GetAccessInfo should return the information on how to access the cluster
func (v *vault) GetAccessInfo(ctx context.Context) (string, error) {
	path := fmt.Sprintf(clusterPathTemplate, v.clusterName)
	secret, err := v.client.KVv2(clusterEngineName).Get(ctx, path)
	if err != nil {
		return "", err
	}
	kubeconfig, ok := secret.Data[clusterKubeconfigKey]
	if !ok {
		return "", fmt.Errorf("can not find kubeconfig in secret store. instance name %s", v.clusterName)
	}
	return kubeconfig.(string), nil
}

func (v *vault) CreateUser(ctx context.Context, user syncer.User) error {
	roleInfo := &vaultproxy.AuthroleRequest_Kubernetes{
		Kubernetes: &vaultproxy.KubernetesAuthRoleMeta{},
	}

	if user.AuthInfo != nil && user.AuthInfo.Kubernetes != nil {
		serviceAccountSet := sets.New[string]()
		namespaceSet := sets.New[string]()
		for _, authInfo := range user.AuthInfo.Kubernetes {
			serviceAccountSet.Insert(authInfo.ServiceAccount)
			namespaceSet.Insert(authInfo.Namespace)
		}
		roleInfo.Kubernetes.Namespaces = namespaceSet.UnsortedList()
		roleInfo.Kubernetes.ServiceAccounts = serviceAccountSet.UnsortedList()
	}

	req := &vaultproxy.AuthroleRequest{
		ClusterName: v.clusterName,
		DestUser:    user.Name,
		Role:        roleInfo,
	}

	if _, err := v.AuthHTTPClient.CreateAuthrole(ctx, req); err != nil {
		return err
	}
	return nil
}

func (v *vault) DeleteUser(ctx context.Context, user syncer.User) error {
	req := &vaultproxy.AuthroleRequest{
		ClusterName: v.clusterName,
		DestUser:    user.Name,
	}

	if _, err := v.AuthHTTPClient.DeleteAuthrole(ctx, req); err != nil {
		return err
	}
	return nil
}

func (v *vault) GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	mgr, ok := v.secMgrMap[repo.Type]
	if !ok {
		return fmt.Errorf("unknow secret type %s", repo.Type)
	}
	return mgr.GrantPermission(ctx, repo, user)
}

func (v *vault) RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	mgr, ok := v.secMgrMap[repo.Type]
	if !ok {
		return fmt.Errorf("unknow secret type %s", repo.Type)
	}
	return mgr.RevokePermission(ctx, repo, user)
}

func newVaultRawClient(cfg configs.SecretRepo) (*vaultapi.Client, error) {
	config := vaultapi.DefaultConfig()
	config.Address = cfg.Vault.Addr

	if strings.HasPrefix(config.Address, "https://") {
		certPool, err := loadcert.GetCertPool()
		if err != nil {
			return nil, fmt.Errorf("load cert failed: %w", err)
		}
		config.HttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: certPool,
				},
			},
		}
	}

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	if cfg.Vault.Token != "" {
		client.SetToken(cfg.Vault.Token)
		return client, nil
	}

	var vaultOpts []vaultauthkubernetes.LoginOption
	vaultOpts = append(vaultOpts, vaultauthkubernetes.WithMountPath(cfg.Vault.MountPath))

	roleName, ok := cfg.OperatorName[configs.OperatorNameRuntime]
	if !ok {
		return nil, fmt.Errorf("role name not found in nautes config")
	}
	k8sAuth, err := vaultauthkubernetes.NewKubernetesAuth(
		roleName,
		vaultOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kubernetes auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to log in with Kubernetes auth: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}
	return client, nil
}

const (
	vaultProxyClientKeypairPath = "/opt/nautes/keypair"
)

func newVaultProxyClient(url string) (vaultproxy.SecretHTTPClient, vaultproxy.AuthHTTPClient, vaultproxy.AuthGrantHTTPClient, error) {
	certPool, err := loadcert.GetCertPool()
	if err != nil {
		return nil, nil, nil, err
	}

	cert, err := tls.LoadX509KeyPair(
		filepath.Join(vaultProxyClientKeypairPath, "client.crt"),
		filepath.Join(vaultProxyClientKeypairPath, "client.key"))
	if err != nil {
		return nil, nil, nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{cert},
	}

	newTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	httpClient, err := kratos.NewClient(context.Background(),
		kratos.WithTransport(newTransport),
		kratos.WithEndpoint(url),
		kratos.WithTLSConfig(tlsConfig))
	if err != nil {
		return nil, nil, nil, err
	}

	return vaultproxy.NewSecretHTTPClient(httpClient),
		vaultproxy.NewAuthHTTPClient(httpClient),
		vaultproxy.NewAuthGrantHTTPClient(httpClient),
		nil
}
