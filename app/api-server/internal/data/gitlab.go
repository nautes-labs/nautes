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

package data

import (
	"context"
	"encoding/json"
	"fmt"

	commonv1 "github.com/nautes-labs/nautes/api/api-server/common/v1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	gitlabclient "github.com/nautes-labs/nautes/app/api-server/pkg/gitlab"
	"github.com/xanzy/go-gitlab"
)

type gitlabRepo struct {
	url    string
	client gitlabclient.GitlabOperator
}

type ProjectDeployKey struct {
}

func NewGitlabRepo(url string, client gitlabclient.GitlabOperator) (*gitlabRepo, error) {
	return &gitlabRepo{url: url, client: client}, nil
}

func (g *gitlabRepo) GetCurrentUser(ctx context.Context) (user string, email string, err error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return
	}
	currentUser, _, err := client.GetCurrentUser()
	if err != nil {
		return
	}

	return currentUser.Username, currentUser.Email, nil
}

func (g *gitlabRepo) CreateCodeRepo(ctx context.Context, gid int, options *biz.GitCodeRepoOptions) (*biz.Project, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(options.Gitlab)
	if err != nil {
		return nil, err
	}

	createProjectOptions := &gitlab.CreateProjectOptions{}
	err = json.Unmarshal(bytes, createProjectOptions)
	if err != nil {
		return nil, err
	}

	if createProjectOptions.NamespaceID == nil {
		createProjectOptions.NamespaceID = &gid
	}

	project, _, err := client.CreateProject(createProjectOptions)
	if err != nil {
		return nil, err
	}

	return &biz.Project{
		ID:                int32(project.ID),
		Name:              project.Name,
		Path:              project.Path,
		WebUrl:            project.WebURL,
		SshUrlToRepo:      project.SSHURLToRepo,
		HttpUrlToRepo:     project.HTTPURLToRepo,
		PathWithNamespace: project.PathWithNamespace,
	}, nil
}

func (g *gitlabRepo) DeleteCodeRepo(ctx context.Context, pid interface{}) error {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return err
	}
	_, err = client.DeleteProject(pid)
	if err != nil {
		return err
	}

	return nil
}

func (g *gitlabRepo) UpdateCodeRepo(ctx context.Context, pid interface{}, options *biz.GitCodeRepoOptions) (*biz.Project, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(options.Gitlab)
	if err != nil {
		return nil, err
	}

	editProjectOptions := &gitlab.EditProjectOptions{}
	err = json.Unmarshal(bytes, editProjectOptions)
	if err != nil {
		return nil, err
	}

	project, _, err := client.UpdateProject(pid, editProjectOptions)
	if err != nil {
		return nil, err
	}

	return &biz.Project{
		ID:                int32(project.ID),
		Name:              project.Name,
		Path:              project.Path,
		Visibility:        string(project.Visibility),
		Description:       project.Description,
		WebUrl:            project.WebURL,
		SshUrlToRepo:      project.SSHURLToRepo,
		HttpUrlToRepo:     project.HTTPURLToRepo,
		PathWithNamespace: project.PathWithNamespace,
	}, nil
}

func (g *gitlabRepo) GetCodeRepo(ctx context.Context, pid interface{}) (*biz.Project, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	project, res, err := client.GetProject(pid, &gitlab.GetProjectOptions{})
	if err != nil && res != nil && res.StatusCode == 403 {
		return nil, commonv1.ErrorNoAuthorization("no permission to get project, err: %s", err)
	}

	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorProjectNotFound("failed to get project, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	return &biz.Project{
		ID:                int32(project.ID),
		Name:              project.Name,
		Visibility:        string(project.Visibility),
		Description:       project.Description,
		Path:              project.Path,
		WebUrl:            project.WebURL,
		SshUrlToRepo:      project.SSHURLToRepo,
		HttpUrlToRepo:     project.HTTPURLToRepo,
		PathWithNamespace: project.PathWithNamespace,
		Namespace: &biz.ProjectNamespace{
			ID:        project.Namespace.ID,
			Name:      project.Namespace.Name,
			Path:      project.Namespace.Path,
			Kind:      project.Namespace.Kind,
			FullPath:  project.Namespace.FullPath,
			AvatarURL: project.Namespace.AvatarURL,
			WebURL:    project.Namespace.WebURL,
		},
	}, nil
}

