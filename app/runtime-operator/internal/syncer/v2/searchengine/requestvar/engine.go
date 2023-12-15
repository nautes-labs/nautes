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

package requestvar

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperjumptech/grule-rule-engine/ast"
	"github.com/hyperjumptech/grule-rule-engine/builder"
	"github.com/hyperjumptech/grule-rule-engine/engine"
	"github.com/hyperjumptech/grule-rule-engine/pkg"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/pkg/nautesconst"
)

const RuleFilePath = "./rules/event_source_path.grl"

const (
	knowledgeBaseNameEventSourcePath = "eventSource"
)

const (
	version = "0.1.0"
)

type RequestVarSearchEngine struct {
	engine        engine.GruleEngine
	knowledgeBase map[string]*ast.KnowledgeBase
}

func NewSearchEngine(_ *component.ComponentInitInfo) (component.EventSourceSearchEngine, error) {
	ruleFilePath := filepath.Join(os.Getenv(nautesconst.EnvNautesHome), RuleFilePath)

	knowledgeLibrary := ast.NewKnowledgeLibrary()
	ruleBuilder := builder.NewRuleBuilder(knowledgeLibrary)
	fileRes := pkg.NewFileResource(ruleFilePath)
	if err := ruleBuilder.BuildRuleFromResource(knowledgeBaseNameEventSourcePath, version, fileRes); err != nil {
		return nil, err
	}

	knowledgeBase := knowledgeLibrary.GetKnowledgeBase(knowledgeBaseNameEventSourcePath, version)
	gruleEngine := engine.NewGruleEngine()
	gruleEngine.ReturnErrOnFailedRuleEvaluation = true
	return &RequestVarSearchEngine{
		engine: *gruleEngine,
		knowledgeBase: map[string]*ast.KnowledgeBase{
			knowledgeBaseNameEventSourcePath: knowledgeBase,
		},
	}, nil
}

type EngineRequest struct {
	Filter component.RequestDataConditions
	Result string
}

func (rvse *RequestVarSearchEngine) GetTargetPathInEventSource(conditions component.RequestDataConditions) (string, error) {
	req := &EngineRequest{
		Filter: conditions,
		Result: "",
	}

	dataCtx := ast.NewDataContext()
	err := dataCtx.Add("req", req)
	if err != nil {
		return "", err
	}

	if err := rvse.engine.Execute(dataCtx, rvse.knowledgeBase[knowledgeBaseNameEventSourcePath]); err != nil {
		return "", err
	}
	if req.Result == "" {
		return "", fmt.Errorf("unable to search for the corresponding rule, event type, %s, event source type, %s, event listener type, %s, request var, %s",
			conditions.EventType,
			conditions.EventSourceType,
			conditions.EventListenerType,
			conditions.RequestVar)
	}

	return req.Result, nil
}
