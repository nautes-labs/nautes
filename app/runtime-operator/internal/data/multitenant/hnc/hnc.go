package hnc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	hncv1alpha2 "sigs.k8s.io/hierarchical-namespaces/api/v1alpha2"
)

func init() {
	utilruntime.Must(hncv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

type hnc struct {
	db                database.Database
	deployer          syncer.Deployment
	k8sClient         client.Client
	clusterWorkerType v1alpha1.ClusterWorkType
	opts              map[string]string
	secretStoreURL    string
	namespace         string
}

func NewHNC(opt v1alpha1.Component, info *syncer.ComponentInitInfo) (syncer.MultiTenant, error) {
	if info.ClusterConnectInfo.Type != v1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.Type)
	}

	k8sClient, err := client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	cluster, err := info.NautesDB.GetCluster(info.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("get cluster failed: %w", err)
	}

	return hnc{
		db:                info.NautesDB,
		deployer:          info.Components.Deployment,
		k8sClient:         k8sClient,
		clusterWorkerType: cluster.Spec.WorkerType,
		opts:              opt.Additions,
		secretStoreURL:    info.NautesConfig.Secret.Vault.Addr,
		namespace:         opt.Namespace,
	}, nil
}

func (h hnc) CleanUp() error {
	return nil
}

var (
	roleBindingTemplate = rbacv1.RoleBinding{
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
)

const (
	OptKeyProductResourcePathPipeline   = "ProductResourcePathPipeline"
	OptKeyProductResourcePathDeployment = "ProductResourcePathDeployment"
	OptKeyProductResourceRevision       = "ProductResourceRevision"
	// OptKeySyncResourceTypes store witch kind of resource will sync between parent and sub namespace
	// format [group/name, group/name]
	OptKeySyncResourceTypes = "SyncResourceTypes"
)

func (h hnc) CreateProduct(ctx context.Context, name string) error {
	if err := h.createNamespace(ctx, name, name); err != nil {
		return err
	}

	app, err := h.getProductApp(name)
	if err != nil {
		return fmt.Errorf("get product app failed: %w", err)
	}

	if app != nil {
		if err := h.deployer.CreateApp(ctx, *app); err != nil {
			return fmt.Errorf("deploy product resources failed: %w", err)
		}
	} else {
		if err := h.deployer.DeleteApp(ctx, h.getEmptyProductApp(name)); err != nil {
			return fmt.Errorf("delete product resources failed: %w", err)
		}
	}

	if err := h.setHNCConfig(ctx); err != nil {
		return err
	}
	return h.addRoleBinding(ctx, name)
}

const (
	nameFormatProductResourceApp = "%s-share"
)

func (h hnc) getEmptyProductApp(productName string) syncer.Application {
	return syncer.Application{
		Resource: syncer.Resource{
			Product: productName,
			Name:    fmt.Sprintf(nameFormatProductResourceApp, productName),
		},
	}
}

func (h hnc) getProductApp(productName string) (*syncer.Application, error) {
	productResourcePath := h.getProductResourcePath()
	revision := h.opts[OptKeyProductResourceRevision]
	if productResourcePath == "" || revision == "" {
		return nil, nil
	}

	coderepo, err := h.db.GetProductCodeRepo(productName)
	if err != nil {
		return nil, err
	}

	app := &syncer.Application{
		Resource: syncer.Resource{
			Product: productName,
			Name:    fmt.Sprintf(nameFormatProductResourceApp, productName),
		},
		Git: &syncer.ApplicationGit{
			URL:      coderepo.Spec.URL,
			Revision: revision,
			Path:     productResourcePath,
			CodeRepo: coderepo.Name,
		},
		Destinations: []syncer.Space{
			{
				Resource: syncer.Resource{
					Product: productName,
					Name:    productName,
				},
				SpaceType: syncer.SpaceTypeKubernetes,
				Kubernetes: syncer.SpaceKubernetes{
					Namespace: productName,
				},
			},
		},
	}

	return app, nil
}

func (h hnc) getProductResourcePath() string {
	switch h.clusterWorkerType {
	case v1alpha1.ClusterWorkTypeDeployment:
		return h.opts[OptKeyProductResourcePathDeployment]
	case v1alpha1.ClusterWorkTypePipeline:
		return h.opts[OptKeyProductResourcePathPipeline]
	default:
		return ""
	}
}

func (h hnc) addRoleBinding(ctx context.Context, name string) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
			Labels:    utils.GetProductLabel(name),
		},
	}

	prodcut, err := h.db.GetProduct(name)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, h.k8sClient, roleBinding, func() error {
		err := utils.CheckResourceOperability(roleBinding, name)
		if err != nil {
			return err
		}

		roleBinding.Subjects = roleBindingTemplate.Subjects
		roleBinding.RoleRef = roleBindingTemplate.RoleRef
		roleBinding.Subjects[0].Name = prodcut.Spec.Name
		return nil
	})
	if err != nil {
		return fmt.Errorf("create or update rolebinding failed: %w", err)
	}

	return nil
}