func (g *gitlabRepo) CreateGroup(ctx context.Context, git *biz.GitGroupOptions) (*biz.Group, error) {
	opts := &gitlab.CreateGroupOptions{}

	if git != nil && git.Gitlab != nil {
		jsonData, _ := json.Marshal(git.Gitlab)
		err := json.Unmarshal(jsonData, opts)
		if err != nil {
			return nil, err
		}
	}

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	group, res, err := client.CreateGroup(opts)
	if err != nil && res != nil && res.StatusCode == 403 {
		return nil, commonv1.ErrorNoAuthorization("no permission to create group, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	return &biz.Group{
		ID:          int32(group.ID),
		Name:        group.Name,
		Visibility:  string(group.Visibility),
		Description: group.Description,
		Path:        group.Path,
		WebUrl:      group.WebURL,
		ParentId:    int32(group.ParentID),
	}, nil
}

// DeleteGroup Deletes group in gitlab
func (g *gitlabRepo) DeleteGroup(ctx context.Context, gid interface{}) error {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return err
	}

	res, err := client.DeleteGroup(gid)
	if err != nil && res != nil && res.StatusCode == 403 {
		return commonv1.ErrorNoAuthorization("no permission to delete group, err: %s", err)
	}

	if err != nil {
		return err
	}

	return nil
}

func (g *gitlabRepo) UpdateGroup(ctx context.Context, gid interface{}, git *biz.GitGroupOptions) (*biz.Group, error) {
	opts := &gitlab.UpdateGroupOptions{}

	if git != nil && git.Gitlab != nil {
		jsonData, _ := json.Marshal(git.Gitlab)
		err := json.Unmarshal(jsonData, opts)
		if err != nil {
			return nil, err
		}
	}

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	group, _, err := client.UpdateGroup(gid, opts)
	if err != nil {
		return nil, err
	}

	return &biz.Group{
		ID:          int32(group.ID),
		Name:        group.Name,
		Visibility:  string(group.Visibility),
		Description: group.Description,
		Path:        group.Path,
		WebUrl:      group.WebURL,
		ParentId:    int32(group.ParentID),
	}, nil
}

func (g *gitlabRepo) GetGroup(ctx context.Context, gid interface{}) (*biz.Group, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	group, res, err := client.GetGroup(gid, &gitlab.GetGroupOptions{})
	if err != nil && res != nil && res.StatusCode == 403 {
		return nil, commonv1.ErrorNoAuthorization("no access to the server, err: %s", err)
	}

	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorGroupNotFound("failed to get group, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	return &biz.Group{
		ID:          int32(group.ID),
		Name:        group.Name,
		Visibility:  string(group.Visibility),
		Description: group.Description,
		Path:        group.Path,
		WebUrl:      group.WebURL,
		ParentId:    int32(group.ParentID),
	}, nil
}

func (g *gitlabRepo) ListGroupCodeRepos(ctx context.Context, gid interface{}, opts *biz.ListGroupProjectsOptions) ([]*biz.Project, error) {
	var projects []*gitlab.Project
	var result []*biz.Project

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	opt := &gitlab.ListGroupProjectsOptions{Search: gitlab.String(opts.Search), ListOptions: gitlab.ListOptions{Page: opts.Page, PerPage: opts.PerPage}}
	projects, _, err = client.ListGroupProjects(gid, opt)
	if err != nil {
		return nil, err
	}

	result = g.convertProject(projects, result)

	return result, nil
}

func (*gitlabRepo) convertProject(projects []*gitlab.Project, result []*biz.Project) []*biz.Project {
	for _, project := range projects {
		result = append(result, &biz.Project{
			ID:                int32(project.ID),
			Name:              project.Name,
			Visibility:        string(project.Visibility),
			Description:       project.Description,
			Path:              project.Path,
			WebUrl:            project.WebURL,
			SshUrlToRepo:      project.SSHURLToRepo,
			HttpUrlToRepo:     project.HTTPURLToRepo,
			PathWithNamespace: project.PathWithNamespace,
			Namespace: &biz.ProjectNamespace{
				ID:        project.Namespace.ID,
				Path:      project.Namespace.Path,
				Name:      project.Namespace.Name,
				Kind:      project.Namespace.Kind,
				FullPath:  project.Namespace.FullPath,
				WebURL:    project.Namespace.WebURL,
				AvatarURL: project.Namespace.AvatarURL,
			},
		})
	}

	return result
}

func (g *gitlabRepo) ListCodeRepos(ctx context.Context, search string) ([]*biz.Project, error) {
	var projects []*gitlab.Project
	var result []*biz.Project

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	projects, _, err = client.ListProjects(search)
	if err != nil {
		return nil, err
	}

	result = g.convertProject(projects, result)

	return result, nil
}

func (g *gitlabRepo) ListAllGroups(ctx context.Context) ([]*biz.Group, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	var Groups []*biz.Group
	groups, _, err := client.ListGroups(nil)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		Groups = append(Groups, &biz.Group{
			ID:          int32(group.ID),
			Name:        group.Name,
			Visibility:  string(group.Visibility),
			Description: group.Description,
			Path:        group.Path,
			WebUrl:      group.WebURL,
			ParentId:    int32(group.ParentID),
		})
	}

	return Groups, nil
}

