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

package data

import (
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	gitlabclient "github.com/nautes-labs/nautes/app/api-server/pkg/gitlab"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"

	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clientCertDir = "/opt/nautes/keypair"
	caCertDir     = "/opt/nautes/cert"
	gitSSLCAInfo  = "GIT_SSL_CAINFO"
	httpsPort     = "443"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewCodeRepo, NewSecretRepo, NewGitRepo, NewDexRepo, NewKubernetes)

func NewData(logger log.Logger, _ *nautesconfigs.Config) (func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return cleanup, nil
}

func NewGitRepo(config *nautesconfigs.Config) (biz.GitRepo, error) {
	if config == nil {
		return nil, fmt.Errorf("failed to get global config when initializing git")
	}

	url, err := utilstring.GetURl(config.Git.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get gitlab url when initializing git, err: %s", err)
	}

	caCertPath := ""
	if url.Port == httpsPort {
		caCertPath = fmt.Sprintf("%s/%s.crt", caCertDir, url.HostName)
	} else {
		caCertPath = fmt.Sprintf("%s/%s_%s.crt", caCertDir, url.HostName, url.Port)
	}

	err = os.Setenv(gitSSLCAInfo, caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to set GIT_SSL_CAINFO, err: %w", err)
	}

	return &gitRepo{config: config}, nil
}

func NewSecretRepo(config *nautesconfigs.Config) (biz.Secretrepo, error) {
	// Get secret platform type according to configuration information
	if config.Secret.RepoType == "vault" {
		return NewVaultClient(config)
	}

	return nil, fmt.Errorf("failed to generate secret repo")
}

func NewCodeRepo(config *nautesconfigs.Config) (biz.CodeRepo, error) {
	// Get secret platform type according to configuration information
	if config.Git.GitType == "gitlab" {
		operator := gitlabclient.NewGitlabOperator()
		return NewGitlabRepo(config.Git.Addr, operator)
	}

	return nil, nil
}

func NewDexRepo(k8sClient client.Client) biz.DexRepo {
	return &Dex{k8sClient: k8sClient}
}