func (h hnc) DeleteProduct(ctx context.Context, name string) error {
	ok, err := h.isRemovableProduct(ctx, name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("space is found, delete failed")
	}

	if err := h.deployer.DeleteApp(ctx, h.getEmptyProductApp(name)); err != nil {
		return fmt.Errorf("clean product app failed: %w", err)
	}

	return h.deleteNamespace(ctx, name, name)
}

func (h hnc) isRemovableProduct(ctx context.Context, name string) (bool, error) {
	namespaces, err := h.listNamespaces(ctx, name, syncer.IgnoreResourceInDeletion())
	if err != nil {
		return false, err
	}

	num := len(namespaces)
	if num >= 2 {
		return false, nil
	} else if num == 0 {
		return true, nil
	}

	if namespaces[0].Name != name {
		return false, nil
	}

	return true, nil
}

func (h hnc) GetProduct(ctx context.Context, name string) (*syncer.ProductStatus, error) {
	namespace, err := h.getNamespace(ctx, name, name)
	if err != nil {
		return nil, err
	}

	userList := newUserList(namespace.Annotations[keyProductUserList])

	status := &syncer.ProductStatus{
		Name:     name,
		Projects: []string{},
		Users:    userList.getUsers(),
	}

	return status, nil
}

func (h hnc) CreateSpace(ctx context.Context, productName string, name string) error {
	if err := h.createNamespace(ctx, productName, name); err != nil {
		return err
	}

	return h.addSpaceParent(ctx, productName, name)
}

func (h hnc) DeleteSpace(ctx context.Context, productName string, name string) error {
	return h.deleteNamespace(ctx, productName, name)
}

func (h hnc) GetSpace(ctx context.Context, productName string, name string) (*syncer.SpaceStatus, error) {
	var spaceStatus syncer.SpaceStatus
	namespace, err := h.getNamespace(ctx, productName, name)
	if err != nil {
		return nil, err
	}

	spaceStatus = getSpace(*namespace)
	return &spaceStatus, nil
}

func (h hnc) ListSpaces(ctx context.Context, productName string, opts ...syncer.ListOption) ([]syncer.SpaceStatus, error) {
	namespaces, err := h.listNamespaces(ctx, productName, opts...)
	if err != nil {
		return nil, err
	}

	spacesStatus := []syncer.SpaceStatus{}

	for _, ns := range namespaces {
		spacesStatus = append(spacesStatus, getSpace(ns))
	}

	return spacesStatus, nil
}

func (h hnc) AddSpaceUser(ctx context.Context, request syncer.PermissionRequest) error {
	switch request.RequestScope {
	case syncer.RequestScopeUser:
		if err := h.addRoleBindingServiceAccount(ctx, request.User, request.Resource.Name); err != nil {
			return fmt.Errorf("grant user %s admin permission in space %s failed: %w", request.User, request.Resource.Name, err)
		}
		return h.addSpaceUsers(ctx, request.Resource.Product, request.Resource.Name, []string{request.User})
	default:
		return fmt.Errorf("unsupported request scope %s", request.RequestScope)
	}
}

func (h hnc) DeleteSpaceUser(ctx context.Context, request syncer.PermissionRequest) error {
	switch request.RequestScope {
	case syncer.RequestScopeUser:
		if err := h.deleteRoleBindingServiceAccount(ctx, request.User, request.Resource.Name); err != nil {
			return fmt.Errorf("revoke user %s admin permission in space %s failed: %w", request.User, request.Resource.Name, err)
		}
		return h.deleteSpaceUsers(ctx, request.Resource.Product, request.Resource.Name, []string{request.User})
	default:
		return fmt.Errorf("unsupported request scope %s", request.RequestScope)
	}
}

func (h hnc) addRoleBindingServiceAccount(ctx context.Context, name, namespace string) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, roleBinding, func() error {
		roleBinding.RoleRef = roleBindingTemplate.RoleRef
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create or update rolebinding failed: %w", err)
	}

	return nil
}

func (h hnc) deleteRoleBindingServiceAccount(ctx context.Context, name, namespace string) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return client.IgnoreNotFound(h.k8sClient.Delete(ctx, roleBinding))
}

