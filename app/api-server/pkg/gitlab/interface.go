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

import "github.com/xanzy/go-gitlab"

type GitlabOperator interface {
	NewGitlabClient(url, token string) (GitlabOperator, error)
	GetCurrentUser() (user *gitlab.User, res *gitlab.Response, err error)

	CreateProject(opt *gitlab.CreateProjectOptions, options ...gitlab.RequestOptionFunc) (project *gitlab.Project, res *gitlab.Response, err error)
	DeleteProject(pid interface{}) (res *gitlab.Response, err error)
	UpdateProject(pid interface{}, opt *gitlab.EditProjectOptions) (project *gitlab.Project, res *gitlab.Response, err error)
	GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
	ListGroupProjects(gid interface{}, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) (projects []*gitlab.Project, res *gitlab.Response, err error)
	ListProjects(search string) (projects []*gitlab.Project, res *gitlab.Response, err error)

	CreateGroup(opt *gitlab.CreateGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error)
	DeleteGroup(gid interface{}, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error)
	UpdateGroup(gid interface{}, opt *gitlab.UpdateGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error)
	GetGroup(gid interface{}, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (group *gitlab.Group, res *gitlab.Response, err error)
	ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) (groups []*gitlab.Group, res *gitlab.Response, err error)

	GetDeployKey(pid interface{}, deployKeyID int, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error)
	ListDeployKeys(pid interface{}, opt *gitlab.ListProjectDeployKeysOptions, options ...gitlab.RequestOptionFunc) (keys []*gitlab.ProjectDeployKey, res *gitlab.Response, err error)
	ListAllDeployKeys(opt *gitlab.ListInstanceDeployKeysOptions, options ...gitlab.RequestOptionFunc) (keys []*gitlab.InstanceDeployKey, res *gitlab.Response, err error)
	AddDeployKey(pid interface{}, opt *gitlab.AddDeployKeyOptions, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error)
	DeleteDeployKey(pid interface{}, deployKey int, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error)
	EnableProjectDeployKey(pid interface{}, deployKey int, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error)
	UpdateProjectDeployKey(pid interface{}, deployKey int, opt *gitlab.UpdateDeployKeyOptions, options ...gitlab.RequestOptionFunc) (key *gitlab.ProjectDeployKey, res *gitlab.Response, err error)
	GetProjectAccessToken(pid interface{}, id int, options ...gitlab.RequestOptionFunc) (token *gitlab.ProjectAccessToken, res *gitlab.Response, err error)
	ListProjectAccessToken(pid interface{}, opt *gitlab.ListProjectAccessTokensOptions, options ...gitlab.RequestOptionFunc) (tokens []*gitlab.ProjectAccessToken, res *gitlab.Response, err error)
	CreateProjectAccessToken(pid interface{}, opt *gitlab.CreateProjectAccessTokenOptions, options ...gitlab.RequestOptionFunc) (token *gitlab.ProjectAccessToken, res *gitlab.Response, err error)
	DeleteProjectAccessToken(pid interface{}, id int, options ...gitlab.RequestOptionFunc) (res *gitlab.Response, err error)
}
