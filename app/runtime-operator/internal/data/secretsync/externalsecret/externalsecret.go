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

package externalsecret

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	externalsecretv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func init() {
	utilruntime.Must(externalsecretv1alpha1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

type ExternalSecret struct {
	k8sClient   client.Client
	secretType  configs.SecretStoreType
	nautesCFG   configs.Config
	clusterName string
}

// NewExternalSecret returns a new instance for secret synchronization.
func NewExternalSecret(_ v1alpha1.Component, info *syncer.ComponentInitInfo) (syncer.SecretSync, error) {
	if info.ClusterConnectInfo.ClusterKind != v1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.ClusterKind)
	}

	k8sClient, err := client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	es := &ExternalSecret{
		k8sClient:   k8sClient,
		secretType:  info.NautesConfig.Secret.RepoType,
		nautesCFG:   info.NautesConfig,
		clusterName: info.ClusterName,
	}
	return es, nil
}

// CleanUp represents when the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (es *ExternalSecret) CleanUp() error {
	return nil
}

func (es *ExternalSecret) GetComponentMachineAccount() *syncer.MachineAccount {
	return nil
}

// CreateSecret creates the secret synchronization object, the synchronization result of the secret depends on the space type in the destination environment.
// If the space type of the request is Kubernetes then is a Secret name else is a folder name on Host.
func (es *ExternalSecret) CreateSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	secretStore, err := buildBaseSecretStore(secretReq)
	if err != nil {
		return fmt.Errorf("build secret store failed: %w", err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, es.k8sClient, secretStore, func() error {
		spec, err := es.convertReqToSecretStoreSpec(secretReq)
		if err != nil {
			return err
		}
		secretStore.Spec = *spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync secret store failed: %w", err)
	}

	externalSecret, err := buildBaseExternalSecret(secretReq)
	if err != nil {
		return fmt.Errorf("build external secret failed: %w", err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, es.k8sClient, externalSecret, func() error {
		spec, err := es.convertReqToExternalSecretSpec(secretReq)
		if err != nil {
			return err
		}
		externalSecret.Spec = *spec
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync external secret failed: %w", err)
	}

	return nil
}

// RemoveSecret removes the secret object from the dest environment of the request.
func (es *ExternalSecret) RemoveSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	secretStore, err := buildBaseSecretStore(secretReq)
	if err != nil {
		return fmt.Errorf("build secret store failed: %w", err)
	}
	if err := es.k8sClient.Delete(ctx, secretStore); client.IgnoreNotFound(err) != nil {
		return err
	}

	externalSecret, err := buildBaseExternalSecret(secretReq)
	if err != nil {
		return fmt.Errorf("build external secret failed: %w", err)
	}
	if err := es.k8sClient.Delete(ctx, externalSecret); client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

// buildBaseSecretStore builds a secret store instance for secret synchronization.
func buildBaseSecretStore(req syncer.SecretRequest) (*externalsecretv1alpha1.SecretStore, error) {
	if req.Destination.Space.SpaceType != syncer.SpaceTypeKubernetes {
		return nil, fmt.Errorf("space type %s is not supported", req.Destination.Space.SpaceType)
	}
	ns := req.Destination.Space.Kubernetes.Namespace
	return &externalsecretv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildSecretStoreName(req.Name),
			Namespace: ns,
		},
	}, nil
}

// buildBaseExternalSecret builds an external secret instance for secret synchronization.
func buildBaseExternalSecret(req syncer.SecretRequest) (*externalsecretv1alpha1.ExternalSecret, error) {
	if req.Destination.Space.SpaceType != syncer.SpaceTypeKubernetes {
		return nil, fmt.Errorf("space type %s is not supported", req.Destination.Space.SpaceType)
	}

	ns := req.Destination.Space.Kubernetes.Namespace
	return &externalsecretv1alpha1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildExternalSecretName(req.Name),
			Namespace: ns,
		},
	}, nil
}

