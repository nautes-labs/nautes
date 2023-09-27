
<div style="margin-top: 40px; margin-left: 20px; margin-right: 20px;">
<p align="center">
<a href="https://nautes.io/"><img src="docs/images/nautes.png" alt="banner" width="147" height="125.4"></a>
</p>
<p align=center>
<a href="https://img.shields.io/badge/License-Apache%202.0-blue.svg"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
<a href="https://img.shields.io/badge/kubernetes-1.21-green"><img src="https://img.shields.io/badge/kubernetes-1.21-green" alt="Kubernetes"></a>
<a href="https://img.shields.io/badge/version-v0.4.0-green"><img src="https://img.shields.io/badge/version-v0.4.0-green" alt="Version"></a>
</p>
</div>

> [English](./README_en.md) | 中文

## Nautes 是什么？

Nautes 是 Kubernetes 原生的开源一站式开发者平台，融合了 DevOps 和 GitOps 的理念和最佳实践，以可插拔的方式集成了业界最优秀的云原生开源项目。

> 当前版本仅用于演示或试用，功能还在不断完善中，不建议用在生产环境。

## 特性

- 覆盖敏捷开发、CI/CD、自动化测试、安全、运维等全流程的一站式开发者平台。
- 遵循 GitOps 最佳实践，以版本库作为唯一可信数据源。当版本库中的数据有变更时，由 Operator 自动识别变更并向 Kubernetes 集群做增量更新。
- 全分布式的多租户架构，租户作为分布式的计算单元和存储单元支持水平扩展，租户所管理的资源同样支持水平扩展。
- 良好的适配性，除了底座 Kubernetes 以及 Git 外，其他组件均可被替换。
- 所有功能均提供声明式的REST API，支持二次开发。
- 对所集成的开源项目，均保持其原生特性，无裁剪式封装，对受管应用不产生二次绑定。
- 通过构建上层数据模型，实现对所集成的开源项目的统一认证、统一授权。
- 支持私有云、混合云的部署模式。

## 架构

Nautes 采用全分布式的多租户架构，平台管理集群负责租户的分配和回收，每个租户独占一套资源（包括代码库、密钥库、制品库、认证服务器、集群等），租户内的资源由租户管理集群进行管理。

租户作为资源的管理单元，可由用户根据自身组织特性进行划分，常见的划分方式有：按产品团队、按部门、按子公司等。

租户内的资源也支持多实例部署，例如：可以在一个租户内部署多个 Harbor 实例，用于隔离不同产品的容器镜像数据。

![](docs/images/brief-architecture.png)

## 开源项目

Nautes 的当前版本主要集成了以下开源项目（序号不代表排序）：

> 我们对这些优秀项目（包括在 Nautes 中使用到，但未在下表列出的所有项目）的作者表示衷心的感谢！

| 序号 | 开源项目           | 用途                          | 版本          | 开源许可     | 项目地址                                                   |
| ---- | ------------------ | ----------------------------- | ------------- | ------------ | ---------------------------------------------------------- |
| 1    | Terraform          | 用于构建基础设施              | v1.3.4        | MPL-2.0      | https://github.com/hashicorp/terraform                     |
| 2    | Ansible            | 安装程序的脚手架              | 2.12.5        | GPL-3.0      | https://github.com/ansible/ansible                         |
| 3    | Kubespray          | Kubernetes 的安装程序         | v2.19.1       | Apache-2.0   | https://github.com/kubernetes-sigs/kubespray               |
| 4    | Ansible Vault      | Vault 的安装程序              | v2.5.8        | BSD-2-Clause | https://github.com/ansible-community/ansible-vault         |
| 5    | Geerlingguy.Gitlab | Gitlab 的安装程序             | 3.2.0         | MIT          | https://github.com/geerlingguy/ansible-role-gitlab         |
| 6    | Kubebuilder        | Operator 脚手架               | v3.2.0        | Apache-2.0   | https://github.com/kubernetes-sigs/kubebuilder             |
| 7    | Kratos             | 微服务脚手架                  | v2.5.4        | MIT          | https://github.com/go-kratos/kratos                        |
| 8    | Vuepress           | 文档站点脚手架                | 1.9.9         | MIT          | https://github.com/vuejs/vuepress                          |
| 9    | Kubernetes         | 容器平台、产品底座            | 1.23.7        | Apache-2.0   | https://github.com/kubernetes/kubernetes                   |
| 10   | K3s                | 轻量级 Kubernetes 发行版      | v1.21.13-k3s1 | Apache-2.0   | https://github.com/k3s-io/k3s                              |
| 11   | Vcluster           | 用于 Kubernetes 的多租户隔离  | 0.10.1        | Apache-2.0   | https://github.com/loft-sh/vcluster                        |
| 12   | HNC                | 用于 Kubernetes 的多租户隔离  | v1.0.0        | Apache-2.0   | https://github.com/kubernetes-sigs/hierarchical-namespaces |
| 13   | Dex                | 用于单点登录、统一认证        | v2.32.0       | Apache-2.0   | https://github.com/dexidp/dex                              |
| 14   | Vault              | 密钥仓库                      | 1.10.4        | MPL-2.0      | https://github.com/hashicorp/vault                         |
| 15   | External Secrets   | 用于同步密钥到 Kubernetes     | 0.5.7         | Apache-2.0   | https://github.com/external-secrets/external-secrets       |
| 16   | Gitlab             | 代码仓库、IDP、基础数据提供者 | 15.10         | MIT          | https://gitlab.com/gitlab-org/gitlab                       |
| 17   | Nexus              | 成品包仓库                    | 3.39.0        | EPL-1.0      | https://github.com/sonatype/nexus-public                   |
| 18   | Harbor             | 容器镜像仓库                  | v2.5.1        | Apache-2.0   | https://github.com/goharbor/harbor                         |
| 19   | Argocd             | 用于持续部署                  | v2.4.0        | Apache-2.0   | https://github.com/argoproj/argo-cd                        |
| 20   | Argorollouts       | 用于渐进式交付                | v1.2.1        | Apache-2.0   | https://github.com/argoproj/argo-rollouts                  |
| 21   | Argoevents         | 事件监听器                    | v1.7.1        | Apache-2.0   | https://github.com/argoproj/argo-events                    |
| 22   | Tekton Pipeline    | 流水线                        | v0.37.5       | Apache-2.0   | https://github.com/tektoncd/pipeline                       |
| 23   | Cert Manager       | 用于自签证书                  | v1.8.0        | Apache-2.0   | https://github.com/cert-manager/cert-manager               |
| 24   | Traefik            | Ingress 控制器                | v2.7.1        | MIT          | https://github.com/traefik/traefik                         |
| 25   | Sonarqube          | 用于代码静态检查              | 9.5           | LGPL-3.0     | https://github.com/SonarSource/sonarqube                   |

