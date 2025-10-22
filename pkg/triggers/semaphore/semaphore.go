package semaphore

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

type Semaphore struct{}

func (s *Semaphore) Name() string {
	return "semaphore"
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Description() string {
	return "Start a new execution chain when something happens in your Semaphore project"
}

func (s *Semaphore) OutputChannels() []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (s *Semaphore) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "Semaphore integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "semaphore",
				},
			},
		},
		{
			Name:     "project",
			Label:    "Project",
			Type:     components.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []components.VisibilityCondition{
				{
					Field:  "integration",
					Values: []string{"*"},
				},
			},
			TypeOptions: &components.TypeOptions{
				Resource: &components.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
	}
}

func (s *Semaphore) Start(ctx triggers.TriggerContext) error {
	return ctx.WebhookContext.Setup("handleWebhook")
}

func (s *Semaphore) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "handleWebhook",
			UserAccessible: false,
		},
	}
}

func (s *Semaphore) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "handleWebhook":
		return s.handleWebhook(ctx, ctx.HttpRequestContext.Request, ctx.HttpRequestContext.Response)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

// TODO: not sure how to surface the errors to the HTTP server
func (s *Semaphore) handleWebhook(ctx triggers.TriggerActionContext, r *http.Request, w http.ResponseWriter) error {
	signature := r.Header.Get("X-Semaphore-Signature-256")
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
