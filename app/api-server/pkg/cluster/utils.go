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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	argocdapplicationv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdapplicationsetv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	gopkgyaml "gopkg.in/yaml.v2"
	yaml "sigs.k8s.io/yaml"
)

const (
	minNodePort = 30000
	maxNodePort = 32767
)

func readFile(filePath string) (data []byte, err error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	data, err = ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	return
}

func GetVclusterNames(filePath string) (vclusterNames []string, err error) {
	data, err := readFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return vclusterNames, nil
		} else {
			return nil, err
		}
	}
	data = []byte(ReplacePlaceholders(string(data), "{{vcluster}}", "vcluster"))
	return GetApplicationSetElements(data)
}

func GetHostClusterNames(filePath string) (hostClusterNames []string, err error) {
	data, err := readFile(filePath)
	if err != nil {
		return hostClusterNames, nil
	}

	data = []byte(ReplacePlaceholders(string(data), "{{cluster}}", "cluster"))
	str, err := GetApplicationSetElements(data)
	if err != nil {
		return
	}

	return str, nil
}

func GetApplicationSetElements(data []byte) ([]string, error) {

	var as argocdapplicationsetv1alpha1.ApplicationSet
	var elements []string
	err := yaml.Unmarshal(data, &as)
	if err != nil {
		return nil, err
	}

	for _, element := range as.Spec.Generators[0].List.Elements {
		var m map[string]interface{}
		err := json.Unmarshal(element.Raw, &m)
		if err != nil {
			return nil, err
		}
		clusterName, ok := m["cluster"].(string)
		if !ok {
			return nil, fmt.Errorf("unable to obtain cluster information")
		}
		elements = append(elements, clusterName)
	}

	return elements, nil
}

func ReplacePlaceholders(data string, placeholder, value string) string {
	replacements := map[string]string{
		placeholder: value,
	}

	for placeholder, value := range replacements {
		data = strings.ReplaceAll(data, placeholder, value)
	}

	return data
}

func AddIfNotExists(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

func GenerateNodePort(usedPorts []int) int {
	for {
		port := rand.Intn(maxNodePort-minNodePort+1) + minNodePort
		if port >= minNodePort && port <= maxNodePort {
			if !contains(usedPorts, port) {
				return port
			}
		}
	}
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseArgocdApplication(fileName string) (*argocdapplicationv1alpha1.Application, error) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil, nil
	}

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var app argocdapplicationv1alpha1.Application
	if yaml.Unmarshal(bytes, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func getTraefikHttpsNodePort(app *argocdapplicationv1alpha1.Application) (int, error) {
	var values struct {
		Ports struct {
			WebSecure struct {
				NodePort int `yaml:"nodePort"`
			} `yaml:"websecure"`
		} `yaml:"ports"`
	}

	if err := yaml.Unmarshal([]byte(app.Spec.Source.Helm.Values), &values); err != nil {
		return 0, fmt.Errorf("failed to unmarshal values YAML: %w", err)
	}

	return values.Ports.WebSecure.NodePort, nil
}

func parseCluster(fileName string) (*resourcev1alpha1.Cluster, error) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var cluster resourcev1alpha1.Cluster
	if err := yaml.Unmarshal(bytes, &cluster); err != nil {
		return nil, err
	}

	return &cluster, nil
}

func RemoveStringFromArray(arr []string, target string) []string {
	for i := 0; i < len(arr); i++ {
		if arr[i] == target {
			arr = append(arr[:i], arr[i+1:]...)
			i--
		}
	}
	return arr
}

func generateNipHost(perfix, name, ip string) string {
	return fmt.Sprintf("%s.%s.%s.nip.io", perfix, name, ip)
}

type Ingress struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Namespace   string `yaml:"namespace"`
		Annotations struct {
			IngressClass string `yaml:"kubernetes.io/ingress.class"`
		} `yaml:"annotations"`
	} `yaml:"metadata"`
	Spec struct {
		TLS []struct {
			Hosts []string `yaml:"hosts"`
		} `yaml:"tls"`
		Rules []struct {
			Host string `yaml:"host"`
			HTTP struct {
				Paths []struct {
					Path     string `yaml:"path"`
					PathType string `yaml:"pathType"`
					Backend  struct {
						Service struct {
							Name string `yaml:"name"`
							Port struct {
								Number int `yaml:"number"`
							} `yaml:"port"`
						} `yaml:"service"`
					} `yaml:"backend"`
				} `yaml:"paths"`
			} `yaml:"http"`
		} `yaml:"rules"`
	} `yaml:"spec"`
}

func parseIngresses(bytes string) ([]Ingress, error) {
	ingresses := make([]Ingress, 0)
	decoder := gopkgyaml.NewDecoder(strings.NewReader(bytes))

	for {
		var ingress Ingress
		err := decoder.Decode(&ingress)
		if err != nil {
			break
		}
		ingresses = append(ingresses, ingress)
	}

	return ingresses, nil
}

func isExistDir(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
