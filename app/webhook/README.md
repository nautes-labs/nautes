# Webhook

Webhook 项目用于检查nautes 的资源是否有效，防止无效的资源创建、修改写入kubernetes， 防止资源被引用时执行删除命令。


## 功能简介

Webhook 主要是提供校验资源有效性的功能，包括：

| 资源类型 | 校验逻辑 |
| --- | ---|
| Cluster | 有虚拟集群的宿主集群无法被删除<br>有运行时环境的运行时集群无法被删除<br>有运行时环境的运行时集群无法变更用途<br>有虚拟集群的宿主集群无法变更用途<br>宿主集群必须是物理集群<br>虚拟集群的 hostCluster 为必填项<br>物理集群的 hostCluster 必须为空 |
| Deployment Runtime | 已经部署过的运行时不可以切换目标环境<br>不允许在同一个环境中创建两个指向相同源的部署运行时 |
| Environment | 被运行时引用的环境无法被删除 |
| Product Provider | 被产品引用的产品提供者无法被删除 |
| Code Repo | 代码仓库地址是否存在且有效 |
