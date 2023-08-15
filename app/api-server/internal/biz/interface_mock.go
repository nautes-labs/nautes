// Code generated by MockGen. DO NOT EDIT.
// Source: app/api-server/internal/biz/interface.go

// Package biz is a generated GoMock package.
package biz

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

// MockCodeRepo is a mock of CodeRepo interface.
type MockCodeRepo struct {
	ctrl     *gomock.Controller
	recorder *MockCodeRepoMockRecorder
}

// MockCodeRepoMockRecorder is the mock recorder for MockCodeRepo.
type MockCodeRepoMockRecorder struct {
	mock *MockCodeRepo
}

// NewMockCodeRepo creates a new mock instance.
func NewMockCodeRepo(ctrl *gomock.Controller) *MockCodeRepo {
	mock := &MockCodeRepo{ctrl: ctrl}
	mock.recorder = &MockCodeRepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCodeRepo) EXPECT() *MockCodeRepoMockRecorder {
	return m.recorder
}

// CreateCodeRepo mocks base method.
func (m *MockCodeRepo) CreateCodeRepo(ctx context.Context, gid int, options *GitCodeRepoOptions) (*Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateCodeRepo", ctx, gid, options)
	ret0, _ := ret[0].(*Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCodeRepo indicates an expected call of CreateCodeRepo.
func (mr *MockCodeRepoMockRecorder) CreateCodeRepo(ctx, gid, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCodeRepo", reflect.TypeOf((*MockCodeRepo)(nil).CreateCodeRepo), ctx, gid, options)
}

// CreateGroup mocks base method.
func (m *MockCodeRepo) CreateGroup(ctx context.Context, gitOptions *GitGroupOptions) (*Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGroup", ctx, gitOptions)
	ret0, _ := ret[0].(*Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateGroup indicates an expected call of CreateGroup.
func (mr *MockCodeRepoMockRecorder) CreateGroup(ctx, gitOptions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGroup", reflect.TypeOf((*MockCodeRepo)(nil).CreateGroup), ctx, gitOptions)
}

// CreateProjectAccessToken mocks base method.
func (m *MockCodeRepo) CreateProjectAccessToken(ctx context.Context, pid interface{}, opt *CreateProjectAccessTokenOptions) (*ProjectAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateProjectAccessToken", ctx, pid, opt)
	ret0, _ := ret[0].(*ProjectAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateProjectAccessToken indicates an expected call of CreateProjectAccessToken.
func (mr *MockCodeRepoMockRecorder) CreateProjectAccessToken(ctx, pid, opt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateProjectAccessToken", reflect.TypeOf((*MockCodeRepo)(nil).CreateProjectAccessToken), ctx, pid, opt)
}

// DeleteCodeRepo mocks base method.
func (m *MockCodeRepo) DeleteCodeRepo(ctx context.Context, pid interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCodeRepo", ctx, pid)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCodeRepo indicates an expected call of DeleteCodeRepo.
func (mr *MockCodeRepoMockRecorder) DeleteCodeRepo(ctx, pid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCodeRepo", reflect.TypeOf((*MockCodeRepo)(nil).DeleteCodeRepo), ctx, pid)
}

// DeleteDeployKey mocks base method.
func (m *MockCodeRepo) DeleteDeployKey(ctx context.Context, pid interface{}, deployKey int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDeployKey", ctx, pid, deployKey)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDeployKey indicates an expected call of DeleteDeployKey.
func (mr *MockCodeRepoMockRecorder) DeleteDeployKey(ctx, pid, deployKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDeployKey", reflect.TypeOf((*MockCodeRepo)(nil).DeleteDeployKey), ctx, pid, deployKey)
}

// DeleteGroup mocks base method.
func (m *MockCodeRepo) DeleteGroup(ctx context.Context, gid interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteGroup", ctx, gid)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteGroup indicates an expected call of DeleteGroup.
func (mr *MockCodeRepoMockRecorder) DeleteGroup(ctx, gid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteGroup", reflect.TypeOf((*MockCodeRepo)(nil).DeleteGroup), ctx, gid)
}

// DeleteProjectAccessToken mocks base method.
func (m *MockCodeRepo) DeleteProjectAccessToken(ctx context.Context, pid interface{}, id int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteProjectAccessToken", ctx, pid, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteProjectAccessToken indicates an expected call of DeleteProjectAccessToken.
func (mr *MockCodeRepoMockRecorder) DeleteProjectAccessToken(ctx, pid, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteProjectAccessToken", reflect.TypeOf((*MockCodeRepo)(nil).DeleteProjectAccessToken), ctx, pid, id)
}

// EnableProjectDeployKey mocks base method.
func (m *MockCodeRepo) EnableProjectDeployKey(ctx context.Context, pid interface{}, deployKey int) (*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnableProjectDeployKey", ctx, pid, deployKey)
	ret0, _ := ret[0].(*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EnableProjectDeployKey indicates an expected call of EnableProjectDeployKey.
func (mr *MockCodeRepoMockRecorder) EnableProjectDeployKey(ctx, pid, deployKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnableProjectDeployKey", reflect.TypeOf((*MockCodeRepo)(nil).EnableProjectDeployKey), ctx, pid, deployKey)
}

// GetCodeRepo mocks base method.
func (m *MockCodeRepo) GetCodeRepo(ctx context.Context, pid interface{}) (*Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCodeRepo", ctx, pid)
	ret0, _ := ret[0].(*Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCodeRepo indicates an expected call of GetCodeRepo.
func (mr *MockCodeRepoMockRecorder) GetCodeRepo(ctx, pid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCodeRepo", reflect.TypeOf((*MockCodeRepo)(nil).GetCodeRepo), ctx, pid)
}

// GetCurrentUser mocks base method.
func (m *MockCodeRepo) GetCurrentUser(ctx context.Context) (string, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentUser", ctx)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetCurrentUser indicates an expected call of GetCurrentUser.
func (mr *MockCodeRepoMockRecorder) GetCurrentUser(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentUser", reflect.TypeOf((*MockCodeRepo)(nil).GetCurrentUser), ctx)
}

// GetDeployKey mocks base method.
func (m *MockCodeRepo) GetDeployKey(ctx context.Context, pid interface{}, deployKeyID int) (*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeployKey", ctx, pid, deployKeyID)
	ret0, _ := ret[0].(*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeployKey indicates an expected call of GetDeployKey.
func (mr *MockCodeRepoMockRecorder) GetDeployKey(ctx, pid, deployKeyID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeployKey", reflect.TypeOf((*MockCodeRepo)(nil).GetDeployKey), ctx, pid, deployKeyID)
}

// GetGroup mocks base method.
func (m *MockCodeRepo) GetGroup(ctx context.Context, gid interface{}) (*Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroup", ctx, gid)
	ret0, _ := ret[0].(*Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGroup indicates an expected call of GetGroup.
func (mr *MockCodeRepoMockRecorder) GetGroup(ctx, gid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroup", reflect.TypeOf((*MockCodeRepo)(nil).GetGroup), ctx, gid)
}

// GetProjectAccessToken mocks base method.
func (m *MockCodeRepo) GetProjectAccessToken(ctx context.Context, pid interface{}, id int) (*ProjectAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProjectAccessToken", ctx, pid, id)
	ret0, _ := ret[0].(*ProjectAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProjectAccessToken indicates an expected call of GetProjectAccessToken.
func (mr *MockCodeRepoMockRecorder) GetProjectAccessToken(ctx, pid, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProjectAccessToken", reflect.TypeOf((*MockCodeRepo)(nil).GetProjectAccessToken), ctx, pid, id)
}

// ListAccessTokens mocks base method.
func (m *MockCodeRepo) ListAccessTokens(ctx context.Context, pid interface{}, opt *ListOptions) ([]*ProjectAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAccessTokens", ctx, pid, opt)
	ret0, _ := ret[0].([]*ProjectAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAccessTokens indicates an expected call of ListAccessTokens.
func (mr *MockCodeRepoMockRecorder) ListAccessTokens(ctx, pid, opt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAccessTokens", reflect.TypeOf((*MockCodeRepo)(nil).ListAccessTokens), ctx, pid, opt)
}

// ListAllDeployKeys mocks base method.
func (m *MockCodeRepo) ListAllDeployKeys(ctx context.Context, opt *ListOptions) ([]*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAllDeployKeys", ctx, opt)
	ret0, _ := ret[0].([]*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAllDeployKeys indicates an expected call of ListAllDeployKeys.
func (mr *MockCodeRepoMockRecorder) ListAllDeployKeys(ctx, opt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAllDeployKeys", reflect.TypeOf((*MockCodeRepo)(nil).ListAllDeployKeys), ctx, opt)
}

// ListAllGroups mocks base method.
func (m *MockCodeRepo) ListAllGroups(ctx context.Context) ([]*Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAllGroups", ctx)
	ret0, _ := ret[0].([]*Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAllGroups indicates an expected call of ListAllGroups.
func (mr *MockCodeRepoMockRecorder) ListAllGroups(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAllGroups", reflect.TypeOf((*MockCodeRepo)(nil).ListAllGroups), ctx)
}

// ListCodeRepos mocks base method.
func (m *MockCodeRepo) ListCodeRepos(ctx context.Context, search string) ([]*Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCodeRepos", ctx, search)
	ret0, _ := ret[0].([]*Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCodeRepos indicates an expected call of ListCodeRepos.
func (mr *MockCodeRepoMockRecorder) ListCodeRepos(ctx, search interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCodeRepos", reflect.TypeOf((*MockCodeRepo)(nil).ListCodeRepos), ctx, search)
}

// ListDeployKeys mocks base method.
func (m *MockCodeRepo) ListDeployKeys(ctx context.Context, pid interface{}, opt *ListOptions) ([]*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListDeployKeys", ctx, pid, opt)
	ret0, _ := ret[0].([]*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDeployKeys indicates an expected call of ListDeployKeys.
func (mr *MockCodeRepoMockRecorder) ListDeployKeys(ctx, pid, opt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDeployKeys", reflect.TypeOf((*MockCodeRepo)(nil).ListDeployKeys), ctx, pid, opt)
}

// ListGroupCodeRepos mocks base method.
func (m *MockCodeRepo) ListGroupCodeRepos(ctx context.Context, gid interface{}, opts *ListGroupProjectsOptions) ([]*Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListGroupCodeRepos", ctx, gid, opts)
	ret0, _ := ret[0].([]*Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListGroupCodeRepos indicates an expected call of ListGroupCodeRepos.
func (mr *MockCodeRepoMockRecorder) ListGroupCodeRepos(ctx, gid, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListGroupCodeRepos", reflect.TypeOf((*MockCodeRepo)(nil).ListGroupCodeRepos), ctx, gid, opts)
}

// SaveDeployKey mocks base method.
func (m *MockCodeRepo) SaveDeployKey(ctx context.Context, pid interface{}, title string, canPush bool, publicKey []byte) (*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveDeployKey", ctx, pid, title, canPush, publicKey)
	ret0, _ := ret[0].(*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SaveDeployKey indicates an expected call of SaveDeployKey.
func (mr *MockCodeRepoMockRecorder) SaveDeployKey(ctx, pid, title, canPush, publicKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveDeployKey", reflect.TypeOf((*MockCodeRepo)(nil).SaveDeployKey), ctx, pid, title, canPush, publicKey)
}

// UpdateCodeRepo mocks base method.
func (m *MockCodeRepo) UpdateCodeRepo(ctx context.Context, pid interface{}, options *GitCodeRepoOptions) (*Project, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCodeRepo", ctx, pid, options)
	ret0, _ := ret[0].(*Project)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateCodeRepo indicates an expected call of UpdateCodeRepo.
func (mr *MockCodeRepoMockRecorder) UpdateCodeRepo(ctx, pid, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCodeRepo", reflect.TypeOf((*MockCodeRepo)(nil).UpdateCodeRepo), ctx, pid, options)
}

// UpdateDeployKey mocks base method.
func (m *MockCodeRepo) UpdateDeployKey(ctx context.Context, pid interface{}, deployKey int, title string, canPush bool) (*ProjectDeployKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateDeployKey", ctx, pid, deployKey, title, canPush)
	ret0, _ := ret[0].(*ProjectDeployKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDeployKey indicates an expected call of UpdateDeployKey.
func (mr *MockCodeRepoMockRecorder) UpdateDeployKey(ctx, pid, deployKey, title, canPush interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDeployKey", reflect.TypeOf((*MockCodeRepo)(nil).UpdateDeployKey), ctx, pid, deployKey, title, canPush)
}

// UpdateGroup mocks base method.
func (m *MockCodeRepo) UpdateGroup(ctx context.Context, gid interface{}, git *GitGroupOptions) (*Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroup", ctx, gid, git)
	ret0, _ := ret[0].(*Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateGroup indicates an expected call of UpdateGroup.
func (mr *MockCodeRepoMockRecorder) UpdateGroup(ctx, gid, git interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroup", reflect.TypeOf((*MockCodeRepo)(nil).UpdateGroup), ctx, gid, git)
}

// MockSecretrepo is a mock of Secretrepo interface.
type MockSecretrepo struct {
	ctrl     *gomock.Controller
	recorder *MockSecretrepoMockRecorder
}

// MockSecretrepoMockRecorder is the mock recorder for MockSecretrepo.
type MockSecretrepoMockRecorder struct {
	mock *MockSecretrepo
}

// NewMockSecretrepo creates a new mock instance.
func NewMockSecretrepo(ctrl *gomock.Controller) *MockSecretrepo {
	mock := &MockSecretrepo{ctrl: ctrl}
	mock.recorder = &MockSecretrepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSecretrepo) EXPECT() *MockSecretrepoMockRecorder {
	return m.recorder
}

// AuthorizationSecret mocks base method.
func (m *MockSecretrepo) AuthorizationSecret(ctx context.Context, id int, destUser, gitType, mountPath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthorizationSecret", ctx, id, destUser, gitType, mountPath)
	ret0, _ := ret[0].(error)
	return ret0
}

// AuthorizationSecret indicates an expected call of AuthorizationSecret.
func (mr *MockSecretrepoMockRecorder) AuthorizationSecret(ctx, id, destUser, gitType, mountPath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthorizationSecret", reflect.TypeOf((*MockSecretrepo)(nil).AuthorizationSecret), ctx, id, destUser, gitType, mountPath)
}

// DeleteSecret mocks base method.
func (m *MockSecretrepo) DeleteSecret(ctx context.Context, id int, user, permission string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecret", ctx, id, user, permission)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecret indicates an expected call of DeleteSecret.
func (mr *MockSecretrepoMockRecorder) DeleteSecret(ctx, id, user, permission interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecret", reflect.TypeOf((*MockSecretrepo)(nil).DeleteSecret), ctx, id, user, permission)
}

// GetDeployKey mocks base method.
func (m *MockSecretrepo) GetDeployKey(ctx context.Context, secretOptions *SecretOptions) (*DeployKeySecretData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeployKey", ctx, secretOptions)
	ret0, _ := ret[0].(*DeployKeySecretData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeployKey indicates an expected call of GetDeployKey.
func (mr *MockSecretrepoMockRecorder) GetDeployKey(ctx, secretOptions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeployKey", reflect.TypeOf((*MockSecretrepo)(nil).GetDeployKey), ctx, secretOptions)
}

// GetProjectAccessToken mocks base method.
func (m *MockSecretrepo) GetProjectAccessToken(ctx context.Context, secretOptions *SecretOptions) (*AccessTokenSecretData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProjectAccessToken", ctx, secretOptions)
	ret0, _ := ret[0].(*AccessTokenSecretData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProjectAccessToken indicates an expected call of GetProjectAccessToken.
func (mr *MockSecretrepoMockRecorder) GetProjectAccessToken(ctx, secretOptions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProjectAccessToken", reflect.TypeOf((*MockSecretrepo)(nil).GetProjectAccessToken), ctx, secretOptions)
}

// GetSecret mocks base method.
func (m *MockSecretrepo) GetSecret(ctx context.Context, secretOptions *SecretOptions) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecret", ctx, secretOptions)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecret indicates an expected call of GetSecret.
func (mr *MockSecretrepoMockRecorder) GetSecret(ctx, secretOptions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecret", reflect.TypeOf((*MockSecretrepo)(nil).GetSecret), ctx, secretOptions)
}

// SaveClusterConfig mocks base method.
func (m *MockSecretrepo) SaveClusterConfig(ctx context.Context, id, config string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveClusterConfig", ctx, id, config)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveClusterConfig indicates an expected call of SaveClusterConfig.
func (mr *MockSecretrepoMockRecorder) SaveClusterConfig(ctx, id, config interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveClusterConfig", reflect.TypeOf((*MockSecretrepo)(nil).SaveClusterConfig), ctx, id, config)
}

// SaveDeployKey mocks base method.
func (m *MockSecretrepo) SaveDeployKey(ctx context.Context, id, key, user, permission string, extendKVs map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveDeployKey", ctx, id, key, user, permission, extendKVs)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveDeployKey indicates an expected call of SaveDeployKey.
func (mr *MockSecretrepoMockRecorder) SaveDeployKey(ctx, id, key, user, permission, extendKVs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveDeployKey", reflect.TypeOf((*MockSecretrepo)(nil).SaveDeployKey), ctx, id, key, user, permission, extendKVs)
}

// SaveProjectAccessToken mocks base method.
func (m *MockSecretrepo) SaveProjectAccessToken(ctx context.Context, id, token, user, permission string, extendKVs map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveProjectAccessToken", ctx, id, token, user, permission, extendKVs)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveProjectAccessToken indicates an expected call of SaveProjectAccessToken.
func (mr *MockSecretrepoMockRecorder) SaveProjectAccessToken(ctx, id, token, user, permission, extendKVs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveProjectAccessToken", reflect.TypeOf((*MockSecretrepo)(nil).SaveProjectAccessToken), ctx, id, token, user, permission, extendKVs)
}

// MockGitRepo is a mock of GitRepo interface.
type MockGitRepo struct {
	ctrl     *gomock.Controller
	recorder *MockGitRepoMockRecorder
}

// MockGitRepoMockRecorder is the mock recorder for MockGitRepo.
type MockGitRepoMockRecorder struct {
	mock *MockGitRepo
}

// NewMockGitRepo creates a new mock instance.
func NewMockGitRepo(ctrl *gomock.Controller) *MockGitRepo {
	mock := &MockGitRepo{ctrl: ctrl}
	mock.recorder = &MockGitRepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGitRepo) EXPECT() *MockGitRepoMockRecorder {
	return m.recorder
}

// Clone mocks base method.
func (m *MockGitRepo) Clone(ctx context.Context, param *CloneRepositoryParam) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Clone", ctx, param)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Clone indicates an expected call of Clone.
func (mr *MockGitRepoMockRecorder) Clone(ctx, param interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clone", reflect.TypeOf((*MockGitRepo)(nil).Clone), ctx, param)
}

// Commit mocks base method.
func (m *MockGitRepo) Commit(ctx context.Context, path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Commit", ctx, path)
	ret0, _ := ret[0].(error)
	return ret0
}

// Commit indicates an expected call of Commit.
func (mr *MockGitRepoMockRecorder) Commit(ctx, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Commit", reflect.TypeOf((*MockGitRepo)(nil).Commit), ctx, path)
}

// Diff mocks base method.
func (m *MockGitRepo) Diff(ctx context.Context, path string, command ...string) (string, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, path}
	for _, a := range command {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Diff", varargs...)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Diff indicates an expected call of Diff.
func (mr *MockGitRepoMockRecorder) Diff(ctx, path interface{}, command ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, path}, command...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Diff", reflect.TypeOf((*MockGitRepo)(nil).Diff), varargs...)
}

// Fetch mocks base method.
func (m *MockGitRepo) Fetch(ctx context.Context, path string, command ...string) (string, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, path}
	for _, a := range command {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Fetch", varargs...)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Fetch indicates an expected call of Fetch.
func (mr *MockGitRepoMockRecorder) Fetch(ctx, path interface{}, command ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, path}, command...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fetch", reflect.TypeOf((*MockGitRepo)(nil).Fetch), varargs...)
}

// Merge mocks base method.
func (m *MockGitRepo) Merge(ctx context.Context, path string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Merge", ctx, path)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Merge indicates an expected call of Merge.
func (mr *MockGitRepoMockRecorder) Merge(ctx, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Merge", reflect.TypeOf((*MockGitRepo)(nil).Merge), ctx, path)
}

// Push mocks base method.
func (m *MockGitRepo) Push(ctx context.Context, path string, command ...string) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, path}
	for _, a := range command {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Push", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Push indicates an expected call of Push.
func (mr *MockGitRepoMockRecorder) Push(ctx, path interface{}, command ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, path}, command...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Push", reflect.TypeOf((*MockGitRepo)(nil).Push), varargs...)
}

// SaveConfig mocks base method.
func (m *MockGitRepo) SaveConfig(ctx context.Context, path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveConfig", ctx, path)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveConfig indicates an expected call of SaveConfig.
func (mr *MockGitRepoMockRecorder) SaveConfig(ctx, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveConfig", reflect.TypeOf((*MockGitRepo)(nil).SaveConfig), ctx, path)
}

// MockDexRepo is a mock of DexRepo interface.
type MockDexRepo struct {
	ctrl     *gomock.Controller
	recorder *MockDexRepoMockRecorder
}

// MockDexRepoMockRecorder is the mock recorder for MockDexRepo.
type MockDexRepoMockRecorder struct {
	mock *MockDexRepo
}

// NewMockDexRepo creates a new mock instance.
func NewMockDexRepo(ctrl *gomock.Controller) *MockDexRepo {
	mock := &MockDexRepo{ctrl: ctrl}
	mock.recorder = &MockDexRepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDexRepo) EXPECT() *MockDexRepoMockRecorder {
	return m.recorder
}

// RemoveRedirectURIs mocks base method.
func (m *MockDexRepo) RemoveRedirectURIs(redirectURIs string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveRedirectURIs", redirectURIs)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveRedirectURIs indicates an expected call of RemoveRedirectURIs.
func (mr *MockDexRepoMockRecorder) RemoveRedirectURIs(redirectURIs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveRedirectURIs", reflect.TypeOf((*MockDexRepo)(nil).RemoveRedirectURIs), redirectURIs)
}

// UpdateRedirectURIs mocks base method.
func (m *MockDexRepo) UpdateRedirectURIs(redirectURI string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRedirectURIs", redirectURI)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateRedirectURIs indicates an expected call of UpdateRedirectURIs.
func (mr *MockDexRepoMockRecorder) UpdateRedirectURIs(redirectURI interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRedirectURIs", reflect.TypeOf((*MockDexRepo)(nil).UpdateRedirectURIs), redirectURI)
}

// MockKubernetes is a mock of Kubernetes interface.
type MockKubernetes struct {
	ctrl     *gomock.Controller
	recorder *MockKubernetesMockRecorder
}

// MockKubernetesMockRecorder is the mock recorder for MockKubernetes.
type MockKubernetesMockRecorder struct {
	mock *MockKubernetes
}

// NewMockKubernetes creates a new mock instance.
func NewMockKubernetes(ctrl *gomock.Controller) *MockKubernetes {
	mock := &MockKubernetes{ctrl: ctrl}
	mock.recorder = &MockKubernetesMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKubernetes) EXPECT() *MockKubernetesMockRecorder {
	return m.recorder
}

// ListCodeRepo mocks base method.
func (m *MockKubernetes) ListCodeRepo(ctx context.Context) (*v1alpha1.CodeRepoList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCodeRepo", ctx)
	ret0, _ := ret[0].(*v1alpha1.CodeRepoList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCodeRepo indicates an expected call of ListCodeRepo.
func (mr *MockKubernetesMockRecorder) ListCodeRepo(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCodeRepo", reflect.TypeOf((*MockKubernetes)(nil).ListCodeRepo), ctx)
}

// ListCodeRepoBindings mocks base method.
func (m *MockKubernetes) ListCodeRepoBindings(ctx context.Context) (*v1alpha1.CodeRepoBindingList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCodeRepoBindings", ctx)
	ret0, _ := ret[0].(*v1alpha1.CodeRepoBindingList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCodeRepoBindings indicates an expected call of ListCodeRepoBindings.
func (mr *MockKubernetesMockRecorder) ListCodeRepoBindings(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCodeRepoBindings", reflect.TypeOf((*MockKubernetes)(nil).ListCodeRepoBindings), ctx)
}
