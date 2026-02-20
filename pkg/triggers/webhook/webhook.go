package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	MaxEventSize           = 64 * 1024
	DefaultHeaderTokenName = "X-Webhook-Token"
)

func init() {
	registry.RegisterTrigger("webhook", &Webhook{})
}

type Webhook struct{}

type Metadata struct {
	URL            string `json:"url"`
	Authentication string `json:"authentication"`
}

type Configuration struct {
	Authentication string `json:"authentication"`
	HeaderName     string `json:"headerName" mapstructure:"headerName"`
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

func (w *Webhook) Documentation() string {
	return `The Webhook trigger starts a new workflow execution when an HTTP request is received at the generated webhook URL.

## Use Cases

- **External system integration**: Receive events from third-party services
- **CI/CD pipelines**: Trigger workflows from build systems
- **Form submissions**: Process data from web forms
- **Event notifications**: Receive notifications from external applications

## How It Works

1. When you add a Webhook trigger to a workflow, SuperPlane generates a unique webhook URL
2. Configure the authentication method for the webhook
3. External systems can send HTTP requests to this URL
4. Each request starts a new workflow execution with the request data

## Authentication Methods

- **Signature (HMAC)**: Verify requests using HMAC-SHA256 signature in the ` + "`X-Signature-256`" + ` header
- **Bearer Token**: Require a Bearer token in the ` + "`Authorization`" + ` header
- **Header Token**: Require a raw token in a custom header (default: ` + "`X-Webhook-Token`" + `)
- **None (unsafe)**: No authentication (not recommended for production)

## Request Data

The webhook payload includes:
- **body**: Parsed request body (JSON if possible, otherwise raw data)
- **headers**: All HTTP headers from the request

## Security

- Each webhook has a unique secret key for authentication
- Secrets can be reset using the "Reset Authentication" action
- Maximum payload size: 64KB

## Example Usage

Send a POST request to the webhook URL with your payload. The workflow will receive the data and start execution.`
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
						{Label: "Header Token", Value: "header_token"},
						{Label: "None (unsafe)", Value: "none"},
					},
				},
			},
		},
		{
			Name:        "headerName",
			Label:       "Header Name",
			Type:        configuration.FieldTypeString,
			Default:     DefaultHeaderTokenName,
			Placeholder: DefaultHeaderTokenName,
			Description: "HTTP header that must contain the authentication token",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authentication", Values: []string{"header_token"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "authentication", Values: []string{"header_token"}},
			},
		},
	}
}

func (w *Webhook) Setup(ctx core.TriggerContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
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

	if metadata.URL == "" {
		webhookURL, err := ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}

		metadata.URL = webhookURL
	}

	metadata.Authentication = config.Authentication

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
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
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
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
		plainKey, _, err = ctx.Webhook.ResetSecret()
	case "bearer":
		plainKey, _, err = ctx.Webhook.ResetSecret()
	case "header_token":
		plainKey, _, err = ctx.Webhook.ResetSecret()
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

	var config Configuration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to parse configuration: %w", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	switch config.Authentication {
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
	case "header_token":
		headerName := config.HeaderTokenName()
		headerToken := ctx.Headers.Get(headerName)
		if headerToken == "" {
			return http.StatusUnauthorized, fmt.Errorf("missing %s header", headerName)
		}

		if headerToken != string(secret) {
			return http.StatusUnauthorized, fmt.Errorf("invalid header token")
		}

		ctx.Headers.Set(headerName, "********")
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

	err = ctx.Events.Emit("webhook", output)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (w *Webhook) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (c Configuration) HeaderTokenName() string {
	if c.HeaderName != "" {
		return c.HeaderName
	}

	return DefaultHeaderTokenName
}
