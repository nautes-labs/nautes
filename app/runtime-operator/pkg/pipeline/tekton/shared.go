package shared

import "github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"

type ResourceType string

const (
	ResourceTypeCodeRepoSSHKey      ResourceType = "sshKey"
	ResourceTypeCodeRepoAccessToken ResourceType = "accessToken"
)

type ResourceRequest struct {
	Type        ResourceType                `json:"type"`
	SSHKey      *ResourceRequestSSHKey      `json:"sshKey,omitempty"`
	AccessToken *ResourceRequestAccessToken `json:"accessToken,omitempty"`
}

type ResourceRequestSSHKey struct {
	ResourceName string               `json:"resourceName"`
	SecretInfo   component.SecretInfo `json:"secretInfo"`
}

type ResourceRequestAccessToken struct {
	ResourceName string               `json:"resourceName"`
	SecretInfo   component.SecretInfo `json:"secretInfo"`
}
