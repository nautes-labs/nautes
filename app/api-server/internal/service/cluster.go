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
	"os"
	"reflect"
	"strings"

	clusterv1 "github.com/nautes-labs/nautes/api/api-server/cluster/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	ClusterRegistration "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	registercluster "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"
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
	thirdPartComponentsList []*Component
}

func NewClusterService(cluster *biz.ClusterUsecase, configs *nautesconfigs.Config) (*ClusterService, error) {
	thirdPartComponentsList, err := loadThirdPartComponentsList(thirdPartComponentsPath)
	if err != nil {
		return nil, err
	}

	return &ClusterService{cluster: cluster, configs: configs, thirdPartComponentsList: thirdPartComponentsList}, nil
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
	err := s.Validate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate save request, err: %v", err)
	}

	ctx = biz.SetResourceContext(ctx, "", biz.SaveMethod, "", "", nodestree.Cluster, req.ClusterName)

	cluster, err := s.constructResourceCluster(req, s.configs.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to construct cluster, err: %v", err)
	}

	var vcluster *registercluster.Vcluster
	if ok := registercluster.IsVirtualDeploymentRuntime(cluster); ok {
		if req.Body.Vcluster != nil {
			vcluster = &registercluster.Vcluster{
				HttpsNodePort: req.Body.Vcluster.HttpsNodePort,
			}
		}
	}

	kubeconfig, err := convertKubeconfig(req.Body.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig, err: %v", err)
	}

	traefik := &ClusterRegistration.Traefik{}
	if req.Body.ClusterType == string(resourcev1alpha1.CLUSTER_TYPE_PHYSICAL) {
		traefik.HttpNodePort = req.Body.Traefik.HttpNodePort
		traefik.HttpsNodePort = req.Body.Traefik.HttpsNodePort
	}
	param := &ClusterRegistration.ClusterRegistrationParam{
		Cluster:    cluster,
		ArgocdHost: req.Body.ArgocdHost,
		TektonHost: req.Body.TektonHost,
		Traefik:    traefik,
		Vcluster:   vcluster,
	}

	if err := s.cluster.SaveCluster(ctx, param, kubeconfig); err != nil {
		return nil, err
	}

	return &clusterv1.SaveReply{
		Msg: fmt.Sprintf("Successfully saved %s cluster", req.ClusterName),
	}, nil
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

