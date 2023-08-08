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

package argocd

import (
	"context"
	"fmt"
	"reflect"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	scheme         = runtime.NewScheme()
	baseAppProject = argocrd.AppProject{
		Spec: argocrd.AppProjectSpec{
			SourceRepos:  []string{},
			Destinations: []argocrd.ApplicationDestination{},
			ClusterResourceWhitelist: []metav1.GroupKind{
				{
					Group: "*",
					Kind:  "*",
				},
			},
		},
	}
	baseApp = argocrd.Application{
		Spec: argocrd.ApplicationSpec{
			Destination: argocrd.ApplicationDestination{
				Server: kubernetesDefaultService,
			},
			SyncPolicy: &argocrd.SyncPolicy{
				Automated: &argocrd.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
		},
	}
)

func init() {
	utilruntime.Must(argocrd.AddToScheme(scheme))
	utilruntime.Must(nautescrd.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

type destCluster struct {
	name        string
	productName string
	client.Client
}

func (d *destCluster) GetClient() client.Client {
	return d.Client
}

func findCodeRepoInProject(project argocrd.AppProject, repoName string) (int, bool) {
	repos := project.Spec.SourceRepos
	isExist := false
	index := -1
	for i, src := range repos {
		if src == repoName {
			isExist = true
			index = i
			break
		}
	}
	return index, isExist
}

func findDestinationInProject(project argocrd.AppProject, namespaceName string) (int, bool) {
	dests := project.Spec.Destinations
	isExist := false
	index := -1
	for i, dest := range dests {
		if dest.Namespace == namespaceName && dest.Server == "*" {
			isExist = true
			index = i
			break
		}
	}
	return index, isExist
}

type appProject struct {
	resource    *argocrd.AppProject
	productName string
	client.Client
}

func NewAppProject(name, namespace, productName string, k8sClient client.Client) (*appProject, error) {
	project := &argocrd.AppProject{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, project)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		project = &argocrd.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: productName},
			},
			Spec: argocrd.AppProjectSpec{},
		}
	}

	if reason, ok := isNotTerminatingAndBelongsToProduct(project, productName); !ok {
		return nil, fmt.Errorf(reason)
	}

	return &appProject{
		resource:    project,
		productName: productName,
		Client:      k8sClient,
	}, nil
}

func (p *appProject) GetName() string {
	return p.resource.Name
}

func (p *appProject) GetNamespace() string {
	return p.resource.Namespace
}

func (p *appProject) SyncAppPermission(ctx context.Context, newURL, oldURL, newNamespace, oldNamespace string) error {
	logger := log.FromContext(ctx)
	p.addSource(newURL)
	p.addNamespace(newNamespace)

	if newURL != oldURL {
		logger.V(1).Info("coderepo url is change", "NewURL", newURL, "OldURL", oldURL)
	}
	deletable, err := p.urlIsDeletable(ctx, oldURL)
	if err != nil {
		return fmt.Errorf("check url is deletable failed: %w", err)
	}
	if deletable {
		logger.V(1).Info("old url with remove from app project", "URL", oldURL)
		p.deleteSource(oldURL)
	}

	p.deleteNamespace(oldNamespace)
	return p.syncToKubernetes(ctx)
}

func (p *appProject) SyncAppProject(ctx context.Context, spec argocrd.AppProjectSpec) error {
	p.resource.Spec = spec
	return p.syncToKubernetes(ctx)
}

func (p *appProject) syncToKubernetes(ctx context.Context) error {
	if p.resource.CreationTimestamp.IsZero() {
		return p.Create(ctx, p.resource)
	}
	return p.Update(ctx, p.resource)
}

func (p *appProject) destroy(ctx context.Context) error {
	if p.resource.CreationTimestamp.IsZero() {
		return nil
	}

	return p.Delete(ctx, p.resource)
}

