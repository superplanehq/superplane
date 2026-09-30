package datadog

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ReceiptField is the canvas payload key for the stored receipt ID.
// Work order source payloads omit this key.
const ReceiptField = "superplaneReceiptId"

type webhookReceiptContextKey struct{}

// WebhookReceiptState carries one receipt from the public webhook handler
// into the integration request. The handler fills the outcome.
type WebhookReceiptState struct {
	ID                uuid.UUID
	Outcome           string
	SubscriptionCount int
}

// WebhookSummary is the metadata stored for one Datadog call.
// It does not include the body or the webhook token.
type WebhookSummary struct {
	EventType       string
	AlertTransition string
	AlertID         string
	Service         string
	IssueID         string
}

func WithWebhookReceipt(r *http.Request, state *WebhookReceiptState) *http.Request {
	if r == nil || state == nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), webhookReceiptContextKey{}, state))
}

func WebhookReceiptFromRequest(r *http.Request) *WebhookReceiptState {
	if r == nil {
		return nil
	}
	state, _ := r.Context().Value(webhookReceiptContextKey{}).(*WebhookReceiptState)
	return state
}

func setWebhookReceipt(r *http.Request, outcome string, subscriptions int) {
	state := WebhookReceiptFromRequest(r)
	if state == nil {
		return
	}
	state.Outcome = outcome
	state.SubscriptionCount = subscriptions
}

func stampWebhookReceipt(payload map[string]any, r *http.Request) {
	state := WebhookReceiptFromRequest(r)
	if state == nil || state.ID == uuid.Nil || payload == nil {
		return
	}
	payload[ReceiptField] = state.ID.String()
}

// SummarizeWebhook reads event metadata from a Datadog webhook body.
func SummarizeWebhook(body []byte) WebhookSummary {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return WebhookSummary{}
	}

	link := webhookField(payload, "link")
	issueID, _ := IssueIDFromURL(link)
	return WebhookSummary{
		EventType:       webhookField(payload, "event_type"),
		AlertTransition: webhookField(payload, "alert_transition"),
		AlertID:         webhookField(payload, "alert_id"),
		Service:         webhookService(payload),
		IssueID:         issueID,
	}
}

func webhookService(payload map[string]any) string {
	for _, key := range []string{"tags", "alert_scope", "alert_query"} {
		if service := firstTagValue(webhookField(payload, key), "service"); service != "" {
			return service
		}
	}
	return ""
}

func webhookField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}
