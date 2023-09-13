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

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	clusterv1 "github.com/nautes-labs/nautes/api/api-server/cluster/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	clusterManager "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	registercluster "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	clusterConfig "github.com/nautes-labs/nautes/pkg/config/cluster"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	clustrFilterFieldRules = map[string]map[string]selector.FieldSelector{
		FieldClusterType: {
			selector.EqualOperator: selector.NewStringSelector(_ClusterType, selector.Eq),
		},
		FieldUsage: {
			selector.EqualOperator: selector.NewStringSelector(_Usage, selector.Eq),
		},
		FieldWorkType: {
			selector.EqualOperator: selector.NewStringSelector(_WorkType, selector.Eq),
		},
	}
)

type ClusterService struct {
	clusterv1.UnimplementedClusterServer
	cluster                 *biz.ClusterUsecase
	configs                 *nautesconfigs.Config
	thirdPartComponentsList []*clusterManager.ThridPartComponent
	componentsType          *clusterManager.ComponentsOfClusterType
}

func NewClusterService(cluster *biz.ClusterUsecase, configs *nautesconfigs.Config) (*ClusterService, error) {
	thirdPartComponentsList, err := clusterManager.GetThirdPartComponentsList()
	if err != nil {
		return nil, err
	}

	componentsType, err := clusterManager.GetComponentsOfClusterType()
	if err != nil {
		return nil, err
	}

	return &ClusterService{
		cluster:                 cluster,
		configs:                 configs,
		thirdPartComponentsList: thirdPartComponentsList,
		componentsType:          componentsType,
	}, nil
}

func (s *ClusterService) GetCluster(ctx context.Context, req *clusterv1.GetRequest) (*clusterv1.GetReply, error) {
	cluster, err := s.cluster.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return nil, err
	}

	reply, err := s.convertClustertoReply(cluster)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (s *ClusterService) ListClusters(ctx context.Context, req *clusterv1.ListsRequest) (*clusterv1.ListsReply, error) {
	clusters, err := s.cluster.ListClusters(ctx)
	if err != nil {
		return nil, err
	}

	listReply := &clusterv1.ListsReply{}
	for _, cluster := range clusters {
		passed, err := selector.Match(req.FieldSelector, cluster, clustrFilterFieldRules)
		if err != nil {
			return nil, err
		}
		if !passed {
			continue
		}

		reply, err := s.convertClustertoReply(cluster)
		if err != nil {
			return nil, err
		}

		listReply.Items = append(listReply.Items, reply)
	}

	return listReply, nil
}

func (s *ClusterService) SaveCluster(ctx context.Context, req *clusterv1.SaveRequest) (*clusterv1.SaveReply, error) {
	err := s.validate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate save request, err: %v", err)
	}

	cluster, err := s.getCluster(req, s.configs.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to construct cluster, err: %v", err)
	}

	vcluster := s.getVcluster(cluster, req)

	kubeconfig, err := convertKubeconfig(req.Body.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig, err: %v", err)
	}

	param := &clusterManager.ClusterRegistrationParam{
		Cluster:  cluster,
		Vcluster: vcluster,
	}

	ctx = biz.SetResourceContext(ctx, "", biz.SaveMethod, "", "", nodestree.Cluster, req.ClusterName)
	if err := s.cluster.SaveCluster(ctx, param, kubeconfig); err != nil {
		return nil, err
	}

	return &clusterv1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %s cluster", req.ClusterName),
	}, nil
}

func (*ClusterService) getVcluster(cluster *resourcev1alpha1.Cluster, req *clusterv1.SaveRequest) *registercluster.Vcluster {
	var vcluster *registercluster.Vcluster

	if ok := registercluster.IsVirtualDeploymentRuntime(cluster); ok {
		vcluster = &registercluster.Vcluster{
			HttpsNodePort: req.Body.Vcluster.HttpsNodePort,
		}
	}

	return vcluster
}

