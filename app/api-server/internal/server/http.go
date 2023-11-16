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

package server

import (
	"context"
	"strings"

	mmd "github.com/go-kratos/kratos/v2/middleware/metadata"
	clusterv1 "github.com/nautes-labs/nautes/api/api-server/cluster/v1"
	coderepov1 "github.com/nautes-labs/nautes/api/api-server/coderepo/v1"
	coderepobindingv1 "github.com/nautes-labs/nautes/api/api-server/coderepobinding/v1"
	deploymentruntimev1 "github.com/nautes-labs/nautes/api/api-server/deploymentruntime/v1"
	environmentv1 "github.com/nautes-labs/nautes/api/api-server/environment/v1"
	productv1 "github.com/nautes-labs/nautes/api/api-server/product/v1"
	projectv1 "github.com/nautes-labs/nautes/api/api-server/project/v1"
	projectpipelineruntimev1 "github.com/nautes-labs/nautes/api/api-server/projectpipelineruntime/v1"
	"github.com/nautes-labs/nautes/app/api-server/internal/conf"
	"github.com/nautes-labs/nautes/app/api-server/internal/service"

	"github.com/go-kratos/grpc-gateway/v2/protoc-gen-openapiv2/generator"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/swagger-api/openapiv2"
)

type ServiceProductGroup struct {
	projectPipelineRuntime *service.ProjectPipelineRuntimeService
	deploymentRuntime      *service.DeploymentruntimeService
	codeRepo               *service.CodeRepoService
	codeRepoBinding        *service.CodeRepoBindingService
	product                *service.ProductService
	project                *service.ProjectService
	environment            *service.EnvironmentService
	cluster                *service.ClusterService
}

//nolint:lll
func NewServiceGroup(projectPipelineRuntime *service.ProjectPipelineRuntimeService, deploymentRuntime *service.DeploymentruntimeService, codeRepo *service.CodeRepoService, codeRepoBinding *service.CodeRepoBindingService, product *service.ProductService, project *service.ProjectService, environment *service.EnvironmentService, cluster *service.ClusterService) *ServiceProductGroup {
	return &ServiceProductGroup{
		projectPipelineRuntime: projectPipelineRuntime,
		deploymentRuntime:      deploymentRuntime,
		codeRepo:               codeRepo,
		codeRepoBinding:        codeRepoBinding,
		product:                product,
		project:                project,
		environment:            environment,
		cluster:                cluster,
	}
}

func (s *ServiceProductGroup) Register(srv *http.Server) {
	productv1.RegisterProductHTTPServer(srv, s.product)
	projectv1.RegisterProjectHTTPServer(srv, s.project)
	environmentv1.RegisterEnvironmentHTTPServer(srv, s.environment)
	clusterv1.RegisterClusterHTTPServer(srv, s.cluster)
	coderepov1.RegisterCodeRepoHTTPServer(srv, s.codeRepo)
	coderepobindingv1.RegisterCodeRepoBindingHTTPServer(srv, s.codeRepoBinding)
	deploymentruntimev1.RegisterDeploymentruntimeHTTPServer(srv, s.deploymentRuntime)
	projectpipelineruntimev1.RegisterProjectPipelineRuntimeHTTPServer(srv, s.projectPipelineRuntime)
}

// NewHTTPServer new a HTTP server.
func NewHTTPServer(c *conf.Server, serviceProductGroup *ServiceProductGroup) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			mmd.Server(),
			recovery.Recovery(),
			TokenWithContext(),
			validate.Validator(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	openAPIhandler := openapiv2.NewHandler(openapiv2.WithGeneratorOptions(generator.UseJSONNamesForFields(true), generator.EnumsAsInts(true)))
	srv := http.NewServer(opts...)
	srv.HandlePrefix("/q/", openAPIhandler)
	serviceProductGroup.Register(srv)
	return srv
}

const BearerToken = "token"
const Authorization = "Authorization"

func TokenWithContext() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				header := tr.RequestHeader()
				bearerToken := header.Get(Authorization)
				token := strings.TrimSpace(strings.Replace(bearerToken, "Bearer", "", 1))
				if token != "" {
					ctx = context.WithValue(ctx, BearerToken, token)
				} else {
					if md, ok := metadata.FromServerContext(ctx); ok {
						token := md.Get(Authorization)
						ctx = context.WithValue(ctx, BearerToken, token)
					}
				}
			}

			return handler(ctx, req)
		}
	}
}
