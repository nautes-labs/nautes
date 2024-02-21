// Copyright 2024 Nautes Authors
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

package middlewaresecret

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// m *Mock SecretIndexDirectoryInterface
type Mock struct {
	Indexes []SecretIndex
	Secrets map[string]v1alpha1.MiddlewareInitAccessInfo
}

func (m *Mock) ReloadSecretIndex(ctx context.Context) error {
	return nil
}

func (m *Mock) AddSecret(ctx context.Context, metadata MiddlewareMetadata, accessInfo v1alpha1.MiddlewareInitAccessInfo,
) (indexID string, err error) {
	indexID = uuid.New().String()

	secIndex := SecretIndex{
		NamespacedName: types.NamespacedName{
			Name:      indexID,
			Namespace: "",
		},
		Key: "",
	}

	m.Indexes = append(m.Indexes, secIndex)
	m.Secrets[indexID] = accessInfo

	return
}

func (m *Mock) CheckSecretIsAvailable(ctx context.Context, indexID string) (bool, error) {
	_, ok := m.Secrets[indexID]
	return ok, nil
}

func (m *Mock) GetSecret(ctx context.Context, indexID string) (*v1alpha1.MiddlewareInitAccessInfo, error) {
	accessInfo, ok := m.Secrets[indexID]
	if !ok {
		return nil, fmt.Errorf("secret not found, indexID %s", indexID)
	}
	return &accessInfo, nil
}
