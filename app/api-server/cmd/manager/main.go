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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nautes-labs/nautes/app/api-server/internal/conf"
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"

	"net/http/pprof"

	nethppt "net/http"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	clustermanagement "github.com/nautes-labs/nautes/app/api-server/pkg/clusters"
	"github.com/nautes-labs/nautes/pkg/log/zap"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"github.com/nautes-labs/nautes/pkg/queue/nautesqueue"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string
	// global config
	globalConfigNamespace string
	globalConfigName      string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.StringVar(&globalConfigName, "global-config-name", "nautes-configs", "The resources name of global config.")
	flag.StringVar(&globalConfigNamespace, "global-config-namespace", "nautes", "The namespace of global config in.")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	pprofMux := nethppt.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	server := &nethppt.Server{Addr: fmt.Sprintf(":%d", 6060), Handler: pprofMux} //nolint:gosec
	go server.ListenAndServe()                                                   //nolint:errcheck

	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(zap.NewLogger(),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	err := logger.Log(-1, "global-config-namespace", globalConfigNamespace, "global-config-name", globalConfigName)
	if err != nil {
		panic(err)
	}

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	k8sClient, err := kubernetes.NewClient()
	if err != nil {
		panic(err)
	}

	nodesTree, err := NewNodestree(k8sClient)
	if err != nil {
		panic(err)
	}

	gloabalConfigs, err := GetNautesConfigs(k8sClient, globalConfigNamespace, globalConfigName)
	if err != nil {
		panic(err)
	}

	clusterOperator, err := NewClusterOperator()
	if err != nil {
		panic(err)
	}

	stop := make(chan struct{})
	q := nautesqueue.NewQueue(stop, 1)

	app, cleanup, err := wireApp(bc.Server, logger, nodesTree, gloabalConfigs, k8sClient, clusterOperator, q)
	if err != nil {
		panic(err)
	}

	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func NewNodestree(k8sClient client.Client) (nodestree.NodesTree, error) {
	resourcesLayout, err := nodestree.NewConfig()
	if err != nil {
		return nil, err
	}

	fileOptions := &nodestree.FileOptions{
		ExclusionsSuffix: []string{".txt", ".md"},
		ContentType:      nodestree.CRDContentType,
	}
	nodesTree := nodestree.NewNodestree(fileOptions, resourcesLayout, k8sClient)

	return nodesTree, nil
}

func GetNautesConfigs(k8sClient client.Client, namespace, configName string) (nautesConfigs *nautesconfigs.Config, err error) {
	c := nautesconfigs.NautesConfigs{
		Namespace: namespace,
		Name:      configName,
	}
	nautesConfigs, err = c.GetConfigByClient(k8sClient)
	if err != nil {
		return
	}
	return
}

func NewClusterOperator() (clustermanagement.ClusterRegistrationOperator, error) {
	clusterFileOperation, err := clustermanagement.NewClusterConfigFile()
	if err != nil {
		return nil, err
	}

	fileOptions := &nodestree.FileOptions{
		ExclusionsSuffix: []string{".txt", ".md"},
		ContentType:      nodestree.StringContentType,
	}
	nodesTree := nodestree.NewNodestree(fileOptions, nil, nil)

	clusterOperator, err := clustermanagement.NewClusterManagement(clusterFileOperation, nodesTree)
	if err != nil {
		return nil, err
	}

	return clusterOperator, nil
}
