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

package configs

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/nautes-labs/nautes/pkg/kubeconvert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ENV_CFG_NAMESPACE      = "NATUESCFGNAMESPACE"
	ENV_CFG_NAME           = "NATUESCFGNAME"
	DEFAULT_CFG_NAMESPACE  = "nautes"
	DEFAULT_CFG_NAME       = "nautes-configs"
	DefaultConfigName      = "nautes"
	DefaultConfigNamespace = "nautes-configs"
)

type NautesConfigs struct {
	Namespace string
	Name      string
}

func (nc *NautesConfigs) GetConfigByClient(client client.Client) (*Config, error) {
	cm := &corev1.ConfigMap{}
	err := client.Get(context.Background(), nc.GetNamespacedName(), cm)
	if err != nil {
		return nil, err
	}
	return NewConfig(cm.Data["config"])
}

func (nc *NautesConfigs) GetNamespacedName() types.NamespacedName {
	var name, namespace string
	var ok bool
	if nc.Namespace == "" {
		namespace, ok = os.LookupEnv(ENV_CFG_NAMESPACE)
		if !ok {
			namespace = DEFAULT_CFG_NAMESPACE
		}
	} else {
		namespace = nc.Namespace
	}

	if nc.Name == "" {
		name, ok = os.LookupEnv(ENV_CFG_NAME)
		if !ok {
			name = DEFAULT_CFG_NAME
		}
	} else {
		name = nc.Name
	}

	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

}

func (nc *NautesConfigs) GetConfigByRest(cfg *rest.Config) (*Config, error) {
	key := nc.GetNamespacedName()
	name := key.Name
	namespace := key.Namespace

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	cm, err := client.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return NewConfig(cm.Data["config"])
}

func NewConfigInstanceForK8s(namespace, configMap string, kubeconfig string) (*Config, error) {
	var client *kubernetes.Clientset
	var err error

	if kubeconfig == "" {
		restCFG, err := ctrl.GetConfig()
		if err != nil {
			return nil, err
		}

		client, err = kubernetes.NewForConfig(restCFG)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = kubeconvert.ConvertStringToKubeClient(kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	cm, err := client.CoreV1().ConfigMaps(namespace).Get(context.Background(), configMap, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return NewConfig(cm.Data["config"])
}

const DefaultNautesConfigPath = "/opt/nautes/config/config"
const EnvNautesConfigPath = "NAUTESCONFIGPATH"

type configOptions struct {
	filePath  string
	name      string
	namespace string
}

type configFunction func(*configOptions)

// NewNautesConfigFromFile will return a nautes config from specify path
// If path is empty. It will try to find files based on environment variables and default values
func NewNautesConfigFromFile(opts ...configFunction) (*Config, error) {
	var filePath string

	options := &configOptions{}
	for _, fn := range opts {
		fn(options)
	}

	filePathFromEnv, envExist := os.LookupEnv(EnvNautesConfigPath)

	if options.filePath != "" {
		filePath = options.filePath
	} else if envExist {
		filePath = filePathFromEnv
	} else {
		filePath = DefaultNautesConfigPath
	}

	configsByte, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return NewConfig(string(configsByte))
}

// NewNautesConfigFromKubernetes will return a nautes config from kubernetes
func NewNautesConfigFromKubernetes(ctx context.Context, k8sClient client.Client, opts ...configFunction) (*Config, error) {
	cm := &corev1.ConfigMap{}

	options := &configOptions{
		namespace: DefaultConfigNamespace,
		name:      DefaultConfigName,
	}

	for _, fn := range opts {
		fn(options)
	}

	namespacedNames := types.NamespacedName{
		Namespace: options.namespace,
		Name:      options.name,
	}

	if err := k8sClient.Get(ctx, namespacedNames, cm); err != nil {
		return nil, err
	}
	return NewConfig(cm.Data["config"])
}

func FilePath(path string) configFunction {
	return func(c *configOptions) { c.filePath = path }
}