const (
	keyProductUserList = "ProductUsers"
	keySpaceUserList   = "SpaceUsers"
)

func (h hnc) addSpaceUsers(ctx context.Context, productName string, spaceName string, users []string) error {
	namespace, err := h.getNamespace(ctx, productName, spaceName)
	if err != nil {
		return err
	}

	if namespace.Annotations == nil {
		namespace.Annotations = map[string]string{}
	}

	userList := newUserList(namespace.Annotations[keySpaceUserList])
	for _, userName := range users {
		userList.addUser(userName)
		namespace.Annotations[keySpaceUserList] = userList.getUsersAsString()
		if err := h.k8sClient.Update(ctx, namespace); err != nil {
			return fmt.Errorf("add user to cache failed: %w", err)
		}

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userName,
				Namespace: spaceName,
			},
		}
		if err := h.k8sClient.Create(ctx, sa); client.IgnoreAlreadyExists(err) != nil {
			return fmt.Errorf("create user service account failed: %w", err)
		}
	}

	return nil
}

func (h hnc) deleteSpaceUsers(ctx context.Context, productName string, spaceName string, users []string) error {
	namespace, err := h.getNamespace(ctx, productName, spaceName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	userList := newUserList(namespace.Annotations[keySpaceUserList])
	for _, userName := range users {
		userList.deleteUser(userName)

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userName,
				Namespace: spaceName,
			},
		}

		if err := h.k8sClient.Delete(ctx, sa); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	namespace.Annotations[keySpaceUserList] = userList.getUsersAsString()

	return h.k8sClient.Update(ctx, namespace)
}

func (h hnc) CreateUser(ctx context.Context, productName string, name string) error {
	namespace, err := h.getNamespace(ctx, productName, productName)
	if err != nil {
		return err
	}

	userString, ok := namespace.Annotations[keyProductUserList]
	if !ok {
		namespace.Annotations = map[string]string{}
	}
	userList := newUserList(userString)
	userList.addUser(name)
	namespace.Annotations[keyProductUserList] = userList.getUsersAsString()

	if err := h.k8sClient.Update(ctx, namespace); err != nil {
		return fmt.Errorf("update user list in product namespace failed: %w", err)
	}
	return nil
}

func (h hnc) DeleteUser(ctx context.Context, productName string, name string) error {
	user, err := h.GetUser(ctx, productName, name)
	if err != nil {
		return fmt.Errorf("get user %s info failed: %w", name, err)
	}

	for _, authInfo := range user.AuthInfo.Kubernetes {
		if err := h.deleteSpaceUsers(ctx, productName, authInfo.Namespace, []string{name}); err != nil {
			return fmt.Errorf("remove user %s from space %s failed: %w", name, authInfo.Namespace, err)
		}
	}

	namespace, err := h.getNamespace(ctx, productName, productName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	userString, ok := namespace.Annotations[keyProductUserList]
	if !ok {
		namespace.Annotations = map[string]string{}
	}
	userList := newUserList(userString)
	userList.deleteUser(name)
	namespace.Annotations[keyProductUserList] = userList.getUsersAsString()

	if err := h.k8sClient.Update(ctx, namespace); err != nil {
		return fmt.Errorf("update user list in product namespace failed: %w", err)
	}

	return nil
}

func (h hnc) GetUser(ctx context.Context, productName string, name string) (user *syncer.User, err error) {
	productNamespace, err := h.getNamespace(ctx, productName, productName)
	if err != nil {
		return nil, syncer.UserNotFound(err, name)
	}
	userList := newUserList(productNamespace.Annotations[keyProductUserList])
	if !userList.hasUser(name) {
		return nil, syncer.UserNotFound(errors.New(""), name)
	}

	namespaces, err := h.listNamespaces(ctx, productName, syncer.ByUser(name), syncer.IgnoreResourceInDeletion())
	if err != nil {
		return nil, err
	}

	user = &syncer.User{
		Resource: syncer.Resource{
			Product: productName,
			Name:    name,
		},
		Role:     []string{},
		UserType: syncer.UserTypeMachine,
		AuthInfo: &syncer.Auth{},
	}

	for _, ns := range namespaces {
		user.AuthInfo.Kubernetes = append(user.AuthInfo.Kubernetes, syncer.AuthKubernetes{
			ServiceAccount: name,
			Namespace:      ns.Name,
		})
	}

	return user, nil
}

const (
	hierarchyConfigurationName = "hierarchy"
)

func (h hnc) addSpaceParent(ctx context.Context, productName, name string) error {
	hncConfig := &hncv1alpha2.HierarchyConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hierarchyConfigurationName,
			Namespace: name,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, hncConfig, func() error {
		hncConfig.Spec.Parent = productName
		return nil
	})
	if err != nil {
		return fmt.Errorf("set %s's parent failed: %w", name, err)
	}

	return nil
}

