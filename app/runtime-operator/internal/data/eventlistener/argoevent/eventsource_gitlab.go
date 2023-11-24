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
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/google/uuid"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	targetPort           = 12000
	secretKeyAccessToken = "token"
	secretKeySecretToken = "token"
)

type GitlabEventSourceGenerator struct {
	Components     *component.ComponentList
	HostEntrypoint utils.EntryPoint
	Namespace      string
	K8sClient      client.Client
	DB             database.Snapshot
	User           component.MachineAccount
	Space          component.Space
	secMgrAuthInfo *component.AuthInfo
}

// CreateEventSource creates an event source resource of GitLab type by event source.
// 1. It builds an event source resource by unique ID.
// 2. It gets code repos that are used by the event source.
// 3. It creates the access token and secret by unique ID and code repo for request event source.
// 4. It deletes the access token and secret by unique ID and code repo for not in request event source.
// 5. It creates the entrypoint for event listener by unique ID.
// 6. It creates the event source resource which in Kubernetes.
func (gel *GitlabEventSourceGenerator) CreateEventSource(ctx context.Context, eventSource component.EventSourceSet) error {
	es := gel.buildBaseEventSource(eventSource.UniqueID)

	var codeRepoList string
	err := gel.K8sClient.Get(ctx, client.ObjectKeyFromObject(es), es)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if es.Annotations != nil {
		codeRepoList = es.Annotations[AnnotationCodeRepoUsage]
	}

	newCodeRepoSets := gel.getNewCodeRepoSets(eventSource.EventSources)
	oldCodeRepoSets := newCodeRepoUsage(codeRepoList)
	deleteCodeRepos := oldCodeRepoSets.Difference(newCodeRepoSets.Set).UnsortedList()

	for _, codeRepo := range newCodeRepoSets.UnsortedList() {
		if err := gel.createOrUpdateAccessToken(ctx, eventSource.UniqueID, codeRepo); err != nil {
			return fmt.Errorf("create access token failed: %w", err)
		}
		if err := gel.createOrUpdateSecretToken(ctx, eventSource.UniqueID, codeRepo); err != nil {
			return fmt.Errorf("create secret token failed: %w", err)
		}
	}

	for _, codeRepo := range deleteCodeRepos {
		if err := gel.deleteAccessToken(ctx, eventSource.UniqueID, codeRepo); err != nil {
			return fmt.Errorf("delete access token failed: %w", err)
		}
		if err := gel.deleteSecretToken(ctx, eventSource.UniqueID, codeRepo); err != nil {
			return fmt.Errorf("delete secret token failed: %w", err)
		}
	}

	if err := gel.CreateEntryPoint(ctx, eventSource.UniqueID); err != nil {
		return fmt.Errorf("create entrypoint failed: %w", err)
	}

	var oldEventSource []byte

	operationResult, err := controllerutil.CreateOrUpdate(ctx, gel.K8sClient, es, func() error {
		oldEventSource, _ = json.Marshal(es)
		gitlabEventSources, err := gel.createGitlabEventSources(eventSource.UniqueID, eventSource)
		if err != nil {
			return fmt.Errorf("generate gitlab event source failed: %w", err)
		}
		es.Spec.Gitlab = gitlabEventSources
		if es.Annotations == nil {
			es.Annotations = map[string]string{}
		}
		es.Annotations[AnnotationCodeRepoUsage] = newCodeRepoSets.ListAsString()
		return nil
	})

	if operationResult != controllerutil.OperationResultNone {
		newEventSource, _ := json.Marshal(es)
		logger.V(1).Info("event source has been modified", "name", es.Name, "operation", operationResult, "oldResource", string(oldEventSource), "newResource", string(newEventSource))
	}
	return err
}

