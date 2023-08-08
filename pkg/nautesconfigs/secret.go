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

type SecretStoreType string

const (
	SECRET_STORE_VAULT SecretStoreType = "vault"
)

type SecretRepo struct {
	RepoType     SecretStoreType   `yaml:"repoType"`
	Vault        Vault             `yaml:"vault"`
	OperatorName map[string]string `yaml:"operatorName"`
}

type Vault struct {
	Addr string `yaml:"addr"`
	// Url for vault proxy
	ProxyAddr string `yaml:"proxyAddr"`
	CABundle  string `yaml:"CABundle"`
	PKIPath   string `yaml:"PKIPath"`
	// The auth name of current cluster
	MountPath string `yaml:"mountPath"`
	// Specify the token connect to vault, this use for debug, do not use it in product
	Token string `yaml:"token"`
	// Valut kubernetes auth service account namespace
	Namesapce string `yaml:"namespace"`
	// The service account name when create kubernetes auth in vault
	ServiceAccount string `yaml:"serviceAccount"`
}