func (p *appProject) addSource(url string) {
	_, codeRepoIsExist := findCodeRepoInProject(*p.resource, url)
	if !codeRepoIsExist && url != "" {
		p.resource.Spec.SourceRepos = append(p.resource.Spec.SourceRepos, url)
	}
}

func (p *appProject) deleteSource(url string) {
	i, isExist := findCodeRepoInProject(*p.resource, url)
	if isExist {
		p.resource.Spec.SourceRepos = append(p.resource.Spec.SourceRepos[:i], p.resource.Spec.SourceRepos[i+1:]...)
	}
}

func (p *appProject) addNamespace(namespace string) {
	_, destinationIsExist := findDestinationInProject(*p.resource, namespace)
	if !destinationIsExist && namespace != "" {
		dest := argocrd.ApplicationDestination{
			Server:    "*",
			Namespace: namespace,
		}
		p.resource.Spec.Destinations = append(p.resource.Spec.Destinations, dest)
	}
}

func (p *appProject) deleteNamespace(namespace string) {
	i, isExist := findDestinationInProject(*p.resource, namespace)
	if isExist {
		p.resource.Spec.Destinations = append(p.resource.Spec.Destinations[:i], p.resource.Spec.Destinations[i+1:]...)
	}
}

func (p *appProject) isDeletable() bool {
	return len(p.resource.Spec.SourceRepos) == 0
}

func (p *appProject) urlIsDeletable(ctx context.Context, repoURL string) (bool, error) {
	apps := &argocrd.ApplicationList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: p.productName}),
		client.InNamespace(p.resource.Namespace),
	}
	err := p.List(ctx, apps, listOpts...)
	if err != nil {
		return false, err
	}

	isDeleteable := true
	for _, app := range apps.Items {
		if app.Spec.Source.RepoURL == repoURL && app.DeletionTimestamp.IsZero() {
			isDeleteable = false
			break
		}
	}

	return isDeleteable, nil
}

type app struct {
	resource    *argocrd.Application
	productName string
	k8sClient   client.Client
}

func newApp(name, namespace, productName string, k8sClient client.Client) (*app, error) {
	appInK8s := &argocrd.Application{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, appInK8s)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		appInK8s = &argocrd.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: productName}},
			Spec: argocrd.ApplicationSpec{},
		}
	}

	if reason, ok := isNotTerminatingAndBelongsToProduct(appInK8s, productName); !ok {
		return nil, fmt.Errorf(reason)
	}

	return &app{
		resource:    appInK8s,
		productName: productName,
		k8sClient:   k8sClient,
	}, nil
}

type updateAppOption func(*argocrd.ApplicationSpec)

func (a *app) SyncApp(ctx context.Context, opts ...updateAppOption) error {
	if err := a.UpdateApp(ctx, opts...); err != nil {
		return fmt.Errorf("update app failed: %w", err)
	}

	return a.syncToKubernetes(ctx)
}

func (a *app) syncToKubernetes(ctx context.Context) error {
	if a.resource.CreationTimestamp.IsZero() {
		return a.k8sClient.Create(ctx, a.resource)
	}
	return a.k8sClient.Update(ctx, a.resource)
}

func (a *app) UpdateApp(ctx context.Context, opts ...updateAppOption) error {
	spec := baseApp.Spec.DeepCopy()
	for _, fn := range opts {
		fn(spec)
	}

	if !a.isSame(*spec) {
		a.resource.Spec = *spec
	}

	return nil
}

func (a *app) GetName() string {
	return a.resource.Name
}

func (a *app) Destroy(ctx context.Context) error {
	if a.resource.CreationTimestamp.IsZero() {
		return nil
	}

	return a.k8sClient.Delete(ctx, a.resource)
}

func (a *app) isSame(spec argocrd.ApplicationSpec) bool {
	if reflect.DeepEqual(a.resource.Spec.Source, spec.Source) &&
		reflect.DeepEqual(a.resource.Spec.Destination, spec.Destination) &&
		a.resource.Spec.Project == spec.Project {
		return true
	}

	return false
}
