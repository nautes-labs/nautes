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

package kubernetes

import (
	"os"
	"os/user"
	"path"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	kubeConfigFileEnv = "KUBECONFIG"
)

func NewKubernetes() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func NewClient() (client.Client, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig, err := getKubeConfigFile()
		if err != nil {
			return nil, err
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	resourcev1alpha1.AddToScheme(scheme.Scheme)
	k8sClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	return k8sClient, nil
}

func getKubeConfigFile() (string, error) {
	kubeConfigFile := os.Getenv(kubeConfigFileEnv)
	if kubeConfigFile == "" {
		user, err := user.Current()
		if err != nil {
			return "", err
		}

		kubeConfigFile = path.Join(user.HomeDir, ".kube", "config")
		_, err = os.Stat(kubeConfigFile)
		if err != nil {
			return "", err
		}
	}

	return kubeConfigFile, nil
}