// convertReqToSecretStoreSpec builds a secret store spec instance for secret synchronization.
func (es *ExternalSecret) convertReqToSecretStoreSpec(req syncer.SecretRequest) (*externalsecretv1alpha1.SecretStoreSpec, error) {
	// The GetCABundle gets CA cert to access Vault.
	caBundle, err := utils.GetCABundle(es.nautesCFG.Secret.Vault.Addr)
	if err != nil {
		return nil, err
	}

	sa := ""
	for _, serviceAccount := range req.AuthInfo.ServiceAccounts {
		if serviceAccount.Namespace == req.Destination.Space.Kubernetes.Namespace {
			sa = serviceAccount.ServiceAccount
			break
		}
	}

	ssSpec := &externalsecretv1alpha1.SecretStoreSpec{
		Provider: &externalsecretv1alpha1.SecretStoreProvider{
			Vault: &externalsecretv1alpha1.VaultProvider{
				Auth: externalsecretv1alpha1.VaultAuth{
					Kubernetes: &externalsecretv1alpha1.VaultKubernetesAuth{
						Path: es.clusterName,
						ServiceAccountRef: &v1.ServiceAccountSelector{
							Name: sa,
						},
						Role: req.AuthInfo.AccountName,
					},
				},
				// The Server is the Vault server's address.
				Server: es.nautesCFG.Secret.Vault.Addr,
				// The Path is the secret path in the Vault.
				Path:     getVaultSecretPath(req.Source.Type),
				Version:  "v2",
				CABundle: caBundle,
			},
		},
	}
	return ssSpec, nil
}

const (
	vaultSecretEngineGitAccessTokenKey = "accesstoken"
	vaultSecretEngineGitReadOnlyKey    = "deploykey"
	vaultSecretEngineGitReadWriteKey   = "deploykey"
	externalSecretRefSecretStoreKind   = "SecretStore"
)

// convertReqToExternalSecretSpec builds an external secret spec instance for secret synchronization.
func (es *ExternalSecret) convertReqToExternalSecretSpec(req syncer.SecretRequest) (*externalsecretv1alpha1.ExternalSecretSpec, error) {
	var key string
	switch req.Source.CodeRepo.Permission {
	case syncer.CodeRepoPermissionAccessToken:
		key = vaultSecretEngineGitAccessTokenKey
	case syncer.CodeRepoPermissionReadOnly:
		key = vaultSecretEngineGitReadOnlyKey
	case syncer.CodeRepoPermissionReadWrite:
		key = vaultSecretEngineGitReadWriteKey
	}

	format := req.Destination.Format
	secPath := es.getVaultSecretPath(req.Source)
	esSpec := &externalsecretv1alpha1.ExternalSecretSpec{
		SecretStoreRef: externalsecretv1alpha1.SecretStoreRef{
			Name: buildSecretStoreName(req.Name),
			Kind: externalSecretRefSecretStoreKind,
		},
		Target: externalsecretv1alpha1.ExternalSecretTarget{
			Name:           req.Destination.Name,
			CreationPolicy: externalsecretv1alpha1.Owner,
		},
		Data: []externalsecretv1alpha1.ExternalSecretData{
			{
				SecretKey: key,
				RemoteRef: externalsecretv1alpha1.ExternalSecretDataRemoteRef{
					Key:                secPath,
					Property:           key,
					ConversionStrategy: externalsecretv1alpha1.ExternalSecretConversionDefault,
				},
			},
		},
	}

	if format != "" {
		data := map[string]string{}
		if err := yaml.Unmarshal([]byte(format), data); err != nil {
			return nil, fmt.Errorf("format analysis failed: %w", err)
		}
		esSpec.Target.Template = &externalsecretv1alpha1.ExternalSecretTemplate{
			Data: data,
		}
	}
	return esSpec, nil
}

// getVaultSecretPath returns a key of secret in Vault.
func (es *ExternalSecret) getVaultSecretPath(secInfo syncer.SecretInfo) string {
	return fmt.Sprintf("%s/%s/%s/%s", secInfo.CodeRepo.ProviderType,
		secInfo.CodeRepo.ID,
		secInfo.CodeRepo.User,
		secInfo.CodeRepo.Permission)
}

func getVaultSecretPath(secretType syncer.SecretType) *string {
	var str string
	switch secretType {
	case syncer.SecretTypeCodeRepo:
		str = "git"
	case syncer.SecretTypeArtifactRepo:
		str = "repo"
	}
	return &str
}

func buildSecretStoreName(reqName string) string {
	return fmt.Sprintf("ss-%s", reqName)
}

func buildExternalSecretName(reqName string) string {
	return fmt.Sprintf("es-%s", reqName)
}
