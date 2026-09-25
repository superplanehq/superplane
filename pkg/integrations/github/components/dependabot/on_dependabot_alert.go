package dependabot

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	// PayloadType is the canvas event type this trigger emits. Intake seeding
	// uses the same type so a seeded alert and a webhook look the same.
	PayloadType = "github.dependabotAlert"

	webhookEventType = "dependabot_alert"
)

type OnAlert struct{}

type OnAlertConfiguration struct {
	Repository string   `json:"repository" mapstructure:"repository"`
	Actions    []string `json:"actions" mapstructure:"actions"`
}

func (a *OnAlert) Name() string {
	return "github.onDependabotAlert"
}

func (a *OnAlert) Label() string {
	return "On Dependabot Alert"
}

func (a *OnAlert) Description() string {
	return "Listen to Dependabot alert events"
}

func (a *OnAlert) Documentation() string {
	return `The On Dependabot Alert trigger starts a workflow execution when GitHub reports a Dependabot alert in a repository.

## Use Cases

- **Security intake**: Create a task when a new vulnerability is reported
- **Reopened alerts**: Start work again when a fixed alert comes back
- **Severity routing**: Filter alerts by severity before they become tasks

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Actions**: Select which alert actions to listen for (created, reopened, reintroduced, and so on)

## Event Data

Each alert event includes:
- **action**: The action that triggered the event (created, reopened, reintroduced, fixed, dismissed)
- **alert**: The Dependabot alert, including the package, manifest, severity, and advisory
- **repository**: Repository information
- **sender**: User or app that triggered the event

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.

The repository must have Dependabot alerts turned on. The GitHub App needs permission to read Dependabot alerts.`
}

func (a *OnAlert) Icon() string {
	return "github"
}

func (a *OnAlert) Color() string {
	return "gray"
}

func (a *OnAlert) ExampleData() map[string]any {
	return map[string]any{
		"type":      PayloadType,
		"timestamp": "2026-09-25T09:00:00Z",
		"data": map[string]any{
			"action": "created",
			"alert": map[string]any{
				"number":   7,
				"html_url": "https://github.com/acme/payments/security/dependabot/7",
				"dependency": map[string]any{
					"package": map[string]any{
						"name":      "lodash",
						"ecosystem": "npm",
					},
					"manifest_path": "package.json",
				},
				"security_advisory": map[string]any{
					"summary":  "Prototype pollution in lodash",
					"severity": "high",
				},
				"security_vulnerability": map[string]any{
					"vulnerable_version_range": "< 4.17.21",
					"first_patched_version": map[string]any{
						"identifier": "4.17.21",
					},
				},
			},
		},
	}
}

func (a *OnAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created", "reopened", "reintroduced"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Reopened", Value: "reopened"},
						{Label: "Reintroduced", Value: "reintroduced"},
						{Label: "Fixed", Value: "fixed"},
						{Label: "Dismissed", Value: "dismissed"},
						{Label: "Auto dismissed", Value: "auto_dismissed"},
					},
				},
			},
		},
	}
}

func (a *OnAlert) Setup(ctx core.TriggerContext) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
	if err != nil {
		return err
	}

	var config OnAlertConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  webhookEventType,
		Repository: config.Repository,
	})
}

func (a *OnAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (a *OnAlert) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (a *OnAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, a.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config := OnAlertConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != webhookEventType {
		ctx.Logger.Infof("Ignoring event - event type %q is not a dependabot alert event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if !common.WhitelistedAction(data, config.Actions) {
		action, ok := common.ExtractAction(data)
		if !ok {
			ctx.Logger.Info("Ignoring event - without a valid action")
			return http.StatusOK, nil, nil
		}

		ctx.Logger.Infof("Ignoring event - action %q is not configured", action)
		return http.StatusOK, nil, nil
	}

	err = ctx.Events.Emit(PayloadType, data)
	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (a *OnAlert) Cleanup(ctx core.TriggerContext) error {
	return nil
}
