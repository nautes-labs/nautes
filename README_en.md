
<div style="margin-top: 40px; margin-left: 20px; margin-right: 20px;">
<p align="center">
<a href="https://nautes.io/"><img src="docs/images/nautes.png" alt="banner" width="147" height="125.4"></a>
</p>
<p align=center>
<a href="https://img.shields.io/badge/License-Apache%202.0-blue.svg"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
<a href="https://img.shields.io/badge/kubernetes-1.21-green"><img src="https://img.shields.io/badge/kubernetes-1.21-green" alt="Kubernetes"></a>
<a href="https://img.shields.io/badge/version-v0.3.0-green"><img src="https://img.shields.io/badge/version-v0.3.0-green" alt="Version"></a>
</p>
</div>

> English | [中文](README.md)

## What is Nautes? 

Nautes is a Kubernetes-native all-in-one Internal Developer Platform that combines the concepts and best practices of DevOps and GitOps. It integrates the industry's best cloud-native open-source projects in a pluggable manner.

> The current version is for demonstration or trial purposes only, and its features are still being continuously improved. It is not recommended for use in production environments.

## Features
- a Kubernetes-native all-in-one Internal Developer Platform that covers the entire process, including agile development, CI/CD, automated testing, security, and operations.
- Following the best practices of GitOps, with the version repository as the only trusted data source. When data in the repository changes, the Operator automatically detects the changes and performs incremental updates to the Kubernetes cluster. 
- A fully distributed multi-tenant architecture, where tenants serve as distributed computing and storage units that support horizontal scaling. The resources hosted by tenants also support horizontal scaling. 
- Good adaptability, In addition to the base Kubernetes and Git, other components can be replaced.
- All features are provided with declarative REST APIs, supporting secondary development. 
- For all integrated open-source projects, their native features are maintained without any trimmed encapsulation, ensuring that there is no secondary binding for the hosted applications. 
- By constructing a higher-level data model, unified authentication and authorization are achieved for all integrated open-source projects.
- Supports deployment modes for private cloud and hybrid cloud. 

## Architecture

Nautes adopts a fully distributed multi-tenant architecture, where the platform management cluster is responsible for tenant allocation and recovery. Each tenant has exclusive access to a set of resources, including code repositories, key repositories, artifact repositories, authentication servers, and clusters. Resources within a tenant are managed by the tenant management cluster. 

Tenants serve as the unit of resource management which can be divided by users based on their organization's characteristics, such as by product teams, departments, or subsidiaries.

Resources within a tenant can also be deployed with multiple instances, for example, multiple Harbor instances can be deployed within a single tenant to isolate container image data for different products. 

![](docs/images/brief-architecture.png)

## Open Source Projects

The current version of Nautes mainly integrates the following open source projects (the order does not represent ranking):

> We express our sincere gratitude to the authors of these outstanding projects (including all projects used in Nautes but not listed in the table below)!

