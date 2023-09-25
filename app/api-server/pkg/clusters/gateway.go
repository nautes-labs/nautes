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
)

type Traefik struct {
	HttpsNodePort string
	HttpNodePort  string
	Middlewares   []*Middleware
}

type Middleware struct {
	OAuth  *MiddlewareOauth
	Errors *MiddlewareError
}

type MiddlewareOauth struct {
	Name string
}

type MiddlewareError struct {
	Name string
	URL  string
}

func NewTraefik() Gateway {
	return &Traefik{}
}

func (t *Traefik) GetDefaultValue(field string, opt *DefaultValueOptions) (string, error) {
	return "", nil
}

func (t *Traefik) GetGatewayServer(param *ClusterRegistrationParams) *GatewayServer {
	if param == nil {
		return nil
	}

	var cluster = param.Cluster
	var hostCluster = param.HostCluster
	var httpsNodePort, httpNodePort string

	if IsPhysical(cluster) {
		httpsNodePort = cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
		httpNodePort = cluster.Spec.ComponentsList.Gateway.Additions[HttpNodePort]
	} else {
		httpsNodePort = hostCluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
		httpNodePort = hostCluster.Spec.ComponentsList.Gateway.Additions[HttpNodePort]
	}

	middlewares := t.getMiddlewares(param)

	return &GatewayServer{
		Traefik: &Traefik{
			HttpsNodePort: httpsNodePort,
			HttpNodePort:  httpNodePort,
			Middlewares:   middlewares,
		},
	}
}

func (t *Traefik) getMiddlewares(param *ClusterRegistrationParams) []*Middleware {
	var middlewares []*Middleware

	if IsPhysicalProjectPipelineRuntime(param.Cluster) {
		host := param.Cluster.Spec.ComponentsList.Pipeline.Additions[Host]
		httpsNodePort := param.Cluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]
		middlewareBase := &Middleware{
			OAuth: &MiddlewareOauth{
				Name: param.Cluster.Name,
			},
		}
		middleware := *middlewareBase
		middleware.Errors = &MiddlewareError{
			Name: param.Cluster.Name,
			URL:  fmt.Sprintf("https://%s:%s", host, httpsNodePort),
		}

		return []*Middleware{&middleware}
	}

	// The host cluster will traverse the virtual pipeline cluster to obtain middleware data.
	hostCluster := param.Cluster
	httpsNodePort := hostCluster.Spec.ComponentsList.Gateway.Additions[HttpsNodePort]

	for _, cluster := range param.Clusters {
		if IsVirtualProjectPipelineRuntime(&cluster) &&
			cluster.Spec.HostCluster == param.Cluster.Name {
			host := cluster.Spec.ComponentsList.Pipeline.Additions[Host]
			middlewares = append(middlewares, &Middleware{
				Errors: &MiddlewareError{
					Name: cluster.Name,
					URL:  fmt.Sprintf("https://%s:%s", host, httpsNodePort),
				},
				OAuth: &MiddlewareOauth{
					Name: cluster.Name,
				},
			})
		}
	}

	return middlewares
}