func (g *gitlabRepo) ListAllDeployKeys(ctx context.Context, opt *biz.ListOptions) ([]*biz.ProjectDeployKey, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	keys, res, err := client.ListAllDeployKeys(&gitlab.ListInstanceDeployKeysOptions{
		ListOptions: gitlab.ListOptions{
			Page:    opt.Page,
			PerPage: opt.PerPage,
		},
	})
	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorDeploykeyNotFound("failed to list deploykeys, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	projectDeployKeys := []*biz.ProjectDeployKey{}
	projectsWithWriteAccess := []*biz.DeployKeyProject{}
	for _, key := range keys {
		for _, deployKeyProject := range key.ProjectsWithWriteAccess {
			projectsWithWriteAccess = append(projectsWithWriteAccess, &biz.DeployKeyProject{
				ID:                deployKeyProject.ID,
				Description:       deployKeyProject.Description,
				Name:              deployKeyProject.Name,
				NameWithNamespace: deployKeyProject.NameWithNamespace,
				Path:              deployKeyProject.Path,
				PathWithNamespace: deployKeyProject.PathWithNamespace,
				CreatedAt:         deployKeyProject.CreatedAt,
			})
		}
		projectDeployKey := &biz.ProjectDeployKey{
			Title:                   key.Title,
			ID:                      key.ID,
			Key:                     key.Key,
			CreatedAt:               key.CreatedAt,
			Fingerprint:             key.Fingerprint,
			ProjectsWithWriteAccess: projectsWithWriteAccess,
		}
		projectDeployKeys = append(projectDeployKeys, projectDeployKey)
	}

	return projectDeployKeys, nil
}

func (g *gitlabRepo) ListDeployKeys(ctx context.Context, pid interface{}, opt *biz.ListOptions) ([]*biz.ProjectDeployKey, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	keys, res, err := client.ListDeployKeys(pid, &gitlab.ListProjectDeployKeysOptions{Page: opt.Page, PerPage: opt.PerPage})
	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorDeploykeyNotFound("failed to list deploykeys, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	projectDeployKeys := []*biz.ProjectDeployKey{}
	for _, key := range keys {
		projectDeployKey := &biz.ProjectDeployKey{
			Title:     key.Title,
			ID:        key.ID,
			Key:       key.Key,
			CreatedAt: key.CreatedAt,
			CanPush:   key.CanPush,
		}
		projectDeployKeys = append(projectDeployKeys, projectDeployKey)
	}

	return projectDeployKeys, nil
}

func (g *gitlabRepo) DeleteDeployKey(ctx context.Context, pid interface{}, deployKey int) error {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return err
	}

	res, err := client.DeleteDeployKey(pid, deployKey)
	if err != nil && res != nil && res.StatusCode == 404 {
		return commonv1.ErrorDeploykeyNotFound("failed to delete deploy key, err: %s", err)
	}
	if err != nil {
		return err
	}

	return nil
}

func (g *gitlabRepo) GetDeployKey(ctx context.Context, pid interface{}, deployKeyID int) (*biz.ProjectDeployKey, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	projectDeployKey, res, err := client.GetDeployKey(pid, deployKeyID)
	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorDeploykeyNotFound("failed to get deploy key, err: %s", err)
	}
	if err != nil {
		return nil, err
	}

	return &biz.ProjectDeployKey{
		ID:    projectDeployKey.ID,
		Title: projectDeployKey.Title,
		Key:   projectDeployKey.Key,
	}, nil
}

func (g *gitlabRepo) SaveDeployKey(ctx context.Context, pid interface{}, title string, canPush bool, publicKey []byte) (*biz.ProjectDeployKey, error) {
	opts := &gitlab.AddDeployKeyOptions{
		Title:   gitlab.String(title),
		Key:     gitlab.String(string(publicKey)),
		CanPush: gitlab.Bool(canPush),
	}

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	projectDeployKey, _, err := client.AddDeployKey(pid, opts)
	if err != nil {
		return nil, err
	}

	return &biz.ProjectDeployKey{
		ID:    projectDeployKey.ID,
		Title: projectDeployKey.Title,
		Key:   projectDeployKey.Key,
	}, nil
}

func (g *gitlabRepo) UpdateDeployKey(ctx context.Context, pid interface{}, deployKey int, title string, canPush bool) (*biz.ProjectDeployKey, error) {
	opts := &gitlab.UpdateDeployKeyOptions{
		Title:   gitlab.String(title),
		CanPush: gitlab.Bool(canPush),
	}

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	projectDeployKey, _, err := client.UpdateProjectDeployKey(pid, deployKey, opts)
	if err != nil {
		return nil, err
	}

	return &biz.ProjectDeployKey{
		ID:    projectDeployKey.ID,
		Title: projectDeployKey.Title,
		Key:   projectDeployKey.Key,
	}, nil
}