func (s *ClusterService) DeleteCluster(ctx context.Context, req *clusterv1.DeleteRequest) (*clusterv1.DeleteReply, error) {
	ctx = biz.SetResourceContext(ctx, "", biz.DeleteMethod, "", "", nodestree.Cluster, req.ClusterName)

	err := s.cluster.DeleteCluster(ctx, req.ClusterName)
	if err != nil {
		return nil, err
	}
	return &clusterv1.DeleteReply{
		Msg: fmt.Sprintf("Successfully deleted %s cluster", req.ClusterName),
	}, nil
}

func convertKubeconfig(kubeconfig string) (string, error) {
	if needsBase64Decoding(kubeconfig) {
		decoded, err := base64.StdEncoding.DecodeString(kubeconfig)
		if err != nil {
			return "", fmt.Errorf("failed to decode kubeconfig: %v", err)
		}

		return string(decoded), nil
	}

	return kubeconfig, nil
}

func needsBase64Decoding(str string) bool {
	if len(str)%4 != 0 {
		return false
	}

	for _, ch := range str {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=') {
			return false
		}
	}

	return true
}

func (s *ClusterService) validate(req *clusterv1.SaveRequest) error {
	ok := utilstring.CheckURL(req.Body.GetApiServer())
	if !ok {
		return fmt.Errorf("the apiserver %s is not an https URL", req.Body.GetApiServer())
	}

	if req.Body.Usage == string(resourcev1alpha1.CLUSTER_USAGE_WORKER) &&
		req.Body.ClusterType == string(resourcev1alpha1.CLUSTER_TYPE_VIRTUAL) &&
		req.Body.HostCluster == "" {
		return fmt.Errorf("the 'Host Cluster' for virtual cluster is required")
	}

	if req.Body.Usage == string(resourcev1alpha1.CLUSTER_USAGE_WORKER) &&
		req.Body.WorkerType == "" {
		return fmt.Errorf("when the cluster usage is 'worker', the 'WorkerType' field is required")
	}

	return nil
}

func (s *ClusterService) convertClustertoReply(cluster *resourcev1alpha1.Cluster) (*clusterv1.GetReply, error) {
	reply := &clusterv1.GetReply{}
	reply.Name = cluster.Name

	reservedNamespacesAllowedProducts, err := transformMap(cluster.Spec.ReservedNamespacesAllowedProducts)
	if err != nil {
		return nil, fmt.Errorf("failed to get reserved namespaces, err: %s", err)
	}

	productAllowedClusterResources, err := transformMap(cluster.Spec.ProductAllowedClusterResources)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster resources, err: %s", err)
	}

	multiTenant, err := s.transformComponent(cluster.Spec.ComponentsList.MultiTenant)
	if err != nil {
		multiTenant = &clusterv1.Component{}
	}

	certManagement, err := s.transformComponent(cluster.Spec.ComponentsList.CertManagement)
	if err != nil {
		multiTenant = &clusterv1.Component{}
	}

	secretSync, err := s.transformComponent(cluster.Spec.ComponentsList.SecretSync)
	if err != nil {
		multiTenant = &clusterv1.Component{}
	}

	gateway, err := s.transformComponent(cluster.Spec.ComponentsList.Gateway)
	if err != nil {
		gateway = &clusterv1.Component{}
	}

	deployment, err := s.transformComponent(cluster.Spec.ComponentsList.Deployment)
	if err != nil {
		deployment = &clusterv1.Component{}
	}

	progressiveDelivery, err := s.transformComponent(cluster.Spec.ComponentsList.ProgressiveDelivery)
	if err != nil {
		progressiveDelivery = &clusterv1.Component{}
	}

	pipeline, err := s.transformComponent(cluster.Spec.ComponentsList.Pipeline)
	if err != nil {
		pipeline = &clusterv1.Component{}
	}

	eventListener, err := s.transformComponent(cluster.Spec.ComponentsList.EventListener)
	if err != nil {
		eventListener = &clusterv1.Component{}
	}

	if spec := &cluster.Spec; spec != nil {
		reply.ApiServer = cluster.Spec.ApiServer
		reply.ClusterKind = string(cluster.Spec.ClusterKind)
		reply.ClusterType = string(cluster.Spec.ClusterType)
		reply.HostCluster = cluster.Spec.HostCluster
		reply.Usage = string(cluster.Spec.Usage)
		reply.WorkerType = string(cluster.Spec.WorkerType)
		reply.PrimaryDomain = cluster.Spec.PrimaryDomain
		reply.ComponentsList = &clusterv1.ComponentsList{
			MultiTenant:         multiTenant.(*clusterv1.Component),
			CertManagement:      certManagement.(*clusterv1.Component),
			SecretSync:          secretSync.(*clusterv1.Component),
			Gateway:             gateway.(*clusterv1.Component),
			Deployment:          deployment.(*clusterv1.Component),
			ProgressiveDelivery: progressiveDelivery.(*clusterv1.Component),
			Pipeline:            pipeline.(*clusterv1.Component),
			EventListener:       eventListener.(*clusterv1.Component),
		}
		reply.ReservedNamespacesAllowedProducts = reservedNamespacesAllowedProducts
		reply.ProductAllowedClusterResources = productAllowedClusterResources
	}

	return reply, nil
}

