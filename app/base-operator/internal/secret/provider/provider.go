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

package provider

import (
	"context"
	"fmt"

	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"

	"github.com/nautes-labs/nautes/app/base-operator/internal/secret/vault"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
)

func init() {
	SecretProviders["vault"] = vault.NewClient
}

type NewClient func(cfg nautescfg.SecretRepo) (baseinterface.SecretClient, error)

var SecretProviders = map[string]NewClient{}

func GetSecretStore(ctx context.Context, cfg nautescfg.SecretRepo) (baseinterface.SecretClient, error) {
	NewClient, ok := SecretProviders[string(cfg.RepoType)]
	if !ok {
		return nil, fmt.Errorf("unknow sercret repo type")
	}
	return NewClient(cfg)
}
