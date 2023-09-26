package argoevent

import (
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
)

func buildServiceName(uniqueID string, eventType syncer.EventType) string {
	return fmt.Sprintf("%s-%s-eventsource-svc", uniqueID, eventType)
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

func buildEventSourceName(uniqueID string, eventType syncer.EventType) string {
	return fmt.Sprintf("%s-%s", uniqueID, eventType)
}

func buildWebhookPath(basePath, eventName string) string {
	return fmt.Sprintf("/%s/%s", basePath, eventName)
}

func buildBasePath(basePath string) string {
	return fmt.Sprintf("/%s", basePath)
}

func buildDependencyName(uniqueID, eventName string, eventType syncer.EventType) string {
	return fmt.Sprintf("%s-%s-%s", uniqueID, eventName, eventType)
}

func buildConsumerLabel(productName, name string) string {
	return fmt.Sprintf("%s-%s", productName, name)
}
