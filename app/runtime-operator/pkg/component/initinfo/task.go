package initinfo

import "k8s.io/client-go/rest"

type ComponentInitInfo struct {
	ClusterConnectInfo ClusterConnectInfo
	ProductName        string
	ClusterName        string
	Labels             map[string]string
}

type ClusterType string

const (
	ClusterTypeKubernetes = "kubernetes"
)

type ClusterConnectInfo struct {
	ClusterType ClusterType
	Kubernetes  *ClusterConnectInfoKubernetes
}

type ClusterConnectInfoKubernetes struct {
	Config *rest.Config
}
