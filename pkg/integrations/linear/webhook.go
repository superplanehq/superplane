package linear

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type LinearWebhookHandler struct{}

type WebhookPayload struct {
	Action string                 `json:"action"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
	URL    string                 `json:"url"`
}

func (h *LinearWebhookHandler) HandleWebhook(ctx core.HTTPRequestContext) {
	if ctx.Request.Method != http.MethodPost {
		http.Error(ctx.Response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read webhook body: %v", err)
		http.Error(ctx.Response, "failed to read body", http.StatusBadRequest)
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to parse webhook payload: %v", err)
		http.Error(ctx.Response, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Store webhook data in metadata for trigger matching
	webhookData := map[string]interface{}{
		"action": payload.Action,
		"type":   payload.Type,
		"data":   payload.Data,
		"url":    payload.URL,
	}

	ctx.Integration.EmitEvent(core.IntegrationEvent{
		Type:     "webhook",
		Metadata: map[string]interface{}{"webhook": webhookData},
	})

	ctx.Response.WriteHeader(http.StatusOK)
	ctx.Response.Write([]byte(`{"success":true}`))
}
