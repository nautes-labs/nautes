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

package mock

import (
	"context"
	"errors"
	"os"

	runtimeinterface "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

type mock struct {
}

// CreateRole implements interfaces.SecretClient
func (*mock) CreateRole(ctx context.Context, clusterName string, role runtimeinterface.Role) error {
	return nil
}

// DeleteRole implements interfaces.SecretClient
func (*mock) DeleteRole(ctx context.Context, clusterName string, role runtimeinterface.Role) error {
	return nil
}

// GetRole implements interfaces.SecretClient
func (*mock) GetRole(ctx context.Context, clusterName string, role runtimeinterface.Role) (*runtimeinterface.Role, error) {
	return &role, nil
}

// GetSecretDatabaseName will return repo.CodeRepo.Permission as dbname
func (*mock) GetSecretDatabaseName(ctx context.Context, repo runtimeinterface.SecretInfo) (string, error) {
	dbName, ok := os.LookupEnv("TEST_SECRET_DB")
	if !ok {
		return "", errors.New("this is error")
	}
	return dbName, nil
}

// GetSecretKey will return repo.CodeRepo.User as key
func (*mock) GetSecretKey(ctx context.Context, repo runtimeinterface.SecretInfo) (string, error) {
	key, ok := os.LookupEnv("TEST_SECRET_KEY")
	if !ok {
		return "", errors.New("this is error")
	}
	return key, nil
}

// GrantPermission implements interfaces.SecretClient
func (*mock) GrantPermission(ctx context.Context, repo runtimeinterface.SecretInfo, destUser string, destEnv string) error {
	return nil
}

// RevokePermission implements interfaces.SecretClient
func (*mock) RevokePermission(ctx context.Context, repo runtimeinterface.SecretInfo, destUser string, destEnv string) error {
	return nil
}

func NewMock(cfg nautescfg.SecretRepo) (runtimeinterface.SecretClient, error) {
	return &mock{}, nil
}

func (m mock) GetAccessInfo(ctx context.Context, name string) (string, error) {
	kubeconfig, ok := os.LookupEnv("TEST_KUBECONFIG")
	if !ok {
		return "", errors.New("this is error")
	}
	return kubeconfig, nil
}

func (m mock) Logout() error {
	return nil
}

func (m mock) GetCABundle(ctx context.Context) (string, error) {
	return "", nil
}
