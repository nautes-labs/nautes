# Runtime Operator

Runtime Operator 项目提供了一组用于调谐 Project Pipeline Runtime 资源和 Deployment Runtime 资源事件的 Controller，调谐内容主要是根据两类运行时资源的声明信息，在目标集群上同步流水线执行或应用部署所需的基础环境。

## 功能简介

### 同步部署运行时

Controller 会根据 Deployment Runtime 资源引用的 Environment 资源找到运行时的目标集群，并在此集群上完成以下操作：

- 在管理命名空间中同步 Deployment Runtime 资源引用的 CodeRepo 资源。
- 根据 Deployment Runtime 资源关联的 Product 资源的名称同步产品命名空间，并在命名空间中同步 Role 为 namespace-admin、Group 为产品名称的 RoleBinding 资源。
- 根据 Deployment Runtime 资源的名称同步运行时命名空间，并在命名空间中同步运行时 SA。
- 在产品命名空间和运行时命名空间之间建立父子关系。
- 根据 Deployment Runtime 资源关联的 Product 资源的名称在 ArgoCD 中同步 AppProject。
- 根据 Deployment Runtime 资源的名称在 ArgoCD 中同步 Application，Application 的源为 Deployment Runtime 资源引用的 CodeRepo 资源、目标为运行时命名空间。
- 根据 Deployment Runtime 资源关联的 Product 资源的产品名称在 ArgoCD 中同步 Group，并为该 Group 授权上述 AppProject 的只读权限和 Application 的管理权限。

### 同步流水线运行时

- Controller 会根据 Project Pipeline Runtime 资源引用的 Environment 资源找到运行时的目标集群，并在此集群上完成以下操作：

  - 创建产品和流水线运行时的命名空间。
  - 根据运行时资源中定义的 Event Sources 创建事件监听器，监听器会接收 GitLab Webhook、Calender 等外部事件源产生的事件并将其转为内部事件。
  - 根据运行时资源中定义的 Pipeline Triggers 创建流水线触发器，触发器会根据事件监听器发出的内部事件触发指定的流水线。
  - 创建流水线模板同步程序，同步程序会将 default.project 项目指定路径下的流水线模板同步至集群，供产品中各个流水线使用。
  - 授权该运行时所属产品下的用户管理流水线实例的权限。

### 添加自定义步骤

用户可以根据自己业务需要在流水线执行前后添加自定义的步骤。该功能通过插件实现，默认不支持任何自定义步骤。
使用该功能需要按照以下步骤操作：
1. 往 runtime-operator 中添加插件。（具体方式请参考[这里](https://github.com/nautes-labs/plugin-sample)）
2. 在 projectPipelineRuntime资源中添加自定义步骤。
```yaml
# 以下是演示配置，假定已经在 runtime operator 中注册了插件，可以提供 ls 的功能, 它会 ls 镜像中指定的路径。
spec:
  hooks:                    # 需要在流水线前后添加的步骤。
    preHooks:               # 在用户流水线执行前执行的步骤。
      - name: ls            # 要执行的步骤，如果有多个，顺序执行。
        vars:               # 该步骤的入参，可选项由插件提供方定义。
          imageName: bash
          printPath: /var
    postHooks:              # 在用户流水线执行前执行的步骤
      - name: ls
        alias: post-log     # 步骤的名字要全局唯一，如果有冲突，可以指定一个别名。
        vars:
          imageName: bash
          printPath: /usr
```