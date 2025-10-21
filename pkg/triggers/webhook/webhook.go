package webhook

import (
	"encoding/json"
	"fmt"
	"io"
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

func (h *Webhook) OutputChannels() []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (h *Webhook) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (h *Webhook) Start(ctx triggers.TriggerContext) error {
	return ctx.WebhookContext.Setup("handleWebhook")
}

func (h *Webhook) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "handleWebhook",
			UserAccessible: false,
		},
	}
}

func (h *Webhook) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "handleWebhook":
		return h.handleWebhook(ctx, ctx.HttpRequestContext.Request, ctx.HttpRequestContext.Response)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

// TODO: not sure how to surface the errors to the HTTP server
func (h *Webhook) handleWebhook(ctx triggers.TriggerActionContext, r *http.Request, w http.ResponseWriter) error {
	signature := r.Header.Get("X-Signature-256")
	if signature == "" {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return nil
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(
				w,
				fmt.Sprintf("Request body is too large - must be up to %d bytes", MaxEventSize),
				http.StatusRequestEntityTooLarge,
			)

			return nil
		}

		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return nil
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil
	}

	if err := crypto.VerifySignature(secret, body, signature); err != nil {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return nil
	}

	data := map[string]any{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	return ctx.EventContext.Emit(data)
}