func (s *ClusterService) Validate(req *clusterv1.SaveRequest) error {
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

	if req.Body.ClusterType == string(resourcev1alpha1.CLUSTER_TYPE_PHYSICAL) &&
		req.Body.Traefik == nil {
		return fmt.Errorf("traefik parameter is required when cluster type is 'Host Cluster' or 'Physical Runtime'")
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

	if spec := &cluster.Spec; spec != nil {
		reply.ApiServer = cluster.Spec.ApiServer
		reply.ClusterKind = string(cluster.Spec.ClusterKind)
		reply.ClusterType = string(cluster.Spec.ClusterType)
		reply.HostCluster = cluster.Spec.HostCluster
		reply.Usage = string(cluster.Spec.Usage)
		reply.WorkerType = string(cluster.Spec.WorkerType)
		reply.PrimaryDomain = cluster.Spec.PrimaryDomain
		reply.ComponentsList = &clusterv1.ComponentsList{
			MultiTenant:         transformComponent(cluster.Spec.ComponentsList.MultiTenant).(*clusterv1.Component),
			CertMgt:             transformComponent(cluster.Spec.ComponentsList.CertMgt).(*clusterv1.Component),
			SecretMgt:           transformComponent(cluster.Spec.ComponentsList.SecretMgt).(*clusterv1.Component),
			SecretSync:          transformComponent(cluster.Spec.ComponentsList.SecretSync).(*clusterv1.Component),
			IngressController:   transformComponent(cluster.Spec.ComponentsList.IngressController).(*clusterv1.Component),
			Deployment:          transformComponent(cluster.Spec.ComponentsList.Deployment).(*clusterv1.Component),
			ProgressiveDelivery: transformComponent(cluster.Spec.ComponentsList.ProgressiveDelivery).(*clusterv1.Component),
			Pipeline:            transformComponent(cluster.Spec.ComponentsList.Pipeline).(*clusterv1.Component),
			EventListener:       transformComponent(cluster.Spec.ComponentsList.EventListener).(*clusterv1.Component),
		}
		reply.ReservedNamespacesAllowedProducts = reservedNamespacesAllowedProducts
		reply.ProductAllowedClusterResources = productAllowedClusterResources
	}

	return reply, nil
}

// transformComponent Mutual conversion between two components, Resource Components are converted to API Components, and vice versa.
func transformComponent(component interface{}) interface{} {
	if component == nil {
		return nil
	}

	val := reflect.ValueOf(component).Elem()
	name := val.FieldByName("Name")
	namespace := val.FieldByName("Namespace")

	_, ok := component.(*resourcev1alpha1.Component)
	if ok {
		return &clusterv1.Component{
			Name:      name.String(),
			Namespace: namespace.String(),
		}
	}

	_, ok = component.(*clusterv1.Component)
	if ok {
		return &resourcev1alpha1.Component{
			Name:      name.String(),
			Namespace: namespace.String(),
		}
	}

	return nil
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

// constructResourceCluster creates a new Cluster resource based on the given SaveRequest and namespace.
func (s *ClusterService) constructResourceCluster(req *clusterv1.SaveRequest, namespace string) (*resourcev1alpha1.Cluster, error) {
	cluster := &resourcev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.Cluster,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.ClusterName,
			Namespace: namespace,
		},
		Spec: resourcev1alpha1.ClusterSpec{
			ApiServer:     req.Body.ApiServer,
			ClusterType:   resourcev1alpha1.ClusterType(req.Body.ClusterType),
			ClusterKind:   resourcev1alpha1.ClusterKind(req.Body.ClusterKind),
			Usage:         resourcev1alpha1.ClusterUsage(req.Body.Usage),
			HostCluster:   req.Body.HostCluster,
			PrimaryDomain: req.Body.PrimaryDomain,
			WorkerType:    resourcev1alpha1.ClusterWorkType(req.Body.WorkerType),
		},
	}

	err := s.setPrimaryDomainIfEmpty(cluster)
	if err != nil {
		return nil, err
	}

	componentsList, err := s.constructResourceComponentsList(req.Body.ComponentsList)
	if err != nil {
		return nil, err
	}

	if componentsList != nil {
		cluster.Spec.ComponentsList = *componentsList
	}

	reservedNamespaces, err := s.constructReservedNamespaces(req.Body.ReservedNamespacesAllowedProducts)
	if err != nil {
		return nil, err
	}

	if reservedNamespaces == nil {
		reservedNamespaces = make(map[string][]string)
	}
	cluster.Spec.ReservedNamespacesAllowedProducts = reservedNamespaces

	clusterResources, err := s.constructAllowedClusterResources(req.Body.ProductAllowedClusterResources)
	if err != nil {
		return nil, err
	}

	if clusterResources == nil {
		clusterResources = make(map[string][]resourcev1alpha1.ClusterResourceInfo)
	}
	cluster.Spec.ProductAllowedClusterResources = clusterResources

	return cluster, nil
}

func (*ClusterService) setPrimaryDomainIfEmpty(cluster *resourcev1alpha1.Cluster) error {
	if cluster.Spec.PrimaryDomain == "" {
		domain := utilstring.GetDomain(cluster.Spec.ApiServer)
		if domain == "" {
			return fmt.Errorf("failed to get domain for cluster")
		}

		isIp, err := utilstring.IsIp(cluster.Spec.ApiServer)
		if err != nil {
			return err
		}

		if isIp {
			cluster.Spec.PrimaryDomain = fmt.Sprintf("%s.nip.io", domain)
		} else {
			cluster.Spec.PrimaryDomain = domain
		}
	}

	return nil
}

func (s *ClusterService) constructResourceComponentsList(componentsList *clusterv1.ComponentsList) (*resourcev1alpha1.ComponentsList, error) {
	if componentsList == nil {
		componentsList = &clusterv1.ComponentsList{}
	}

	err := fillEmptyWithDefaultInComponentsList(componentsList, s.thirdPartComponentsList)
	if err != nil {
		return nil, err
	}

	resourceComponentList := convertComponentsList(componentsList)

	return resourceComponentList, nil
}

func fillEmptyWithDefaultInComponentsList(componentsList *clusterv1.ComponentsList, thirdPartComponentsList []*Component) error {
	val := reflect.Indirect(reflect.ValueOf(componentsList))
	err := setDefaultValuesIfEmpty(val, thirdPartComponentsList)
	if err != nil {
		return err
	}

	return nil
}

func loadThirdPartComponentsList(path string) ([]*Component, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		path := os.Getenv(EnvthirdPartComponents)
		if path == "" {
			return nil, err
		}

		bytes, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}

	components := []*Component{}
	err = yaml.Unmarshal(bytes, &components)
	if err != nil {
		return nil, err
	}

	errMsgs := validateMissingFieldValues(components)
	if len(errMsgs) > 0 {
		return nil, fmt.Errorf("failed to verify default component configuration, err: %s", strings.Join(errMsgs, ", "))
	}

	return components, nil
}

