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

	"k8s.io/kops/pkg/kubeconfig"
	"sigs.k8s.io/yaml"
)

func (r *ClusterReconciler) ConvertKubeconfig(data []byte) (*kubeconfig.KubectlConfig, error) {
	kubeconfig := &kubeconfig.KubectlConfig{}
	jsonData, err := yaml.YAMLToJSONStrict(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonData), kubeconfig)
	if err != nil {
		return nil, err
	}

	if kubeconfig.Clusters == nil {
		return nil, fmt.Errorf("kubeconfig parsing failed")
	}

	return kubeconfig, nil
}
