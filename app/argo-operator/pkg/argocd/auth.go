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

package argocd

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	c client.Client
)

type AuthOperation interface {
	Login() error
	GetPassword() (string, error)
}

type auth struct {
	c *ArgocdClient
}

func NewArgocdAuth(c *ArgocdClient) AuthOperation {
	return &auth{c: c}
}

func init() {
	c, _ = client.New(config.GetConfigOrDie(), client.Options{})
}

type argoUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type argoResp struct {
	Token string `json:"token"`
}

// get token from argocd
func (a *auth) Login() error {
	url := a.c.client.url
	if url == "" {
		return fmt.Errorf("argocd server address cannot be empty")
	}

	requestAddress := fmt.Sprintf("%s/api/v1/session", url)
	username := "admin"
	password, err := a.GetPassword()
	if err != nil {
		return err
	}

	if password == "" {
		return fmt.Errorf("the password is empty")
	}

	user := &argoUser{
		Username: username,
		Password: password,
	}

	var res = &argoResp{}
	result, err := a.c.http.R().
		SetBody(user).
		SetResult(res).
		Post(requestAddress)
	if err != nil {
		return err
	}

	if result.StatusCode() == 200 {
		a.c.client.token = res.Token
		return nil
	}

	return fmt.Errorf("login argocd result: %s, argocd server: %s, http status code: %d", string(result.Body()), url, result.StatusCode())
}

// get argocd password from the secret key of kubernetes
func (a *auth) GetPassword() (string, error) {
	secretKey := "password"
	secret := &corev1.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: DefaultNamespace,
		Name:      AdminInitialAdminSecret,
	}
	err := c.Get(context.Background(), secretNamespaceName, secret)

	if err != nil {
		return "", err
	}

	return string(secret.Data[secretKey]), nil
}
