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

package coderepo

import (
	"context"
	"encoding/json"
	"time"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	condition metav1.Condition
)

func setCondition(codeRepoStatus resourcev1alpha1.CodeRepoStatus, message string, conditionStatus metav1.ConditionStatus) resourcev1alpha1.CodeRepoStatus {
	condition = metav1.Condition{Type: CodeRepoConditionType, Message: message, Reason: RegularUpdate, Status: conditionStatus}
	codeRepoStatus.SetConditions([]metav1.Condition{condition}, map[string]bool{condition.Type: true})
	return codeRepoStatus
}

func (r *CodeRepoReconciler) setConditionAndUpdateStatus(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo, message string, conditionStatus metav1.ConditionStatus) error {
	codeRepoStatus := setCondition(codeRepo.Status, message, conditionStatus)
	codeRepo.Status = codeRepoStatus
	err := r.updateStatus(ctx, codeRepo)
	if err != nil {
		return err
	}
	return nil
}

func (r *CodeRepoReconciler) updateStatus(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo) error {
	err := r.Status().Update(ctx, codeRepo)
	if err != nil {
		return err
	}

	return nil
}

func (r *CodeRepoReconciler) setSync2ArgoStatus(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo, url, secret_id string) error {
	if codeRepo.Status.Sync2ArgoStatus == nil {
		codeRepo.Status.Sync2ArgoStatus = &resourcev1alpha1.SyncCodeRepo2ArgoStatus{}
	}

	if codeRepo != nil {
		jsonSpec, err := json.Marshal(codeRepo.Spec)
		if err != nil {
			return err
		}
		codeRepo.Status.Sync2ArgoStatus.LastSuccessSpec = string(jsonSpec)
		codeRepo.Status.Sync2ArgoStatus.SecretID = secret_id
		codeRepo.Status.Sync2ArgoStatus.Url = url
		codeRepo.Status.Sync2ArgoStatus.LastSuccessTime = metav1.NewTime(time.Now())
	}

	return nil
}
