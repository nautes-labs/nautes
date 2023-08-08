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

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/vault"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	runtimeinterface "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

func init() {
	SecretProviders["vault"] = vault.NewClient
}

type NewClient func(cfg nautescfg.SecretRepo) (runtimeinterface.SecretClient, error)

var SecretProviders = map[string]NewClient{}

func GetSecretClient(ctx context.Context) (runtimeinterface.SecretClient, error) {
	cfg, ok := runtimecontext.FromNautesConfigContext(ctx)
	if !ok {
		return nil, fmt.Errorf("get nautes config from context failed")
	}

	NewClient, ok := SecretProviders[string(cfg.Secret.RepoType)]
	if !ok {
		return nil, fmt.Errorf("unknow sercret repo type")
	}
	return NewClient(cfg.Secret)

}
