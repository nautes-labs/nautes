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

	clusterv1 "github.com/nautes-labs/nautes/api/api-server/cluster/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/internal/biz"
	clustermanagement "github.com/nautes-labs/nautes/app/api-server/pkg/clusters"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"github.com/nautes-labs/nautes/app/api-server/pkg/selector"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	clusterconfig "github.com/nautes-labs/nautes/pkg/config/cluster"
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
	cluster                *biz.ClusterUsecase
	nautesConfigs          *nautesconfigs.Config
	clusterComponentConfig *clusterconfig.ClusterComponentConfig
	components             *clustermanagement.ComponentsList
}

func NewClusterService(cluster *biz.ClusterUsecase, nautesConfigs *nautesconfigs.Config) (*ClusterService, error) {
	clusterComponentConfig, err := clusterconfig.NewClusterComponentConfig()
	if err != nil {
		return nil, err
	}

	return &ClusterService{
		cluster:                cluster,
		nautesConfigs:          nautesConfigs,
		clusterComponentConfig: clusterComponentConfig,
		components:             clustermanagement.NewComponentsList(),
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

	cluster, err := s.getCluster(req)
	if err != nil {
		return nil, fmt.Errorf("failed to construct cluster, err: %v", err)
	}

	vcluster, err := s.getVcluster(cluster, req)
	if err != nil {
		return nil, err
	}

	kubeconfig, err := convertKubeconfig(req.Body.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig, err: %v", err)
	}

	param := &clustermanagement.ClusterRegistrationParams{
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

func (*ClusterService) getVcluster(cluster *resourcev1alpha1.Cluster, req *clusterv1.SaveRequest) (*clustermanagement.VclusterInfo, error) {
	if clustermanagement.IsPhysical(cluster) {
		return nil, nil
	}

	var httpsNodePort string
	var err error

	if req.Body.Vcluster != nil && req.Body.Vcluster.HttpsNodePort != "" {
		httpsNodePort = req.Body.Vcluster.HttpsNodePort
	} else {
		httpsNodePort, err = utilstring.ExtractPortFromURL(cluster.Spec.ApiServer)
		if err != nil {
			return nil, err
		}
	}

	return &clustermanagement.VclusterInfo{
		HttpsNodePort: httpsNodePort,
	}, nil
}

func convertKubeconfig(kubeconfig string) (string, error) {
	if utilstring.IsBase64Decoding(kubeconfig) {
		decoded, err := base64.StdEncoding.DecodeString(kubeconfig)
		if err != nil {
			return "", fmt.Errorf("failed to decode kubeconfig: %v", err)
		}

		return string(decoded), nil
	}

	return kubeconfig, nil
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
	var reservedNamespacesAllowedProducts, productAllowedClusterResources map[string]*structpb.ListValue
	var err error

	if cluster.Spec.ReservedNamespacesAllowedProducts != nil {
		reservedNamespacesAllowedProducts, err = transformMap(cluster.Spec.ReservedNamespacesAllowedProducts)
		if err != nil {
			return nil, fmt.Errorf("failed to get reserved namespaces, err: %s", err)
		}
	}

	if cluster.Spec.ProductAllowedClusterResources != nil {
		productAllowedClusterResources, err = transformMap(cluster.Spec.ProductAllowedClusterResources)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster resources, err: %s", err)
		}
	}

	return &clusterv1.GetReply{
		Name:                              cluster.Name,
		ApiServer:                         cluster.Spec.ApiServer,
		ClusterKind:                       string(cluster.Spec.ClusterKind),
		ClusterType:                       string(cluster.Spec.ClusterType),
		HostCluster:                       cluster.Spec.HostCluster,
		PrimaryDomain:                     cluster.Spec.PrimaryDomain,
		WorkerType:                        string(cluster.Spec.WorkerType),
		Usage:                             string(cluster.Spec.Usage),
		ReservedNamespacesAllowedProducts: reservedNamespacesAllowedProducts,
		ProductAllowedClusterResources:    productAllowedClusterResources,
		ComponentsList: &clusterv1.ComponentsList{
			MultiTenant:         getClusterComponent(cluster.Spec.ComponentsList.MultiTenant),
			CertManagement:      getClusterComponent(cluster.Spec.ComponentsList.CertManagement),
			SecretSync:          getClusterComponent(cluster.Spec.ComponentsList.SecretSync),
			Gateway:             getClusterComponent(cluster.Spec.ComponentsList.Gateway),
			Deployment:          getClusterComponent(cluster.Spec.ComponentsList.Deployment),
			ProgressiveDelivery: getClusterComponent(cluster.Spec.ComponentsList.ProgressiveDelivery),
			Pipeline:            getClusterComponent(cluster.Spec.ComponentsList.Pipeline),
		},
	}, nil
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
func (s *ClusterService) getCluster(req *clusterv1.SaveRequest) (*resourcev1alpha1.Cluster, error) {
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

	err = s.setDefaultComponent(componentsList, req)
	if err != nil {
		return nil, err
	}

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
			Namespace: s.nautesConfigs.Nautes.Namespace,
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

// setDefaultComponent secret management component is not open to user.
// oauth proxy component is not open to user.
func (s *ClusterService) setDefaultComponent(list *resourcev1alpha1.ComponentsList, req *clusterv1.SaveRequest) error {
	secretManagementComponent, err := s.setComponentDefaults(string(clusterconfig.SecretManagement), "vault", req)
	if err != nil {
		return err
	}
	list.SecretManagement = &resourcev1alpha1.Component{
		Name:      secretManagementComponent.Name,
		Namespace: secretManagementComponent.Namespace,
		Additions: secretManagementComponent.Additions,
	}

	if (req.Body.ClusterType == string(resourcev1alpha1.CLUSTER_TYPE_PHYSICAL) &&
		req.Body.WorkerType == string(resourcev1alpha1.ClusterWorkTypePipeline)) ||
		req.Body.Usage == string(resourcev1alpha1.CLUSTER_USAGE_HOST) {
		oauthproxyComponent, err := s.setComponentDefaults(string(clusterconfig.OauthProxy), "oauth2-proxy", req)
		if err != nil {
			return err
		}
		list.OauthProxy = &resourcev1alpha1.Component{
			Name:      oauthproxyComponent.Name,
			Namespace: oauthproxyComponent.Namespace,
			Additions: oauthproxyComponent.Additions,
		}
	}

	return nil
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
	config, err := clusterconfig.NewClusterComponentConfig()
	if err != nil {
		return nil, err
	}

	componetsDefinition, err := config.GetClusterComponentsDefinition(&clusterconfig.ClusterInfo{
		Name:        req.GetClusterName(),
		ClusterType: body.GetClusterType(),
		WorkType:    body.GetWorkerType(),
		Usage:       body.GetUsage(),
	})
	if err != nil {
		return nil, err
	}

	componentsList, err := s.setDefaultValuesIfEmpty(componetsDefinition, req)
	if err != nil {
		return nil, err
	}

	return s.convertComponentsList(componentsList), nil
}

func (s *ClusterService) setDefaultValuesIfEmpty(componetsDefinition []string, req *clusterv1.SaveRequest) (*clusterv1.ComponentsList, error) {
	componentsList := req.Body.GetComponentsList()
	if componentsList == nil {
		componentsList = &clusterv1.ComponentsList{}
	}

	val := reflect.Indirect(reflect.ValueOf(componentsList))
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if field.Type().Kind() != reflect.Ptr {
			continue
		}

		componentType := utilstring.FirstCharToLower(val.Type().Field(i).Name)
		isContain := utilstring.ContainsString(componetsDefinition, componentType)
		if !isContain {
			if !field.IsNil() {
				field.Set(reflect.ValueOf(nil))
			}
			continue
		}

		// Set default component if component is empty.
		// If the component field is missing a value, set the default value.
		if field.IsNil() {
			defaultComponent, err := s.getDefaultComponent(componentType, req)
			if err != nil {
				return nil, err
			}
			field.Set(reflect.ValueOf(defaultComponent))
		} else {
			// todo
			// add check component name function.

			err := s.fillDefaultFields(field, componentType, req)
			if err != nil {
				return nil, err
			}
		}
	}

	return componentsList, nil
}

func (s *ClusterService) getDefaultComponent(componentType string, req *clusterv1.SaveRequest) (*clusterv1.Component, error) {
	defaultComponent, err := s.setComponentDefaults(componentType, "", req)
	if err != nil {
		return nil, err
	}

	return defaultComponent, nil
}

func (s *ClusterService) fillDefaultFields(field reflect.Value, componentType string, req *clusterv1.SaveRequest) error {
	component := field.Elem()
	name := component.FieldByName("Name")
	namespace := component.FieldByName("Namespace")

	if name.String() == "" {
		defaultComponent, err := s.getDefaultComponent(componentType, req)
		if err != nil {
			return fmt.Errorf("failed to get default component for the component type '%s'", componentType)
		}

		name.SetString(defaultComponent.Name)
		namespace.SetString(defaultComponent.Namespace)

		return nil
	}

	if namespace.String() == "" {
		component, err := s.setComponentDefaults(componentType, name.String(), req)
		if err != nil {
			return fmt.Errorf("failed to get %s component for the component type '%s'", name.String(), componentType)
		}
		namespace.SetString(component.Namespace)
	}

	return nil
}

func (s *ClusterService) setComponentDefaults(componentType, componentName string, req *clusterv1.SaveRequest) (*clusterv1.Component, error) {
	thirdPartComponent, err := s.getThirdComponent(componentType, componentName)
	if err != nil {
		return nil, err
	}

	additions := s.setDefaultValue(thirdPartComponent, componentType, req)

	return &clusterv1.Component{
		Name:      thirdPartComponent.Name,
		Namespace: thirdPartComponent.Namespace,
		Additions: additions,
	}, nil
}

func (s *ClusterService) setDefaultValue(thirdPartComponent *clusterconfig.ThridPartComponent, componentType string, req *clusterv1.SaveRequest) map[string]string {
	var additions = make(map[string]string)
	for _, prop := range thirdPartComponent.Properties {
		if prop.Default != "" {
			additions[prop.Name] = prop.Default
			continue
		}

		component, _ := s.components.GetComponent(componentType, thirdPartComponent.Name)
		if component == nil {
			continue
		}

		getDefaultValueFn := reflect.ValueOf(component).MethodByName("GetDefaultValue")
		if !getDefaultValueFn.IsValid() {
			continue
		}

		attribute := reflect.ValueOf(prop.Name)
		opt := reflect.ValueOf(&clustermanagement.DefaultValueOptions{
			Cluster: &clustermanagement.Cluster{
				Name:        req.ClusterName,
				ApiServer:   req.Body.ApiServer,
				Usage:       req.Body.Usage,
				WorkerType:  req.Body.WorkerType,
				ClusterType: req.Body.ClusterType,
			},
		})
		args := []reflect.Value{attribute, opt}
		result := getDefaultValueFn.Call(args)
		if len(result) == 0 {
			continue
		}

		prop.Default = result[0].String()
		additions[prop.Name] = prop.Default
	}
	return additions
}

// getThirdComponent  Load third-party component configuration based on component name. If the name is empty, use the default component.
func (s *ClusterService) getThirdComponent(componentType, componentName string) (*clusterconfig.ThridPartComponent, error) {
	var thirdPartComponent *clusterconfig.ThridPartComponent
	var err error

	if componentName == "" {
		thirdPartComponent, err = s.clusterComponentConfig.GetDefaultThirdPartComponentByType(componentType)
		if err != nil {
			return nil, err
		}
	} else {
		thirdPartComponent, err = s.clusterComponentConfig.GetThirdPartComponentByName(componentName)
		if err != nil {
			return nil, err
		}
	}
	return thirdPartComponent, nil
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
	list := &resourcev1alpha1.ComponentsList{
		MultiTenant:         getResourceComponent(components.MultiTenant),
		CertManagement:      getResourceComponent(components.CertManagement),
		SecretSync:          getResourceComponent(components.SecretSync),
		Gateway:             getResourceComponent(components.Gateway),
		Deployment:          getResourceComponent(components.Deployment),
		ProgressiveDelivery: getResourceComponent(components.ProgressiveDelivery),
		Pipeline:            getResourceComponent(components.Pipeline),
		EventListener:       getResourceComponent(components.EventListener),
	}

	return list
}

func getResourceComponent(component *clusterv1.Component) *resourcev1alpha1.Component {
	if component == nil {
		return nil
	}

	return &resourcev1alpha1.Component{
		Name:      component.Name,
		Namespace: component.Namespace,
		Additions: component.Additions,
	}
}

func getClusterComponent(component *resourcev1alpha1.Component) *clusterv1.Component {
	if component == nil {
		return nil
	}

	return &clusterv1.Component{
		Name:      component.Name,
		Namespace: component.Namespace,
		Additions: component.Additions,
	}
}