## 主体功能

Nautes 的主体流程以及参与角色如下：

**租户管理员**：负责管理租户内的资源组件，如注册集群、接入制品库等。

**配置管理员**：负责管理IT系统在开发和运行过程中所需的环境和资源，如维护产品基础数据、创建代码库、分配制品库、定义运行时等。

**产品团队**：使用平台功能进行IT系统的研发和运行，如提交代码、上传依赖包、配置流水线、探索性测试等。

![](docs/images/main-process.png)

## 实体定义

- **产品**：对应一个软件系统，包含团队、项目、环境、代码库、制品库、及运行时。产品可以被租户管理员授权使用指定的 Kubernetes 集群。
- **项目**：对应一个微服务，每个项目有自己的代码库。您可以使用集群进行项目的集成和部署，也可以使用产品的制品库对项目的制品进行存储和版本管理。 一个产品下可以包含多个项目。
- **环境**：使用集群（目前只支持 Kubernetes集群）来承载产品中各个项目的集成和部署的管理单元。一个产品包含多个环境，如：开发环境、测试环境、预生产环境和生产环境等。
- **代码库**：用于存储项目的源代码、流水线配置、部署清单的版本库。只支持 Git。
- **流水线运行时**：定义用于集成项目的流水线的配置声明，如：流水线配置的存储位置、流水线的触发方式、运行流水线的目标环境等。
- **部署运行时**：定义用于部署项目的配置声明，如：部署清单的存储位置、部署到的目标环境等。

## 核心组件

Nautes 主要包含以下组件：

<details>
  <summary><b>👤 Base Operator</b></summary>
  处理产品实体和权限实体从提供者到目标服务的同步。<a href="./app/base-operator">了解更多</a>。
</details>

<details>
  <summary><b>🖥️ Cluster Operator</b></summary>
  提供了一个用于调谐 Cluster 资源事件的 Controller，调谐内容主要是管理 Cluster 资源所声明的 Kubernetes 集群的密钥信息，使参与集群管理的其他组件可以从租户的密钥管理系统中正确获取到集群的密钥。<a href="./app/cluster-operator">了解更多</a>。
</details>

<details>
  <summary><b>🔗 Argo Operator</b></summary>
  提供了一组用于调谐 Cluster 资源和 CodeRepo 资源事件的 Controller，调谐内容主要是将 Cluster 资源所声明的 Kubernetes 集群和 CodeRepo 资源所声明的代码库同步到同集群的 ArgoCD 中，使 ArgoCD 中使用了这些 Kubernetes 集群和代码库的 Application 可以正常工作。<a href="./app/argo-operator">了解更多</a>。
</details>

<details>
  <summary><b>⚙️ Runtime Operator</b></summary>
  提供了一组用于调谐 Project Pipeline Runtime 资源和 Deployment Runtime 资源事件的 Controller，调谐内容主要是根据两类运行时资源的声明信息，在目标集群上同步流水线执行或应用部署所需的基础环境。<a href="./app/runtime-operator">了解更多</a>。
</details>

<details>
  <summary><b>🤖 Installer</b></summary>
  提供了一键部署功能，支持基础设施、资源组件、管理组件、以及各组件初始化的自动化安装。<a href="https://github.com/nautes-labs/installer">了解更多</a>。
</details>

<details>
  <summary><b>🌐 API Server</b></summary>
  Nautes 的设计是遵循了 GitOps 的最佳实践，用户应用环境以及 Nautes 自身环境的配置声明均存储在版本库中。声明数据分为两类：密钥数据是存储在 Vault 中，其他数据是存储在 Git（目前只支持 GitLab）仓库中，API Server 项目则提供了一组用于操作这些配置声明的 REST API。<a href="./app/api-server">了解更多</a>。
</details>

<details>
  <summary><b>➡️ CLI</b></summary>
  通过封装 API Server 的 REST API 提供了一个简单的命令行工具，用于简化用户使用 API 的操作。<a href="https://github.com/nautes-labs/cli">了解更多</a>。
</details>

## 安装

Nautes 支持基于公有云、私有云、主机、及 Kubernets 集群进行安装，您可以通过[这里](https://nautes.io/guide/user-guide/installation.html)了解如何在阿里云上一键安装 Nautes。

## 快速开始

我们提供了一份简要的指南，您可以通过这份指南快速地部署出[第一个流水线](https://nautes.io/guide/user-guide/run-a-pipeline.html)和[应用](https://nautes.io/guide/user-guide/deploy-an-application.html)。

## 路线图

我们通过这个[项目](https://github.com/orgs/nautes-labs/projects/1)提供了关于产品路线图以及进度等信息。
