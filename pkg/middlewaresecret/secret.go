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
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretIndexDirectoryInterface interface {
	ReloadSecretIndex(ctx context.Context) error
	// AddSecret adds a secret index to the secret index directory.
	AddSecret(ctx context.Context, middlewareMetadata MiddlewareMetadata, accessInfo v1alpha1.MiddlewareInitAccessInfo) (indexID string, err error)
	// CheckSecretIsAvailable checks if the secret is available.
	// If secret never fetched, return true.
	// If secret is fetched, return false.
	CheckSecretIsAvailable(ctx context.Context, indexID string) (bool, error)
	// GetSecret gets the secret data.
	// Secret can only be fetched once.
	GetSecret(ctx context.Context, indexID string) (*v1alpha1.MiddlewareInitAccessInfo, error)
}

type MiddlewareMetadata struct {
	ProductName     string `json:"productName"`
	EnvironmentName string `json:"environmentName"`
	ClusterName     string `json:"clusterName,omitempty"`
	RuntimeName     string `json:"runtimeName"`
	MiddlewareName  string `json:"middlewareName"`
}

// SecretIndexDirectory is a directory of secret indexes
type SecretIndexDirectory struct {
	// middlewareSecretIndexes is a map of middleware secrets
	// key: ID
	middlewareSecretIndexes map[string]middlewareSecretIndex
	// cmKey is the configmap name and namespace which stores the secret indexes
	cmKey types.NamespacedName
	// cmRevision is the configmap revision
	cmRevision string
	// tenantK8sClient is the k8s client of tenant cluster
	tenantK8sClient client.Client
}

// middlewareSecretIndex is the index of a middleware secret
type middlewareSecretIndex struct {
	ID       string             `json:"id,omitempty"`
	Metadata MiddlewareMetadata `json:"metadata,omitempty"`
	// Index is the index of the secret which stores the secret data
	Index SecretIndex `json:"index,omitempty"`
	// CreationTimestamp is the timestamp when the secret index is created
	CreationTimestamp metav1.Time `json:"creationTimestamp"`
	// FetchedTimestamp is the timestamp when the secret is fetched
	FetchedTimestamp *metav1.Time `json:"fetchedTimestamp,omitempty"`
	// Status is the status of the secret index
	// - InitPasswordNotFetched: the secret index is created but the secret is not fetched
	// - InitPasswordFetched: the secret index is created and the secret is fetched
	Status InitPasswordStatus `json:"status"`
}

type InitPasswordStatus string

const (
	InitPasswordFetched    InitPasswordStatus = "InitPasswordFetched"
	InitPasswordNotFetched InitPasswordStatus = "InitPasswordNotFetched"
)

type SecretIndex struct {
	types.NamespacedName
	Key string `json:"key"`
}

func NewSecretIndexDirectory(ctx context.Context, cmKey types.NamespacedName, tenantK8sClient client.Client) (SecretIndexDirectoryInterface, error) {
	return newSecretIndexDirectory(ctx, cmKey, tenantK8sClient)
}

func newSecretIndexDirectory(ctx context.Context, cmKey types.NamespacedName, tenantK8sClient client.Client) (*SecretIndexDirectory, error) {
	indexes, rev, err := getSecretIndexesFromTenantCluster(ctx, cmKey, tenantK8sClient)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("new secret indexes failed: %w", err)
		}
		indexes = map[string]middlewareSecretIndex{}
		rev = ""
	}

	secretData := &SecretIndexDirectory{
		middlewareSecretIndexes: indexes,
		cmKey:                   cmKey,
		cmRevision:              rev,
		tenantK8sClient:         tenantK8sClient,
	}

	return secretData, nil
}

// ReloadSecretIndex updates the secret index directory from tenant cluster
func (s *SecretIndexDirectory) ReloadSecretIndex(ctx context.Context) error {
	indexes, rev, err := getSecretIndexesFromTenantCluster(ctx, s.cmKey, s.tenantK8sClient)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("new secret indexes failed: %w", err)
	}

	s.middlewareSecretIndexes = indexes
	s.cmRevision = rev

	return nil
}