// DeleteEventSource deletes an event source by event source collection unique ID.
// 1. It builds an event source resource by unique ID.
// 2. It gets code repos that are used by the event source.
// 3. It deletes the access token and secret by unique ID and code repo.
// 4. It deletes the entrypoint for event listener by unique ID.
// 5. It deletes the event source resource which in Kubernetes.
func (gel *GitlabEventSourceGenerator) DeleteEventSource(ctx context.Context, uniqueID string) error {
	es := gel.buildBaseEventSource(uniqueID)

	logger.V(1).Info("delete gitlab event source", "UniqueID", uniqueID)
	var codeRepoList string
	err := gel.K8sClient.Get(ctx, client.ObjectKeyFromObject(es), es)
	if err != nil {
		return client.IgnoreNotFound(err)
	} else if es.Annotations != nil {
		codeRepoList = es.Annotations[AnnotationCodeRepoUsage]
	}

	deleteCodeRepos := newCodeRepoUsage(codeRepoList).UnsortedList()
	for _, codeRepo := range deleteCodeRepos {
		if err := gel.deleteAccessToken(ctx, uniqueID, codeRepo); err != nil {
			return fmt.Errorf("delete access token failed: %w", err)
		}
		if err := gel.deleteSecretToken(ctx, uniqueID, codeRepo); err != nil {
			return fmt.Errorf("delete secret token failed: %w", err)
		}
	}

	if err := gel.DeleteEntryPoint(ctx, uniqueID); err != nil {
		return fmt.Errorf("create entrypoint failed: %w", err)
	}

	return gel.K8sClient.Delete(ctx, es)
}

// createGitlabEventSources transforms the event source to Gitlab event source instance.
func (gel *GitlabEventSourceGenerator) createGitlabEventSources(uniqueID string, eventSource component.EventSourceSet) (map[string]eventsourcev1alpha1.GitlabEventSource, error) {
	gitlabEventSources := map[string]eventsourcev1alpha1.GitlabEventSource{}

	for _, event := range eventSource.EventSources {
		if event.Gitlab == nil {
			continue
		}
		gitlabEventSource, err := gel.buildGitlabEventSource(uniqueID, event.Name, *event.Gitlab)
		if err != nil {
			return nil, err
		}
		gitlabEventSources[event.Name] = *gitlabEventSource
	}

	return gitlabEventSources, nil
}

// buildGitlabEventSource builds the event source instance.
func (gel *GitlabEventSourceGenerator) buildBaseEventSource(uniqueID string) *eventsourcev1alpha1.EventSource {
	return &eventsourcev1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildEventSourceName(uniqueID, component.EventTypeGitlab),
			Namespace: gel.Namespace,
		},
		Spec: eventsourcev1alpha1.EventSourceSpec{
			Template: gel.buildSpecTemplate(),
			Service: &eventsourcev1alpha1.Service{
				Ports: []corev1.ServicePort{
					{
						Port:       12000,
						TargetPort: intstr.FromInt(12000),
					},
				},
			},
		},
	}
}

// buildGitlabEventSource builds the Gitlab event source instance which uses the access token and secret.
func (gel *GitlabEventSourceGenerator) buildGitlabEventSource(uniqueID, eventName string, event component.EventSourceGitlab) (*eventsourcev1alpha1.GitlabEventSource, error) {
	webhookEvents, err := convertArgoEventSourceEventsFromCodeRepo(event.Events)
	if err != nil {
		return nil, err
	}

	return &eventsourcev1alpha1.GitlabEventSource{
		Webhook: &eventsourcev1alpha1.WebhookContext{
			Endpoint: buildWebhookPath(uniqueID, eventName),
			Method:   "POST",
			Port:     strconv.Itoa(targetPort),
			URL:      gel.HostEntrypoint.GetURL(),
		},
		Events: webhookEvents,
		AccessToken: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildAccessTokenName(uniqueID, event.CodeRepo),
			},
			Key: secretKeyAccessToken,
		},
		EnableSSLVerification: false,
		GitlabBaseURL:         event.APIServer,
		DeleteHookOnFinish:    true,
		Projects:              []string{event.RepoID},
		SecretToken: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildSecretTokenName(uniqueID, event.CodeRepo),
			},
			Key: secretKeySecretToken,
		},
	}, nil
}

