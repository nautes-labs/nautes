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

package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewSecretSync func(opt v1alpha1.Component, info *ComponentInitInfo) (SecretSync, error)

type SecretSync interface {
	Component

	// CreateSecret will create a secret object (sercret in kubernetes, file in host) from secret database to dest environment.
	CreateSecret(ctx context.Context, secretReq SecretRequest) error
	RemoveSecret(ctx context.Context, secretReq SecretRequest) error
}

type SecretRequest struct {
	Name        string
	Source      SecretInfo
	User        User
	Destination SecretRequestDestination
}

type SecretRequestDestination struct {
	Name   string
	Space  Space
	Format string
}