// s.transformComponent Mutual conversion between two components, Resource Components are converted to API Components, and vice versa.
func (s *ClusterService) transformComponent(component interface{}) (interface{}, error) {
	if component == nil || isInterfaceNil(component) {
		return nil, fmt.Errorf("component is nil")
	}

	val := reflect.ValueOf(component).Elem()
	nameField := val.FieldByName("Name")
	namespaceField := val.FieldByName("Namespace")
	additionalField := val.FieldByName("Additional")

	if !nameField.IsValid() || !namespaceField.IsValid() || !additionalField.IsValid() {
		return nil, fmt.Errorf("expected fields not found in component")
	}

	switch component.(type) {
	case *resourcev1alpha1.Component:
		return &clusterv1.Component{
			Name:      nameField.String(),
			Namespace: namespaceField.String(),
			Additions: additionalField.Interface().(map[string]string),
		}, nil
	case *clusterv1.Component:
		return &resourcev1alpha1.Component{
			Name:      nameField.String(),
			Namespace: namespaceField.String(),
			Additions: additionalField.Interface().(map[string]string),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported component type: %T", component)
	}
}

func isInterfaceNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// transformMap takes an interface and transforms it to a map[string]*structpb.ListValue based on its actual type.
func transformMap(inputMap interface{}) (map[string]*structpb.ListValue, error) {
	switch actual := inputMap.(type) {
	case map[string][]string:
		return transformValues(actual, convertStringToValue)
	case map[string][]resourcev1alpha1.ClusterResourceInfo:
		return transformValues(actual, convertStructToValue)
	default:
		return nil, fmt.Errorf("unsupported input type: %T", inputMap)
	}
}

// transformValues converts a map of string and slice of a certain type to a map of string and list value.
func transformValues(inputMap interface{}, convertFunc func(interface{}) (*structpb.Value, error)) (map[string]*structpb.ListValue, error) {
	listValuesMap := make(map[string]*structpb.ListValue)

	switch actual := inputMap.(type) {
	case map[string][]string:
		for componentName, valueSlice := range actual {
			values, err := convertSliceToValues(valueSlice, convertFunc)
			if err != nil {
				return nil, err
			}
			listValuesMap[componentName] = &structpb.ListValue{Values: values}
		}
	case map[string][]resourcev1alpha1.ClusterResourceInfo:
		for resourceType, valueSlice := range actual {
			values, err := convertSliceToValues(valueSlice, convertFunc)
			if err != nil {
				return nil, err
			}
			listValuesMap[resourceType] = &structpb.ListValue{Values: values}
		}
	}

	return listValuesMap, nil
}

// convertSliceToValues converts a slice of a certain type to a slice of values using the provided convert function.
func convertSliceToValues(slice interface{}, convertFunc func(interface{}) (*structpb.Value, error)) ([]*structpb.Value, error) {
	values := make([]*structpb.Value, 0)
	switch actual := slice.(type) {
	case []string:
		for _, elem := range actual {
			value, err := convertFunc(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to convert string to value: %w", err)
			}
			values = append(values, value)
		}
	case []resourcev1alpha1.ClusterResourceInfo:
		for _, elem := range actual {
			value, err := convertFunc(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to convert struct to value: %w", err)
			}
			values = append(values, value)
		}
	}

	return values, nil
}

// convertStringToValue creates a new structpb.Value from a string.
func convertStringToValue(s interface{}) (*structpb.Value, error) {
	return structpb.NewValue(s)
}

// convertStructToValue converts a struct to a structpb.Value by marshalling and unmarshalling it.
func convertStructToValue(s interface{}) (*structpb.Value, error) {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	pbValue := &structpb.Value{}
	if err := json.Unmarshal(jsonBytes, pbValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to structpb.Value: %w", err)
	}

	return pbValue, nil
}

// getCluster creates a new Cluster resource based on the given SaveRequest and namespace.
func (s *ClusterService) getCluster(req *clusterv1.SaveRequest, namespace string) (*resourcev1alpha1.Cluster, error) {
	var err error
	var primaryDomain string
	var componentsList *resourcev1alpha1.ComponentsList
	var reservedNamespaces map[string][]string
	var clusterResources map[string][]resourcev1alpha1.ClusterResourceInfo

	primaryDomain, err = s.getPrimaryDomain(req)
	if err != nil {
		return nil, err
	}

	componentsList, err = s.constructResourceComponentsList(req)
	if err != nil {
		return nil, err
	}

	// secret management component is not open to user.
	// oauth proxy component is not open to user.
	s.setDefaultComponent(componentsList)

	reservedNamespaces, err = s.constructReservedNamespaces(req.Body.ReservedNamespacesAllowedProducts)
	if err != nil {
		return nil, err
	}

	clusterResources, err = s.constructAllowedClusterResources(req.Body.ProductAllowedClusterResources)
	if err != nil {
		return nil, err
	}

	return &resourcev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.Cluster,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.ClusterName,
			Namespace: namespace,
		},
		Spec: resourcev1alpha1.ClusterSpec{
			ApiServer:                         req.Body.ApiServer,
			ClusterType:                       resourcev1alpha1.ClusterType(req.Body.ClusterType),
			ClusterKind:                       resourcev1alpha1.ClusterKind(req.Body.ClusterKind),
			Usage:                             resourcev1alpha1.ClusterUsage(req.Body.Usage),
			HostCluster:                       req.Body.HostCluster,
			PrimaryDomain:                     primaryDomain,
			WorkerType:                        resourcev1alpha1.ClusterWorkType(req.Body.WorkerType),
			ComponentsList:                    *componentsList,
			ReservedNamespacesAllowedProducts: reservedNamespaces,
			ProductAllowedClusterResources:    clusterResources,
		},
	}, nil
}

