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

package controller

import (
	"os"
	"strconv"
	"time"

	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetNautesConfigs(c client.Client, namspace, name string) (nautesConfigs *nautesconfigs.Config, err error) {
	config := nautesconfigs.NautesConfigs{
		Namespace: namspace,
		Name:      name,
	}
	nautesConfigs, err = config.GetConfigByClient(c)
	if err != nil {
		return
	}
	return
}

const ReconcileTime = "ReconcileTime"

func GetReconcileTime() time.Duration {
	defaultTime := 60 * time.Second

	value := os.Getenv(ReconcileTime)
	if value != "" {
		newTime, err := strconv.Atoi(value)
		if err != nil {
			return defaultTime
		}

		return time.Duration(newTime) * time.Second
	}

	return defaultTime
}