func validateMissingFieldValues(components []*Component) []string {
	resourceComponentsList := resourcev1alpha1.ComponentsList{}
	val := reflect.ValueOf(resourceComponentsList)

	componentsType := []string{}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if field.Type().Kind() != reflect.Ptr {
			continue
		}

		componentType := val.Type().Field(i).Name
		componentsType = append(componentsType, componentType)
	}

	tmp := make(map[string]*Component)
	for _, component := range components {
		key := strings.ToLower(component.Type)
		tmp[key] = component
	}

	errMsgs := make([]string, 0)
	for _, componentType := range componentsType {
		key := strings.ToLower(componentType)
		component, ok := tmp[key]
		if !ok {
			errMsgs = append(errMsgs, fmt.Sprintf("component '%s' value not found", componentType))
		}

		if component.Name == "" ||
			component.Namespace == "" ||
			len(component.InstallPath) == 0 {
			errMsgs = append(errMsgs, fmt.Sprintf("component '%s' value incomplete, the name and namespace of the component are not empty, and the installation path must exist", componentType))
		}
	}

	return errMsgs
}

func setDefaultValuesIfEmpty(val reflect.Value, componentsList []*Component) error {
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if field.Type().Kind() != reflect.Ptr {
			continue
		}

		componentType := val.Type().Field(i).Name

		if field.IsNil() {
			defaultComponent, err := getDefaultComponent(componentsList, componentType)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(defaultComponent))
		} else {
			err := fillDefaultFields(field, componentsList, componentType)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDefaultComponent(componentsList []*Component, componentType string) (*clusterv1.Component, error) {
	defaultComponent := getComponent(componentsList, componentType, "")
	if defaultComponent == nil {
		return nil, fmt.Errorf("failed to get default component for the component type '%s'", componentType)
	}
	return defaultComponent, nil
}

func fillDefaultFields(field reflect.Value, componentsList []*Component, componentType string) error {
	component := field.Elem()
	name := component.FieldByName("Name")
	namespace := component.FieldByName("Namespace")

	if name.String() == "" {
		defaultComponent, err := getDefaultComponent(componentsList, componentType)
		if err != nil {
			return fmt.Errorf("failed to get default component for the component type '%s'", componentType)
		}
		name.SetString(defaultComponent.Name)
	}

	if namespace.String() == "" {
		component := getComponent(componentsList, componentType, name.String())
		if component == nil {
			return fmt.Errorf("failed to get %s component for the component type '%s'", name.String(), componentType)
		}
		namespace.SetString(component.Namespace)
	}
	return nil
}

func getComponent(componentsList []*Component, componentType, componentName string) *clusterv1.Component {
	for _, component := range componentsList {
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

			tmp := resourcev1alpha1.ClusterResourceInfo{}
			err = json.Unmarshal(bytes, &tmp)
			if err != nil {
				return nil, fmt.Errorf("failed to parsing cluster resource information of %s to struct: 'ClusterResourceInfo', err: %s", productName, err)
			}

			if tmp.Group == "" ||
				tmp.Kind == "" {
				return nil, fmt.Errorf("failed to get cluster resource information of %s, the group and kind of the cluster resource cannot be empty", productName)
			}

			clusterResourcesInfo = append(clusterResourcesInfo, tmp)
		}

		clusterResourceInfoMap[productName] = clusterResourcesInfo
	}

	return clusterResourceInfoMap, nil
}

func convertComponentsList(components *clusterv1.ComponentsList) *resourcev1alpha1.ComponentsList {
	return &resourcev1alpha1.ComponentsList{
		MultiTenant:         transformComponent(components.MultiTenant).(*resourcev1alpha1.Component),
		CertMgt:             transformComponent(components.CertMgt).(*resourcev1alpha1.Component),
		SecretMgt:           transformComponent(components.SecretMgt).(*resourcev1alpha1.Component),
		SecretSync:          transformComponent(components.SecretSync).(*resourcev1alpha1.Component),
		IngressController:   transformComponent(components.IngressController).(*resourcev1alpha1.Component),
		Deployment:          transformComponent(components.Deployment).(*resourcev1alpha1.Component),
		ProgressiveDelivery: transformComponent(components.ProgressiveDelivery).(*resourcev1alpha1.Component),
		Pipeline:            transformComponent(components.Pipeline).(*resourcev1alpha1.Component),
		EventListener:       transformComponent(components.EventListener).(*resourcev1alpha1.Component),
	}
}
