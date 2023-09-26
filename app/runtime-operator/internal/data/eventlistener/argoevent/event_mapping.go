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
