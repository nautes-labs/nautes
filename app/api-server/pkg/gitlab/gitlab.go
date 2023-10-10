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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/xanzy/go-gitlab"
)

const (
	_ApiServerCertPath = "/opt/nautes/keypair/client.crt"
	_ApiServerKeyPath  = "/opt/nautes/keypair/client.key"
	SSLDirectory       = "/opt/nautes/cert"
)

type GitlabClient struct {
	client *gitlab.Client
}

func NewGitlabOperator() GitlabOperator {
	return &GitlabClient{}
}

func (g *GitlabClient) NewGitlabClient(address, token string) (GitlabOperator, error) {
	cert, err := GetCertificate(address)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(cert))

	srvCert, _ := tls.LoadX509KeyPair(_ApiServerCertPath, _ApiServerKeyPath)
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{srvCert},
		MinVersion:   tls.VersionTLS12,
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	oauthClient, err := gitlab.NewOAuthClient(token, gitlab.WithBaseURL(address), gitlab.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to initial gitlab client, %w", err)
	}

	g.client = oauthClient

	return g, nil
}

func GetCertificate(addr string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", err
	}

	hostname := u.Hostname()
	port := u.Port()
	path := fmt.Sprintf("%s/%s_%s.crt", SSLDirectory, hostname, port)
	cert, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		path := fmt.Sprintf("%s/%s.crt", SSLDirectory, hostname)
		cert, err = os.ReadFile(path)
		if err != nil {
			return "", err
		}
	}

	return string(cert), nil
}

func (g *GitlabClient) GetCurrentUser() (user *gitlab.User, res *gitlab.Response, err error) {
	user, res, err = g.client.Users.CurrentUser()
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) CreateProject(opt *gitlab.CreateProjectOptions, options ...gitlab.RequestOptionFunc) (project *gitlab.Project, res *gitlab.Response, err error) {
	opt.InitializeWithReadme = gitlab.Bool(true)
	project, res, err = g.client.Projects.CreateProject(opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) DeleteProject(pid interface{}) (res *gitlab.Response, err error) {
	res, err = g.client.Projects.DeleteProject(pid)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) UpdateProject(pid interface{}, opt *gitlab.EditProjectOptions) (project *gitlab.Project, res *gitlab.Response, err error) {
	project, res, err = g.client.Projects.EditProject(pid, opt)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (project *gitlab.Project, res *gitlab.Response, err error) {
	project, res, err = g.client.Projects.GetProject(pid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) (projects []*gitlab.Project, res *gitlab.Response, err error) {
	projects, res, err = g.client.Groups.ListGroupProjects(gid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) CreateGroup(opt *gitlab.CreateGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error) {
	group, res, err = g.client.Groups.CreateGroup(opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) DeleteGroup(gid interface{}, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error) {
	res, err = g.client.Groups.DeleteGroup(gid, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) UpdateGroup(gid interface{}, opt *gitlab.UpdateGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error) {
	group, res, err = g.client.Groups.UpdateGroup(gid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) GetGroup(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error) {
	group, res, err = g.client.Groups.GetGroup(gid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) (groups []*gitlab.Group, res *gitlab.Response, err error) {
	groups, res, err = g.client.Groups.ListGroups(opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) GetDeployKey(pid interface{}, deployKeyID int, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error) {
	key, res, err = g.client.DeployKeys.GetDeployKey(pid, deployKeyID, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListDeployKeys(pid interface{}, opt *gitlab.ListProjectDeployKeysOptions, options ...gitlab.RequestOptionFunc) (keys []*gitlab.ProjectDeployKey, res *gitlab.Response, err error) {
	keys, res, err = g.client.DeployKeys.ListProjectDeployKeys(pid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListAllDeployKeys(opt *gitlab.ListInstanceDeployKeysOptions, options ...gitlab.RequestOptionFunc) (keys []*gitlab.InstanceDeployKey, res *gitlab.Response, err error) {
	keys, res, err = g.client.DeployKeys.ListAllDeployKeys(opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) AddDeployKey(pid interface{}, opt *gitlab.AddDeployKeyOptions, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error) {
	key, res, err = g.client.DeployKeys.AddDeployKey(pid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) DeleteDeployKey(pid interface{}, deployKey int, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error) {
	res, err = g.client.DeployKeys.DeleteDeployKey(pid, deployKey, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) UpdateProjectDeployKey(pid interface{}, deployKey int, opt *gitlab.UpdateDeployKeyOptions, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error) {
	key, res, err = g.client.DeployKeys.UpdateDeployKey(pid, deployKey, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListProjects(search string) (projects []*gitlab.Project, res *gitlab.Response, err error) {
	projects, res, err = g.client.Projects.ListProjects(&gitlab.ListProjectsOptions{
		Search: gitlab.String(search),
	})
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) EnableProjectDeployKey(pid interface{}, deployKey int, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error) {
	key, res, err = g.client.DeployKeys.EnableDeployKey(pid, deployKey, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) GetProjectAccessToken(pid interface{}, id int, options ...gitlab.RequestOptionFunc) (token *gitlab.ProjectAccessToken, res *gitlab.Response, err error) {
	token, res, err = g.client.ProjectAccessTokens.GetProjectAccessToken(pid, id, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) ListProjectAccessToken(pid interface{}, opt *gitlab.ListProjectAccessTokensOptions, options ...gitlab.RequestOptionFunc) (tokens []*gitlab.ProjectAccessToken, res *gitlab.Response, err error) {
	tokens, res, err = g.client.ProjectAccessTokens.ListProjectAccessTokens(pid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) CreateProjectAccessToken(pid interface{}, opt *gitlab.CreateProjectAccessTokenOptions, options ...gitlab.RequestOptionFunc) (token *gitlab.ProjectAccessToken, res *gitlab.Response, err error) {
	token, res, err = g.client.ProjectAccessTokens.CreateProjectAccessToken(pid, opt, options...)
	if err != nil {
		return
	}

	return
}

func (g *GitlabClient) DeleteProjectAccessToken(pid interface{}, id int, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error) {
	res, err = g.client.ProjectAccessTokens.RevokeProjectAccessToken(pid, id, options...)
	if err != nil {
		return
	}

	return
}
