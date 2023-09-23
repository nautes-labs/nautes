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
	"html/template"
	"io"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
)

const (
	minNodePort = 30000
	maxNodePort = 32767
)

type TemplateOperation interface {
	RegisterFunc(funcMap template.FuncMap)
	Parse(text string) error
	Execute(wr io.Writer, data any) error
	Clone() (TemplateOperation, error)
}

type Template struct {
	t *template.Template
}

type TemplateFunc func(funcMap *template.FuncMap, customFunc func())

func NewTemplate(name string) TemplateOperation {
	var ins = &Template{}

	t := template.New(name).Funcs(template.FuncMap{
		"split":                    strings.Split,
		"getClusterVariable":       getClusterVariable,
		"randomString":             utilstring.RandStr,
		"getDomain":                utilstring.GetDomain,
		"encodeToString":           utilstring.EncodeToString,
		"getVclusterHttpsNodePort": getVclusterHttpsNodePort,
		"getHostClusterApiServer":  getHostClusterApiServer,
		"getVclusterTLSSan":        getVclusterTLSSan,
		"getPipelineServer":        ins.getPipelineServer,
		"getOauthProxyServer":      getOauthProxyServer,
		"getGatewayServer":         getGatewayServer,
		"getDeploymentServer":      getDeploymentServer,
	})
	ins.t = t

	return ins
}

func (t *Template) RegisterFunc(funcMap template.FuncMap) {
	t.t.Funcs(funcMap)
}

func (t *Template) Parse(text string) error {
	if _, err := t.t.Parse(text); err != nil {
		return err
	}

	return nil
}

func (t *Template) Execute(wr io.Writer, data any) error {
	return t.t.Execute(wr, data)
}

func (t *Template) Clone() (TemplateOperation, error) {
	tmp, err := t.t.Clone()
	if err != nil {
		return nil, err
	}

	return &Template{t: tmp}, nil
}

func getClusterVariable() string {
	return "{{cluster}}"
}

func getVclusterHttpsNodePort(vcluster *VclusterInfo) string {
	if vcluster != nil && vcluster.HttpsNodePort != "" {
		return vcluster.HttpsNodePort
	}

	randPort := rand.Intn(maxNodePort-minNodePort+1) + minNodePort
	return strconv.Itoa(randPort)
}

func getVclusterTLSSan(hostCluster *resourcev1alpha1.Cluster) string {
	if hostCluster == nil {
		return ""
	}

	hostClusterIP, err := utilstring.ParseUrl(hostCluster.Spec.ApiServer)
	if err != nil {
		return ""
	}

	return hostClusterIP
}

func getHostClusterApiServer(clusters []resourcev1alpha1.Cluster, hostClusterName string) string {
	for _, cluster := range clusters {
		if cluster.Name == hostClusterName {
			return cluster.Spec.ApiServer
		}
	}

	return ""
}

func (t *Template) getPipelineServer(param *ClusterRegistrationParams) *PipelineServer {
	if param == nil || param.Cluster == nil || param.Cluster.Spec.ComponentsList.Pipeline == nil {
		return nil
	}

	var pipeline = param.Cluster.Spec.ComponentsList.Pipeline.Name

	components := NewComponentsList()
	for componentName, component := range components.Pipeline {
		if string(componentName) == pipeline {
			fn := reflect.ValueOf(component).MethodByName("GetPipelineServer")
			if !fn.IsValid() {
				return nil
			}
			arg1 := reflect.ValueOf(param)
			args := []reflect.Value{arg1}
			results := fn.Call(args)
			if len(results) == 0 {
				return nil
			}

			result, ok := results[0].Interface().(*PipelineServer)
			if !ok {
				return nil
			}

			return result
		}
	}

	return nil
}

func getOauthProxyServer(param *ClusterRegistrationParams) *OAuthProxyServer {
	if param == nil || param.Cluster == nil || param.Cluster.Spec.ComponentsList.OauthProxy == nil {
		return nil
	}

	var cluster = param.Cluster
	var componentName = cluster.Spec.ComponentsList.OauthProxy.Name

	switch componentName {
	case "oauth2-proxy":
		t := NewOAuth2Proxy()
		return t.GetOauthProxyServer(param)
	}

	return nil
}

func getGatewayServer(param *ClusterRegistrationParams) *GatewayServer {
	if param == nil || param.Cluster == nil || param.Cluster.Spec.ComponentsList.Gateway == nil {
		return nil
	}

	var componentName = param.Cluster.Spec.ComponentsList.Gateway.Name

	switch componentName {
	case "traefik":
		t := NewTraefik()
		return t.GetGatewayServer(param)
	}

	return nil
}

func getDeploymentServer(param *ClusterRegistrationParams) *DeploymentServer {
	if param == nil || param.Cluster == nil || param.Cluster.Spec.ComponentsList.Deployment == nil {
		return nil
	}

	var cluster = param.Cluster
	var componentName = cluster.Spec.ComponentsList.Deployment.Name

	switch componentName {
	case "argocd":
		t := NewArgocd()
		return t.GetDeploymentServer(param)
	}

	return nil
}
