// Copyright 2024 Nautes Authors
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

package middlewareruntime_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	runtimeconfig "github.com/nautes-labs/nautes/app/runtime-operator/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMiddlewareruntime(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Middlewareruntime Suite")
}

var (
	providerType = "MockProvider"
	callerType   = "MockBasicCaller"
	cfgPath      = "/tmp/runtimeconfig.yaml"
)

var _ = BeforeSuite(func() {
	err := os.Setenv(runtimeconfig.EnvNameRuntimeOperatorConfigPath, cfgPath)
	Expect(err).To(BeNil())

	cfgFile := `
providerCallerMapping:
  %s: %s
`
	cfgFile = fmt.Sprintf(cfgFile, providerType, callerType)
	err = os.WriteFile(cfgPath, []byte(cfgFile), 0644)
	Expect(err).To(BeNil())

	caller := &mockBasicCaller{
		CallerMeta: component.CallerMeta{
			Type:               callerType,
			ImplementationType: component.CallerImplBasic,
		},
	}

	component.AddFunctionNewCaller(callerType, func(info component.ProviderInfo) (component.Caller, error) {
		return caller, nil
	})
})

var _ = AfterSuite(func() {
	component.ClearCallerFactory()
	err := os.Remove(cfgPath)
	Expect(err).To(BeNil())
})

