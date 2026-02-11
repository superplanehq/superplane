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

type OnDeploymentEventConfiguration struct {
	Project  string   `json:"project"  mapstructure:"project"`
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

type OnDeploymentEventMetadata struct {
	Project    *ProjectInfo `json:"project"              mapstructure:"project"`
	WebhookURL string       `json:"webhookUrl,omitempty" mapstructure:"webhookUrl,omitempty"`
}

type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (t *OnDeploymentEvent) Name() string {
	return "railway.onDeploymentEvent"
}

func (t *OnDeploymentEvent) Label() string {
	return "On Deployment Event"
}

func (t *OnDeploymentEvent) Description() string {
	return "Trigger when a Railway deployment status changes"
}

func (t *OnDeploymentEvent) Documentation() string {
	return `The On Deployment Event trigger starts a workflow when Railway sends deployment status webhooks.

## Setup

After configuring this trigger:
1. Copy the webhook URL shown in the trigger settings
2. Go to Railway → Your Project → Settings → Webhooks
3. Add the webhook URL and select "Deploy" events
4. Save the webhook configuration

## Use Cases

- **Deployment notifications**: Notify Slack when deployments succeed or fail
- **Incident creation**: Create tickets when deployments crash
- **Pipeline chaining**: Trigger downstream workflows after successful deployments

## Configuration

- **Project**: Select the Railway project to monitor
- **Event Filter**: Optionally filter by deployment event type (succeeded, failed, crashed, etc.)
  - Leave empty to receive all deployment events

## Event Data

Each deployment event includes:
- ` + "`type`" + `: Event type (e.g., Deployment.succeeded, Deployment.failed)
- ` + "`details.status`" + `: Deployment status
- ` + "`resource.deployment.id`" + `: Deployment ID
- ` + "`resource.service`" + `: Service name and ID
- ` + "`resource.environment`" + `: Environment name and ID
- ` + "`resource.project`" + `: Project information
- ` + "`timestamp`" + `: When the event occurred`
}

func (t *OnDeploymentEvent) Icon() string {
	return "railway"
}

func (t *OnDeploymentEvent) Color() string {
	return "purple"
}

func (t *OnDeploymentEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Railway project to monitor for deployment events",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "statuses",
			Label:       "Event Filter",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for these deployment events. Leave empty to receive all events.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Succeeded", Value: "succeeded"},
						{Label: "Failed", Value: "failed"},
						{Label: "Crashed", Value: "crashed"},
						{Label: "Building", Value: "building"},
						{Label: "Deploying", Value: "deploying"},
						{Label: "Initializing", Value: "initializing"},
						{Label: "Removing", Value: "removing"},
						{Label: "Removed", Value: "removed"},
					},
				},
			},
		},
	}
}

func (t *OnDeploymentEvent) Setup(ctx core.TriggerContext) error {
	var metadata OnDeploymentEventMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	config := OnDeploymentEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	// If already set up with matching project and webhook URL, nothing to do
	if metadata.Project != nil && metadata.Project.ID == config.Project && metadata.WebhookURL != "" {
		return nil
	}

	// Fetch project details to validate it exists
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Setup webhook and get the URL for manual configuration in Railway
	webhookURL := metadata.WebhookURL
	if webhookURL == "" {
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}
	}

	// Store metadata with webhook URL
	if err := ctx.Metadata.Set(OnDeploymentEventMetadata{
		Project: &ProjectInfo{
			ID:   project.ID,
			Name: project.Name,
		},
		WebhookURL: webhookURL,
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

func (t *OnDeploymentEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Note: Railway does NOT provide webhook signatures
	// We cannot verify the request authenticity

	// Parse the webhook payload
	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Check if this is a deployment event (format: Deployment.succeeded, Deployment.failed, etc.)
	eventType, _ := payload["type"].(string)
	if !strings.HasPrefix(eventType, "Deployment.") {
		// Not a deployment event, ignore silently
		return http.StatusOK, nil
	}

	// Extract event action from type (e.g., "succeeded" from "Deployment.succeeded")
	eventAction := extractEventAction(eventType)

	// Load configuration to check status filter
	config := OnDeploymentEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Filter by event action if configured
	if len(config.Statuses) > 0 {
		// Reject events with empty action when filter is active
		if eventAction == "" || !slices.Contains(config.Statuses, eventAction) {
			return http.StatusOK, nil
		}
	}

	// Emit the event
	if err := ctx.Events.Emit("railway.deployment", payload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnDeploymentEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnDeploymentEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnDeploymentEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// extractEventAction extracts the event action from the type field
// e.g., "Deployment.succeeded" -> "succeeded"
func extractEventAction(eventType string) string {
	parts := strings.SplitN(eventType, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
