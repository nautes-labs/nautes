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

package v1

import (
	"bytes"
	fmt "fmt"
	"text/template"
)

const (
	GitSecretName     = "git"
	RepoSecretName    = "repo"
	ClusterSecretName = "cluster"
	TenantSecretName  = "tenant"
)

type SecretType int

const (
	GIT SecretType = iota
	REPO
	CLUSTER
	TENANTGIT
	TENANTREPO
)

func (s SecretType) String() string {
	switch s {
	case GIT:
		return "git"
	case REPO:
		return "repo"
	case CLUSTER:
		return "cluster"
	case TENANTGIT:
		return "tenant-git"
	case TENANTREPO:
		return "tenant-repo"
	}
	return ""
}

var (
	SecretPolicy string = "path \"%s\" {\n    capabilities = [\"read\"]\n}"
	GitPolicy    string = `
path "git/data/%[1]s" {
    capabilities = ["read"]
}

path "git/metadata/%[1]s" {
    capabilities = ["read"]
}`
	ClusterPolicy string = `
path "cluster/data/%s" {
    capabilities = ["read"]
}

path "auth/%s/role/*" {
    capabilities = ["read"]
}`
)

type VaultTemplate struct {
	name     string
	template string
}

var (
	GitPathTemplate VaultTemplate = VaultTemplate{
		name:     "gitPath",
		template: "{{.ProviderType}}/{{.Id}}/{{.Username}}/{{.Permission}}",
	}
	GitPolicyPathTemplate VaultTemplate = VaultTemplate{
		name:     "gitPolicyPath",
		template: "{{.ProviderType}}-{{.Id}}-{{.Username}}-{{.Permission}}",
	}
	RepoPathTemplate VaultTemplate = VaultTemplate{
		name:     "repoPath",
		template: "{{.ProviderId}}/{{.Type}}/{{.Id}}/{{.Username}}/{{.Permission}}",
	}
	RepoPolicyPathTemplate VaultTemplate = VaultTemplate{
		name:     "repoPolicy",
		template: "{{.ProviderId}}-{{.Type}}-{{.Id}}-{{.Username}}-{{.Permission}}",
	}
	ClusterPathTemplate VaultTemplate = VaultTemplate{
		name:     "clusterPath",
		template: "{{.Type}}/{{.Id}}/{{.Username}}/{{.Permission}}",
	}
	ClusterPolicyPathTemplate VaultTemplate = VaultTemplate{
		name:     "clusterPolicyPath",
		template: "{{.Type}}-{{.Id}}-{{.Username}}-{{.Permission}}",
	}
	TenantGitPathTemplate VaultTemplate = VaultTemplate{
		name:     "tenantGitPath",
		template: "git/{{.Id}}/{{.Permission}}",
	}
	TenantGitPolicyPathTemplate VaultTemplate = VaultTemplate{
		name:     "tenantGitPolicyPath",
		template: "tenant-git-{{.Id}}-{{.Permission}}",
	}
	TenantRepoPathTemplate VaultTemplate = VaultTemplate{
		name:     "tenantRepoPath",
		template: "repo/{{.Id}}/{{.Permission}}",
	}
	TenantRepoPolicyPathTemplate VaultTemplate = VaultTemplate{
		name:     "tenantRepoPolicyPath",
		template: "tenant-repo-{{.Id}}-{{.Permission}}",
	}
	RolePathTemplate = VaultTemplate{
		name:     "rolePath",
		template: "auth/{{.ClusterName}}/role/{{.Projectid}}",
	}
)

func GetPath(vars interface{}, vault_tmpl VaultTemplate) (string, error) {
	tmpl, err := template.New(vault_tmpl.name).Parse(vault_tmpl.template)
	if err != nil {
		return "", err
	}

	var path bytes.Buffer
	err = tmpl.ExecuteTemplate(&path, vault_tmpl.name, vars)
	if err != nil {
		return "", err
	}
	return path.String(), nil
}

func (x *GitKVs) getData() map[string]interface{} {
	secretData := make(map[string]interface{}, 0)
	for k, v := range x.Additionals {
		secretData[k] = v
	}
	if x.GetDeployKey() != "" {
		secretData["deploykey"] = x.GetDeployKey()
	}
	if x.GetAccessToken() != "" {
		secretData["accesstoken"] = x.GetAccessToken()
	}
	return secretData
}

func (x *RepoAccount) getData() map[string]interface{} {
	return map[string]interface{}{
		"username": x.GetUsername(),
		"password": x.GetPassword(),
	}
}

