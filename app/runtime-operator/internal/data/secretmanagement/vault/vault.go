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
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	vaultproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	loadcert "github.com/nautes-labs/nautes/pkg/util/loadcerts"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// secretManager grants the machine account access to the secret or revokes the machine account access to the secret.
type secretManager interface {
	GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error
	RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error
}

type vault struct {
	client *vaultapi.Client
	vaultproxy.SecretHTTPClient
	vaultproxy.AuthHTTPClient
	vaultproxy.AuthGrantHTTPClient
	clusterName string
	secMgrMap   map[component.SecretType]secretManager
}

func NewSecretManagement(opt v1alpha1.Component, info *component.ComponentInitInfo) (component.SecretManagement, error) {
	return NewVaultClient(opt, info)
}

// NewVaultProxy defines a function of Vault proxy.
type NewVaultProxy func(url string) (vaultproxy.SecretHTTPClient, vaultproxy.AuthHTTPClient, vaultproxy.AuthGrantHTTPClient, error)

type newOptions struct {
	newVaultProxyClient NewVaultProxy
}

type NewOption func(*newOptions)

func SetNewVaultProxyClientFunction(fn NewVaultProxy) NewOption {
	return func(no *newOptions) { no.newVaultProxyClient = fn }
}

// NewVaultClient builds a client instance to access the Vault, including read and write operations.
func NewVaultClient(_ v1alpha1.Component, info *component.ComponentInitInfo, opts ...NewOption) (component.SecretManagement, error) {
	options := &newOptions{
		newVaultProxyClient: newVaultProxyClient,
	}

	for _, fn := range opts {
		fn(options)
	}

	vaultRawClient, err := newVaultRawClient(info.NautesConfig.Secret)
	if err != nil {
		return nil, err
	}

	secClient, authClient, grantClient, err := options.newVaultProxyClient(info.NautesConfig.Secret.Vault.ProxyAddr)
	if err != nil {
		return nil, err
	}

	vaultClient := &vault{
		client:              vaultRawClient,
		SecretHTTPClient:    secClient,
		AuthHTTPClient:      authClient,
		AuthGrantHTTPClient: grantClient,
		clusterName:         info.ClusterName,
		secMgrMap:           map[component.SecretType]secretManager{},
	}

	vaultClient.secMgrMap[component.SecretTypeCodeRepo] = &codeRepoManager{
		AuthGrantHTTPClient: vaultClient.AuthGrantHTTPClient,
		clusterName:         vaultClient.clusterName,
	}
	vaultClient.secMgrMap[component.SecretTypeArtifactRepo] = &artifactRepoManager{
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

// CleanUp revokes a connection with Vault.
func (v *vault) CleanUp() error {
	err := v.client.Auth().Token().RevokeSelf("")
	if err != nil {
		logf.Log.Error(err, "revoke token failed")
		return err
	}
	return nil
}

func (v *vault) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

// GetAccessInfo returns the access information of the cluster.
func (v *vault) GetAccessInfo(ctx context.Context) (string, error) {
	path := fmt.Sprintf(clusterPathTemplate, v.clusterName)
	secret, err := v.client.KVv2(clusterEngineName).Get(ctx, path)
	if err != nil {
		return "", err
	}
	kubeConfig, ok := secret.Data[clusterKubeconfigKey]
	if !ok {
		return "", fmt.Errorf("can not find kubeconfig in secret store. instance name %s", v.clusterName)
	}
	return kubeConfig.(string), nil
}

const OriginName = "vault"

// CreateAccount creates the machine account in Vault by the specified Kubernetes authentication method.
func (v *vault) CreateAccount(ctx context.Context, account component.MachineAccount) (*component.AuthInfo, error) {
	roleInfo := &vaultproxy.AuthroleRequest_Kubernetes{
		Kubernetes: &vaultproxy.KubernetesAuthRoleMeta{},
	}
	roleInfo.Kubernetes.Namespaces = account.Spaces
	roleInfo.Kubernetes.ServiceAccounts = []string{account.Name}

	req := &vaultproxy.AuthroleRequest{
		ClusterName: v.clusterName,
		DestUser:    account.Name,
		Role:        roleInfo,
	}

	if _, err := v.AuthHTTPClient.CreateAuthrole(ctx, req); err != nil {
		return nil, err
	}

	authInfo := &component.AuthInfo{
		OriginName:      OriginName,
		AccountName:     account.Name,
		AuthType:        component.AuthTypeKubernetesServiceAccount,
		ServiceAccounts: []component.AuthInfoServiceAccount{},
	}

	for _, ns := range roleInfo.Kubernetes.Namespaces {
		authInfo.ServiceAccounts = append(authInfo.ServiceAccounts, component.AuthInfoServiceAccount{
			ServiceAccount: account.Name,
			Namespace:      ns,
		})
	}

	return authInfo, nil
}

// DeleteAccount deletes the machine account in Vault.
func (v *vault) DeleteAccount(ctx context.Context, account component.MachineAccount) error {
	req := &vaultproxy.AuthroleRequest{
		ClusterName: v.clusterName,
		DestUser:    account.Name,
	}

	if _, err := v.AuthHTTPClient.DeleteAuthrole(ctx, req); err != nil {
		return err
	}
	return nil
}

// GrantPermission grants the machine account access to the secret.
// It's implemented differently depending on the secret type.
func (v *vault) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	mgr, ok := v.secMgrMap[repo.Type]
	if !ok {
		return fmt.Errorf("unknow secret type %s", repo.Type)
	}
	return mgr.GrantPermission(ctx, repo, account)
}

// RevokePermission revokes the machine account access to the secret.
// It's implemented differently depending on the secret type.
func (v *vault) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	mgr, ok := v.secMgrMap[repo.Type]
	if !ok {
		return fmt.Errorf("unknow secret type %s", repo.Type)
	}
	return mgr.RevokePermission(ctx, repo, account)
}

// newVaultRawClient returns a read-only client to access Vault.
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
					RootCAs:    certPool,
					MinVersion: tls.VersionTLS12,
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

// newVaultProxyClient returns three read-write clients to access Vault.
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
		MinVersion:   tls.VersionTLS12,
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
