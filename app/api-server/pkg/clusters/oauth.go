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
)

func NewOAuthProxyServer(cluster *resourcev1alpha1.Cluster) (OAuthProxy, error) {
	if cluster.Spec.ComponentsList.OauthProxy == nil {
		return nil, fmt.Errorf("failed to get oauth proxy component")
	}

	component := cluster.Spec.ComponentsList.OauthProxy
	if component.Name == "oauth2-proxy" {
		return NewOAuth2Proxy(), nil
	}

	return nil, fmt.Errorf("failed to get oauth proxy component")
}

type OAuth2Proxy struct {
	Hosts           string
	WhitelistDomain string
	Redirect        string
	Ingresses       []*Ingress
}

type Ingress struct {
	Name string
	Host string
}

func NewOAuth2Proxy() OAuthProxy {
	return &OAuth2Proxy{}
}

func (o *OAuth2Proxy) GetOauthProxyServer(param *ClusterRegistrationParams) *OAuthProxyServer {
	cluster := param.Cluster
	redirect := o.GenerateOAuthProxyRedirect(cluster)
	whitelistDomain := o.generateOAuthProxyWhitelistDomain(cluster)
	hosts := o.getOAuthProxyHosts(cluster)
	ingresses := o.getIngress(param)

	return &OAuthProxyServer{
		OAuth2Proxy: &OAuth2Proxy{
			Hosts:           hosts,
			WhitelistDomain: whitelistDomain,
			Redirect:        redirect,
			Ingresses:       ingresses,
		},
	}
}

func (o *OAuth2Proxy) GenerateOAuthProxyRedirect(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil {
		return ""
	}

	httpsNodePort, ok := cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	if !ok || httpsNodePort == "" {
		return "[err: failed to get https node port]"
	}
	redirectURL := fmt.Sprintf("https://auth.%s.%s:%s/oauth2/callback", cluster.Name, cluster.Spec.PrimaryDomain, httpsNodePort)

	return redirectURL
}

func (o *OAuth2Proxy) generateOAuthProxyWhitelistDomain(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil {
		return ""
	}

	httpsNodePort := cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
	if httpsNodePort == "" {
		return ""
	}

	whitelistDomain := fmt.Sprintf(".%s:%s", cluster.Spec.PrimaryDomain, httpsNodePort)
	return whitelistDomain
}

func (o *OAuth2Proxy) getOAuthProxyHosts(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil {
		return ""
	}
	tlsHosts := fmt.Sprintf("auth.%s.%s", cluster.Name, cluster.Spec.PrimaryDomain)
	return tlsHosts
}

func (o *OAuth2Proxy) getIngress(param *ClusterRegistrationParams) []*Ingress {
	var hosts []*Ingress
	var clusters = param.Clusters

	if IsPhysicalProjectPipelineRuntime(param.Cluster) {
		host := o.getIngressHost(param.Cluster)
		hosts = append(hosts, &Ingress{
			Name: param.Cluster.GetName(),
			Host: host,
		})

		return hosts
	}

	for i, cluster := range clusters {
		if IsVirtualProjectPipelineRuntime(&clusters[i]) &&
			cluster.Spec.HostCluster == param.Cluster.Name {
			host := o.getIngressHost(&clusters[i])
			hosts = append(hosts, &Ingress{
				Name: cluster.GetName(),
				Host: host,
			})
		}
	}

	return hosts
}

func (o *OAuth2Proxy) getIngressHost(cluster *resourcev1alpha1.Cluster) string {
	if cluster == nil || cluster.Spec.WorkerType != resourcev1alpha1.ClusterWorkTypePipeline {
		return ""
	}

	return cluster.Spec.ComponentsList.Pipeline.Additions["host"]
}
