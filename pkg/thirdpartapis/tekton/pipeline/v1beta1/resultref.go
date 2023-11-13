/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"regexp"
)

// ResultRef is a type that represents a reference to a task run result
type ResultRef struct {
	PipelineTask string `json:"pipelineTask"`
	Result       string `json:"result"`
	ResultsIndex int    `json:"resultsIndex"`
	Property     string `json:"property"`
}

const (
	resultExpressionFormat = "tasks.<taskName>.results.<resultName>"
	// Result expressions of the form <resultName>.<attribute> will be treated as object results.
	// If a string result name contains a dot, brackets should be used to differentiate it from an object result.
	// https://github.com/tektoncd/community/blob/main/teps/0075-object-param-and-result-types.md#collisions-with-builtin-variable-replacement
	objectResultExpressionFormat = "tasks.<taskName>.results.<objectResultName>.<individualAttribute>"
	// ResultTaskPart Constant used to define the "tasks" part of a pipeline result reference
	ResultTaskPart = "tasks"
	// ResultResultPart Constant used to define the "results" part of a pipeline result reference
	ResultResultPart = "results"
	// TODO(#2462) use one regex across all substitutions
	// variableSubstitutionFormat matches format like $result.resultname, $result.resultname[int] and $result.resultname[*]
	variableSubstitutionFormat = `\$\([_a-zA-Z0-9.-]+(\.[_a-zA-Z0-9.-]+)*(\[([0-9])*\*?\])?\)`
	// arrayIndexing will match all `[int]` and `[*]` for parseExpression
	arrayIndexing = `\[([0-9])*\*?\]`
	// ResultNameFormat Constant used to define the the regex Result.Name should follow
	ResultNameFormat = `^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`
)

var variableSubstitutionRegex = regexp.MustCompile(variableSubstitutionFormat)
var resultNameFormatRegex = regexp.MustCompile(ResultNameFormat)
var arrayIndexingRegex = regexp.MustCompile(arrayIndexing)
