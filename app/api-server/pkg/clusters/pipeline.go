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

package cluster

import (
	"fmt"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
)

func NewPipelineServer(cluster *resourcev1alpha1.Cluster) (Pipeline, error) {
	if cluster.Spec.ComponentsList.Pipeline == nil {
		return nil, fmt.Errorf("failed to get deployment component")
	}

	component := cluster.Spec.ComponentsList.Pipeline
	if component.Name == "tekton" {
		return NewTekton(), nil
	}

	return nil, fmt.Errorf("failed to get deployment component")
}

type Tekton struct {
	Hosts string
	URL   string
}

func NewTekton() Pipeline {
	return &Tekton{}
}

func (t *Tekton) GetDefaultValue(_ string, opt *DefaultValueOptions) (string, error) {
	if opt.Cluster == nil {
		return "", nil
	}

	ip, err := utilstring.ParseUrl(opt.Cluster.ApiServer)
	if err != nil {
		return "", fmt.Errorf("tekton host not filled in and automatic parse host cluster IP failed, err: %s", err)
	}

	return generateNipHost("tekton", opt.Cluster.Name, ip), nil
}

func (t *Tekton) GetPipelineServer(param *ClusterRegistrationParams) *PipelineServer {
	hosts, ok := param.Cluster.Spec.ComponentsList.Pipeline.Additions[Host]
	if !ok {
		return nil
	}

	url := t.getURL(param)

	return &PipelineServer{
		Tekton: &Tekton{
			Hosts: hosts,
			URL:   url,
		},
	}
}

func (t *Tekton) getURL(param *ClusterRegistrationParams) string {
	var cluster = param.Cluster
	var hostCluster = param.HostCluster
	var httpsNodePort string

	host := t.getHost(cluster)
	if IsPhysical(cluster) {
		httpsNodePort = cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	} else {
		if hostCluster == nil {
			return "[failed to get host cluster when get argocd url]"
		}

		httpsNodePort = hostCluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	}

	// eg: https://tekton.dev-pipeline.10.204.118.214.nip.io:30443
	url := fmt.Sprintf("https://%s:%s", host, httpsNodePort)

	return url
}

func (t *Tekton) getHost(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil {
		return "[the cluster is nil pointer dereference]"
	}

	host := cluster.Spec.ComponentsList.Pipeline.Additions[Host]

	return host
}
