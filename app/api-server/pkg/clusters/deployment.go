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
	"html/template"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
)

func NewDeploymentServer(cluster *resourcev1alpha1.Cluster) (Deployment, error) {
	if cluster.Spec.ComponentsList.Deployment == nil {
		return nil, fmt.Errorf("failed to get deployment component")
	}

	component := cluster.Spec.ComponentsList.Deployment

	switch component.Name {
	case "argocd":
		return NewArgocd(), nil
	}

	return nil, fmt.Errorf("failed to get deployment component")
}

type Argocd struct {
	Host string
	URL  string
	Port string
}

func NewArgocd() Deployment {
	return &Argocd{}
}

func (t *Argocd) GetDefaultValue(field string, opt *DefaultValueOptions) (string, error) {
	if opt.Cluster == nil {
		return "", nil
	}

	ip, err := utilstring.ParseUrl(opt.Cluster.ApiServer)
	if err != nil {
		return "", fmt.Errorf("tekton host not filled in and automatic parse host cluster IP failed, err: %s", err)
	}

	return generateNipHost("argocd", opt.Cluster.Name, ip), nil
}

func (a *Argocd) RegisterTemplateFuncs() template.FuncMap {
	var funcMap = template.FuncMap{}

	funcMap["getDeploymentServer"] = a.GetDeploymentServer

	return funcMap
}

func (a *Argocd) GetDeploymentServer(param *ClusterRegistrationParams) *DeploymentServer {
	if param == nil {
		return &DeploymentServer{}
	}

	host := a.getHost(param.Cluster)
	url := a.getURL(param)

	return &DeploymentServer{Argocd: &Argocd{
		Host: host,
		URL:  url,
	}}
}

func (a *Argocd) GetOauthURL(param *ClusterRegistrationParams) (string, error) {
	if param == nil {
		return "", fmt.Errorf("failed to get argocd oauth url")
	}

	url := a.getURL(param)
	oauth := fmt.Sprintf("%s/%s", url, _ArgocdOAuthSuffix)

	return oauth, nil
}

func (a *Argocd) getHost(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil {
		return "[the cluster is nil pointer dereference]"
	}

	host := cluster.Spec.ComponentsList.Deployment.Additions[Host]

	return host
}

func (a *Argocd) getURL(param *ClusterRegistrationParams) string {
	var cluster = param.Cluster
	var httpsNodePort string
	var hostCluster = param.HostCluster

	host := a.getHost(cluster)
	if IsPhysical(cluster) {
		httpsNodePort = cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	} else {
		if hostCluster == nil {
			return "[failed to get host cluster when get argocd url]"
		}

		httpsNodePort = hostCluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	}

	// eg: https://argocd.dev-pipeline.10.204.118.214.nip.io:30443
	return fmt.Sprintf("https://%s:%s", host, httpsNodePort)
}

func generateNipHost(perfix, name, ip string) string {
	return fmt.Sprintf("%s.%s.%s.nip.io", perfix, name, ip)
}
