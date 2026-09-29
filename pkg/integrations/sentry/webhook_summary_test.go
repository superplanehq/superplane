package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeWebhook(t *testing.T) {
	body := []byte(`{
		"action":"created",
		"installation":{"uuid":"install-1"},
		"data":{
			"issue":{
				"id":123,
				"shortId":"JS-1",
				"title":"secret stack",
				"project":{"slug":"javascript-react-f"}
			}
		}
	}`)

	summary := SummarizeWebhook("issue", body)

	assert.Equal(t, "issue", summary.Resource)
	assert.Equal(t, "created", summary.Action)
	assert.Equal(t, "install-1", summary.InstallationUUID)
	assert.Equal(t, "javascript-react-f", summary.ProjectSlug)
	assert.Equal(t, "123", summary.IssueID)
	assert.Equal(t, "JS-1", summary.IssueShortID)
	assert.NotContains(t, summary.IssueID+summary.ProjectSlug, "secret stack")
}

func TestHostedIssueWebhookMismatch(t *testing.T) {
	expected := "https://app.superplane.com/api/v1/sentry/app/webhook"

	assert.Empty(t, hostedIssueWebhookMismatch(expected, []string{"issue"}, expected))
	assert.Contains(t, hostedIssueWebhookMismatch(expected, []string{"error"}, expected), "issue event")
	assert.Contains(t, hostedIssueWebhookMismatch("https://wrong.example/hook", []string{"issue"}, expected), expected)
}
