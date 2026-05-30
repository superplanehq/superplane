package railway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnDeploymentEvent struct{}

type OnDeploymentConfiguration struct {
	Project    string   `json:"project" mapstructure:"project"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type OnDeploymentPayload struct {
	Type          string `json:"type"`
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
	ServiceID     string `json:"serviceId"`
	DeploymentID  string `json:"deploymentId"`
	Status        string `json:"status,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

var defaultEventTypes = []string{"Deployment.deployed", "Deployment.failed"}

var eventTypeOptions = []configuration.FieldOption{
	{Label: "Deployed", Value: "Deployment.deployed"},
	{Label: "Failed", Value: "Deployment.failed"},
	{Label: "Crashed", Value: "Deployment.crashed"},
	{Label: "Redeployed", Value: "Deployment.redeployed"},
	{Label: "Building", Value: "Deployment.building"},
}

func (t *OnDeploymentEvent) Name() string {
	return "railway.onDeployment"
}

func (t *OnDeploymentEvent) Label() string {
	return "On Deployment Event"
}

func (t *OnDeploymentEvent) Description() string {
	return "Trigger when a Railway deployment status changes"
}

func (t *OnDeploymentEvent) Documentation() string {
	return `The On Deployment Event trigger fires when a deployment changes status in the specified Railway project.

## Use Cases

- **Slack Notification**: Send notifications when build/deploy fails or succeeds.
- **Auto-verification**: Run integration test workflows on successful deploys.

## Configuration

- **Project**: The Railway project to watch.
- **Event Types**: Deployment statuses to listen for (defaults to Deployed and Failed).`
}

func (t *OnDeploymentEvent) Icon() string {
	return "railway"
}

func (t *OnDeploymentEvent) Color() string {
	return "gray"
}

func (t *OnDeploymentEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Railway project to watch",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "eventTypes",
			Label:       "Event Types",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Default:     defaultEventTypes,
			Description: "Deployment statuses to trigger on",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: eventTypeOptions,
				},
			},
		},
	}
}

func decodeOnDeploymentConfiguration(config any) (OnDeploymentConfiguration, error) {
	spec := OnDeploymentConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return OnDeploymentConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Project = strings.TrimSpace(spec.Project)
	if spec.Project == "" {
		return OnDeploymentConfiguration{}, fmt.Errorf("project is required")
	}

	if len(spec.EventTypes) == 0 {
		spec.EventTypes = defaultEventTypes
	}

	return spec, nil
}

func (t *OnDeploymentEvent) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	// Request webhook for the specific project and event types
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectID:  config.Project,
		EventTypes: config.EventTypes,
	})
}

func (t *OnDeploymentEvent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnDeploymentEvent) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func parseRailwayDeploymentWebhook(body []byte) (OnDeploymentPayload, error) {
	var raw struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Details   struct {
			Status string `json:"status"`
		} `json:"details"`
		Resource struct {
			Project struct {
				ID string `json:"id"`
			} `json:"project"`
			Environment struct {
				ID string `json:"id"`
			} `json:"environment"`
			Service struct {
				ID string `json:"id"`
			} `json:"service"`
			Deployment struct {
				ID string `json:"id"`
			} `json:"deployment"`
		} `json:"resource"`
		ProjectID     string `json:"projectId"`
		EnvironmentID string `json:"environmentId"`
		ServiceID     string `json:"serviceId"`
		DeploymentID  string `json:"deploymentId"`
		Status        string `json:"status"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return OnDeploymentPayload{}, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	if raw.ProjectID != "" {
		return OnDeploymentPayload{
			Type:          raw.Type,
			ProjectID:     raw.ProjectID,
			EnvironmentID: raw.EnvironmentID,
			ServiceID:     raw.ServiceID,
			DeploymentID:  raw.DeploymentID,
			Status:        raw.Status,
			Timestamp:     raw.Timestamp,
		}, nil
	}

	status := raw.Details.Status
	if status == "" {
		status = raw.Status
	}

	return OnDeploymentPayload{
		Type:          raw.Type,
		ProjectID:     raw.Resource.Project.ID,
		EnvironmentID: raw.Resource.Environment.ID,
		ServiceID:     raw.Resource.Service.ID,
		DeploymentID:  raw.Resource.Deployment.ID,
		Status:        status,
		Timestamp:     raw.Timestamp,
	}, nil
}

func (t *OnDeploymentEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config, err := decodeOnDeploymentConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	payload, err := parseRailwayDeploymentWebhook(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	// Filter by project ID
	if payload.ProjectID != config.Project {
		return http.StatusOK, nil, nil
	}

	// Filter by event type
	if !slices.Contains(config.EventTypes, payload.Type) {
		return http.StatusOK, nil, nil
	}

	// Emit event data
	if err := ctx.Events.Emit(t.Name(), []any{payload}); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, nil, nil
}

func (t *OnDeploymentEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnDeploymentEvent) ExampleData() map[string]any {
	return map[string]any{
		"type":          "Deployment.deployed",
		"projectId":     "8db400fa-357e-4646-90f0-c7eb36e88a92",
		"environmentId": "9a1d7a89-2cf4-4446-9b69-4cde850918aa",
		"serviceId":     "2a345678-bcde-4fgh-1234-567812345678",
		"deploymentId":  "ebda9796-09e4-456f-af60-d1a66dee66a0",
		"status":        "SUCCESS",
		"timestamp":     "2026-05-30T19:46:09.816Z",
	}
}