| SN   | Projects           | Usage                                        | Version       | License      | URL                                                        |
| ---- | ------------------ | -------------------------------------------- | ------------- | ------------ | ---------------------------------------------------------- |
| 1    | Terraform          | For building infrastructure                  | v1.3.4        | MPL-2.0      | https://github.com/hashicorp/terraform                     |
| 2    | Ansible            | The scaffold of the installation program     | 2.12.5        | GPL-3.0      | https://github.com/ansible/ansible                         |
| 3    | Kubespray          | The installation program of Kubernetes       | v2.19.1       | Apache-2.0   | https://github.com/kubernetes-sigs/kubespray               |
| 4    | Ansible Vault      | The installation program of Vault            | v2.5.8        | BSD-2-Clause | https://github.com/ansible-community/ansible-vault         |
| 5    | Geerlingguy.Gitlab | The installation program of Gitlab           | 3.2.0         | MIT          | https://github.com/geerlingguy/ansible-role-gitlab         |
| 6    | Kubebuilder        | The scaffold of Operator                     | v3.2.0        | Apache-2.0   | https://github.com/kubernetes-sigs/kubebuilder             |
| 7    | Kratos             | The scaffold of Microservice                 | v2.5.4        | MIT          | https://github.com/go-kratos/kratos                        |
| 8    | Vuepress           | The scaffold of document site                | 1.9.9         | MIT          | https://github.com/vuejs/vuepress                          |
| 9    | Kubernetes         | Container platform, product base             | 1.23.7        | Apache-2.0   | https://github.com/kubernetes/kubernetes                   |
| 10   | K3s                | Lightweight Kubernetes Distribution          | v1.21.13-k3s1 | Apache-2.0   | https://github.com/k3s-io/k3s                              |
| 11   | Vcluster           | Multi-tenancy isolation for Kubernetes       | 0.10.1        | Apache-2.0   | https://github.com/loft-sh/vcluster                        |
| 12   | HNC                | Multi-tenancy isolation for Kubernetes       | v1.0.0        | Apache-2.0   | https://github.com/kubernetes-sigs/hierarchical-namespaces |
| 13   | Dex                | For SSO and unified authentication           | v2.32.0       | Apache-2.0   | https://github.com/dexidp/dex                              |
| 14   | Vault              | Secret repository                            | 1.10.4        | MPL-2.0      | https://github.com/hashicorp/vault                         |
| 15   | External Secrets   | For synchronizing secrets to Kubernetes      | 0.5.7         | Apache-2.0   | https://github.com/external-secrets/external-secrets       |
| 16   | Gitlab             | Code repository, IDP, and base-data provider | 15.10         | MIT          | https://gitlab.com/gitlab-org/gitlab                       |
| 17   | Nexus              | Package repository                           | 3.39.0        | EPL-1.0      | https://github.com/sonatype/nexus-public                   |
| 18   | Harbor             | Container image repository                   | v2.5.1        | Apache-2.0   | https://github.com/goharbor/harbor                         |
| 19   | Argocd             | For continuous deployment                    | v2.4.0        | Apache-2.0   | https://github.com/argoproj/argo-cd                        |
| 20   | Argorollouts       | For progressive delivery.                    | v1.2.1        | Apache-2.0   | https://github.com/argoproj/argo-rollouts                  |
| 21   | Argoevents         | Event listener                               | v1.7.1        | Apache-2.0   | https://github.com/argoproj/argo-events                    |
| 22   | Tekton Pipeline    | Continuous Integration Pipeline              | v0.37.5       | Apache-2.0   | https://github.com/tektoncd/pipeline                       |
| 23   | Cert Manager       | For self-signed certificate                  | v1.8.0        | Apache-2.0   | https://github.com/cert-manager/cert-manager               |
| 24   | Traefik            | Ingress Controller                           | v2.7.1        | MIT          | https://github.com/traefik/traefik                         |
| 25   | Sonarqube          | For static code analysis                     | 9.5           | LGPL-3.0     | https://github.com/SonarSource/sonarqube                   |

## Core Functions

The main processes and roles involved in Nautes are as follows:

**Tenant Manager:** Responsible for managing the resource components within a tenant, such as registering clusters and artifact repositories.  
**Configuration Manager:** Responsible for managing the environment and resources required by the IT system during development and operation, such as maintaining product metadata, creating code repositories, assigning artifact repositories, and defining runtime environments.  
**Product Team:** Uses the platform to develop and operate the IT system, such as submitting code, uploading dependencies, configuring pipelines, conducting exploratory testing, and so on.

![](docs/images/main-process.png)

## Entity Definition