func convertArgoEventSourceEventsFromCodeRepo(events []string) ([]string, error) {
	webhookEvents := []string{}
	for _, ev := range events {
		webhookEvent, ok := gitlabWebhookEventToArgoEventMapping[ev]
		if !ok {
			return nil, fmt.Errorf("event %s is unsupported", ev)
		}
		webhookEvents = append(webhookEvents, webhookEvent)
	}
	return webhookEvents, nil
}

// CreateEntryPoint creates an entry point for listening to the event source, the default is the Ingress of Kubernetes.
func (gel *GitlabEventSourceGenerator) CreateEntryPoint(ctx context.Context, uniqueID string) error {
	entrypoint := gel.buildEntrypoint(uniqueID)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      entrypoint.Name,
			Namespace: entrypoint.Destination.KubernetesService.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, gel.K8sClient, ingress, func() error {
		pathType := networkingv1.PathTypeImplementationSpecific
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: entrypoint.Source.Domain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     entrypoint.Source.Path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: entrypoint.Destination.KubernetesService.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(entrypoint.Destination.KubernetesService.Port),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	return err
}

// DeleteEntryPoint deletes the entry point of listening to the event source by event source collection unique ID.
func (gel *GitlabEventSourceGenerator) DeleteEntryPoint(ctx context.Context, uniqueID string) error {
	entrypoint := gel.buildEntrypoint(uniqueID)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      entrypoint.Name,
			Namespace: entrypoint.Destination.KubernetesService.Namespace,
		},
	}
	return client.IgnoreNotFound(gel.K8sClient.Delete(ctx, ingress))
}

// buildEntrypoint builds access entry using the logic entry point resource by event source collection unique ID.
func (gel *GitlabEventSourceGenerator) buildEntrypoint(uniqueID string) component.EntryPoint {
	return component.EntryPoint{
		Name: buildGatewayName(uniqueID),
		Source: component.EntryPointSource{
			Domain:   gel.HostEntrypoint.Domain,
			Path:     buildBasePath(uniqueID),
			Port:     gel.HostEntrypoint.Port,
			Protocol: gel.HostEntrypoint.Protocol,
		},
		Destination: component.EntryPointDestination{
			Type: component.DestinationTypeKubernetes,
			KubernetesService: &component.KubernetesService{
				Name:      buildServiceName(uniqueID, component.EventTypeGitlab),
				Namespace: gel.Namespace,
				Port:      targetPort,
			},
		},
	}
}

const (
	codeRepoProviderCAMountName = "certs-volume"
	gitlabCAConfigMapName       = "ca-certificates"
	gitlabCAMountPath           = "/etc/ssl/certs"
)

// buildSpecTemplate builds template resources for the event source.
func (gel *GitlabEventSourceGenerator) buildSpecTemplate() *eventsourcev1alpha1.Template {
	return &eventsourcev1alpha1.Template{
		Container: &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      codeRepoProviderCAMountName,
					MountPath: gitlabCAMountPath,
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: codeRepoProviderCAMountName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: gitlabCAConfigMapName,
						},
					},
				},
			},
		},
	}
}

// createOrUpdateSecretToken creates or updates secret which was used to GitLab webhook.
// It generates a random for the secret by uuid.
func (gel *GitlabEventSourceGenerator) createOrUpdateSecretToken(ctx context.Context, uniqueID, repoName string) error {
	name := buildSecretTokenName(uniqueID, repoName)

	secretToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gel.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(ctx, gel.K8sClient, secretToken, func() error {
		if secretToken.Data == nil {
			token := uuid.New().String()
			secretToken.Data = map[string][]byte{
				"token": []byte(token),
			}
		}
		return nil
	})

	if operation != controllerutil.OperationResultNone {
		log.FromContext(ctx).V(1).Info("resource modified", "name", name, "operation", operation)
	}

	return err
}

