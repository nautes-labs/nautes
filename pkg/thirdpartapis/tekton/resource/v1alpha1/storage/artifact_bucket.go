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
	resource "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

// For some reason gosec thinks this string has enough entropy to be a potential secret.
// The nosec comment disables it for this line.
/* #nosec */
const secretVolumeMountPath = "/var/bucketsecret"

// ArtifactBucket contains the Storage bucket configuration defined in the
// Bucket config map.

type ArtifactBucket struct {
	Name     string
	Location string
	Secrets  []resource.SecretParam

	ShellImage  string
	GsutilImage string
}
