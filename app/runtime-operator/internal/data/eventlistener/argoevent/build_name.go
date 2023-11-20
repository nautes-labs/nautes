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

package argoevent

import (
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
)

func buildServiceName(uniqueID string, eventSourceType component.EventSourceType) string {
	return fmt.Sprintf("%s-%s-eventsource-svc", uniqueID, eventSourceType)
}

func buildGatewayName(uniqueID string) string {
	return fmt.Sprintf("%s-gitlab", uniqueID)
}

func buildAccessTokenName(uniqueID, repoName string) string {
	return fmt.Sprintf("%s-%s-access-token", uniqueID, repoName)
}

func buildSecretTokenName(uniqueID, repoName string) string {
	return fmt.Sprintf("%s-%s-secret-token", uniqueID, repoName)
}

func buildSensorName(productName, name string, num int) string {
	return fmt.Sprintf("%s-%s-%d", productName, name, num)
}

func buildEventSourceName(uniqueID string, eventSourceType component.EventSourceType) string {
	return fmt.Sprintf("%s-%s", uniqueID, eventSourceType)
}

func buildWebhookPath(basePath, eventSourceName string) string {
	return fmt.Sprintf("/%s/%s", basePath, eventSourceName)
}

func buildBasePath(basePath string) string {
	return fmt.Sprintf("/%s", basePath)
}

func buildDependencyName(uniqueID, eventSourceName string, eventSourceType component.EventSourceType) string {
	return fmt.Sprintf("%s-%s-%s", uniqueID, eventSourceName, eventSourceType)
}

func buildConsumerLabel(productName, name string) string {
	return fmt.Sprintf("%s-%s", productName, name)
}
