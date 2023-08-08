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

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	utilString "github.com/nautes-labs/nautes/app/argo-operator/util/strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addFinalizer(codeRepo *resourcev1alpha1.CodeRepo, finalizerName string) {
	if ok := utilString.ContainsString(codeRepo.Finalizers, finalizerName); !ok {
		codeRepo.Finalizers = append(codeRepo.Finalizers, finalizerName)
	}
}

func deleteFinalizer(codeRepo *resourcev1alpha1.CodeRepo, finalizerName string) {
	if ok := utilString.ContainsString(codeRepo.Finalizers, finalizerName); ok {
		finalizers := utilString.RemoveString(codeRepo.Finalizers, finalizerName)
		codeRepo.Finalizers = finalizers
	}
}

func (r *CodeRepoReconciler) AddFinalizerAndUpdateStatus(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo, finalizerName string) error {
	addFinalizer(codeRepo, finalizerName)
	if err := r.Update(ctx, codeRepo); err != nil {
		if err := r.setConditionAndUpdateStatus(ctx, codeRepo, err.Error(), metav1.ConditionFalse); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *CodeRepoReconciler) DeleteFinalizerAndUpdateStatus(ctx context.Context, codeRepo *resourcev1alpha1.CodeRepo, finalizerName string) error {
	deleteFinalizer(codeRepo, finalizerName)
	if err := r.Update(ctx, codeRepo); err != nil {
		err := r.setConditionAndUpdateStatus(ctx, codeRepo, err.Error(), metav1.ConditionFalse)
		if err != nil {
			return err
		}
	}

	return nil
}
