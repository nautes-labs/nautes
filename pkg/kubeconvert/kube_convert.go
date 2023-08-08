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

package kubeconvert

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func ConvertStringToRawConfig(kubeconfig string) (clientcmdapi.Config, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		return clientcmdapi.Config{}, err
	}
	return clientConfig.RawConfig()
}

func ConvertStringToRestConfig(kubeconfig string) (*rest.Config, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}

func ConvertStringToKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
	restConfig, err := ConvertStringToRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

func ConvertRestConfigToApiConfig(rst rest.Config) clientcmdapi.Config {
	cfg := clientcmdapi.Config{
		Kind:        "Config",
		APIVersion:  "v1",
		Preferences: clientcmdapi.Preferences{},
		Clusters: map[string]*clientcmdapi.Cluster{"default": {

			Server:                   rst.Host,
			InsecureSkipTLSVerify:    false,
			CertificateAuthorityData: rst.TLSClientConfig.CAData,
		}},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{"default": {
			ClientCertificateData: rst.TLSClientConfig.CertData,
			ClientKeyData:         rst.TLSClientConfig.KeyData,
		}},
		Contexts: map[string]*clientcmdapi.Context{"default": {
			Cluster:   "default",
			AuthInfo:  "default",
			Namespace: "default",
		}},
		CurrentContext: "default",
	}
	return cfg
}