func (x *ClusterAccount) getData() map[string]interface{} {
	data := make(map[string]interface{})
	if x.GetKubeconfig() != "" {
		data["kubeconfig"] = x.GetKubeconfig()
	}
	return data
}

// Store the key info for authorize and vault request
type SecretRequest struct {
	SecretMeta
	SecretData map[string]interface{}
	PolicyData string
}

type SecRequest interface {
	ConvertRequest() (*SecretRequest, error)
}

type SecretMeta struct {
	SecretName string // secret name , use for vault api
	SecretPath string // secret path , use for vault api
	SecretType string // secret data type
	FullPath   string // full path of secret use for create policy and authorize
	PolicyName string // vault policy name, use for authorize and policy create
}

func (x *GitMeta) GetNames() (*SecretMeta, error) {
	secretName := GitSecretName
	secretPath, err := GetPath(x, GitPathTemplate)
	if err != nil {
		return nil, err
	}
	fullPath := fmt.Sprintf("%s/data/%s", secretName, secretPath)
	policyName, err := GetPath(x, GitPolicyPathTemplate)
	if err != nil {
		return nil, err
	}

	return &SecretMeta{
		SecretName: secretName,
		SecretPath: secretPath,
		SecretType: GIT.String(),
		FullPath:   fullPath,
		PolicyName: policyName,
	}, nil
}

func (x *GitRequest) ConvertRequest() (*SecretRequest, error) {
	secretMeta, err := x.Meta.GetNames()
	if err != nil {
		return nil, err
	}

	secretData := make(map[string]interface{}, 0)
	if x.Kvs != nil {
		secretData = x.Kvs.getData()
	}
	policyData := fmt.Sprintf(GitPolicy, secretMeta.SecretPath)

	return &SecretRequest{
		SecretMeta: *secretMeta,
		SecretData: secretData,
		PolicyData: policyData,
	}, nil
}

func (x *RepoMeta) GetNames() (*SecretMeta, error) {
	secretName := RepoSecretName
	secretPath, err := GetPath(x, RepoPathTemplate)
	if err != nil {
		return nil, err
	}
	fullPath := fmt.Sprintf("%s/data/%s", secretName, secretPath)
	policyName, err := GetPath(x, RepoPolicyPathTemplate)
	if err != nil {
		return nil, err
	}

	return &SecretMeta{
		SecretName: secretName,
		SecretPath: secretPath,
		SecretType: REPO.String(),
		FullPath:   fullPath,
		PolicyName: policyName,
	}, nil
}

func (x *RepoRequest) ConvertRequest() (*SecretRequest, error) {
	secretMeta, err := x.Meta.GetNames()
	if err != nil {
		return nil, err
	}

	secretData := x.GetAccount().getData()
	policyData := fmt.Sprintf(SecretPolicy, secretMeta.FullPath)

	return &SecretRequest{
		SecretMeta: *secretMeta,
		SecretData: secretData,
		PolicyData: policyData,
	}, nil
}

func (x *ClusterMeta) GetNames() (*SecretMeta, error) {
	secretName := ClusterSecretName
	secretPath, err := GetPath(x, ClusterPathTemplate)
	if err != nil {
		return nil, err
	}
	fullPath := fmt.Sprintf("%s/data/%s", secretName, secretPath)
	policyName, err := GetPath(x, ClusterPolicyPathTemplate)
	if err != nil {
		return nil, err
	}

	return &SecretMeta{
		SecretName: secretName,
		SecretPath: secretPath,
		SecretType: CLUSTER.String(),
		FullPath:   fullPath,
		PolicyName: policyName,
	}, nil
}

func (x *ClusterRequest) ConvertRequest() (*SecretRequest, error) {
	secretMeta, err := x.Meta.GetNames()
	if err != nil {
		return nil, err
	}

	secretData := x.GetAccount().getData()
	policyData := fmt.Sprintf(ClusterPolicy, secretMeta.SecretPath, x.Meta.GetId())

	return &SecretRequest{
		SecretMeta: *secretMeta,
		SecretData: secretData,
		PolicyData: policyData,
	}, nil
}

func (x *TenantGitMeta) GetNames() (*SecretMeta, error) {
	secretName := TenantSecretName
	secretPath, err := GetPath(x, TenantGitPathTemplate)
	if err != nil {
		return nil, err
	}
	fullPath := fmt.Sprintf("%s/data/%s", secretName, secretPath)
	policyName, err := GetPath(x, TenantGitPolicyPathTemplate)
	if err != nil {
		return nil, err
	}

	return &SecretMeta{
		SecretName: secretName,
		SecretPath: secretPath,
		SecretType: TENANTGIT.String(),
		FullPath:   fullPath,
		PolicyName: policyName,
	}, nil
}

