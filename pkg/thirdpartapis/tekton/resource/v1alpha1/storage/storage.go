/*
Copyright 2019-2020 The Tekton Authors

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

package storage

import (
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1beta1"
	resource "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

// PipelineStorageResourceInterface adds a function to the PipelineResourceInterface for retrieving
// secrets that are usually needed for storage PipelineResources.
type PipelineStorageResourceInterface interface {
	v1beta1.PipelineResourceInterface
	GetSecretParams() []resource.SecretParam
}
