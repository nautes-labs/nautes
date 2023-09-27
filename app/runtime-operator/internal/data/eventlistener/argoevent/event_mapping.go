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

package argoevent

var (
	// The mapping relationship between gitlab webhook and the argo event source event
	// GitLab API docs:
	// https://docs.gitlab.com/ce/api/projects.html#list-project-hooks
	// Dest
	// https://github.com/xanzy/go-gitlab/blob/v0.79.1/projects.go
	gitlabWebhookEventToArgoEventMapping = map[string]string{
		"confidential_issues_events": "ConfidentialIssuesEvents",
		// "confidential_note_events":   "", go-gitlab not supported
		// "deployment_events":          "", go-gitlab not supported
		"issues_events":         "IssuesEvents",
		"job_events":            "JobEvents",
		"merge_requests_events": "MergeRequestsEvents",
		"note_events":           "NoteEvents",
		"pipeline_events":       "PipelineEvents",
		"push_events":           "PushEvents",
		// "releases_events":            "", // go-gitlab not supported
		"tag_push_events": "TagPushEvents",
	}
)
