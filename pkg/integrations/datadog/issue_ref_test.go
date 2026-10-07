package datadog

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIssueIDFromURL(t *testing.T) {
	issueID := "11111111-1111-4111-8111-111111111111"

	found, ok := IssueIDFromURL("https://app.datadoghq.eu/error-tracking/issue/" + issueID + "?from_ts=1")
	assert.True(t, ok)
	assert.Equal(t, issueID, found)

	_, ok = IssueIDFromURL("https://app.datadoghq.com/monitors/98765")
	assert.False(t, ok)

	_, ok = IssueIDFromURL("")
	assert.False(t, ok)
}

func TestIssueIDFromEventData(t *testing.T) {
	issueID := "11111111-1111-4111-8111-111111111111"
	found, ok := IssueIDFromEventData(map[string]any{
		"type": ErrorTrackingAlertPayloadType,
		"data": map[string]any{
			"link": "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
		},
	})
	assert.True(t, ok)
	assert.Equal(t, issueID, found)

	_, ok = IssueIDFromEventData(map[string]any{
		"type": ErrorTrackingAlertPayloadType,
		"data": map[string]any{
			"link": "https://app.datadoghq.com/monitors/98765",
		},
	})
	assert.False(t, ok)

	found, ok = IssueIDFromEventData(map[string]any{
		"type": ErrorTrackingAlertPayloadType,
		"data": map[string]any{
			"link":        "https://app.datadoghq.com/monitors/98765",
			"description": "See [the issue](https://app.datadoghq.eu/error-tracking/issue/" + issueID + ").",
		},
	})
	assert.True(t, ok)
	assert.Equal(t, issueID, found)

	_, ok = IssueIDFromEventData(map[string]any{
		"type": "sentry.issue",
		"data": map[string]any{
			"link": "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
		},
	})
	assert.False(t, ok)
}

func TestReceiptIDFromEventData(t *testing.T) {
	receiptID := uuid.New()
	found, ok := ReceiptIDFromEventData(map[string]any{
		"type": ErrorTrackingAlertPayloadType,
		"data": map[string]any{
			ReceiptField: receiptID.String(),
		},
	})
	assert.True(t, ok)
	assert.Equal(t, receiptID, found)

	_, ok = ReceiptIDFromEventData(map[string]any{
		"type": "sentry.issue",
		"data": map[string]any{
			ReceiptField: receiptID.String(),
		},
	})
	assert.False(t, ok)
}
