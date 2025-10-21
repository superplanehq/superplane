package webhook

import (
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/triggers"
)

const MaxEventSize = 64 * 1024

type Webhook struct{}

func (w *Webhook) Name() string {
	return "webhook"
}

func (w *Webhook) Label() string {
	return "Webhook"
}

func (w *Webhook) Description() string {
	return "Start a new execution chain with a webhook"
}

func (w *Webhook) OutputChannels() []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (w *Webhook) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (h *Webhook) Setup(ctx triggers.SetupContext) error {
	return ctx.WebhookContext.Create()
}

func (h *Webhook) Start(ctx triggers.TriggerContext) error {
	return ctx.WebhookContext.RegisterActionCall("handleWebhook")
}

func (h *Webhook) handleWebhook(ctx triggers.TriggerActionContext, r *http.Request, w http.ResponseWriter) error {
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

	return ctx.EventContext.Emit(body)
}

func (w *Webhook) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "handleWebhook",
			UserAccessible: false,
		},
	}
}

func (w *Webhook) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "handleWebhook":
		return w.handleWebhook(ctx, ctx.WebhookContext.Request, ctx.WebhookContext.Response)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