const (
	hncConfigName = "config"
)

func (h hnc) setHNCConfig(ctx context.Context) error {
	resources := h.opts[OptKeySyncResourceTypes]
	if resources == "" {
		return nil
	}

	var syncResources []hncv1alpha2.ResourceSpec
	for _, res := range strings.Split(resources, ",") {
		elements := strings.Split(strings.Replace(res, " ", "", -1), "/")
		if len(elements) != 2 {
			return fmt.Errorf("decode resource type failed")
		}

		resourceDefination := hncv1alpha2.ResourceSpec{
			Group:    elements[0],
			Resource: elements[1],
			Mode:     "",
		}

		syncResources = append(syncResources, resourceDefination)
	}

	hncConfig := &hncv1alpha2.HNCConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: hncConfigName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, hncConfig, func() error {
		hncConfig.Spec.Resources = syncResources
		return nil
	})
	return err
}

func (h hnc) createNamespace(ctx context.Context, productName, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: utils.GetProductLabel(productName),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, h.k8sClient, namespace, func() error {
		return utils.CheckResourceOperability(namespace, productName)
	})
	if err != nil {
		return fmt.Errorf("create or update namespace failed: %w", err)
	}
	if err := h.createSecretStoreCA(ctx, namespace.Name); err != nil {
		return fmt.Errorf("create secret store ca in namespace failed: %w", err)
	}
	return nil
}

func (h hnc) deleteNamespace(ctx context.Context, productName string, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := h.k8sClient.Get(ctx, client.ObjectKeyFromObject(namespace), namespace); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !utils.IsBelongsToProduct(namespace, productName) {
		return fmt.Errorf("space %s is not belongs to product %s", name, productName)
	}

	return h.k8sClient.Delete(ctx, namespace)
}

func (h hnc) getNamespace(ctx context.Context, productName, name string) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if err := h.k8sClient.Get(ctx, client.ObjectKeyFromObject(namespace), namespace); err != nil {
		return nil, err
	}

	if err := utils.CheckResourceOperability(namespace, productName); err != nil {
		return nil, err
	}

	return namespace, nil
}

func (h hnc) listNamespaces(ctx context.Context, productName string, opts ...syncer.ListOption) ([]corev1.Namespace, error) {
	listOpts := &syncer.ListOptions{}
	for _, fn := range opts {
		fn(listOpts)
	}

	namespaceList := &corev1.NamespaceList{}
	nsListOpt := client.MatchingLabels(utils.GetProductLabel(productName))
	if err := h.k8sClient.List(ctx, namespaceList, nsListOpt); err != nil {
		return nil, fmt.Errorf("list namespace failed: %w", err)
	}

	namespaces := []corev1.Namespace{}

	for _, ns := range namespaceList.Items {
		if listOpts.User != "" {
			userList := newUserList(ns.Annotations[keySpaceUserList])
			if !userList.hasUser(listOpts.User) {
				continue
			}
		}

		if listOpts.IgnoreDataInDeletion {
			if ns.DeletionTimestamp != nil {
				continue
			}
		}

		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}

func getSpace(namespace corev1.Namespace) syncer.SpaceStatus {
	userList := newUserList(namespace.Annotations[keySpaceUserList])
	spaceStatus := syncer.SpaceStatus{
		Space: syncer.Space{
			Resource: syncer.Resource{
				Product: namespace.Labels[v1alpha1.LABEL_BELONG_TO_PRODUCT],
				Name:    namespace.Name,
			},
			SpaceType: syncer.SpaceTypeKubernetes,
			Kubernetes: syncer.SpaceKubernetes{
				Namespace: namespace.Name,
			},
		},
		Users: userList.getUsers(),
	}
	return spaceStatus
}

const (
	secretNameSecretStoreCA = "ca"
	fileNameCA              = "ca.crt"
)

func (h hnc) createSecretStoreCA(ctx context.Context, namespace string) error {
	if h.secretStoreURL == "" {
		return nil
	}
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNameSecretStoreCA,
			Namespace: namespace,
		},
		Data: map[string][]byte{},
	}
	caBundle, err := utils.GetCABundle(h.secretStoreURL)
	if err != nil {
		return err
	}
	_, err = controllerutil.CreateOrUpdate(ctx, h.k8sClient, caSecret, func() error {
		caSecret.Data[fileNameCA] = caBundle
		return nil
	})
	return err
}
