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

package argoevents

import (
	"bytes"
	"text/template"
)

var (
	tmplEventSourceGitlab            = "eventSourceGitlab"
	tmplEventSourceGitlabEventName   = "eventSourceGitlabEventName"
	tmplGitlabAccessToken            = "gitlabAccessToken"
	tmplGitlabSecretToken            = "gitlabSecretToken"
	tmplGitlabEventSourcePath        = "gitlabEventSourcePath"
	tmplGitlabEndPoint               = "gitlabEndPoint"
	tmplGitlabIngressName            = "gitlabIngressName"
	tmplGitlabServiceName            = "gitlabServiceName"
	tmplEventSourceCalendar          = "eventSourceCalendar"
	tmplEventSourceCalendarEventName = "eventSourceCalendarEventName"
	tmplTriggerName                  = "triggerName"
	tmplDependencyName               = "dependencyName"
	tmplSensorName                   = "sensorName"
	tmplVaultEngineGitAcessTokenPath = "vaultEngineGitAcessTokenPath"
	nameAndInitPipelineTemplates     = map[string]string{
		tmplEventSourceGitlab:            "{{ .productName }}-{{ .runtimeName }}-gitlab",
		tmplEventSourceGitlabEventName:   "{{ .runtimeName }}-{{ .eventName }}-{{ .pipelineName }}",
		tmplGitlabAccessToken:            "{{ .productName }}-{{ .runtimeName }}-{{ .repoName }}-accesstoken",
		tmplGitlabSecretToken:            "{{ .productName }}-{{ .runtimeName }}-{{ .repoName }}-secrettoken",
		tmplGitlabEventSourcePath:        "/{{ .clusterName }}-{{ .productName }}-{{ .runtimeName }}-gitlab",
		tmplGitlabEndPoint:               "/{{ .clusterName }}-{{ .productName }}-{{ .runtimeName }}-gitlab/{{ .eventName }}/{{ .pipelineName }}",
		tmplGitlabIngressName:            "{{ .clusterName }}-{{ .productName }}-{{ .runtimeName }}-gitlab",
		tmplGitlabServiceName:            "{{ .clusterName }}-{{ .productName }}-{{ .runtimeName }}-gitlab",
		tmplEventSourceCalendar:          "{{ .productName }}-{{ .runtimeName }}-calendar",
		tmplEventSourceCalendarEventName: "{{ .runtimeName }}-{{ .eventName }}-{{ .pipelineName }}",
		tmplDependencyName:               "{{ .runtimeName }}-{{ .eventSourceType }}-{{ .eventName }}",
		tmplTriggerName:                  "{{ .eventName }}-{{ .pipelineName }}-{{ .eventSourceType }}",
		tmplSensorName:                   "{{ .productName }}-{{ .runtimeName }}",
		tmplVaultEngineGitAcessTokenPath: "{{ .pipelineRepoProviderType }}/{{ .repoName }}/default/accesstoken-api",
	}
)

var (
	keyEventName                = "eventName"
	keyPipelineName             = "pipelineName"
	keyProductName              = "productName"
	keyRuntimeName              = "runtimeName"
	keyRepoName                 = "repoName"
	keyClusterName              = "clusterName"
	keyEventSourceType          = "eventSourceType"
	keyPipelineRepoProviderType = "pipelineRepoProviderType"
	keyPipelineRepoID           = "pipelineRepoID"
	keyPipelineRepoURL          = "pipelineRepoURL"
	keyPipelinePath             = "pipelinePath"
	keyIsCodeRepoTrigger        = "isCodeRepoTrigger"
	keyPipelineLabel            = "pipelineLabel"
	keyServiceAccountName       = "serviceAccountName"
)

type eventType string

var (
	eventTypeGitlab   eventType = "gitlab"
	eventTypeCalendar eventType = "calendar"
)

func getStringFromTemplate(templateName string, vars interface{}) (string, error) {
	tmpl, err := template.New(templateName).Parse(nameAndInitPipelineTemplates[templateName])
	if err != nil {
		return "", err
	}

	var path bytes.Buffer
	err = tmpl.Execute(&path, vars)
	if err != nil {
		return "", err
	}
	return path.String(), nil
}
