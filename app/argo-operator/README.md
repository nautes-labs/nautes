# Argo Operator

Argo Operator 项目提供了一组用于调谐 Cluster 资源和 CodeRepo 资源事件的 Controller，调谐内容主要是将 Cluster 资源所声明的 Kubernetes 集群和 CodeRepo 资源所声明的代码库同步到同集群的 ArgoCD 中，使 ArgoCD 中使用了这些 Kubernetes 集群和代码库的 Application 可以正常工作。

## 功能简介

Controller 除了可以通过响应资源的增删改事件进行调谐外，还以定时轮询的方式检查 Kubernetes 集群和代码库信息的变化并进行调谐。

当 Controller 监听到 Cluster 资源有增改或检测到密钥管理系统中的 Kubernetes 集群 kubeconfig 信息有变化时，会主动从密钥管理系统中获取最新的 kubeconfig 内容，并通过 API 将集群同步到 ArgoCD 中。同步成功后 Controller 会在 Cluster 资源中记录此次同步的 kubeconfig 内容的版本，作为下一次同步时检查数据差异的基准值。当 Controller 监听到 Cluster 资源被删除时，会通过 API 从 ArgoCD 中移除相应的 Kubernetes 集群。

当 Controller 监听到 CodeRepo 资源有增改或检测到密钥管理系统中的代码库 DeployKey 信息有变化时，会主动从密钥管理系统中获取最新的 DeployKey 内容，并通过 API 将代码库同步到 ArgoCD 中。同步成功后 Controller 会在 CodeRepo 资源中记录此次同步的 DeployKey 内容的版本，作为下一次同步时检查数据差异的基准值。当 Controller 监听到 CodeRepo 资源被删除时，会通过 API 从 ArgoCD 中移除相应的代码库。
