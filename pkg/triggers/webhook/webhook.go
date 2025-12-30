package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const MaxEventSize = 64 * 1024

func init() {
	registry.RegisterTrigger("webhook", &Webhook{})
}

type Webhook struct{}

type Metadata struct {
	URL            string `json:"url"`
	Authentication string `json:"authentication"`
}

type Configuration struct {
	URL            string `json:"url"`
	Authentication string `json:"authentication"`
}

func (w *Webhook) Name() string {
	return "webhook"
}

func (w *Webhook) Label() string {
	return "Webhook"
}

func (w *Webhook) Description() string {
	return "Start a new execution chain when a webhook is called"
}

func (w *Webhook) Icon() string {
	return "webhook"
}

func (w *Webhook) Color() string {
	return "black"
}

func (w *Webhook) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "Webhook URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			ReadOnly:    true,
			Description: "This URL will be generated automatically after saving the trigger",
		},
		{
			Name:     "authentication",
			Label:    "Authentication",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "signature",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Signature (HMAC)", Value: "signature"},
						{Label: "Bearer Token", Value: "bearer"},
						{Label: "None (unsafe)", Value: "none"},
					},
				},
			},
		},
	}
}

func (w *Webhook) Setup(ctx core.TriggerContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := Configuration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if metadata.URL != "" && metadata.Authentication == config.Authentication {

		return nil
	}

	var webhookURL string
	if metadata.URL == "" {
		if ctx.WebhookContext == nil {
			return fmt.Errorf("webhook context is required")
		}

		webhookID, err := ctx.WebhookContext.Setup(nil)
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}

		baseURL := getWebhooksBaseURL()
		webhookURL = fmt.Sprintf("%s/webhooks/%s",
			strings.TrimSuffix(baseURL, "/"),
			webhookID.String())

		metadata.URL = webhookURL
		config.URL = webhookURL
	} else {
		webhookURL = metadata.URL
		config.URL = webhookURL
	}

	metadata.Authentication = config.Authentication

	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return updateTriggerConfiguration(ctx, config)
}

func updateTriggerConfiguration(ctx core.TriggerContext, config Configuration) error {
	nodeMetadataCtx, ok := ctx.MetadataContext.(*contexts.NodeMetadataContext)
	if !ok {
		return nil
	}

	configMap := map[string]any{
		"url":            config.URL,
		"authentication": config.Authentication,
	}

	return nodeMetadataCtx.UpdateConfiguration(configMap)
}

func (w *Webhook) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "resetAuthentication",
			Description:    "Reset/regenerate authentication key",
			UserAccessible: true,
			Parameters:     []configuration.Field{},
		},
	}
}

func (w *Webhook) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "resetAuthentication":
		return w.resetAuthentication(ctx)
	}
	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (w *Webhook) resetAuthentication(ctx core.TriggerActionContext) (map[string]any, error) {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	var config Configuration
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	plainKey, err := []byte{}, nil
	switch metadata.Authentication {
	case "signature":
		plainKey, _, err = ctx.WebhookContext.ResetSecret()
	case "bearer":
		plainKey, _, err = ctx.WebhookContext.ResetSecret()
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", metadata.Authentication)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to reset authentication: %w", err)
	}

	result := map[string]any{
		"secret": string(plainKey),
	}

	return result, nil
}

func (w *Webhook) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if len(ctx.Body) > MaxEventSize {
		return http.StatusRequestEntityTooLarge, fmt.Errorf("payload too large")
	}

	var metadata Metadata
	err := mapstructure.Decode(ctx.Configuration, &metadata)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to parse configuration: %w", err)
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	switch metadata.Authentication {
	case "signature":
		signature := ctx.Headers.Get("X-Signature-256")
		if signature == "" {
			return http.StatusForbidden, fmt.Errorf("missing signature header")
		}

		signature = strings.TrimPrefix(signature, "sha256=")
		if signature == "" {
			return http.StatusForbidden, fmt.Errorf("invalid signature format")
		}

		if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
			return http.StatusForbidden, fmt.Errorf("invalid signature")
		}
	case "bearer":
		authHeader := ctx.Headers.Get("Authorization")
		if authHeader == "" {
			return http.StatusUnauthorized, fmt.Errorf("missing Authorization header")
		}

		expectedToken := "Bearer " + string(secret)
		if authHeader != expectedToken {
			return http.StatusUnauthorized, fmt.Errorf("invalid Bearer token")
		}

		ctx.Headers.Set("Authorization", "Bearer ********")
	}

	var parsedData any
	err = json.Unmarshal(ctx.Body, &parsedData)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	output := map[string]any{
		"body":    parsedData,
		"headers": ctx.Headers,
	}

	err = ctx.EventContext.Emit("webhook", output)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func getWebhooksBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	basePath := os.Getenv("PUBLIC_API_BASE_PATH")

	return baseURL + basePath
}
