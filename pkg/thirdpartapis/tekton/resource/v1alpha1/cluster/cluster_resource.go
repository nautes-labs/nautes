/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	resource "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

// Resource represents a cluster configuration (kubeconfig)
// that can be accessed by tasks in the pipeline
type Resource struct {
	Name string                        `json:"name"`
	Type resource.PipelineResourceType `json:"type"`
	// URL must be a host string
	URL      string `json:"url"`
	Revision string `json:"revision"`
	// Server requires Basic authentication
	Username  string `json:"username"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"`
	// Server requires Bearer authentication. This client will not attempt to use
	// refresh tokens for an OAuth2 flow.
	// Token overrides userame and password
	Token string `json:"token"`
	// Server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	CAData []byte `json:"cadata"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS.
	ClientKeyData []byte `json:"clientKeyData"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS.
	ClientCertificateData []byte `json:"clientCertificateData"`
	// Secrets holds a struct to indicate a field name and corresponding secret name to populate it
	Secrets []resource.SecretParam `json:"secrets"`

	KubeconfigWriterImage string `json:"-"`
	ShellImage            string `json:"-"`
}
