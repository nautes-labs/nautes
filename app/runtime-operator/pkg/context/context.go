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

package context

import (
	"context"

	runtimesecretclient "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	nautesctx "github.com/nautes-labs/nautes/pkg/context"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

const (
	ContextKeyConfig nautesctx.ContextKey = "runtime.deployment.config"
	ContextKeySecret nautesctx.ContextKey = "runtime.deployment.secretclient"
)

func NewNautesConfigContext(ctx context.Context, cfg nautescfg.Config) context.Context {
	return context.WithValue(ctx, ContextKeyConfig, cfg)
}

func FromNautesConfigContext(ctx context.Context) (nautescfg.Config, bool) {
	cfg, ok := ctx.Value(ContextKeyConfig).(nautescfg.Config)
	return cfg, ok
}

func NewSecretClientContext(ctx context.Context, client runtimesecretclient.SecretClient) context.Context {
	return context.WithValue(ctx, ContextKeySecret, client)
}

func FromSecretClientConetxt(ctx context.Context) (runtimesecretclient.SecretClient, bool) {
	client, ok := ctx.Value(ContextKeySecret).(runtimesecretclient.SecretClient)
	return client, ok
}