- **Product:** Corresponds to a software system, which includes teams, projects, environments, code repositories, artifact repositories, and runtimes. The tenant manager can authorize the product to use designated Kubernetes clusters.
- **Project:** Corresponds to a microservice, and each project has its own code repositories. You can integrate and deploy projects using Kubernetes clusters, and store versioned artifacts in artifact repositories. A product can contain multiple projects.
- **Environment:** A management unit that uses a cluster to host the integration and deployment of various microservices in the product. Currently, we only support the Kubernetes cluster type. A product contains multiple environments, such as development, testing, pre-production, and production environments.
- **Code Repository:** A repository used for storing a project's source code, pipeline configurations, or deployment manifests. Only Git is supported.
- **Pipeline Runtime:** The configuration declaration of a pipeline used to integrate projects, such as the storage location of pipeline configurations, the pipeline's triggering method and the target environment for running the pipeline, etc.
- **Deployment Runtime:** The configuration declaration used to deploy projects, such as the storage location of deployment manifests and the target environment to deploy to, etc.

## Core Components

Nautes consists of the following components: 

<details>
  <summary><b>👤 Base Operator</b></summary>
Handles the synchronization of product entities and permission entities from the provider to the target service. Learn more. <a href="https://github.com/nautes-labs/base-operator">Learn more</a>。 

</details>

<details>
  <summary><b>🖥️ Cluster Operator</b></summary>

Provides a Controller for reconciling Cluster resource events, managing the key information of the Kubernetes clusters declared by the Cluster resources, enabling other components involved in cluster management to correctly obtain the cluster's keys from the tenant's key management system. <a href="https://github.com/nautes-labs/cluster-operator">Learn more</a>。 

</details>

<details>
  <summary><b>🔗 Argo Operator</b></summary>

 Provides a set of Controllers for reconciling Cluster resource events and CodeRepo resource events, synchronizing the Kubernetes clusters declared by the Cluster resources and the code repositories declared by the CodeRepo resources to the ArgoCD in the same cluster, enabling Applications in ArgoCD using these Kubernetes clusters and code repositories to work properly. <a href="https://github.com/nautes-labs/argo-operator">Learn more</a>。 

</details>

<details>
  <summary><b>⚙️ Runtime Operator</b></summary>

 Provides a set of Controllers for reconciling Project Pipeline Runtime resource events and Deployment Runtime resource events, synchronizing the basic environment required for pipeline execution or application deployment on the target cluster according to the declaration information of the two types of runtime resources. <a href="https://github.com/nautes-labs/runtime-operator">Learn more</a>。 

</details>

<details>
  <summary><b>🤖 Installer</b></summary>

 Provides a one-click deployment feature, supporting automated installation of infrastructure, resource components, management components, and component initialization. <a href="https://github.com/nautes-labs/installer">Learn more</a>。 

</details>

<details>
  <summary><b>🌐 API Server</b></summary>

 Nautes follows the best practices of GitOps, with user application environment and Nautes' own environment configuration declarations stored in version repositories. Declaration data is divided into two categories: key data is stored in Vault, while other data is stored in Git repositories(currently only supports GitLab) . The API Server project provides a set of REST APIs for operating these configuration declarations. <a href="https://github.com/nautes-labs/api-server">Learn more</a>。 

</details>

<details>
  <summary><b>➡️ CLI</b></summary>

The REST API of the API Server is encapsulated to provide a simple command-line tool, which simplifies user operations with the API.  <a href="https://github.com/nautes-labs/cli">Learn more</a>。 

</details>


## Installation

Nautes supports installation on public cloud, private cloud, host, and Kubernetes clusters. You can learn how to install Nautes with one-click on Alibaba Cloud [here](https://nautes.io/guide/user-guide/installation.html).

## Quick start

We offer a brief guide that can help you quickly deploy your first [pipeline](https://nautes.io/en/guide/user-guide/run-a-pipeline.html) and [application](https://nautes.io/en/guide/user-guide/deploy-an-application.html) with ease. 

## Roadmap

We provide information about product roadmap and progress through this [project](https://github.com/orgs/nautes-labs/projects/1).