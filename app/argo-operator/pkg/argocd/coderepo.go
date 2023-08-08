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
	"encoding/json"
	"fmt"
	"net/url"
)

type CoderepoOperation interface {
	CreateRepository(repoName, key, repourl string) error
	UpdateRepository(repoName, key, repourl string) error
	GetRepositoryInfo(repoURL string) (*CodeRepoResponse, error)
	DeleteRepository(repoURL string) error
}

type codeRepo struct {
	argocd *ArgocdClient
}

type CodeRepoResponse struct {
	Name            string          `json:"name"`
	ConnectionState ConnectionState `json:"connectionState"`
	Repo            string          `json:"repo"`
}

func NewArgocdCodeRepo(argocd *ArgocdClient) CoderepoOperation {
	return &codeRepo{argocd: argocd}
}

type RepositoryData struct {
	Name          string `json:"name"`
	Project       string `json:"project"`
	Repo          string `json:"repo"`
	Type          string `json:"type"`
	SSHPrivateKey string `json:"sshPrivateKey"`
}

func (c *codeRepo) CreateRepository(repoName, key, url string) error {
	repoType := "git"
	data := &RepositoryData{
		Name:          repoName,
		Repo:          url,
		Type:          repoType,
		SSHPrivateKey: key,
	}
	requestAddress := fmt.Sprintf("%s/api/v1/repositories", c.argocd.client.url)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		SetBody(data).
		Post(requestAddress)
	if err != nil {
		return err
	}

	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("failed to create repository, code: %d, reponse: %s", result.StatusCode(), string(result.Body()))
}

func (c *codeRepo) UpdateRepository(repoName, key, repourl string) error {
	repoType := "git"
	data := &RepositoryData{
		Name:          repoName,
		Repo:          repourl,
		Type:          repoType,
		SSHPrivateKey: key,
	}
	encodedurl := url.QueryEscape(repourl)
	requestAddress := fmt.Sprintf("%s/api/v1/repositories/%s", c.argocd.client.url, encodedurl)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		SetBody(data).
		Put(requestAddress)
	if err != nil {
		return err
	}

	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("failed to update repository, code: %d, reponse: %s", result.StatusCode(), string(result.Body()))
}

func (c *codeRepo) GetRepositoryInfo(repo string) (*CodeRepoResponse, error) {
	escapeRepo := url.QueryEscape(repo)
	url := spliceRequestCodeRepoUrl(c.argocd.client.url, escapeRepo)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		Get(url)

	if err != nil {
		return nil, err
	}

	if result.StatusCode() == 404 {
		errNotFound := &ErrorNotFound{}
		err = json.Unmarshal(result.Body(), errNotFound)
		if err != nil {
			return nil, err
		}
		return nil, errNotFound
	}

	if result.StatusCode() == 200 {
		response := &CodeRepoResponse{}
		err = json.Unmarshal(result.Body(), response)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, fmt.Errorf("failed to get repository result: %s", result.Body())
}

func (c *codeRepo) DeleteRepository(repourl string) error {
	repoName := url.QueryEscape(repourl)
	url := spliceRequestCodeRepoUrl(c.argocd.client.url, repoName)
	bearerToken := spliceBearerToken(c.argocd.client.token)
	authorization := "Authorization"

	result, err := c.argocd.http.R().
		SetHeader(authorization, bearerToken).
		Delete(url)
	if err != nil {
		return err
	}

	if result.StatusCode() == 200 {
		return nil
	}

	return fmt.Errorf("failed to delete repository info, code: %d, reponse: %s", result.StatusCode(), string(result.Body()))
}

func spliceRequestCodeRepoUrl(host, name string) string {
	return fmt.Sprintf("%s/api/v1/repositories/%s", host, name)
}