func (g *gitlabRepo) EnableProjectDeployKey(ctx context.Context, pid interface{}, deployKey int) (*biz.ProjectDeployKey, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	projectDeployKey, _, err := client.EnableProjectDeployKey(pid, deployKey)
	if err != nil {
		return nil, err
	}

	return &biz.ProjectDeployKey{
		ID:    projectDeployKey.ID,
		Title: projectDeployKey.Title,
		Key:   projectDeployKey.Key,
	}, nil
}

func (g *gitlabRepo) GetProjectAccessToken(ctx context.Context, pid interface{}, id int) (*biz.ProjectAccessToken, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	token, res, err := client.GetProjectAccessToken(pid, id)
	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorAccesstokenNotFound("failed to get access token, err: %s", err)
	}
	if err != nil {
		return nil, err
	}

	return &biz.ProjectAccessToken{
		Name:        token.Name,
		ID:          token.ID,
		UserID:      token.UserID,
		Scopes:      token.Scopes,
		CreatedAt:   token.CreatedAt,
		LastUsedAt:  token.LastUsedAt,
		ExpiresAt:   (*biz.ISOTime)(token.ExpiresAt),
		Active:      token.Active,
		Revoked:     token.Revoked,
		Token:       token.Token,
		AccessLevel: biz.AccessLevelValue(token.AccessLevel),
	}, nil
}

func (g *gitlabRepo) ListAccessTokens(ctx context.Context, pid interface{}, opt *biz.ListOptions) ([]*biz.ProjectAccessToken, error) {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}

	accessTokens, res, err := client.ListProjectAccessToken(pid, &gitlab.ListProjectAccessTokensOptions{Page: opt.Page, PerPage: opt.PerPage})
	if err != nil && res != nil && res.StatusCode == 404 {
		return nil, commonv1.ErrorAccesstokenNotFound("failed to list access tokens, err: %s", err)
	}

	if err != nil {
		return nil, err
	}

	projectAccessTokens := []*biz.ProjectAccessToken{}
	for _, token := range accessTokens {
		projectAccessToken := &biz.ProjectAccessToken{
			Name:        token.Name,
			ID:          token.ID,
			UserID:      token.UserID,
			Scopes:      token.Scopes,
			CreatedAt:   token.CreatedAt,
			LastUsedAt:  token.LastUsedAt,
			ExpiresAt:   (*biz.ISOTime)(token.ExpiresAt),
			Active:      token.Active,
			Revoked:     token.Revoked,
			Token:       token.Token,
			AccessLevel: biz.AccessLevelValue(token.AccessLevel),
		}
		projectAccessTokens = append(projectAccessTokens, projectAccessToken)
	}

	return projectAccessTokens, nil
}

func (g *gitlabRepo) CreateProjectAccessToken(ctx context.Context, pid interface{}, opt *biz.CreateProjectAccessTokenOptions) (*biz.ProjectAccessToken, error) {
	tokenOptions := &gitlab.CreateProjectAccessTokenOptions{
		Name:        opt.Name,
		Scopes:      opt.Scopes,
		AccessLevel: (*gitlab.AccessLevelValue)(opt.AccessLevel),
		ExpiresAt:   (*gitlab.ISOTime)(opt.ExpiresAt),
	}

	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return nil, err
	}
	token, _, err := client.CreateProjectAccessToken(pid, tokenOptions)
	if err != nil {
		return nil, err
	}

	return &biz.ProjectAccessToken{
		Name:        token.Name,
		ID:          token.ID,
		UserID:      token.UserID,
		Scopes:      token.Scopes,
		CreatedAt:   token.CreatedAt,
		LastUsedAt:  token.LastUsedAt,
		ExpiresAt:   (*biz.ISOTime)(token.ExpiresAt),
		Active:      token.Active,
		Revoked:     token.Revoked,
		Token:       token.Token,
		AccessLevel: biz.AccessLevelValue(token.AccessLevel),
	}, nil
}

func (g *gitlabRepo) DeleteProjectAccessToken(ctx context.Context, pid interface{}, id int) error {
	client, err := NewGitlabClient(ctx, g)
	if err != nil {
		return err
	}

	if _, err := client.DeleteProjectAccessToken(pid, id); err != nil {
		return err
	}

	return nil
}

func NewGitlabClient(ctx context.Context, g *gitlabRepo) (gitlabclient.GitlabOperator, error) {
	token := ctx.Value("token")
	if token == nil {
		return nil, fmt.Errorf("token is not found")
	}
	t, ok := token.(string)
	if !ok {
		return nil, fmt.Errorf("token type error, it must be string")
	}

	client, err := g.client.NewGitlabClient(g.url, t)
	if err != nil {
		return nil, err
	}

	return client, nil
}
