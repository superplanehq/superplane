package datadog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorTrackingIssuePayload_MatchesTheAlertTheGraphReads(t *testing.T) {
	payload := ErrorTrackingIssuePayload(ErrorTrackingIssue{
		ID:           "11111111-1111-4111-8111-111111111111",
		ErrorType:    "TimeoutError",
		ErrorMessage: "checkout timed out",
		Service:      "checkout",
		URL:          "https://app.datadoghq.eu/error-tracking/issue/11111111-1111-4111-8111-111111111111",
		Sample:       &ErrorSample{Env: "Prod"},
	})

	assert.Equal(t, "TimeoutError: checkout timed out", payload.Title)
	assert.Equal(t, payload.Body, payload.Description)
	assert.Contains(t, payload.Description, "checkout timed out")
	assert.Contains(t, payload.Description, payload.Link)
	assert.Equal(t, ErrorTrackingAlertEventType, payload.EventType)
	assert.Equal(t, AlertTransitionTriggered, payload.AlertTransition)
	assert.Equal(t, "service:checkout", payload.AlertQuery)
	assert.Equal(t, "service:checkout,env:prod", payload.Tags)
	assert.Equal(t, "prod", payload.Environment)
	assert.True(t, strings.HasPrefix(payload.Link, "https://app.datadoghq.eu/error-tracking/issue/"))
}
