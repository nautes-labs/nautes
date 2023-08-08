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

package factory

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/app/cluster-operator/internal/secretclient/vault"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
	nautesconfig "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	factory = newFactory()
)

type NewSecretClient func(context.Context, *nautesconfig.Config, client.Client) (secretclient.SecretClient, error)

type SecretClientFactory struct {
	newFuntionMenu map[string]NewSecretClient
}

func newFactory() SecretClientFactory {
	return SecretClientFactory{
		newFuntionMenu: map[string]NewSecretClient{
			string(nautesconfig.SECRET_STORE_VAULT): vault.NewVaultClient,
		},
	}
}

func GetFactory() SecretClientFactory {
	return factory
}

func (s SecretClientFactory) NewSecretClient(ctx context.Context, cfg *nautesconfig.Config, k8sClient client.Client) (secretclient.SecretClient, error) {
	logger := log.FromContext(ctx)

	newSecretClient, ok := s.newFuntionMenu[string(cfg.Secret.RepoType)]
	if !ok {
		return nil, fmt.Errorf("unknow type of secret client")
	}

	logger.V(1).Info("create a new secret client", "type", cfg.Secret.RepoType)
	secretStore, err := newSecretClient(ctx, cfg, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("get secret client failed: %w", err)
	}

	return secretStore, nil
}

func (s SecretClientFactory) AddNewMethod(name string, newFunc NewSecretClient) {
	s.newFuntionMenu[name] = newFunc
}