// deleteSecretToken deletes the secret that was used to GitLab webhook.
func (gel *GitlabEventSourceGenerator) deleteSecretToken(ctx context.Context, uniqueID, repoName string) error {
	name := buildSecretTokenName(uniqueID, repoName)

	secretToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gel.Namespace,
		},
	}

	err := gel.K8sClient.Delete(ctx, secretToken)
	return client.IgnoreNotFound(err)
}

// createOrUpdateAccessToken creates or updates access token in Kubernetes and grants secret policy from the vault, which was used to GitLab webhook.
// 1. It builds a secret request body by event source collection unique ID and repo name.
// 2. It grants repo permission for argo event user.
// 3. It syncs an access token to argo event namespace, which the access token is used to access the repository.
func (gel *GitlabEventSourceGenerator) createOrUpdateAccessToken(ctx context.Context, uniqueID, repoName string) error {
	secReq, err := gel.buildCodeRepoRequest(uniqueID, repoName)
	if err != nil {
		return fmt.Errorf("create secret request failed: %w", err)
	}

	if err := gel.Components.SecretManagement.GrantPermission(ctx, secReq.Source, gel.User); err != nil {
		return fmt.Errorf("grant coderepo access token permission to argo event user failed: %w", err)
	}

	return gel.Components.SecretSync.CreateSecret(ctx, *secReq)
}

// deleteAccessToken deletes access token in Kubernetes, which was used for GitLab webhook.
// 1. It builds a secret request body by event source collection unique ID and repo name.
// 2. It revokes repo permission for argo-event user.
// 3. It removes the access token which is used to access the repository.
func (gel *GitlabEventSourceGenerator) deleteAccessToken(ctx context.Context, uniqueID, repoName string) error {
	secReq, err := gel.buildCodeRepoRequest(uniqueID, repoName)
	if err != nil {
		return fmt.Errorf("create secret request failed: %w", err)
	}

	if err := gel.Components.SecretManagement.RevokePermission(ctx, secReq.Source, gel.User); err != nil {
		return fmt.Errorf("revoke code repo access token permission from argo event user failed: %w", err)
	}

	return gel.Components.SecretSync.RemoveSecret(ctx, *secReq)
}

// getNewCodeRepoSets transforms an array of the Event to Set.
func (gel *GitlabEventSourceGenerator) getNewCodeRepoSets(events []component.EventSource) codeRepoUsage {
	newSet := sets.New[string]()
	for i := range events {
		if events[i].Gitlab == nil {
			continue
		}
		newSet.Insert(events[i].Gitlab.CodeRepo)
	}
	return codeRepoUsage{newSet}
}

// buildCodeRepoRequest builds a secret request body that was used to create or delete a secret in secret management.
func (gel *GitlabEventSourceGenerator) buildCodeRepoRequest(uniqueID, repoName string) (*component.SecretRequest, error) {
	name := buildAccessTokenName(uniqueID, repoName)

	codeRepo, err := gel.DB.GetCodeRepo(repoName)
	if err != nil {
		return nil, fmt.Errorf("get code repo failed: %w", err)
	}
	provider, err := gel.DB.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	return &component.SecretRequest{
		Name: name,
		Source: component.SecretInfo{
			Type: component.SecretTypeCodeRepo,
			CodeRepo: &component.CodeRepo{
				ProviderType: provider.Spec.ProviderType,
				ID:           codeRepo.Name,
				User:         "default",
				Permission:   component.CodeRepoPermissionAccessToken,
			},
		},
		AuthInfo: gel.secMgrAuthInfo,
		Destination: component.SecretRequestDestination{
			Name:   name,
			Space:  gel.Space,
			Format: `token: '{{ .accesstoken | toString }}'`,
		},
	}, nil
}
