package rootly

// WebhookPayload represents the Rootly webhook payload
type WebhookPayload struct {
	Event WebhookEvent   `json:"event"`
	Data  map[string]any `json:"data"`
}

// WebhookEvent represents the event metadata in a Rootly webhook
type WebhookEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	IssuedAt string `json:"issued_at"`
}

// buildPayload creates a payload map from a webhook payload.
// This is shared between OnEvent and OnIncident triggers.
func buildPayload(webhook WebhookPayload) map[string]any {
	payload := map[string]any{
		"event":     webhook.Event.Type,
		"event_id":  webhook.Event.ID,
		"issued_at": webhook.Event.IssuedAt,
	}

	if webhook.Data != nil {
		payload["incident"] = webhook.Data
	}

	return payload
}
