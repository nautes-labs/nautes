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

package argocd

import resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

type tlsClientConfig struct {
	Insecure bool   `json:"insecure"`
	CaData   string `json:"caData"`
	CertData string `json:"certData"`
	KeyData  string `json:"keyData"`
}

type ClusterDataConfig struct {
	TlsClientConfig tlsClientConfig `json:"tlsClientConfig"`
}

type Info struct {
	serverVersion string
}

type ConnectionState struct {
	AttemptedAt string `json:"attemptedAt"`
	Message     string `json:"message"`
	Status      string `json:"status"`
}

type ClusterResponse struct {
	Name            string          `json:"name"`
	ConnectionState ConnectionState `json:"connectionState"`
	Server          string          `json:"server"`
}

type ClusterData struct {
	Server string            `json:"server"`
	Name   string            `json:"name"`
	Config ClusterDataConfig `json:"config"`
	Info   Info              `json:"info"`
	Labels map[string]string `json:"labels"`
}

type ClusterInstance struct {
	Name            string                        `json:"name"`
	ApiServer       string                        `json:"apiServer"`
	ClusterType     string                        `json:"clusterType"`
	HostCluster     string                        `json:"hostType"`
	Provider        string                        `json:"provider"`
	CertificateData string                        `json:"certficateData"`
	CaData          string                        `json:"caData"`
	KeyData         string                        `json:"keyData"`
	Usage           resourcev1alpha1.ClusterUsage `json:"usage"`
}

type CodeRepo struct {
	argocd *ArgocdClient
}

type CodeRepoResponse struct {
	Name            string          `json:"name"`
	ConnectionState ConnectionState `json:"connectionState"`
	Repo            string          `json:"repo"`
}
