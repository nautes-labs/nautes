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
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	vault "github.com/hashicorp/vault/api"
	kubernetesauth "github.com/hashicorp/vault/api/auth/kubernetes"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Vault struct {
	*vault.Client
}

const (
	TenantNamespace             = "tenant"
	CodeHostingPlatformRootPath = "git/%s/root"
	CodeHostingPlatformRootKey  = "access_token"
)

func (v *Vault) GetGitRepoRootToken(ctx context.Context, name string) (string, error) {
	path := fmt.Sprintf(CodeHostingPlatformRootPath, name)
	secret, err := v.KVv2(TenantNamespace).Get(ctx, path)
	if err != nil {
		return "", err
	}
	token, ok := secret.Data[CodeHostingPlatformRootKey]
	if !ok {
		return "", fmt.Errorf("can not find access token in secret store. instance name %s", name)
	}
	return token.(string), nil
}

func NewClient(cfg nautescfg.SecretRepo) (baseinterface.SecretClient, error) {
	return NewVault(cfg)
}

func (v *Vault) Logout() {
	err := v.Auth().Token().RevokeSelf("")
	if err != nil {
		logf.Log.Error(err, "revoke token failed")
	}
}

func NewVault(cfg nautescfg.SecretRepo) (*Vault, error) {
	config := vault.DefaultConfig()
	config.Address = cfg.Vault.Addr

	if strings.HasPrefix(config.Address, "https://") {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(cfg.Vault.CABundle))
		config.HttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					RootCAs:    caCertPool,
				},
			},
		}
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	if cfg.Vault.Token != "" {
		client.SetToken(cfg.Vault.Token)
		return &Vault{
			Client: client,
		}, nil
	}

	var vaultOpts []kubernetesauth.LoginOption
	vaultOpts = append(vaultOpts, kubernetesauth.WithMountPath(cfg.Vault.MountPath))

	k8sAuth, err := kubernetesauth.NewKubernetesAuth(
		cfg.OperatorName["Base"],
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
	return &Vault{
		Client: client,
	}, nil
}
