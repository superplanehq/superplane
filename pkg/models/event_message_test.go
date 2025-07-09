package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateEventMessage(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		headers  string
		expected string
	}{
		{
			name:     "GitHub pull request opened",
			raw:      `{"action":"opened","pull_request":{"title":"Add new feature"},"repository":{"name":"my-repo"}}`,
			headers:  `{"X-Hub-Signature-256":"sha256=abc123"}`,
			expected: "Pull request opened: Add new feature in my-repo",
		},
		{
			name:     "GitHub push event",
			raw:      `{"ref":"refs/heads/main","commits":[{"id":"abc123","message":"Add new feature"}],"repository":{"name":"my-repo"}}`,
			headers:  `{"X-Hub-Signature-256":"sha256=abc123"}`,
			expected: "Add new feature",
		},
		{
			name:     "GitHub push event multiple commits",
			raw:      `{"ref":"refs/heads/main","commits":[{"id":"abc123","message":"Fix bug"},{"id":"def456","message":"Add tests"}],"repository":{"name":"my-repo"}}`,
			headers:  `{"X-Hub-Signature-256":"sha256=abc123"}`,
			expected: "Add tests",
		},
		{
			name:     "Semaphore pipeline passed",
			raw:      `{"pipeline":{"name":"my-pipeline","result":"passed"}}`,
			headers:  `{"X-Semaphore-Signature-256":"sha256=abc123"}`,
			expected: "Pipeline my-pipeline passed",
		},
		{
			name:     "Semaphore pipeline failed",
			raw:      `{"pipeline":{"name":"my-pipeline","result":"failed"}}`,
			headers:  `{"X-Semaphore-Signature-256":"sha256=abc123"}`,
			expected: "Pipeline my-pipeline failed",
		},
		{
			name:     "Semaphore job passed",
			raw:      `{"blocks":[{"jobs":[{"name":"my-job","result":"passed"}]}]}`,
			headers:  `{"X-Semaphore-Signature-256":"sha256=abc123"}`,
			expected: "Job my-job passed",
		},
		{
			name:     "Unknown event",
			raw:      `{"unknown":"data"}`,
			headers:  `{"Content-Type":"application/json"}`,
			expected: "Event received",
		},
		{
			name:     "Invalid JSON",
			raw:      `{invalid json}`,
			headers:  `{"Content-Type":"application/json"}`,
			expected: "Event received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateEventMessage(SourceTypeEventSource, []byte(tt.raw), []byte(tt.headers))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateGitHubEventMessage(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected string
	}{
		{
			name: "Pull request closed",
			payload: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"title": "Fix bug",
				},
				"repository": map[string]interface{}{
					"name": "test-repo",
				},
			},
			expected: "Pull request closed: Fix bug in test-repo",
		},
		{
			name: "Issue opened",
			payload: map[string]interface{}{
				"action": "opened",
				"issue": map[string]interface{}{
					"title": "Bug report",
				},
				"repository": map[string]interface{}{
					"name": "test-repo",
				},
			},
			expected: "Issue opened: Bug report in test-repo",
		},
		{
			name: "Repository with no action",
			payload: map[string]interface{}{
				"repository": map[string]interface{}{
					"name": "test-repo",
				},
			},
			expected: "GitHub event in test-repo",
		},
		{
			name:     "No repository info",
			payload:  map[string]interface{}{"action": "opened"},
			expected: "GitHub event received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGitHubEventMessage(tt.payload)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSemaphoreEventMessage(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected string
	}{
		{
			name: "Pipeline running",
			payload: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"name":   "test-pipeline",
					"result": "running",
				},
			},
			expected: "Pipeline test-pipeline started",
		},
		{
			name: "Pipeline passed with commit message",
			payload: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"name":   "test-pipeline",
					"result": "passed",
				},
				"revision": map[string]interface{}{
					"commit_message": "Add new feature",
				},
			},
			expected: "Pipeline test-pipeline passed: Add new feature",
		},
		{
			name: "Pipeline failed with empty commit message",
			payload: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"name":   "test-pipeline",
					"result": "failed",
				},
				"revision": map[string]interface{}{
					"commit_message": "empty",
				},
			},
			expected: "Pipeline test-pipeline failed",
		},
		{
			name: "Pipeline canceled using state field",
			payload: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"name":  "test-pipeline",
					"state": "canceled",
				},
			},
			expected: "Pipeline test-pipeline canceled",
		},
		{
			name: "Job failed",
			payload: map[string]interface{}{
				"blocks": []interface{}{
					map[string]interface{}{
						"jobs": []interface{}{
							map[string]interface{}{
								"name":   "test-job",
								"result": "failed",
							},
						},
					},
				},
			},
			expected: "Job test-job failed",
		},
		{
			name: "Job passed",
			payload: map[string]interface{}{
				"blocks": []interface{}{
					map[string]interface{}{
						"jobs": []interface{}{
							map[string]interface{}{
								"name":   "test-job",
								"result": "passed",
							},
						},
					},
				},
			},
			expected: "Job test-job passed",
		},
		{
			name: "Pipeline without status",
			payload: map[string]interface{}{
				"pipeline": map[string]interface{}{
					"name": "test-pipeline",
				},
			},
			expected: "Pipeline test-pipeline event",
		},
		{
			name:     "No pipeline or job info",
			payload:  map[string]interface{}{"unknown": "data"},
			expected: "Semaphore event received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSemaphoreEventMessage(tt.payload)
			assert.Equal(t, tt.expected, result)
		})
	}
}