# 如何添加一个自定义的中间件

## 运行时是如何转换成中间件的

运行时转换成具体的中间件大概有以下几个步骤
1. 把运行时中的中间件变成一组部署任务。
2. 把 中间件 根据 服务提供方 和 服务实现方式 转换成一组中间件资源（*）。
// 调用方法和调用工具不等价， 通讯工具用词调整
3. 把 中间件资源 根据 服务提供方和通讯工具 转换成 通讯工具 能识别的格式。
4. 使用 通讯工具 通知 服务提供方 创建中间件和相关资源。
5. 返回中间件的状态信息。
6. 把所有中间件的状态信息合并成运行时的部署结果并更新运行时状态。

> 1. 中间件源是为了分离服务提供方和通讯工具而定义的一类资源，服务提供方和中间件对应了一组具体资源。

## 添加一个自定义中间件需要完成的步骤

在不修改代码的情况下，需要完成以下内容
1. 定义中间件的类型以及可选的字段。
2. 定义中间件对应的中间件资源。
3. 编写中间件和中间件资源的转换关系。
4. 编写转换关系，中间件资源如何转换成调用工具能识别的格式，如何解析调用工具返回的数据格式。

修改代码的情况下，需要完成以下内容
1. 定义中间件的类型，并且在 Middleware 结构体中添加新的字段。
2. 定义新的中间件资源类型。
3. 定义中间件与中间件资源的转换关系。
4. 编写转换关系，中间件资源如何转换成调用工具能识别的格式，如何解析调用工具返回的数据格式。

// 基础说明和比较
两种方式的比较
|项目|不修改代码|修改代码|
|--|--|--|
|添加难度|可以随时添加删除|需要重新编译项目，修改资源定义|
|唯一性|通过通用数据类型实现，类型没有强制约束力，可能会出现类型冲突。|有明确的类型定义，可以明显区别于其他数据类型。|
|复杂属性的支持|不支持复杂的数据结构，需要把复杂的属性转换成字符串字典|支持复杂的数据结构|
|可维护性|属性字段不直观，中间件有哪些字段不会体现在项目中的任何地方，只存在于设计者的脑中|有明确的可选字段|



### 定义中间件类型并使用（不改代码）
定义一个类型名字
定义中间件可选的属性，格式为 map[string]string。

使用时按以下格式填写中间件部分。
```yaml
name: 中间件的名字
type: 自定义的中间件类型名字。
commonMiddlewareInfo:
  key1: value
  key2: value
  key3: value
```

### 定义中间件资源类型

非必要操作，但是可以帮助你更好的完成转换关系的定义。
大多数情况下，中间件资源应该是被定义好了的，优先复用已有的资源定义。

```yaml
name:
# 资源的类型名字
type:
dependencies:
- name:
  type:
# 资源的属性字段
spec:
  key1: value
  key2: value
# 资源的状态字段
status:
  key1: value
  key2: value
```

### 编写中间件与中间件资源的转换关系

转换关系为一份yaml文件，里面包含了完成一个中间件部署需要多少中间件资源。
不同的 服务提供方 + 中间实现方式是一个独立的转换定义
不同的资源通过"---"进行分割。
该文件的格式是 golang 的 template包的模板格式。

```yaml
name: {{ .Name }}-redis
type: redis
labels:
  app: {{ .Name }}
spec:
  deployType: cluster
---
name: {{ .Name }}-service
type: service
dependencies:
- name: {{ .Name }}-redis
  type: redis
spec:
  selectorKey: app
  selectorValue: {{ .Name }}
```

定义好的文件放在 runtime-operator 所在环境中 /opt/nautes/middleware-transform-rules/{{ 服务提供方名字 }}/{{ 中间件名字 }}/{{ 实现方式名 }}.yaml 的路径中, 即可生效。

### 编写中间件资源转换成调用工具能识别的格式的方法

转换关系是一个资源对应一份yaml文件，里面包含了这个资源增删改查的翻译关系。
一份转换关系对应的是一个服务提供方 + 调用工具 + 中间件资源。
该文件的个别字段是 golang 的 template包的模板格式。

以上面的 redis 对 http 为例
```yaml
Create:
  # 这一部分根据调用工具不同格式会不一样。具体需要看调用工具的定义。
  Generation:
    Request: POST
    # 支持写模板
    Body: '{"deployType": {{ .Spec.DeployType }} }'
    # 支持写模板
    URI: '/create'
    # 每个字段支持模板, 如果渲染出来为空，则不会有这个头部信息。
    # http 会根据认证信息自动补充常见的头部信息，具体看调用工具 http 的实现。
    Header:
      CustomHeader: ""
    # 逻辑同Header，如果渲染结果为空，则不会有这个查询条件。
    Query:
      CustomQuery: ""
  Parse:
    KeyName: id
    KeyPath: data.id
Get:
  Generation:
    Request: GET
    URI: '/redis/{{ .Name }}'
  Parse:
    KeyName: data
    KeyPath: data
Update:
  Generation:
    Request: PUT
    Body: '{"key": "new_value"}'
    URI: '/redis/{{ .Name }}'
  Parse:
    KeyName: result
    KeyPath: .data.result
Delete:
  Generation:
    Request: DELETE
    URI: '/redis/{{ .Name }}'
```

定义好的文件放在 runtime-operator 所在环境中 /opt/nautes/resource-transform-rules/{{ 服务提供方名字 }}/{{ 调用工具名字 }}/{{ 资源名字 }}.yaml 的路径中, 即可生效。