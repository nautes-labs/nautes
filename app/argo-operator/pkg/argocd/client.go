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
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	AdminInitialAdminSecret = "argocd-initial-admin-secret"
	DefaultNamespace        = "argocd"
	DefaultServiceAccount   = "default"
)

type Client struct {
	token string
	url   string
}

type ArgocdClient struct {
	client         *Client
	http           *resty.Client
	ArgocdAuth     AuthOperation
	ArgocdCluster  ClusterOperation
	ArgocdCodeRepo CoderepoOperation
}

func NewArgocd(url string) (c *ArgocdClient) {
	c = &ArgocdClient{
		client: &Client{url: url},
		http:   resty.New().SetTimeout(30 * time.Second),
	}
	c.http.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	c.ArgocdAuth = NewArgocdAuth(c)
	c.ArgocdCluster = NewArgocdCluster(c)
	c.ArgocdCodeRepo = NewArgocdCodeRepo(c)
	return
}

func (c *ArgocdClient) Auth() AuthOperation {
	return c.ArgocdAuth
}

func (c *ArgocdClient) Cluster() ClusterOperation {
	return c.ArgocdCluster
}

func (c *ArgocdClient) CodeRepo() CoderepoOperation {
	return c.ArgocdCodeRepo
}

func spliceBearerToken(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}
