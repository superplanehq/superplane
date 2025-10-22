package github

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

type GitHub struct{}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) Label() string {
	return "GitHub"
}

func (g *GitHub) Description() string {
	return "Start a new execution chain when something happens in your GitHub repository"
}

func (g *GitHub) OutputChannels() []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (g *GitHub) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "integration",
			Label:    "GitHub integration",
			Type:     components.FieldTypeIntegration,
			Required: true,
			TypeOptions: &components.TypeOptions{
				Integration: &components.IntegrationTypeOptions{
					Type: "github",
				},
			},
		},
		{
			Name:     "repository",
			Label:    "Repository",
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
					Type: "repository",
				},
			},
		},
	}
}

func (g *GitHub) Start(ctx triggers.TriggerContext) error {
	return ctx.WebhookContext.Setup("handleWebhook")
}

func (g *GitHub) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "handleWebhook",
			UserAccessible: false,
		},
	}
}

func (g *GitHub) HandleAction(ctx triggers.TriggerActionContext) error {
	switch ctx.Name {
	case "handleWebhook":
		return g.handleWebhook(ctx, ctx.HttpRequestContext.Request, ctx.HttpRequestContext.Response)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

// TODO: not sure how to surface the errors to the HTTP server
func (g *GitHub) handleWebhook(ctx triggers.TriggerActionContext, r *http.Request, w http.ResponseWriter) error {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return nil
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "Invalid X-GitHub-Event", http.StatusForbidden)
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