func (s *ClusterService) setDefaultComponent(list *resourcev1alpha1.ComponentsList) {
	secretManagementComponent := s.getComponent("secretManagement", "vault")
	list.SecretManagement = &resourcev1alpha1.Component{
		Name:      secretManagementComponent.Name,
		Namespace: secretManagementComponent.Namespace,
		Additions: secretManagementComponent.Additions,
	}

	oauthproxyComponent := s.getComponent("oauthproxy", "oauth2-proxy")
	list.OauthProxy = &resourcev1alpha1.Component{
		Name:      oauthproxyComponent.Name,
		Namespace: oauthproxyComponent.Namespace,
		Additions: oauthproxyComponent.Additions,
	}
}

func (*ClusterService) getPrimaryDomain(req *clusterv1.SaveRequest) (string, error) {
	if req.Body.PrimaryDomain != "" {
		return req.Body.PrimaryDomain, nil
	}

	domain := utilstring.GetDomain(req.Body.ApiServer)

	isIp, err := utilstring.IsIp(req.Body.ApiServer)
	if err != nil {
		return "", err
	}

	if isIp {
		return fmt.Sprintf("%s.nip.io", domain), nil
	}

	return domain, nil
}

func (s *ClusterService) constructResourceComponentsList(req *clusterv1.SaveRequest) (*resourcev1alpha1.ComponentsList, error) {
	body := req.GetBody()
	config, err := clusterConfig.NewClusterComponentConfig()
	if err != nil {
		return nil, err
	}

	componetsDefinition, err := config.GetClusterComponentsDefinition(&clusterConfig.ClusterInfo{
		Name:        req.GetClusterName(),
		ClusterType: body.GetClusterType(),
		WorkType:    body.GetWorkerType(),
		Usage:       body.GetUsage(),
	})
	if err != nil {
		return nil, err
	}

	componentsList := body.GetComponentsList()
	err = s.setDefaultValuesIfEmpty(componentsList, componetsDefinition)
	if err != nil {
		return nil, err
	}

	return s.convertComponentsList(componentsList), nil
}

