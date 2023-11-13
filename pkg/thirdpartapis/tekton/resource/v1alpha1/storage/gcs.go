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
	resourcev1alpha1 "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

const (
	gcsSecretVolumeMountPath     = "/var/secret"
	activateServiceAccountScript = `#!/usr/bin/env bash
if [[ "${GOOGLE_APPLICATION_CREDENTIALS}" != "" ]]; then
  echo GOOGLE_APPLICATION_CREDENTIALS is set, activating Service Account...
  gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
fi
`
)

// GCSResource is a GCS endpoint from which to get artifacts which is required
// by a Build/Task for context (e.g. a archive from which to build an image).
type GCSResource struct {
	Name     string                                `json:"name"`
	Type     resourcev1alpha1.PipelineResourceType `json:"type"`
	Location string                                `json:"location"`
	TypeDir  bool                                  `json:"typeDir"`
	// Secret holds a struct to indicate a field name and corresponding secret name to populate it
	Secrets []resourcev1alpha1.SecretParam `json:"secrets"`

	ShellImage  string `json:"-"`
	GsutilImage string `json:"-"`
}
