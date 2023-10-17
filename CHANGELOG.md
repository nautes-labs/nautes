# Change Log

## v0.4.1

> Change log since v0.4.0

### Upgrade Notice

> No, really, you must read this before you upgrade.

### Fixed
1. Fixed to delete the dex config that was old when the cluster updated some attributes.

### Changes
1. Codes of Nautes had been verified by GolangCI-lint.
2. The config folder which is in base-operator and argo-operator and cluster-operator had been deleted.
3. The coverage of unit tests was increased not lower than 80%.
4. Null and {} of cluster resource was deleted.
5. Created and rewrote the CreateOrUpdate function with controller-runtime which is used native reflect.DeepEqual function when creating an application for argocd.
6. Renamed the attribute of additions of the multi-tenant such as ProductResourcePathPipeline, ProductResourceRevision, and SyncResourceTypes.
```yaml
  componentsList:
     multiTenant:
        name: hnc
        namespace: hnc-system
        additions:
           productResourceKustomizeFileFolder: templates/pipelines
           productResourceRevision: main
           syncResourceTypes: tekton.dev/Pipeline
```

### New Feature
1. By default, the runtime creates an account with the name of the runtime. You can also specify an account or not.
   What does mean the account for runtime which is the deployment runtime, or project pipeline runtime? It's a ServiceAccount for Kubernetes and a Role for Vault.
- DeploymentRuntime
```yaml
apiVersion: nautes.resource.nautes.io/v1alpha1
kind: DeploymentRuntime
spec:
  name: dr-demo
  account: dr-demo-account
```
- ProjectPipelineRuntime
```yaml
apiVersion: nautes.resource.nautes.io/v1alpha1
kind: ProjectPipelineRuntime
spec:
  name: pr-demo
  account: pr-demo-account
```

## v0.4.0

> Change log since v0.3.9

### Upgrade Notice

> No, really, you must read this before you upgrade.

### Changes
1. Deleted the argocdHost tektonHost and traefik attributes when adding a cluster.
2. Classified by cluster component type, more customized components are provided for clusters when creating.
3. Refactored the RuntimeOperator which uses the interface and implementation of components, enhances expandability and maintainability.
   Now there would have been more implementation for every component. such as Gateway may use APISIX or NginxIngress as an implementer.

### New Feature
1. Supported custom components for cluster resource. It would be best to use the componentsList attribute when adding a cluster. 
The componentsList includes three properties which are name and namespace and additions which are additional properties. The Key is the component attribute, the Value is value of the component attribute.
eg: If the traefik as gateway, it can be set attributes of traefik by the additions attribute.
```yaml
  componentsList:
    gateway:
      name: traefik
      namespace: traefik
      additions:
        httpNodePort: "30080"
        httpsNodePort: "30443"
```

eg: At least be used multiTenant component and gateway of the cluster when adding a pipeline runtime cluster.
```yaml
  componentsList:
    multiTenant:
      name: hnc
      namespace: hnc-system
      additions:
        ProductResourcePathPipeline: templates/pipelines
        ProductResourceRevision: main
        SyncResourceTypes: tekton.dev/Pipeline
    gateway:
      name: traefik
      namespace: traefik
      additions:
        httpNodePort: "30080"
        httpsNodePort: "30443"
```

## v0.3.9

> Change log since v0.3.8

### Upgrade Notice

> No, really, you must read this before you upgrade.

### Changes
1. Supported custom namespace for ProjectPipelineRuntime and changed the Destination attribute of ProjectPipelineRuntime resource to an object the result includes Environment and Namespace. The type of Namespace is a string.
2. Optimized CodeRepo and Project list functions，refactored CodeRepoBinding authorized feature.

### New Feature
1.  Added optional additional resources for ProjectPipelineRuntime resource.

## v0.3.8

### Changes
1. Merge api server, argo operator, base operator, cluster operator, runtime operator into one repo.
