package datadog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeWebhook(t *testing.T) {
	issueID := "11111111-1111-4111-8111-111111111111"
	summary := SummarizeWebhook([]byte(`{
		"event_type":"error_tracking_alert",
		"alert_transition":"Triggered",
		"alert_id":"867",
		"title":"do not store this title",
		"body":"do not store this body",
		"tags":"monitor,service:checkout,env:prod",
		"link":"https://app.datadoghq.com/error-tracking/issue/` + issueID + `"
	}`))

	assert.Equal(t, WebhookSummary{
		EventType:       "error_tracking_alert",
		AlertTransition: "Triggered",
		AlertID:         "867",
		Service:         "checkout",
		IssueID:         issueID,
	}, summary)

	assert.Equal(t, WebhookSummary{}, SummarizeWebhook([]byte(`{"title":`)))
}

func TestStampWebhookReceipt(t *testing.T) {
	receiptID := uuid.New()
	request := WithWebhookReceipt(httptest.NewRequest(http.MethodPost, "/events", nil), &WebhookReceiptState{ID: receiptID})
	payload := map[string]any{"event_type": "error_tracking_alert"}

	stampWebhookReceipt(payload, request)

	require.Equal(t, receiptID.String(), payload[ReceiptField])
}
