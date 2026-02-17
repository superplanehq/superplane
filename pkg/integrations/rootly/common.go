package rootly

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// NodeMetadata contains metadata stored on trigger and component nodes
type NodeMetadata struct {
	Service *Service `json:"service,omitempty"`
}

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

type webhookRequest struct {
	payload WebhookPayload
}

func decodeAndVerifyWebhook[T any](ctx core.WebhookRequestContext, config *T) (*webhookRequest, int, error) {
	if err := mapstructure.Decode(ctx.Configuration, config); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature
	signature := ctx.Headers.Get("X-Rootly-Signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return nil, http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	var payload WebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	return &webhookRequest{payload: payload}, http.StatusOK, nil
}
