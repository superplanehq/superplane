package sentry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IssueDescription(t *testing.T) {
	fullIssue := map[string]any{
		"id":        "123",
		"shortId":   "IPE-1",
		"title":     "Error #1: This is a test error!",
		"culprit":   "SentryCustomError(frontend/src/util)",
		"level":     "error",
		"status":    "unresolved",
		"substatus": "new",
		"firstSeen": "2022-04-04T18:17:18.320000Z",
		"lastSeen":  "2022-04-04T18:17:18.320000Z",
		"permalink": "https://your-org.sentry.io/issues/123/",
		"web_url":   "https://your-org.sentry.io/issues/123/",
		"count":     "42",
		"project": map[string]any{
			"id":   "456",
			"name": "ipe",
			"slug": "ipe",
		},
		"assignedTo": map[string]any{
			"type": "user",
			"id":   "789",
			"name": "Person",
		},
		"metadata": map[string]any{
			"type":  "Error",
			"value": "This is a test error!",
		},
	}
	fullJSON, err := json.MarshalIndent(fullIssue, "", "  ")
	require.NoError(t, err)

	sparseIssue := map[string]any{
		"title":     "Broken deploy",
		"permalink": "https://your-org.sentry.io/issues/9/",
	}

	tests := []struct {
		name  string
		issue any
		want  string
	}{
		{
			name:  "full payload",
			issue: fullIssue,
			want:  "https://your-org.sentry.io/issues/123/\n\n```json\n" + string(fullJSON) + "\n```",
		},
		{
			name:  "sparse payload",
			issue: sparseIssue,
			want:  "https://your-org.sentry.io/issues/9/\n\n```json\n{\n  \"permalink\": \"https://your-org.sentry.io/issues/9/\",\n  \"title\": \"Broken deploy\"\n}\n```",
		},
		{
			name:  "missing issue",
			issue: nil,
			want:  "",
		},
		{
			name:  "non-map issue",
			issue: "not a map",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := IssueDescription(tt.issue)
			assert.Equal(t, tt.want, body)
			assert.NotContains(t, body, "TODO")
			assert.NotContains(t, body, "placeholder")
		})
	}

	t.Run("full payload keeps every delivered key", func(t *testing.T) {
		body := IssueDescription(fullIssue)
		for _, key := range []string{
			"id", "shortId", "title", "culprit", "level", "status", "substatus",
			"firstSeen", "lastSeen", "permalink", "web_url", "count", "project",
			"assignedTo", "metadata",
		} {
			assert.Contains(t, body, `"`+key+`"`)
		}
	})
}
