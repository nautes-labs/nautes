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

package argoevents_test

import (
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/eventbus/argoevents"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/constant"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var syncer interfaces.EventBus
var task interfaces.RuntimeSyncTask

var _ = Describe("Argoevents", func() {
	var accessInfo interfaces.AccessInfo
	var runtime *nautescrd.ProjectPipelineRuntime
	var cluster nautescrd.Cluster
	var product nautescrd.Product
	var pipelineRepo nautescrd.CodeRepo
	var sourceRepo nautescrd.CodeRepo
	BeforeEach(func() {
		accessInfo = interfaces.AccessInfo{
			Type:       interfaces.ACCESS_TYPE_K8S,
			Kubernetes: cfg,
		}

		cluster = nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%s", randNum()),
				Namespace: nautesCFG.Nautes.Namespace,
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:   "https://127.0.0.1:6443",
				ClusterType: nautescrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind: nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:       nautescrd.CLUSTER_USAGE_WORKER,
				HostCluster: "",
			},
			Status: nautescrd.ClusterStatus{
				EntryPoints: map[string]nautescrd.ClusterEntryPoint{
					"go": {
						HTTPPort:  31080,
						HTTPSPort: 31443,
						Type:      nautescrd.ServiceTypeNodePort,
					},
				},
			},
		}

		product = nautescrd.Product{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("product-%s", randNum()),
				Namespace: nautesCFG.Nautes.Namespace,
			},
			Spec: nautescrd.ProductSpec{
				Name:         fmt.Sprintf("testProduct%s", randNum()),
				MetaDataPath: "",
			},
		}

		pipelineRepo = nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  fakeProvider.Name,
				Product:           product.Name,
				Project:           "",
				RepoName:          "test",
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   true,
				Webhook:           &nautescrd.Webhook{},
			},
		}

		sourceRepo = nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-src-%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  fakeProvider.Name,
				Product:           product.Name,
				Project:           "",
				RepoName:          "src",
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook: &nautescrd.Webhook{
					Events: []string{
						"push_events",
					},
				},
			},
		}

		destination := nautescrd.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("env-%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.EnvironmentSpec{
				Product: product.Name,
				Cluster: cluster.Name,
				EnvType: "kubernetes",
			},
		}

		pipelineName := fmt.Sprintf("pipelineName-%s", randNum())
		evName := fmt.Sprintf("ev-name-%s", randNum())
		runtime = &nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pipeline%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.ProjectPipelineRuntimeSpec{
				Project:        "",
				PipelineSource: pipelineRepo.Name,
				Pipelines: []nautescrd.Pipeline{
					{
						Name:  pipelineName,
						Label: "",
						Path:  "pipelines",
					},
				},
				Destination: nautescrd.ProjectPipelineDestination{
					Environment: destination.Name,
				},
				EventSources: []nautescrd.EventSource{
					{
						Name: evName,
						Gitlab: &nautescrd.Gitlab{
							RepoName: sourceRepo.Name,
							Revision: "main",
							Events: []string{
								"push_events",
							},
						},
					},
				},
				Isolation: "",
				PipelineTriggers: []nautescrd.PipelineTrigger{
					{
						EventSource: evName,
						Pipeline:    pipelineName,
					},
				},
			},
		}

		fakeCodeRepos = map[string]nautescrd.CodeRepo{
			pipelineRepo.Name: pipelineRepo,
			sourceRepo.Name:   sourceRepo,
		}

		task = interfaces.RuntimeSyncTask{
			AccessInfo:         accessInfo,
			Product:            product,
			Cluster:            cluster,
			NautesCfg:          *nautesCFG,
			Runtime:            runtime,
			RuntimeType:        interfaces.RUNTIME_TYPE_PIPELINE,
			ServiceAccountName: constant.ServiceAccountDefault,
		}

		syncer = argoevents.NewSyncer(mockK8SClient)
	})

	AfterEach(func() {
		var err error
		err = syncer.RemoveEvents(ctx, task)
		Expect(err).Should(BeNil())

		isCleaned := false
		for i := 0; i < 30; i++ {
			esList := &eventsourcev1alpha1.EventSourceList{}
			sensorList := &sensorv1alpha1.SensorList{}
			err = k8sClient.List(ctx, esList)
			Expect(err).Should(BeNil())
			err = k8sClient.List(ctx, sensorList)
			Expect(err).Should(BeNil())
			if len(esList.Items) == 0 && len(sensorList.Items) == 0 {
				isCleaned = true
				break
			}
			time.Sleep(time.Second)
		}
		Expect(isCleaned).Should(BeTrue())
	})

	It("when eventsource has one type of event, pipeline runtime will create a eventsource and one sensor", func() {
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		esList := &eventsourcev1alpha1.EventSourceList{}
		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, esList)
		Expect(err).Should(BeNil())
		Expect(len(esList.Items)).Should(Equal(1))
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
	})

	It("when event source has two types of event, pipeline runtime will create two eventsource and one sensor", func() {
		runtime.Spec.EventSources[0].Calendar = &nautescrd.Calendar{
			Schedule: "15 * * * *",
		}
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		esList := &eventsourcev1alpha1.EventSourceList{}
		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, esList)
		Expect(err).Should(BeNil())
		Expect(len(esList.Items)).Should(Equal(2))
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
		Expect(len(sensorList.Items[0].Spec.Dependencies)).Should(Equal(2))
		Expect(len(sensorList.Items[0].Spec.Triggers)).Should(Equal(2))
	})

	It("when two event sources have same type, pipeline runtime will create on eventsource and one sensor", func() {
		codeRepo := nautescrd.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("repo-%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.CodeRepoSpec{
				CodeRepoProvider:  fakeProvider.Name,
				Product:           product.Name,
				Project:           "",
				RepoName:          "test2",
				URL:               "",
				DeploymentRuntime: false,
				PipelineRuntime:   false,
				Webhook: &nautescrd.Webhook{
					Events: []string{"push_events"},
				},
			},
		}
		evName := fmt.Sprintf("ev-name-%s", randNum())
		fakeCodeRepos[codeRepo.Name] = codeRepo
		runtime.Spec.EventSources = append(runtime.Spec.EventSources, nautescrd.EventSource{
			Name: evName,
			Gitlab: &nautescrd.Gitlab{
				RepoName: codeRepo.Name,
				Revision: "main",
				Events:   []string{},
			},
		})
		runtime.Spec.PipelineTriggers = append(runtime.Spec.PipelineTriggers, nautescrd.PipelineTrigger{
			EventSource: evName,
			Pipeline:    runtime.Spec.Pipelines[0].Name,
		})
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		esList := &eventsourcev1alpha1.EventSourceList{}
		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, esList)
		Expect(err).Should(BeNil())
		Expect(len(esList.Items)).Should(Equal(1))
		Expect(len(esList.Items[0].Spec.Gitlab)).Should(Equal(2))
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
		Expect(len(sensorList.Items[0].Spec.Dependencies)).Should(Equal(2))
		Expect(len(sensorList.Items[0].Spec.Triggers)).Should(Equal(2))
	})

	It("when nautes eventsource events and code repo webhook events is diff, argo event eventsource will use coderepo define, sensor trigger will use eventsource define", func() {
		runtime.Spec.EventSources[0].Gitlab.Events = []string{"tag_push_events"}
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		esList := &eventsourcev1alpha1.EventSourceList{}
		err = k8sClient.List(ctx, esList)
		Expect(err).Should(BeNil())
		Expect(len(esList.Items)).Should(Equal(1))
		evName := fmt.Sprintf("%s-%s-%s",
			runtime.Name,
			runtime.Spec.PipelineTriggers[0].EventSource,
			runtime.Spec.PipelineTriggers[0].Pipeline,
		)
		isSame := reflect.DeepEqual(esList.Items[0].Spec.Gitlab[evName].Events, []string{"PushEvents"})
		Expect(isSame).Should(BeTrue())

		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
		isSame = false
		for _, filter := range sensorList.Items[0].Spec.Dependencies[0].Filters.Data {
			if filter.Path == "headers.X-Gitlab-Event" {
				isSame = reflect.DeepEqual(filter.Value, []string{"\"Tag Push Hook\""})
			}
		}
		Expect(isSame).Should(BeTrue())
	})

	It("when nautes pipelinetrigger revision is defined, trigger will use it instead of pass through", func() {
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())
		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
		Expect(len(sensorList.Items[0].Spec.Triggers)).Should(Equal(1))
		paras := sensorList.Items[0].Spec.Triggers[0].Template.K8s.Parameters
		Expect(paras[0].Src.Value).Should(BeNil())
		Expect(paras[0].Src.DataKey).Should(Equal("body.ref"))
		dependencyName := fmt.Sprintf("%s-gitlab-%s", runtime.Name, runtime.Spec.EventSources[0].Name)
		Expect(paras[0].Src.DependencyName).Should(Equal(dependencyName))

		runtime.Spec.PipelineTriggers[0].Revision = "dev"
		_, err = syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		sensorList = &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(1))
		Expect(len(sensorList.Items[0].Spec.Triggers)).Should(Equal(1))
		paras = sensorList.Items[0].Spec.Triggers[0].Template.K8s.Parameters
		Expect(*paras[0].Src.Value).Should(Equal("dev"))
		Expect(paras[0].Src.DataKey).Should(Equal(""))

	})

	It("when runtime has two, eventsource and sensor will create two", func() {
		_, err := syncer.SyncEvents(ctx, task)
		Expect(err).Should(BeNil())

		task2 := task
		task2.Runtime = &nautescrd.ProjectPipelineRuntime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("runtime-%s", randNum()),
				Namespace: product.Name,
			},
			Spec: nautescrd.ProjectPipelineRuntimeSpec{
				Project:        "",
				PipelineSource: pipelineRepo.Name,
				Pipelines: []nautescrd.Pipeline{
					{
						Name:  pipelineRepo.Name,
						Label: "",
						Path:  "template",
					},
				},
				Destination:  runtime.Spec.Destination,
				EventSources: runtime.Spec.EventSources,
				PipelineTriggers: []nautescrd.PipelineTrigger{
					{
						EventSource: runtime.Spec.EventSources[0].Name,
						Pipeline:    pipelineRepo.Name,
						Revision:    "",
					},
				},
			},
		}

		_, err = syncer.SyncEvents(ctx, task2)
		Expect(err).Should(BeNil())

		esList := &eventsourcev1alpha1.EventSourceList{}
		err = k8sClient.List(ctx, esList)
		Expect(err).Should(BeNil())
		Expect(len(esList.Items)).Should(Equal(2))

		sensorList := &sensorv1alpha1.SensorList{}
		err = k8sClient.List(ctx, sensorList)
		Expect(err).Should(BeNil())
		Expect(len(sensorList.Items)).Should(Equal(2))

		for _, sensor := range sensorList.Items {
			name := sensor.Spec.Dependencies[0].EventSourceName
			eventName := sensor.Spec.Dependencies[0].EventName
			existed := false
			for _, es := range esList.Items {
				if es.Name == name {
					existed = true
					_, ok := es.Spec.Gitlab[eventName]
					Expect(ok).Should(BeTrue())
				}
			}
			Expect(existed).Should(BeTrue())

			for _, trigger := range sensor.Spec.Triggers {
				Expect(trigger.Template.Conditions).Should(Equal(sensor.Spec.Dependencies[0].Name))
			}
		}

		err = syncer.RemoveEvents(ctx, task2)
		Expect(err).Should(BeNil())
	})
})

