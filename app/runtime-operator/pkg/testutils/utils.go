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

package testutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RandNum() string {
	return fmt.Sprintf("%04d", rand.Intn(999999))
}

func WaitForDelete(client client.Client, obj client.Object) error {
	for i := 0; i < 10; i++ {
		err := client.Delete(context.Background(), obj)
		if apierrors.IsNotFound(err) {
			return nil
		}

		time.Sleep(time.Second)
	}
	return fmt.Errorf("wait for delete %s timeout", obj.GetName())
}

func GenerateNames(format string, len int) []string {
	names := make([]string, len)
	for i := 0; i < len; i++ {
		names[i] = fmt.Sprintf(format, i)
	}
	return names
}

func NamespaceIsNotExist(k8sClient client.Client, obj client.Object) (bool, error) {
	err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	if obj.GetDeletionTimestamp() == nil {
		return false, nil
	}

	return true, nil
}
