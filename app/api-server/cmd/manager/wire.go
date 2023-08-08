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

//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	"github.com/nautes-labs/nautes/app/api-server/internal/conf"
	"github.com/nautes-labs/nautes/app/api-server/internal/data"
	"github.com/nautes-labs/nautes/app/api-server/internal/server"
	"github.com/nautes-labs/nautes/app/api-server/internal/service"
	cluster "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// wireApp init kratos application.
func wireApp(confServer *conf.Server, logger log.Logger, nodesTree nodestree.NodesTree, config *configs.Config, client client.Client, clusteroperator cluster.ClusterRegistrationOperator) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
