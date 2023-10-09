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

package gitlab

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	loadcert "github.com/nautes-labs/nautes/pkg/util/loadcerts"
	"github.com/xanzy/go-gitlab"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GitLab struct {
	*gitlab.Client
	DefaultProjectName string
	provider           baseinterface.CodeRepoProvider
}

func NewProvider(token string, codeRepoProvider nautescrd.CodeRepoProvider, cfg nautescfg.Config) (baseinterface.ProductProvider, error) {
	return NewGitlab(token, codeRepoProvider, cfg)
}

func NewGitlab(token string, codeRepoProvider nautescrd.CodeRepoProvider, cfg nautescfg.Config) (*GitLab, error) {
	apiURL := fmt.Sprintf("%s/api/v4", codeRepoProvider.Spec.ApiServer)
	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(apiURL),
	}
	if strings.HasPrefix(apiURL, "https://") {
		httpClient, err := getHttpsClient()
		if err != nil {
			return nil, err
		}
		opts = append(opts, gitlab.WithHTTPClient(httpClient))
	}
	client, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get gitlab client: %w", err)
	}

	return &GitLab{
		Client:             client,
		DefaultProjectName: cfg.Git.DefaultProductName,
		provider: baseinterface.CodeRepoProvider{
			Name: codeRepoProvider.Name,
		},
	}, nil
}

func getHttpsClient() (*http.Client, error) {
	caCertPool, err := loadcert.GetCertPool()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    caCertPool,
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	return client, nil
}

func (g *GitLab) GetProducts() ([]nautescrd.Product, error) {
	products := []nautescrd.Product{}
	_, resp, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
		},
		Search: &g.DefaultProjectName,
	})
	if err != nil {
		return nil, fmt.Errorf("get project list failed: %w", err)
	}

	loopTimes := (float32(resp.TotalItems) / float32(resp.ItemsPerPage)) + 1
	orderKey := "id"
	for i := float32(1); i < loopTimes; i++ {
		gitlabProjects, _, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    int(i),
				PerPage: 50,
			},
			Search:  &g.DefaultProjectName,
			OrderBy: &orderKey,
		})
		if err != nil {
			return nil, fmt.Errorf("get project list failed: %w", err)
		}
		for _, project := range gitlabProjects {
			products = append(products, nautescrd.Product{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("product-%d", project.Namespace.ID),
				},
				Spec: nautescrd.ProductSpec{
					Name:         project.Namespace.Path,
					MetaDataPath: project.SSHURLToRepo,
				},
			})
		}
	}

	newProducts := []nautescrd.Product{}
	var restProducts []nautescrd.Product
	for i, product := range products {
		restProducts = products[i+1:]
		isDuplicate := false
		for _, nextProduct := range restProducts {
			if product.Name == nextProduct.Name {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			newProducts = append(newProducts, product)
		}
	}

	return newProducts, nil
}

func (g *GitLab) GetProductMeta(_ context.Context, id string) (baseinterface.ProductMeta, error) {
	productMeta := baseinterface.ProductMeta{}

	group, resp, err := g.Groups.GetGroup(id, nil)
	if err != nil {
		return productMeta, fmt.Errorf("get group info failed. code %d: %w", resp.Response.StatusCode, err)
	}

	nameWithNamespace := fmt.Sprintf("%s/%s", group.Path, g.DefaultProjectName)
	searchWithNamespace := true
	projects, resp, err := g.Projects.ListProjects(&gitlab.ListProjectsOptions{
		SearchNamespaces: &searchWithNamespace,
		Search:           &nameWithNamespace,
	})
	if err != nil {
		return productMeta, fmt.Errorf("get meta data failed, code %d: %w", resp.Response.StatusCode, err)
	}
	if len(projects) != 1 {
		return productMeta, fmt.Errorf("meta data is nil or more than one")
	}

	return baseinterface.ProductMeta{
		ID:     id,
		MetaID: fmt.Sprintf("%d", projects[0].ID),
	}, nil
}

func (g *GitLab) GetCodeRepoProvider(_ context.Context) (baseinterface.CodeRepoProvider, error) {
	return g.provider, nil
}
