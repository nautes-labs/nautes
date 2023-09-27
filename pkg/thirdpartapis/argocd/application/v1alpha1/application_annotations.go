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

package v1alpha1

const (
	// AnnotationKeyRefresh is the annotation key which indicates that app needs to be refreshed. Removed by application controller after app is refreshed.
	// Might take values 'normal'/'hard'. Value 'hard' means manifest cache and target cluster state cache should be invalidated before refresh.
	AnnotationKeyRefresh string = "argocd.argoproj.io/refresh"

	// AnnotationKeyManifestGeneratePaths is an annotation that contains a list of semicolon-separated paths in the
	// manifests repository that affects the manifest generation. Paths might be either relative or absolute. The
	// absolute path means an absolute path within the repository and the relative path is relative to the application
	// source path within the repository.
	AnnotationKeyManifestGeneratePaths = "argocd.argoproj.io/manifest-generate-paths"
)
