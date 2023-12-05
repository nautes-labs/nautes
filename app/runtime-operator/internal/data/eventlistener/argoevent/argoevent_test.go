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

package argoevent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/argoproj/argo-events/pkg/apis/common"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/eventlistener/argoevent"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Gitlab eventsource", func() {
	var ctx context.Context
	var evListener component.EventListener
	var es component.EventSourceSet
	var seed string
	var esNames []string
	var evNames []string
	var apiServer string
	var codeRepoNames []string
	var codeRepoIDs []string
	var uuids []string
	var clusterBaseURL string
	var argoEventCache *corev1.ConfigMap
	BeforeEach(func() {
		var err error
		ctx = context.TODO()
		opt := v1alpha1.Component{
			Name:      "argoevent",
			Namespace: argoEventNamespace,
			Additions: map[string]string{},
		}
		apiServer = "https://127.0.0.1:443"
		seed = RandNum()

		esNames = GenerateNames(fmt.Sprintf("esname-%s-%%d", seed), 5)
		evNames = GenerateNames(fmt.Sprintf("evname-%s-%%d", seed), 5)
		codeRepoNames = GenerateNames(fmt.Sprintf("coderepo-%s-%%d", seed), 5)
		codeRepoIDs = GenerateNames("%d", 5)
		uuids = GenerateNames(fmt.Sprintf("uuid-%s-%%d", seed), 5)

		argoEventCache = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      argoevent.CacheName,
				Namespace: argoEventNamespace,
			},
		}

		cluster := v1alpha1.Cluster{
			Spec: v1alpha1.ClusterSpec{
				ApiServer:     apiServer,
				ClusterType:   v1alpha1.CLUSTER_TYPE_PHYSICAL,
				PrimaryDomain: "domain.nip.io",
			},
			Status: v1alpha1.ClusterStatus{
				EntryPoints: map[string]v1alpha1.ClusterEntryPoint{
					"ppt": {
						HTTPPort:  0,
						HTTPSPort: 32443,
						Type:      v1alpha1.ServiceTypeNodePort,
					},
				},
			},
		}
		clusterName := fmt.Sprintf("cluster-%s", seed)
		clusterBaseURL = fmt.Sprintf("https://webhook.%s.domain.nip.io:32443", clusterName)
		provider := v1alpha1.CodeRepoProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "x",
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.CodeRepoProviderSpec{
				ApiServer:    apiServer,
				ProviderType: "gitlab",
			},
		}

		codeRepo := v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      codeRepoNames[0],
				Namespace: nautesNamespace,
			},
			Spec: v1alpha1.CodeRepoSpec{
				CodeRepoProvider: provider.Name,
				Product:          "",
				Project:          "",
				RepoName:         "",
				URL:              "",
			},
		}
		codeRepo2 := codeRepo.DeepCopy()
		codeRepo2.Name = codeRepoNames[1]

		db := &mockDB{
			cluster: cluster,
			codeRepos: []v1alpha1.CodeRepo{
				codeRepo,
				*codeRepo2,
			},
			provider: provider,
		}
		info := &component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &component.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName:            clusterName,
			NautesResourceSnapshot: db,
			NautesConfig: configs.Config{
				Nautes: configs.Nautes{
					Namespace: nautesNamespace,
				},
			},
			Components: &component.ComponentList{
				SecretManagement: &mockSecMgr{},
				SecretSync:       &mockSecSyncer{},
			},
			EventSourceSearchEngine: &mockRuleEngine{},
		}

		evListener, err = argoevent.NewArgoEvent(opt, info)
		Expect(err).Should(BeNil())

		es = component.EventSourceSet{
			ResourceMetaData: component.ResourceMetaData{
				Product: "",
				Name:    esNames[0],
			},
			UniqueID: uuids[0],
			EventSources: []component.EventSource{
				{
					Name: evNames[0],
					Gitlab: &component.EventSourceGitlab{
						APIServer: apiServer,
						Events:    []string{"push_events"},
						CodeRepo:  codeRepo.Name,
						RepoID:    codeRepoIDs[0],
					},
				},
			},
		}

	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, argoEventCache)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
	})

	It("can create gitlab eventsource", func() {
		err := evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())

		currentEs := &eventsourcev1alpha1.EventSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", uuids[0], component.EventTypeGitlab),
				Namespace: argoEventNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(currentEs), currentEs)
		Expect(err).Should(BeNil())
		ev := es.EventSources[0]
		gitlabEvent, ok := currentEs.Spec.Gitlab[ev.Name]
		Expect(ok).Should(BeTrue())
		Expect(gitlabEvent.GitlabBaseURL).Should(Equal(apiServer))
		Expect(gitlabEvent.Projects).Should(Equal([]string{codeRepoIDs[0]}))

		destWebHook := &eventsourcev1alpha1.WebhookContext{
			Endpoint: fmt.Sprintf("/%s/%s", uuids[0], ev.Name),
			Method:   "POST",
			Port:     "12000",
			URL:      clusterBaseURL,
		}
		Expect(reflect.DeepEqual(destWebHook, gitlabEvent.Webhook)).Should(BeTrue())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoEventCache), argoEventCache)
		Expect(err).Should(BeNil())
		_, ok = argoEventCache.Data[es.UniqueID]
		Expect(ok).Should(BeTrue())
	})

	It("if code repo changed, webhook and token will change", func() {
		err := evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())

		oldSecretTokenName := fmt.Sprintf("%s-%s-secret-token", es.UniqueID, es.EventSources[0].Gitlab.CodeRepo)
		es.EventSources[0].Gitlab.CodeRepo = codeRepoNames[1]
		es.EventSources[0].Gitlab.RepoID = codeRepoIDs[1]
		newSecretTokenName := fmt.Sprintf("%s-%s-secret-token", es.UniqueID, es.EventSources[0].Gitlab.CodeRepo)

		err = evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())

		currentEs := &eventsourcev1alpha1.EventSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", uuids[0], component.EventTypeGitlab),
				Namespace: argoEventNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(currentEs), currentEs)
		Expect(err).Should(BeNil())
		ev := es.EventSources[0]
		gitlabEvent, ok := currentEs.Spec.Gitlab[ev.Name]
		Expect(ok).Should(BeTrue())
		Expect(gitlabEvent.GitlabBaseURL).Should(Equal(apiServer))
		Expect(gitlabEvent.Projects).Should(Equal([]string{codeRepoIDs[1]}))

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoEventCache), argoEventCache)
		Expect(err).Should(BeNil())
		_, ok = argoEventCache.Data[es.UniqueID]
		Expect(ok).Should(BeTrue())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      oldSecretTokenName,
				Namespace: argoEventNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		secret.Name = newSecretTokenName
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		Expect(err).Should(BeNil())
	})

	It("can remove gitlab eventsource", func() {
		err := evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())
		err = evListener.DeleteEventSource(ctx, es.UniqueID)
		Expect(err).Should(BeNil())

		currentEs := &eventsourcev1alpha1.EventSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", uuids[0], component.EventTypeGitlab),
				Namespace: argoEventNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(currentEs), currentEs)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoEventCache), argoEventCache)
		Expect(err).Should(BeNil())
		_, ok := argoEventCache.Data[es.UniqueID]
		Expect(ok).ShouldNot(BeTrue())
	})

	It("can change event type", func() {
		err := evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())
		es.EventSources = []component.EventSource{
			{
				Name: evNames[1],
				Calendar: &component.EventSourceCalendar{
					Schedule:       "* * * * *",
					Interval:       "2",
					ExclusionDates: []string{"20"},
					Timezone:       "utc-8",
				},
			},
		}
		err = evListener.CreateEventSource(ctx, es)
		Expect(err).Should(BeNil())

		currentEs := &eventsourcev1alpha1.EventSource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", uuids[0], component.EventTypeGitlab),
				Namespace: argoEventNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(currentEs), currentEs)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		currentEs.Name = fmt.Sprintf("%s-%s", uuids[0], component.EventTypeCalendar)
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(currentEs), currentEs)
		Expect(err).Should(BeNil())

		destSpecCalendar := map[string]eventsourcev1alpha1.CalendarEventSource{
			es.EventSources[0].Name: {
				Schedule:       es.EventSources[0].Calendar.Schedule,
				Interval:       es.EventSources[0].Calendar.Interval,
				ExclusionDates: es.EventSources[0].Calendar.ExclusionDates,
				Timezone:       es.EventSources[0].Calendar.Timezone,
			},
		}
		ok := reflect.DeepEqual(currentEs.Spec.Calendar, destSpecCalendar)
		Expect(ok).Should(BeTrue())
	})
})

