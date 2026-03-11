package flyio

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	onAppStateChangePayloadType = "flyio.app.stateChange"
)

type OnAppStateChange struct{}

type OnAppStateChangeConfiguration struct {
	App string `json:"app" mapstructure:"app"`
}

func (t *OnAppStateChange) Name() string {
	return "flyio.onAppStateChange"
}

func (t *OnAppStateChange) Label() string {
	return "On App State Change"
}

func (t *OnAppStateChange) Description() string {
	return "Triggers when a Fly.io application's status changes"
}

func (t *OnAppStateChange) Documentation() string {
	return `The On App State Change trigger fires when a Fly.io application's status changes (e.g., deployed, suspended, dead).

## Use Cases

- **Deployment notifications**: React when an app finishes deploying
- **Health monitoring**: Trigger alerts or remediation when an app goes down
- **Lifecycle automation**: Start downstream workflows when an app's state transitions

## How It Works

This trigger listens for incoming webhook events about Fly.io app state changes. When a matching event is received, it emits the app details as trigger output.

## Configuration

- **App**: The Fly.io application to monitor for state changes

## Event Data

Each event includes:
- **appName**: Name of the application
- **appId**: Application ID
- **status**: Current application status
- **event**: The type of state change event`
}

func (t *OnAppStateChange) Icon() string {
	return "refresh-cw"
}

func (t *OnAppStateChange) Color() string {
	return "purple"
}

func (t *OnAppStateChange) ExampleData() map[string]any {
	return map[string]any{
		"appName": "my-fly-app",
		"appId":   "app-123",
		"status":  "deployed",
		"event":   "app.state_change",
	}
}

func (t *OnAppStateChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "app",
			Label:    "App",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "app",
					UseNameAsValue: true,
				},
			},
			Description: "Fly.io application to monitor for state changes",
		},
	}
}

func (t *OnAppStateChange) Setup(ctx core.TriggerContext) error {
	config := OnAppStateChangeConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.App == "" {
		return fmt.Errorf("app is required")
	}

	_, err := ctx.Webhook.Setup()
	return err
}

func (t *OnAppStateChange) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnAppStateChangeConfiguration{}
	if ctx.Configuration != nil {
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
		}
	}

	// Parse the incoming webhook payload
	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Extract app name from the payload and filter
	appName, _ := payload["appName"].(string)
	if config.App != "" && appName != config.App {
		// Event is for a different app — ignore
		return http.StatusOK, nil
	}

	err := ctx.Events.Emit(onAppStateChangePayloadType, payload)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnAppStateChange) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAppStateChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAppStateChange) Cleanup(ctx core.TriggerContext) error {
	return nil
}
