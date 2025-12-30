package webhook

import (
	"crypto/rand"
	"encoding/hex"
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
	SignatureKey   string `json:"signatureKey,omitempty"`
	Authentication string `json:"authentication"`
	HeaderKeyName  string `json:"headerKeyName,omitempty"`
	HeaderKeyValue string `json:"headerKeyValue,omitempty"`
}

type Configuration struct {
	URL            string `json:"url"`
	Authentication string `json:"authentication"`
	SignatureKey   string `json:"signatureKey,omitempty"`
	HeaderKeyName  string `json:"headerKeyName,omitempty"`
	HeaderKeyValue string `json:"headerKeyValue,omitempty"`
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
	return "blue"
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
			Default:  "none",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "none"},
						{Label: "Signature (HMAC)", Value: "signature"},
						{Label: "Header Key", Value: "headerkey"},
					},
				},
			},
		},
		{
			Name:        "signatureKey",
			Label:       "Signature Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			ReadOnly:    true,
			Sensitive:   true,
			Description: "HMAC signature key for webhook verification",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authentication",
					Values: []string{"signature"},
				},
			},
		},
		{
			Name:        "headerKeyName",
			Label:       "Header Key Header Name",
			Type:        configuration.FieldTypeString,
			Default:     "X-API-Key",
			Description: "Name of the header containing the Header Key",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authentication",
					Values: []string{"headerkey"},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "authentication",
					Values: []string{"headerkey"},
				},
			},
		},
		{
			Name:        "headerKeyValue",
			Label:       "Header Key Value",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "The Header Key value to verify against",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "authentication",
					Values: []string{"headerkey"},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "authentication",
					Values: []string{"headerkey"},
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

	configHeaderKeyName := config.HeaderKeyName
	configHeaderKeyValue := config.HeaderKeyValue

	metadata.SignatureKey = ""
	metadata.HeaderKeyName = ""
	metadata.HeaderKeyValue = ""
	config.SignatureKey = ""
	config.HeaderKeyName = ""
	config.HeaderKeyValue = ""

	switch config.Authentication {
	case "signature":
		key, err := generateSignatureKey()
		if err != nil {
			return fmt.Errorf("failed to generate signature key: %w", err)
		}
		metadata.SignatureKey = key
		config.SignatureKey = key
	case "headerkey":
		metadata.HeaderKeyName = configHeaderKeyName
		metadata.HeaderKeyValue = configHeaderKeyValue
		config.HeaderKeyName = configHeaderKeyName
		config.HeaderKeyValue = configHeaderKeyValue
	}

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
		"signatureKey":   config.SignatureKey,
		"headerKeyName":  config.HeaderKeyName,
		"headerKeyValue": config.HeaderKeyValue,
	}

	return nodeMetadataCtx.UpdateConfiguration(configMap)
}

func (w *Webhook) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "resetAuth",
			Description:    "Reset/regenerate authentication credentials",
			UserAccessible: true,
			Parameters:     []configuration.Field{},
		},
	}
}

func (w *Webhook) HandleAction(ctx core.TriggerActionContext) error {
	switch ctx.Name {
	case "resetAuth":
		return w.resetAuth(ctx)
	}
	return fmt.Errorf("action %s not supported", ctx.Name)
}

func (w *Webhook) resetAuth(ctx core.TriggerActionContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	var config Configuration
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	preserveHeaderKeyName := metadata.HeaderKeyName
	if metadata.Authentication == "headerkey" && preserveHeaderKeyName == "" {
		preserveHeaderKeyName = config.HeaderKeyName
	}

	metadata.SignatureKey = ""
	metadata.HeaderKeyName = ""
	metadata.HeaderKeyValue = ""
	config.SignatureKey = ""
	config.HeaderKeyName = ""
	config.HeaderKeyValue = ""

	switch metadata.Authentication {
	case "signature":
		key, err := generateSignatureKey()
		if err != nil {
			return fmt.Errorf("failed to generate signature key: %w", err)
		}
		metadata.SignatureKey = key
		config.SignatureKey = key
	default:
		return fmt.Errorf("unsupported authentication method: %s", metadata.Authentication)
	}

	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	triggerCtx := core.TriggerContext{
		Logger:                 ctx.Logger,
		Configuration:          ctx.Configuration,
		MetadataContext:        ctx.MetadataContext,
		RequestContext:         ctx.RequestContext,
		EventContext:           ctx.EventContext,
		WebhookContext:         ctx.WebhookContext,
		IntegrationContext:     nil,
		AppInstallationContext: ctx.AppInstallationContext,
	}

	return updateTriggerConfiguration(triggerCtx, config)
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

		if err := crypto.VerifySignature([]byte(metadata.SignatureKey), ctx.Body, signature); err != nil {
			return http.StatusForbidden, fmt.Errorf("invalid signature")
		}
	case "headerkey":
		headerKeyHeader := ctx.Headers.Get(metadata.HeaderKeyName)
		if headerKeyHeader == "" {
			return http.StatusUnauthorized, fmt.Errorf("missing Header Key header: %s", metadata.HeaderKeyName)
		}

		if headerKeyHeader != metadata.HeaderKeyValue {
			return http.StatusUnauthorized, fmt.Errorf("invalid Header Key")
		}
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

func generateSignatureKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func getWebhooksBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	basePath := os.Getenv("PUBLIC_API_BASE_PATH")

	return baseURL + basePath
}