func (s *ClusterService) setDefaultValuesIfEmpty(componentsList *clusterv1.ComponentsList, componetsDefinition []string) error {
	val := reflect.Indirect(reflect.ValueOf(componentsList))
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if field.Type().Kind() != reflect.Ptr {
			continue
		}

		componentType := val.Type().Field(i).Name
		isContain := utilstring.ContainsString(componetsDefinition, componentType)
		if !isContain {
			continue
		}

		// Set default component if component is empty.
		// If the component field is missing a value, set the default value.
		if field.IsNil() {
			defaultComponent, err := s.getDefaultComponent(componentType)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(defaultComponent))
		} else {
			err := s.fillDefaultFields(field, componentType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ClusterService) getDefaultComponent(componentType string) (*clusterv1.Component, error) {
	defaultComponent := s.getComponent(componentType, "")
	if defaultComponent == nil {
		return nil, fmt.Errorf("failed to get default component for the component type '%s'", componentType)
	}
	return defaultComponent, nil
}

func (s *ClusterService) fillDefaultFields(field reflect.Value, componentType string) error {
	component := field.Elem()
	name := component.FieldByName("Name")
	namespace := component.FieldByName("Namespace")

	if name.String() == "" {
		defaultComponent, err := s.getDefaultComponent(componentType)
		if err != nil {
			return fmt.Errorf("failed to get default component for the component type '%s'", componentType)
		}
		name.SetString(defaultComponent.Name)
	}

	if namespace.String() == "" {
		component := s.getComponent(componentType, name.String())
		if component == nil {
			return fmt.Errorf("failed to get %s component for the component type '%s'", name.String(), componentType)
		}
		namespace.SetString(component.Namespace)
	}

	return nil
}

func (s *ClusterService) getComponent(componentType, componentName string) *clusterv1.Component {
	for _, component := range s.thirdPartComponentsList {
		if !strings.EqualFold(component.Type, componentType) {
			continue
		}

		if componentName != "" && component.Name == componentName {
			return &clusterv1.Component{
				Name:      component.Name,
				Namespace: component.Namespace,
			}
		} else if componentName == "" && component.Default {
			return &clusterv1.Component{
				Name:      component.Name,
				Namespace: component.Namespace,
			}
		}
	}

	return nil
}

func (s *ClusterService) constructReservedNamespaces(listValue map[string]*structpb.ListValue) (map[string][]string, error) {
	if listValue == nil {
		return nil, nil
	}

	reservedNamespaces := make(map[string][]string)
	for key, value := range listValue {
		productNames := make([]string, 0)

		for _, val := range value.Values {
			val, ok := val.Kind.(*structpb.Value_StringValue)
			if !ok {
				return nil, fmt.Errorf("the value of the reserved namespace must be a string")
			}

			productNames = append(productNames, val.StringValue)
		}

		reservedNamespaces[key] = productNames
	}

	return reservedNamespaces, nil
}

func (s *ClusterService) constructAllowedClusterResources(listValue map[string]*structpb.ListValue) (map[string][]resourcev1alpha1.ClusterResourceInfo, error) {
	clusterResourceInfoMap := make(map[string][]resourcev1alpha1.ClusterResourceInfo)
	if listValue == nil {
		return nil, nil
	}

	for productName, value := range listValue {
		clusterResourcesInfo := make([]resourcev1alpha1.ClusterResourceInfo, 0)

		for _, val := range value.Values {
			val, ok := val.Kind.(*structpb.Value_StructValue)
			if !ok {
				return nil, fmt.Errorf("the cluster information of %s must be an struct 'ClusterResourceInfo' type", productName)
			}

			bytes, err := val.StructValue.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("failed to parsing cluster resource information of %s to json data, err: %s", productName, err)
			}

			resource := resourcev1alpha1.ClusterResourceInfo{}
			err = json.Unmarshal(bytes, &resource)
			if err != nil {
				return nil, fmt.Errorf("failed to parsing cluster resource information of %s to struct: 'ClusterResourceInfo', err: %s", productName, err)
			}

			if resource.Group == "" ||
				resource.Kind == "" {
				return nil, fmt.Errorf("failed to get cluster resource information of %s, the group and kind of the cluster resource cannot be empty", productName)
			}

			clusterResourcesInfo = append(clusterResourcesInfo, resource)
		}

		clusterResourceInfoMap[productName] = clusterResourcesInfo
	}

	return clusterResourceInfoMap, nil
}