func (x *TenantGitRequest) ConvertRequest() (*SecretRequest, error) {
	secretMeta, err := x.Meta.GetNames()
	if err != nil {
		return nil, err
	}

	secretData := make(map[string]interface{}, 0)
	if x.Kvs != nil {
		secretData = x.Kvs.getData()
	}
	policyData := fmt.Sprintf(SecretPolicy, secretMeta.FullPath)

	return &SecretRequest{
		SecretMeta: *secretMeta,
		SecretData: secretData,
		PolicyData: policyData,
	}, nil
}

func (x *TenantRepoMeta) GetNames() (*SecretMeta, error) {
	secretName := TenantSecretName
	secretPath, err := GetPath(x, TenantRepoPathTemplate)
	if err != nil {
		return nil, err
	}
	fullPath := fmt.Sprintf("%s/data/%s", secretName, secretPath)
	policyName, err := GetPath(x, TenantRepoPolicyPathTemplate)
	if err != nil {
		return nil, err
	}

	return &SecretMeta{
		SecretName: secretName,
		SecretPath: secretPath,
		SecretType: TENANTREPO.String(),
		FullPath:   fullPath,
		PolicyName: policyName,
	}, nil
}

func (x *TenantRepoRequest) ConvertRequest() (*SecretRequest, error) {
	secretMeta, err := x.Meta.GetNames()
	if err != nil {
		return nil, err
	}

	secretData := x.GetAccount().getData()
	policyData := fmt.Sprintf(SecretPolicy, secretMeta.FullPath)

	return &SecretRequest{
		SecretMeta: *secretMeta,
		SecretData: secretData,
		PolicyData: policyData,
	}, nil
}

func (x *AuthRequest) ConvertRequest() (*SecretRequest, error) {

	fullPath := fmt.Sprintf("auth/%s", x.ClusterName)

	return &SecretRequest{
		SecretMeta: SecretMeta{FullPath: fullPath},
		SecretData: nil,
		PolicyData: "",
	}, nil
}

func (x *AuthroleRequest) ConvertRequest() (*SecretRequest, error) {

	fullPath := fmt.Sprintf("auth/%s/role/%s", x.ClusterName, x.DestUser)

	return &SecretRequest{
		SecretMeta: SecretMeta{FullPath: fullPath},
		SecretData: nil,
		PolicyData: "",
	}, nil
}

type GrantTarget struct {
	RolePath string
	Name     string
}

type AuthGrantRequest interface {
	ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error)
}

func ConvertAuthGrantRequest(cluster, user string, sec *SecretMeta) (*GrantTarget, *SecretRequest, error) {
	rolePath, err := GetPath(map[string]string{"ClusterName": cluster, "Projectid": user}, RolePathTemplate)
	if err != nil {
		return nil, nil, err
	}

	return &GrantTarget{
			RolePath: rolePath,
			Name:     user,
		}, &SecretRequest{
			SecretMeta: *sec,
		}, nil
}

func (req *AuthroleGitPolicyRequest) ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error) {
	secretMeta, err := req.Secret.GetNames()
	if err != nil {
		return nil, nil, err
	}
	return ConvertAuthGrantRequest(req.ClusterName, req.DestUser, secretMeta)
}

func (req *AuthroleRepoPolicyRequest) ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error) {
	secretMeta, err := req.Secret.GetNames()
	if err != nil {
		return nil, nil, err
	}
	return ConvertAuthGrantRequest(req.ClusterName, req.DestUser, secretMeta)
}

func (req *AuthroleClusterPolicyRequest) ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error) {
	secretMeta, err := req.Secret.GetNames()
	if err != nil {
		return nil, nil, err
	}
	return ConvertAuthGrantRequest(req.ClusterName, req.DestUser, secretMeta)
}

func (req *AuthroleTenantGitPolicyRequest) ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error) {
	secretMeta, err := req.Secret.GetNames()
	if err != nil {
		return nil, nil, err
	}
	return ConvertAuthGrantRequest(req.ClusterName, req.DestUser, secretMeta)
}

func (req *AuthroleTenantRepoPolicyRequest) ConvertToAuthPolicyReqeuest() (*GrantTarget, *SecretRequest, error) {
	secretMeta, err := req.Secret.GetNames()
	if err != nil {
		return nil, nil, err
	}
	return ConvertAuthGrantRequest(req.ClusterName, req.DestUser, secretMeta)
}