var testDataKubeconfig = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkakNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUzTURNME9EUTJPVFV3SGhjTk1qTXhNakkxTURZeE1UTTFXaGNOTXpNeE1qSXlNRFl4TVRNMQpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUzTURNME9EUTJPVFV3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFRUEMxVlVkdWFuNU1ML25OdTBlNUZVZEphWXpNZU5VeTdza1pqb3ZkTWgKRXJiYnlwVmZScnBsOW1EUk5sS1JrV0pkdkhwK0JIZVBYanBEdjhWeXdZWTVvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXF2bkJya3Njc1FIWTdhb2hMcjdyCnBDRWcraW93Q2dZSUtvWkl6ajBFQXdJRFJ3QXdSQUlnTWtoaUFaYkdNeUxvOVFNK2RaY2pUNHpYcklxSm0xd3oKOFFHV0JFMzJnSFFDSUhTaTNoTkU0UHRONWVwT3pZQmVweC9FMEpON3diWjZuejJaVk9rR3h6VksKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://127.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrVENDQVRlZ0F3SUJBZ0lJQldjY1c5Sk9UZ1F3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOekF6TkRnME5qazFNQjRYRFRJek1USXlOVEEyTVRFek5Wb1hEVEkwTVRJeQpOREEyTVRFek5Wb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJDb2IxSTlzc0c3YVZCZ2cKVGoxcWNWY1ZhRFl2c2NiQ09OK0lxeVBxZ3NuYTd3OUNsNzlXTENkeXY5NWtKdHVmQzlKSDdZVTlXakl3RFllYwprTU4xODhhalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUUlvZW1FWURzbFJhOHhmQVFoTENZa0pSWWt6REFLQmdncWhrak9QUVFEQWdOSUFEQkYKQWlCdFAyNnQ0TU84OWUyZGJyTkkxcVo3Yy9VeGZHTGZBeVlJUkxjM0ZuK3kyQUloQUs0aHdubFJhaUFpQ1pNQwp3UWszOElBb3c3Rzl5dndNcmRDN3hvMVhueFJDCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUzTURNME9EUTJPVFV3SGhjTk1qTXhNakkxTURZeE1UTTFXaGNOTXpNeE1qSXlNRFl4TVRNMQpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUzTURNME9EUTJPVFV3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFTQWJjd2NSbzlaVjVuYzFYY2czNzNteGZPanlGaG14ZTIxVXJMNFFvS3MKVXJGVngxM0pTTGFtakRyZTBhV2trRUc0M3E1RTdSTDArY05QZ2QyMGRkekFvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUNLSHBoR0E3SlVXdk1Yd0VJU3dtCkpDVVdKTXd3Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQU00S2ladFltL1ViZkt5S3NPWXZDREt4U2ppZ2RFa2wKRzlOK1AzbURnK1ZZQWlFQWltcXVNS1ErK2hSNmFTUmZtcGhkOUh6Z0xSRjlVOEozZ0RVS29VWVR6clE9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUg2WnJzeVVqWmJ1aWFacnlYTWM4VWR2TXF2YjY1cnNPemMrR1Qzbm8vNFBvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFS2h2VWoyeXdidHBVR0NCT1BXcHhWeFZvTmkreHhzSTQzNGlySStxQ3lkcnZEMEtYdjFZcwpKM0svM21RbTI1OEwwa2Z0aFQxYU1qQU5oNXlRdzNYenhnPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
`

type mockBasicCaller struct {
	component.CallerMeta
}

type mockRequest struct {
	PostResp []byte
	PostErr  error
}

func (m *mockBasicCaller) Post(ctx context.Context, request interface{}) ([]byte, error) {
	req, ok := request.(*mockRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}

	return req.PostResp, req.PostErr
}

type mockRequestTransformer struct {
	PostResp  []byte
	PostErr   error
	ParseResp *map[string]string
}

func (m *mockRequestTransformer) GenerateRequest(resource resources.Resource) (req interface{}, err error) {
	return &mockRequest{
		PostResp: m.PostResp,
		PostErr:  m.PostErr,
	}, nil
}

func (m *mockRequestTransformer) ParseResponse(response []byte) (state *resources.Status, err error) {
	if m.ParseResp == nil {
		return nil, nil
	}
	state = &resources.Status{
		Properties: *m.ParseResp,
	}
	return
}

// mkc *mockKubernetesClient client.Client
type mockKubernetesClient struct {
	SubResourceWriter client.SubResourceWriter
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (mkc *mockKubernetesClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	panic("not implemented") // TODO: Implement
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (mkc *mockKubernetesClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	panic("not implemented") // TODO: Implement
}

// Create saves the object obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (mkc *mockKubernetesClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	panic("not implemented") // TODO: Implement
}

// Delete deletes the given obj from Kubernetes cluster.
func (mkc *mockKubernetesClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	panic("not implemented") // TODO: Implement
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (mkc *mockKubernetesClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	panic("not implemented") // TODO: Implement
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (mkc *mockKubernetesClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	panic("not implemented") // TODO: Implement
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (mkc *mockKubernetesClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	panic("not implemented") // TODO: Implement
}

func (mkc *mockKubernetesClient) Status() client.SubResourceWriter {
	return mkc.SubResourceWriter
}

// SubResourceClientConstructor returns a subresource client for the named subResource. Known
// upstream subResources usages are:
//
//   - ServiceAccount token creation:
//     sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     token := &authenticationv1.TokenRequest{}
//     c.SubResourceClient("token").Create(ctx, sa, token)
//
//   - Pod eviction creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     c.SubResourceClient("eviction").Create(ctx, pod, &policyv1.Eviction{})
//
//   - Pod binding creation:
//     pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     binding := &corev1.Binding{Target: corev1.ObjectReference{Name: "my-node"}}
//     c.SubResourceClient("binding").Create(ctx, pod, binding)
//
//   - CertificateSigningRequest approval:
//     csr := &certificatesv1.CertificateSigningRequest{
//     ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"},
//     Status: certificatesv1.CertificateSigningRequestStatus{
//     Conditions: []certificatesv1.[]CertificateSigningRequestCondition{{
//     Type: certificatesv1.CertificateApproved,
//     Status: corev1.ConditionTrue,
//     }},
//     },
//     }
//     c.SubResourceClient("approval").Update(ctx, csr)
//
//   - Scale retrieval:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{}
//     c.SubResourceClient("scale").Get(ctx, dep, scale)
//
//   - Scale update:
//     dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}
//     scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 2}}
//     c.SubResourceClient("scale").Update(ctx, dep, client.WithSubResourceBody(scale))
func (mkc *mockKubernetesClient) SubResource(subResource string) client.SubResourceClient {
	panic("not implemented") // TODO: Implement
}

// Scheme returns the scheme this client is using.
func (mkc *mockKubernetesClient) Scheme() *runtime.Scheme {
	panic("not implemented") // TODO: Implement
}

// RESTMapper returns the rest this client is using.
func (mkc *mockKubernetesClient) RESTMapper() meta.RESTMapper {
	panic("not implemented") // TODO: Implement
}

// mksc *mockKubernetesSubResourceClient client.SubResourceWriter
type mockKubernetesSubResourceClient struct{}

// Create saves the subResource object in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (mksc *mockKubernetesSubResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	panic("not implemented") // TODO: Implement
}

// Update updates the fields corresponding to the status subresource for the
// given obj. obj must be a struct pointer so that obj can be updated
// with the content returned by the Server.
func (mksc *mockKubernetesSubResourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return nil
}

// Patch patches the given object's subresource. obj must be a struct
// pointer so that obj can be updated with the content returned by the
// Server.
func (mksc *mockKubernetesSubResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	panic("not implemented") // TODO: Implement
}
