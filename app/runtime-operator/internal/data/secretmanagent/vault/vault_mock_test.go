package vault_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	vaultapi "github.com/hashicorp/vault/api"
	vaultproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	vproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
	vproxyMock "github.com/nautes-labs/nautes/test/mock/vaultproxy"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

const (
	mockClient = "MockClient"
)

func initMock() {
	ctl := gomock.NewController(GinkgoT())
	secClient = vproxyMock.NewMockSecretHTTPClient(ctl)
	secClient.EXPECT().CreateCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.ClusterRequest, optxs ...interface{}) (*vproxy.CreateClusterReply, error) {
			cluster, err := req.ConvertRequest()
			Expect(err).Should(BeNil())
			sec, err := vaultClientRoot.KVv2(cluster.SecretName).Put(ctx, cluster.SecretPath, map[string]interface{}{
				"kubeconfig": req.GetAccount().Kubeconfig,
			})
			Expect(err).Should(BeNil())

			return &vproxy.CreateClusterReply{
				Secret: &vproxy.SecretInfo{
					Name:    cluster.SecretName,
					Path:    cluster.SecretPath,
					Version: int32(sec.VersionMetadata.Version),
				},
			}, nil
		})
	secClient.EXPECT().DeleteCluster(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.ClusterRequest, optxs ...interface{}) (*vproxy.DeleteClusterReply, error) {
			cluster, err := req.ConvertRequest()
			Expect(err).Should(BeNil())
			err = vaultClientRoot.KVv2(cluster.SecretName).DeleteMetadata(ctx, cluster.SecretPath)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)

	authClient = vproxyMock.NewMockAuthHTTPClient(ctl)
	authClient.EXPECT().CreateAuth(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthRequest, optxs ...interface{}) (*vproxy.CreateAuthReply, error) {
			configPath := fmt.Sprintf("auth/%s/config", req.ClusterName)

			cfg, err := vaultClientRoot.Logical().Read(configPath)
			Expect(err).Should(BeNil())

			if cfg == nil {
				err := vaultClientRoot.Sys().EnableAuthWithOptions(req.ClusterName, &vaultapi.MountInput{Type: "kubernetes"})
				Expect(err).Should(BeNil())
			}

			opts := map[string]interface{}{
				"kubernetes_host":    req.Kubernetes.Url,
				"kubernetes_ca_cert": req.Kubernetes.Cabundle,
				"token_reviewer_jwt": req.Kubernetes.Token,
			}
			_, err = vaultClientRoot.Logical().Write(configPath, opts)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	authClient.EXPECT().DeleteAuth(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthRequest, optxs ...interface{}) (*vproxy.DeleteAuthReply, error) {
			err := vaultClientRoot.Sys().DisableAuth(req.ClusterName)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	authClient.EXPECT().CreateAuthrole(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthroleRequest, optxs ...interface{}) (*vproxy.CreateAuthReply, error) {
			path := fmt.Sprintf("auth/%s/role/%s", req.ClusterName, req.DestUser)
			opts := map[string]interface{}{
				"bound_service_account_namespaces": req.GetKubernetes().Namespaces,
				"bound_service_account_names":      req.GetKubernetes().ServiceAccounts,
			}
			_, err := vaultClientRoot.Logical().Write(path, opts)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	authClient.EXPECT().DeleteAuthrole(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, req *vproxy.AuthroleRequest, optxs ...interface{}) (*vproxy.CreateAuthReply, error) {
			path := fmt.Sprintf("auth/%s/role/%s", req.ClusterName, req.DestUser)
			_, err := vaultClientRoot.Logical().Delete(path)
			Expect(err).Should(BeNil())
			return nil, nil
		},
	)
	grantClient = vproxyMock.NewMockAuthGrantHTTPClient(ctl)
	grantClient.EXPECT().GrantAuthroleGitPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(grantPermission)
	grantClient.EXPECT().RevokeAuthroleGitPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(revokePermission)
	grantClient.EXPECT().GrantAuthroleRepoPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(grantPermission)
	grantClient.EXPECT().RevokeAuthroleRepoPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(revokePermission)
	grantClient.EXPECT().GrantAuthroleClusterPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(grantPermission)
	grantClient.EXPECT().RevokeAuthroleClusterPolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(revokePermission)
}

func grantPermission(ctx context.Context, req vproxy.AuthGrantRequest, optxs ...interface{}) (*vproxy.GrantAuthrolePolicyReply, error) {
	authMeta, secretMeta, err := req.ConvertToAuthPolicyReqeuest()
	Expect(err).Should(BeNil())
	role, err := vaultClientRoot.Logical().Read(authMeta.RolePath)
	Expect(err).Should(BeNil())
	Expect(role).ShouldNot(BeNil())
	hasKey := false
	for _, policy := range role.Data["token_policies"].([]interface{}) {
		if policy == secretMeta.PolicyName {
			hasKey = true
			break
		}
	}
	if !hasKey {
		role.Data["token_policies"] = append(role.Data["token_policies"].([]interface{}), secretMeta.PolicyName)
	}
	_, err = vaultClientRoot.Logical().Write(authMeta.RolePath, role.Data)
	Expect(err).Should(BeNil())
	return nil, nil
}

func revokePermission(ctx context.Context, req vproxy.AuthGrantRequest, optxs ...interface{}) (*vproxy.RevokeAuthrolePolicyReply, error) {
	policyMeta, secretMeta, err := req.ConvertToAuthPolicyReqeuest()
	Expect(err).Should(BeNil())
	role, err := vaultClientRoot.Logical().Read(policyMeta.RolePath)
	Expect(err).Should(BeNil())
	if role == nil {
		return nil, vproxy.ErrorResourceNotFound("resource not found")
	}
	var newPolicyList []interface{}
	for _, v := range role.Data["token_policies"].([]interface{}) {
		if v != secretMeta.PolicyName {
			newPolicyList = append(newPolicyList, v)
		}
	}
	role.Data["token_policies"] = newPolicyList
	_, err = vaultClientRoot.Logical().Write(policyMeta.RolePath, role.Data)
	Expect(err).Should(BeNil())
	return nil, nil
}

func getMockVaultProxyClient(_ string) (vaultproxy.SecretHTTPClient, vaultproxy.AuthHTTPClient, vaultproxy.AuthGrantHTTPClient, error) {
	return secClient, authClient, grantClient, nil
}
