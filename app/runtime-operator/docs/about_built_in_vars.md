## 内置变量
本文档主要存放关于流水线内置变量的相关说明

| 名字 | 说明 |
| -- | -- |
| EventSourceCodeRepoName | the name of the code repo that triggered the event.
| EventSourceCodeRepoURL  | the url of the code repo that triggered the event.
| ServiceAccount          | service account that runs hooks.
| Namespace               | The namespace name of the running user pipeline.
| PipelineCodeRepoName    | Name of the code repo where the pipeline file is stored.
| PipelineCodeRepoURL     | URL of the code repo where the pipeline file is stored.
| PipelineRevision        | User-specified pipeline branches.
| PipelineFilePath        | The path of the pipeline file specified by the user.
| ProviderType            | Type of code repository provider.
| CodeRepoProviderURL     | The API server URL of the code repo provider.
| PipelineDashBoardURL    | The url address to view the pipeline running status.
| PipelineLabel           | The tags to be placed on the user pipeline are used for the execution status of the end user pipeline.
