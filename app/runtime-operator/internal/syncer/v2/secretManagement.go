package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewSecretManagement func(opt v1alpha1.Component, info ComponentInitInfo) (SecretManagement, error)

type SecretManagement interface {
	// GetAccessInfo should return the infomation on how to access the cluster
	GetAccessInfo(ctx context.Context, clusterName string) (string, error)

	//// The cache will be stored and passed based on the product name + user name.

	CreateUser(ctx context.Context, user User, cache interface{}) (interface{}, error)
	DeleteUser(ctx context.Context, user User, cache interface{}) (interface{}, error)
	GrantPermission(ctx context.Context, repo SecretInfo, user User) error
	RevokePermission(ctx context.Context, repo SecretInfo, user User) error
}

type SecretType int32

const (
	SecretTypeCodeRepo = iota
	SecretTypeArtifactRepo
)

type SecretInfo struct {
	Type SecretType
	// CodeRepo 类型的密钥
	CodeRepo *CodeRepo
	// Artifact 类型的密钥
	ArtifactAccount *ArtifactAccount
}

type CodeRepoPermission string

const (
	CodeRepoPermissionReadOnly    = "readonly"
	CodeRepoPermissionReadWrite   = "readwrite"
	CodeRepoPermissionAccessToken = "accesstoken-api"
)

type CodeRepo struct {
	ProviderType string
	ID           string
	User         string
	Permission   CodeRepoPermission
}

type ArtifactAccount struct {
	ProviderName string
	Product      string
	Project      string
}