var _ = Describe("Test func GetURLFromCluster", func() {
	var cluster nautescrd.Cluster
	var hostCluster *nautescrd.Cluster
	var isSecret bool
	BeforeEach(func() {
		hostCluster = &nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("host-cluster-%s", randNum()),
			},
			Spec: nautescrd.ClusterSpec{
				PrimaryDomain: "nautes.io",
			},
		}
		cluster = nautescrd.Cluster{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("cluster-%s", randNum()),
			},
			Spec: nautescrd.ClusterSpec{
				ApiServer:     "https://127.0.0.1:6443",
				ClusterType:   nautescrd.CLUSTER_TYPE_VIRTUAL,
				ClusterKind:   nautescrd.CLUSTER_KIND_KUBERNETES,
				Usage:         nautescrd.CLUSTER_USAGE_WORKER,
				HostCluster:   hostCluster.Name,
				PrimaryDomain: "cluster.io",
			},
			Status: nautescrd.ClusterStatus{
				EntryPoints: map[string]nautescrd.ClusterEntryPoint{
					"test": {
						HTTPPort:  80,
						HTTPSPort: 443,
						Type:      nautescrd.ServiceTypeNodePort,
					},
				},
			},
		}

		isSecret = true
	})

	It("cluster has primary domain, url should come from domain", func() {
		url, err := argoevents.GetURLFromCluster(cluster, hostCluster, isSecret)
		Expect(err).Should(BeNil())
		targetURL := fmt.Sprintf("https://webhook.%s.%s:443", cluster.Name, cluster.Spec.PrimaryDomain)
		Expect(url).Should(Equal(targetURL))
	})

	It("if cluster does not has primary domain and host cluster has primary domain, url should come from host cluster domain", func() {
		cluster.Spec.PrimaryDomain = ""
		url, err := argoevents.GetURLFromCluster(cluster, hostCluster, isSecret)
		Expect(err).Should(BeNil())
		targetURL := fmt.Sprintf("https://webhook.%s.%s:443", cluster.Name, hostCluster.Spec.PrimaryDomain)
		Expect(url).Should(Equal(targetURL))
	})

	It("if cluster does not has primary domain and host cluster doesn't exist, url should come from cluster api", func() {
		cluster.Spec.PrimaryDomain = ""
		cluster.Spec.HostCluster = ""
		cluster.Spec.ApiServer = "https://test.domain.io:6443"
		url, err := argoevents.GetURLFromCluster(cluster, nil, isSecret)
		Expect(err).Should(BeNil())
		targetURL := fmt.Sprintf("https://webhook.%s.test.domain.io:443", cluster.Name)
		Expect(url).Should(Equal(targetURL))
	})

	It("If the URL is obtained through the apiserver and the URL format of the apiserver is an IP address, the returned URL address is attached with the suffix .nip.io", func() {
		cluster.Spec.PrimaryDomain = ""
		cluster.Spec.HostCluster = ""
		url, err := argoevents.GetURLFromCluster(cluster, nil, isSecret)
		Expect(err).Should(BeNil())
		targetURL := fmt.Sprintf("https://webhook.%s.127.0.0.1.nip.io:443", cluster.Name)
		Expect(url).Should(Equal(targetURL))
	})

	It("if get unsafe url, protocal and port is different", func() {
		isSecret = false
		url, err := argoevents.GetURLFromCluster(cluster, hostCluster, isSecret)
		Expect(err).Should(BeNil())
		targetURL := fmt.Sprintf("http://webhook.%s.%s:80", cluster.Name, cluster.Spec.PrimaryDomain)
		Expect(url).Should(Equal(targetURL))
	})

})
