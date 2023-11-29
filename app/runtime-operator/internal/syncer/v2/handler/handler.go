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

package handler

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

func NewHandler(info *component.ComponentInitInfo) (task.ResourceRequestHandler, error) {
	var k8sClient client.Client
	if info.ClusterConnectInfo.ClusterKind == v1alpha1.CLUSTER_KIND_KUBERNETES {
		var err error
		k8sClient, err = client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
		if err != nil {
			return nil, err
		}
	}

	return &deployer{
		k8sClient:  k8sClient,
		components: info.Components,
	}, nil
}

type deployer struct {
	k8sClient  client.Client
	components *component.ComponentList
}

func (t *deployer) CreateResource(ctx context.Context, space component.Space, authInfo component.AuthInfo, req component.RequestResource) error {
	switch req.Type {
	case component.ResourceTypeCodeRepoSSHKey:
		return t.CreateSSHKey(ctx, space, authInfo, req)
	case component.ResourceTypeCodeRepoAccessToken:
		return t.CreateAccessToken(ctx, space, authInfo, req)
	case component.ResourceTypeCAFile:
		return t.CreateCAFile(ctx, space, authInfo, req)
	case component.ResourceTypeEmptyDir:
		return nil
	default:
		return fmt.Errorf("unknown request type %s", req.Type)
	}
}

func (t *deployer) DeleteResource(ctx context.Context, space component.Space, req component.RequestResource) error {
	switch req.Type {
	case component.ResourceTypeCodeRepoSSHKey:
		return t.DeleteSSHKey(ctx, space, req)
	case component.ResourceTypeCodeRepoAccessToken:
		return t.DeleteAccessToken(ctx, space, req)
	case component.ResourceTypeCAFile:
		return t.DeleteCAFile(ctx, space, req)
	case component.ResourceTypeEmptyDir:
		return nil
	default:
		return fmt.Errorf("unknown request type %s", req.Type)
	}
}

func (t *deployer) CreateSSHKey(ctx context.Context, space component.Space, authInfo component.AuthInfo, req component.RequestResource) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.CreateSecret(ctx, component.SecretRequest{
		Name:     req.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: &authInfo,
		Destination: component.SecretRequestDestination{
			Name:   req.ResourceName,
			Space:  space,
			Format: `id_rsa: '{{ .deploykey | toString }}'`,
		},
	})
	if err != nil {
		return fmt.Errorf("create secret %s in space failed: %w", req.ResourceName, err)
	}

	return nil
}

func (t *deployer) DeleteSSHKey(ctx context.Context, space component.Space, req component.RequestResource) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.RemoveSecret(ctx, component.SecretRequest{
		Name:     req.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: nil,
		Destination: component.SecretRequestDestination{
			Name:  req.ResourceName,
			Space: space,
		},
	})

	if err != nil {
		return fmt.Errorf("delete secret %s in space failed: %w", req.ResourceName, err)
	}
	return nil
}

func (t *deployer) CreateAccessToken(ctx context.Context, space component.Space, authInfo component.AuthInfo, req component.RequestResource) error {
	tokenReq := req.AccessToken

	err := t.components.SecretSync.CreateSecret(ctx, component.SecretRequest{
		Name:     req.ResourceName,
		Source:   tokenReq.SecretInfo,
		AuthInfo: &authInfo,
		Destination: component.SecretRequestDestination{
			Name:   req.ResourceName,
			Space:  space,
			Format: `token: '{{ .accesstoken | toString }}'`,
		},
	})
	if err != nil {
		return fmt.Errorf("create secret %s in space failed: %w", req.ResourceName, err)
	}

	return nil
}

func (t *deployer) DeleteAccessToken(ctx context.Context, space component.Space, req component.RequestResource) error {
	tokenReq := req.AccessToken

	err := t.components.SecretSync.RemoveSecret(ctx, component.SecretRequest{
		Name:     req.ResourceName,
		Source:   tokenReq.SecretInfo,
		AuthInfo: nil,
		Destination: component.SecretRequestDestination{
			Name:  req.ResourceName,
			Space: space,
		},
	})

	if err != nil {
		return fmt.Errorf("delete secret %s in space failed: %w", req.ResourceName, err)
	}
	return nil
}

func (t *deployer) CreateCAFile(ctx context.Context, space component.Space, authInfo component.AuthInfo, req component.RequestResource) error {
	caReq := req.CAFile

	caByte, err := utils.GetCABundle(caReq.URL)
	if err != nil {
		return fmt.Errorf("load ca failed, domain, %s: %w", caReq.URL, err)
	}
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.ResourceName,
			Namespace: space.Kubernetes.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, t.k8sClient, sec, func() error {
		sec.Data = map[string][]byte{
			"ca.crt": caByte,
		}
		return nil
	})
	return err
}

func (t *deployer) DeleteCAFile(ctx context.Context, space component.Space, req component.RequestResource) error {
	err := t.k8sClient.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.ResourceName,
			Namespace: space.Kubernetes.Namespace,
		},
	})

	return client.IgnoreNotFound(err)
}