var _ = Describe("Sensor", func() {
	var ctx context.Context
	var evListener component.EventListener
	var seed string
	var evNames []string
	var apiServer string
	var uuids []string
	var productName string
	var serviceAccounts []string
	var consumerNames []string
	var consumer component.ConsumerSet
	var resBytes []byte

	type res struct {
		Go string
	}
	BeforeEach(func() {
		var err error
		ctx = context.TODO()
		opt := v1alpha1.Component{
			Name:      "argoevent",
			Namespace: argoEventNamespace,
			Additions: map[string]string{},
		}
		apiServer = "https://127.0.0.1:443"
		seed = RandNum()

		evNames = GenerateNames(fmt.Sprintf("evname-%s-%%d", seed), 5)
		uuids = GenerateNames(fmt.Sprintf("uuid-%s-%%d", seed), 5)
		productName = fmt.Sprintf("product-%s", seed)
		consumerNames = GenerateNames(fmt.Sprintf("cs-%s-%%d", seed), 5)
		serviceAccounts = GenerateNames(fmt.Sprintf("sa-%s-%%d", seed), 5)

		cluster := v1alpha1.Cluster{
			Spec: v1alpha1.ClusterSpec{
				ApiServer:     apiServer,
				ClusterType:   v1alpha1.CLUSTER_TYPE_PHYSICAL,
				PrimaryDomain: "domain.nip.io",
			},
			Status: v1alpha1.ClusterStatus{
				EntryPoints: map[string]v1alpha1.ClusterEntryPoint{
					"ppt": {
						HTTPPort:  0,
						HTTPSPort: 32443,
						Type:      v1alpha1.ServiceTypeNodePort,
					},
				},
			},
		}

		db := &mockDB{
			cluster: cluster,
		}
		info := &component.ComponentInitInfo{
			ClusterConnectInfo: component.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &component.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName:            "",
			NautesResourceSnapshot: db,
			NautesConfig: configs.Config{
				Nautes: configs.Nautes{
					Namespace: nautesNamespace,
				},
			},
			Components: &component.ComponentList{
				SecretManagement: &mockSecMgr{},
				SecretSync:       &mockSecSyncer{},
			},
			EventSourceSearchEngine: &mockRuleEngine{},
		}

		evListener, err = argoevent.NewArgoEvent(opt, info)
		Expect(err).Should(BeNil())

		res := &res{
			Go: "123",
		}
		resBytes, err = json.Marshal(res)
		Expect(err).Should(BeNil())

		consumer = component.ConsumerSet{
			ResourceMetaData: component.ResourceMetaData{
				Product: productName,
				Name:    consumerNames[0],
			},
			Account: component.MachineAccount{
				Product: productName,
				Name:    serviceAccounts[0],
			},
			Consumers: []component.Consumer{
				{
					UniqueID:        uuids[0],
					EventSourceName: evNames[0],
					EventSourceType: component.EventTypeGitlab,
					Filters: []component.Filter{
						{
							Key:        "headers.X-Gitlab-Event",
							Value:      "Tag Push Hook",
							Comparator: component.EqualTo,
						},
					},
					Task: component.EventTask{
						Type: component.EventTaskTypeRaw,
						Vars: []component.InputOverWrite{
							{
								RequestVar: component.EventSourceVarRef,
								Dest:       "spec.params.1.value",
							},
						},
						Raw: res,
					},
				},
			},
		}
	})

	It("can create sensor", func() {
		err := evListener.CreateConsumer(ctx, consumer)
		Expect(err).Should(BeNil())

		sensorName := fmt.Sprintf("%s-%s-0", consumer.Product, consumer.Name)
		sensor := &sensorv1alpha1.Sensor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sensorName,
				Namespace: argoEventNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(sensor), sensor)
		Expect(err).Should(BeNil())

		dependName := fmt.Sprintf("%s-%s-%s",
			consumer.Consumers[0].UniqueID,
			consumer.Consumers[0].EventSourceName,
			consumer.Consumers[0].EventSourceType,
		)
		destSpec := sensorv1alpha1.SensorSpec{
			Dependencies: []sensorv1alpha1.EventDependency{
				{
					Name:            dependName,
					EventSourceName: fmt.Sprintf("%s-%s", uuids[0], component.EventTypeGitlab),
					EventName:       evNames[0],
					Filters: &sensorv1alpha1.EventDependencyFilter{
						Data: []sensorv1alpha1.DataFilter{
							{
								Path: consumer.Consumers[0].Filters[0].Key,
								Type: "string",
								Value: []string{
									consumer.Consumers[0].Filters[0].Value,
								},
								Comparator: "=",
							},
						},
					},
				},
			},
			Triggers: []sensorv1alpha1.Trigger{
				{
					Template: &sensorv1alpha1.TriggerTemplate{
						Name:       dependName,
						Conditions: dependName,
						K8s: &sensorv1alpha1.StandardK8STrigger{
							Operation: sensorv1alpha1.Create,
							Source: &sensorv1alpha1.ArtifactLocation{
								Resource: &common.Resource{
									Value: resBytes,
								},
							},
							Parameters: []sensorv1alpha1.TriggerParameter{
								{
									Src: &sensorv1alpha1.TriggerParameterSource{
										DependencyName: dependName,
										DataKey:        "body.ref",
										UseRawData:     false,
									},
									Dest: consumer.Consumers[0].Task.Vars[0].Dest,
								},
							},
						},
					},
				},
			},
			Template: &sensorv1alpha1.Template{
				ServiceAccountName: argoevent.ServiceAccountArgoEvents,
			},
		}
		ok := reflect.DeepEqual(destSpec.Dependencies, sensor.Spec.Dependencies)
		Expect(ok).Should(BeTrue())
		ok = reflect.DeepEqual(destSpec.Triggers, sensor.Spec.Triggers)
		Expect(ok).Should(BeTrue())
		ok = reflect.DeepEqual(destSpec.Template, sensor.Spec.Template)
		Expect(ok).Should(BeTrue())
	})

	It("can delete sensor", func() {
		err := evListener.CreateConsumer(ctx, consumer)
		Expect(err).Should(BeNil())

		err = evListener.DeleteConsumer(ctx, consumer.Product, consumer.Name)
		Expect(err).Should(BeNil())

		listOpt := []client.ListOption{
			client.MatchingLabels{argoevent.LabelConsumer: fmt.Sprintf("%s-%s", consumer.Product, consumer.Name)},
			client.InNamespace(argoEventNamespace),
		}
		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, sensorList, listOpt...)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(0))
	})
})