const secretKey = "secret"

// AddSecretIndex adds a secret index to the secret index directory
func (s *SecretIndexDirectory) AddSecret(ctx context.Context, middlewareMetadata MiddlewareMetadata, accessInfo v1alpha1.MiddlewareInitAccessInfo,
) (indexID string, err error) {
	indexID = uuid.New().String()

	if err := s.ReloadSecretIndex(ctx); err != nil {
		return "", fmt.Errorf("update secret index directory failed: %w", err)
	}

	index := SecretIndex{
		NamespacedName: types.NamespacedName{
			Name:      fmt.Sprintf("middleware-secret-%s", indexID),
			Namespace: s.cmKey.Namespace,
		},
		Key: secretKey,
	}

	middleware := middlewareSecretIndex{
		ID:                indexID,
		Metadata:          middlewareMetadata,
		Index:             index,
		CreationTimestamp: metav1.Now(),
		Status:            InitPasswordNotFetched,
	}

	if err := s.createSecret(ctx, accessInfo, index); err != nil {
		return "", fmt.Errorf("create secret failed: %w", err)
	}

	s.middlewareSecretIndexes[indexID] = middleware

	if err := s.uploadIndexesToTenantCluster(ctx); err != nil {
		return "", fmt.Errorf("update secret indexes to tenant cluster failed: %w", err)
	}

	return indexID, nil
}

// GetSecretIndex retrieves the secret index for a specific middleware in the given product, runtime, environment, and cluster.
// If index is found and the secret is not fetched, return the secret index.
func (s *SecretIndexDirectory) CheckSecretIsAvailable(_ context.Context, indexID string) (bool, error) {
	_, ok := s.middlewareSecretIndexes[indexID]
	if !ok {
		return false, nil
	}

	if s.middlewareSecretIndexes[indexID].Status == InitPasswordFetched {
		return false, nil
	}

	return true, nil
}

// GetSecret gets the secret data and disable the secret index.
// Secret will be deleted after get.
func (s *SecretIndexDirectory) GetSecret(ctx context.Context, indexID string) (*v1alpha1.MiddlewareInitAccessInfo, error) {
	if err := s.ReloadSecretIndex(ctx); err != nil {
		return nil, fmt.Errorf("update secret index directory failed: %w", err)
	}

	// Get secret index.
	secIndex, ok := s.middlewareSecretIndexes[indexID]
	if !ok {
		return nil, ErrorNotFound(fmt.Errorf("index %s not found", indexID))
	}

	sec, err := s.getSecret(ctx, secIndex.Index)
	if err != nil {
		return nil, fmt.Errorf("get secret failed: %w", err)
	}

	// If secret is fetched, delete it.
	if err := s.deleteSecret(ctx, secIndex.Index); err != nil {
		return nil, fmt.Errorf("delete secret failed: %w", err)
	}

	// Disable secret index.
	if err := s.disableIndex(indexID); err != nil {
		return sec, ErrorDisableIndexFailed(err)
	}

	if err := s.uploadIndexesToTenantCluster(ctx); err != nil {
		return sec, ErrorDisableIndexFailed(fmt.Errorf("update secret index directory failed: %w", err))
	}

	return sec, nil
}

// uploadIndexesToTenantCluster updates the secret index directory to tenant cluster
func (s *SecretIndexDirectory) uploadIndexesToTenantCluster(ctx context.Context) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.cmKey.Name,
			Namespace: s.cmKey.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, s.tenantK8sClient, cm, func() error {
		if cm.ResourceVersion != s.cmRevision {
			return fmt.Errorf("configmap %s has been updated", cm.Name)
		}

		data, err := convertSecretIndexesToConfigMapData(s.middlewareSecretIndexes)
		if err != nil {
			return err
		}
		cm.Data = data
		return nil
	})

	if err != nil {
		return fmt.Errorf("update secret index directory failed: %w", err)
	}
	return nil
}