func (s *ClusterService) convertComponentsList(components *clusterv1.ComponentsList) *resourcev1alpha1.ComponentsList {
	multiTenant, err := s.transformComponent(components.MultiTenant)
	if err != nil {
		multiTenant = &resourcev1alpha1.Component{}
	}

	certManagement, err := s.transformComponent(components.CertManagement)
	if err != nil {
		certManagement = &resourcev1alpha1.Component{}
	}

	secretSync, err := s.transformComponent(components.SecretSync)
	if err != nil {
		secretSync = &resourcev1alpha1.Component{}
	}

	gateway, err := s.transformComponent(components.Gateway)
	if err != nil {
		gateway = &resourcev1alpha1.Component{}
	}

	deployment, err := s.transformComponent(components.Deployment)
	if err != nil {
		deployment = &resourcev1alpha1.Component{}
	}

	progressiveDelivery, err := s.transformComponent(components.ProgressiveDelivery)
	if err != nil {
		progressiveDelivery = &resourcev1alpha1.Component{}
	}

	pipeline, err := s.transformComponent(components.Pipeline)
	if err != nil {
		pipeline = &resourcev1alpha1.Component{}
	}

	eventListener, err := s.transformComponent(components.EventListener)
	if err != nil {
		eventListener = &resourcev1alpha1.Component{}
	}

	list := &resourcev1alpha1.ComponentsList{
		MultiTenant:         multiTenant.(*resourcev1alpha1.Component),
		CertManagement:      certManagement.(*resourcev1alpha1.Component),
		SecretSync:          secretSync.(*resourcev1alpha1.Component),
		Gateway:             gateway.(*resourcev1alpha1.Component),
		Deployment:          deployment.(*resourcev1alpha1.Component),
		ProgressiveDelivery: progressiveDelivery.(*resourcev1alpha1.Component),
		Pipeline:            pipeline.(*resourcev1alpha1.Component),
		EventListener:       eventListener.(*resourcev1alpha1.Component),
	}

	return list
}
