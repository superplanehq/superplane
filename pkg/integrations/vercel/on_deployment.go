package vercel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnDeployment struct{}

func (t *OnDeployment) Name() string {
	return "vercel.onDeployment"
}

func (t *OnDeployment) Label() string {
	return "On Deployment"
}

func (t *OnDeployment) Description() string {
	return "Listen to Vercel deployment events for a project, or for all projects"
}

func (t *OnDeployment) Documentation() string {
	return `The On Deployment trigger emits Vercel deployment events.

## Use Cases

- **Deploy notifications**: Notify Slack or PagerDuty when deployments succeed or fail
- **Post-deploy automation**: Run smoke tests after a production deployment
- **Release orchestration**: Start downstream workflows when a build finishes

## Configuration

- **Project**: Optional Vercel project. Leave empty to listen to all projects.
- **Event Types**: Deployment states to listen for. Defaults to ` + "`deployment.succeeded`" + `.

## Webhook Verification

SuperPlane creates the webhook in Vercel automatically and verifies every request using the ` + "`x-vercel-signature`" + ` header.

## Event Data

Emitted data includes ` + "`eventType`" + `, ` + "`deploymentId`" + `, ` + "`url`" + `, ` + "`readyState`" + `, ` + "`target`" + `, and ` + "`projectId`" + `, plus the raw Vercel payload.`
}

func (t *OnDeployment) Icon() string {
	return "rocket"
}

func (t *OnDeployment) Color() string {
	return "gray"
}

func (t *OnDeployment) Configuration() []configuration.Field {
	options := make([]configuration.FieldOption, 0, len(allowedDeploymentEventTypes))
	for _, eventType := range allowedDeploymentEventTypes {
		state := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(eventType, "deployment."), ".", " "))
		options = append(options, configuration.FieldOption{Label: state, Value: eventType})
	}

	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Vercel project to listen to. Leave empty to listen to all projects.",
		},
		{
			Name:     "eventTypes",
			Label:    "Event Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default:  defaultDeploymentEventTypes,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: options,
				},
			},
		},
	}
}

func (t *OnDeployment) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnEventConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Project != "" {
		client, clientErr := NewClient(ctx.HTTP, ctx.Integration)
		if clientErr != nil {
			return clientErr
		}

		if _, projectErr := client.GetProject(config.Project); projectErr != nil {
			return fmt.Errorf("failed to fetch Vercel project %s: %w", config.Project, projectErr)
		}
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{})
}

func (t *OnDeployment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnDeployment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnDeployment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnDeployment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if err := verifyWebhookSignature(ctx); err != nil {
		return http.StatusForbidden, nil, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %w", err)
	}

	eventType := readString(payload["type"])
	if !slices.Contains(allowedDeploymentEventTypes, eventType) {
		return http.StatusOK, nil, nil
	}

	config, err := decodeOnEventConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventPayload := readMap(payload["payload"])
	deployment := readMap(eventPayload["deployment"])
	projectID := readString(deployment["projectId"])
	if projectID == "" {
		projectID = readString(eventPayload["projectId"])
	}

	if config.Project != "" && projectID != config.Project {
		return http.StatusOK, nil, nil
	}

	if !slices.Contains(selectedEventTypes(config), eventType) {
		return http.StatusOK, nil, nil
	}

	data := map[string]any{
		"eventType":    eventType,
		"projectId":    projectID,
		"name":         readString(deployment["name"]),
		"url":          readString(deployment["url"]),
		"readyState":   readString(deployment["readyState"]),
		"target":       readString(eventPayload["target"]),
		"deploymentId": readString(deployment["id"]),
	}
	if data["target"] == "" {
		data["target"] = readString(deployment["target"])
	}
	for key, value := range payload {
		data[key] = value
	}

	if err := ctx.Events.Emit(eventPayloadType(eventType), data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}