func (s *SecretIndexDirectory) getSecret(ctx context.Context, index SecretIndex) (*v1alpha1.MiddlewareInitAccessInfo, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      index.Name,
			Namespace: index.Namespace,
		},
	}

	if err := s.tenantK8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, fmt.Errorf("get secret %s failed: %w", secret.Name, err)
	}

	sec := &v1alpha1.MiddlewareInitAccessInfo{}
	if err := json.Unmarshal(secret.Data[index.Key], sec); err != nil {
		return nil, fmt.Errorf("unmarshal secret %s failed: %w", secret.Name, err)
	}

	return sec, nil
}

func (s *SecretIndexDirectory) createSecret(ctx context.Context,
	accessInfo v1alpha1.MiddlewareInitAccessInfo,
	secretIndex SecretIndex,
) error {
	switch accessInfo.GetType() {
	case v1alpha1.MiddlewareAccessInfoTypeUserPassword:
		secretByte, err := json.Marshal(accessInfo.UserPassword)
		if err != nil {
			return fmt.Errorf("failed to marshal user password: %w", err)
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: secretIndex.Namespace,
				Name:      secretIndex.Name,
			},
			Data: map[string][]byte{
				secretIndex.Key: secretByte,
			},
		}
		if err := s.tenantK8sClient.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret in tenant cluster: %w", err)
		}
	default:
		return fmt.Errorf("unsupported access info type: %s", accessInfo.GetType())
	}

	return nil
}

func (s *SecretIndexDirectory) deleteSecret(ctx context.Context, secretIndex SecretIndex) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretIndex.Namespace,
			Name:      secretIndex.Name,
		},
	}
	if err := s.tenantK8sClient.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete secret in tenant cluster: %w", err)
	}
	return nil
}

func (s *SecretIndexDirectory) disableIndex(indexID string) error {
	middlewareSecretIndex, ok := s.middlewareSecretIndexes[indexID]
	if !ok {
		return fmt.Errorf("index %s not found", indexID)
	}
	middlewareSecretIndex.Status = InitPasswordFetched
	fetchTime := metav1.Now()
	middlewareSecretIndex.FetchedTimestamp = &fetchTime
	s.middlewareSecretIndexes[indexID] = middlewareSecretIndex

	return nil
}

// getSecretIndexesFromTenantCluster gets the secret indexes from tenant cluster
func getSecretIndexesFromTenantCluster(ctx context.Context, cmKey types.NamespacedName, tenantK8sClient client.Client,
) (indexes map[string]middlewareSecretIndex, configMapRevision string, err error) {
	cm := &corev1.ConfigMap{}
	if err := tenantK8sClient.Get(ctx, cmKey, cm); err != nil {
		return nil, "", fmt.Errorf("get config map %s failed: %w", cmKey.String(), err)
	}

	indexes, err = convertConfigMapDataToSecretIndexes(cm.Data)

	return indexes, cm.ResourceVersion, err
}

func convertConfigMapDataToSecretIndexes(data map[string]string) (map[string]middlewareSecretIndex, error) {
	indexes := map[string]middlewareSecretIndex{}

	for id, secretByte := range data {
		secIndex := &middlewareSecretIndex{}
		if err := json.Unmarshal([]byte(secretByte), secIndex); err != nil {
			return nil, fmt.Errorf("unmarshal secret data failed: %w", err)
		}
		indexes[id] = *secIndex
	}

	return indexes, nil
}

func convertSecretIndexesToConfigMapData(indexes map[string]middlewareSecretIndex) (map[string]string, error) {
	data := map[string]string{}

	for id, secrets := range indexes {
		secretByte, err := json.Marshal(secrets)
		if err != nil {
			return nil, fmt.Errorf("marshal secret data failed: %w", err)
		}
		data[id] = string(secretByte)
	}

	return data, nil
}
