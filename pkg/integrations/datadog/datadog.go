package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Datadog to work with SuperPlane:

1. **Get API Keys**: In Datadog, go to Organization Settings > API Keys to get your API Key
2. **Get Application Key**: Go to Organization Settings > Application Keys to create an Application Key with **Webhooks Write** permission
3. **Select Site**: Choose the Datadog site that matches your account (US1, US3, US5, EU, or AP1)
4. **Enter Credentials**: Provide your API Key, Application Key, and Site in the integration configuration
5. **Add the webhook to a monitor**: In an Error Tracking New Issue monitor, add ` + "`@webhook-superplane`" + ` to the notification message
`

func init() {
	registry.RegisterIntegration("datadog", &Datadog{})
}

type Datadog struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
	AppKey string `json:"appKey"`
	Site   string `json:"site"`
}

func (d *Datadog) Name() string {
	return "datadog"
}

func (d *Datadog) Label() string {
	return "Datadog"
}

func (d *Datadog) Icon() string {
	return "chart-bar"
}

func (d *Datadog) Description() string {
	return "React to Error Tracking alerts and create events in Datadog"
}

func (d *Datadog) Instructions() string {
	return installationInstructions
}

func (d *Datadog) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "site",
			Label:    "Datadog Site",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "datadoghq.com",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US1 (datadoghq.com)", Value: "datadoghq.com"},
						{Label: "US3 (us3.datadoghq.com)", Value: "us3.datadoghq.com"},
						{Label: "US5 (us5.datadoghq.com)", Value: "us5.datadoghq.com"},
						{Label: "EU (datadoghq.eu)", Value: "datadoghq.eu"},
						{Label: "AP1 (ap1.datadoghq.com)", Value: "ap1.datadoghq.com"},
					},
				},
			},
		},
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Datadog API Key for authentication",
		},
		{
			Name:        "appKey",
			Label:       "Application Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Datadog Application Key for authentication and webhook setup",
		},
	}
}

func (d *Datadog) Actions() []core.Action {
	return []core.Action{
		&CreateEvent{},
	}
}

func (d *Datadog) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnErrorTrackingAlert{},
	}
}

func (d *Datadog) Cleanup(ctx core.IntegrationCleanupContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to create datadog client during cleanup: %v", err)
		}
		return nil
	}

	if err := deleteWebhook(client); err != nil && ctx.Logger != nil {
		ctx.Logger.Warnf("failed to delete datadog webhook during cleanup: %v", err)
	}

	return nil
}

func (d *Datadog) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.Site == "" {
		return fmt.Errorf("site is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	if config.AppKey == "" {
		return fmt.Errorf("appKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	if err := reconcileWebhook(ctx, client); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (d *Datadog) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	if ctx.Request.Method != http.MethodPost {
		ctx.Response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := verifyWebhookRequest(ctx.Integration, ctx.Request); err != nil {
		ctx.Logger.Warnf("rejected datadog webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("failed to read datadog webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		ctx.Logger.Errorf("failed to decode datadog webhook body: %v", err)
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	if !shouldDispatchErrorTrackingAlert(payload) {
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	if err := d.dispatchWebhookMessage(ctx, payload); err != nil {
		ctx.Logger.Errorf("failed to dispatch datadog webhook: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func shouldDispatchErrorTrackingAlert(payload map[string]any) bool {
	eventType, _ := payload["event_type"].(string)
	if !strings.EqualFold(eventType, ErrorTrackingAlertEventType) {
		return false
	}

	transition, _ := payload["alert_transition"].(string)
	return strings.EqualFold(strings.TrimSpace(transition), AlertTransitionTriggered)
}

func (d *Datadog) dispatchWebhookMessage(ctx core.HTTPRequestContext, payload map[string]any) error {
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to list datadog subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		if err := subscription.SendMessage(payload); err != nil {
			ctx.Logger.Errorf("failed to send datadog message to subscription: %v", err)
		}
	}

	return nil
}

func (d *Datadog) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (d *Datadog) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *Datadog) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
