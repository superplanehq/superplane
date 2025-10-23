package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/triggers"
)

const MaxEventSize = 64 * 1024

type Webhook struct{}

func (h *Webhook) Name() string {
	return "webhook"
}

func (h *Webhook) Label() string {
	return "Webhook"
}

func (h *Webhook) Description() string {
	return "Start a new execution chain with a webhook"
}

func (h *Webhook) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (h *Webhook) Setup(ctx triggers.TriggerContext) error {
	return ctx.WebhookContext.Setup(nil)
}

func (h *Webhook) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "resetSecret",
			UserAccessible: true,
		},
	}
}

func (h *Webhook) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "resetSecret":
		return fmt.Errorf("TODO")
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

// TODO: not sure how to surface the errors to the HTTP server
func (h *Webhook) HandleWebhook(ctx triggers.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	err = ctx.EventContext.Emit(data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
