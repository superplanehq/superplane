package logfire

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"strings"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("logfire", &Logfire{}, &LogfireWebhookHandler{})
}

type Logfire struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
}

type Metadata struct {
	ExternalOrganizationID string `json:"externalOrganizationId,omitempty"`
	SupportsWebhookSetup   bool   `json:"supportsWebhookSetup"`
	SupportsQueryAPI       bool   `json:"supportsQueryApi"`
}

func (l *Logfire) Name() string {
	return "logfire"
}

func (l *Logfire) Label() string {
	return "Logfire"
}

func (l *Logfire) Icon() string {
	return "flame"
}

func (l *Logfire) Description() string {
	return "Set up Logfire for AI Observability"
}

func (l *Logfire) Instructions() string {
	return `## Create a Logfire API key for SuperPlane

1. Open **Settings** in Logfire.
2. Under **ORG: <your-username>**, select **API Keys**.
3. Click **New API Key**.
4. Enter a key name.
5. Enable these **five** scopes:
   - **Organization scopes**: ` + "`organization:write_channel`" + ` (required for auto-creating webhook channels)
   - **Project scopes**: ` + "`project:read `, " + "`project:read_token `, " + "`project:read_alert `, " + "and " + "`project:write_alert`" + `
6. Select **All Project** or a specific project from the dropdown.
7. Click **Create API Key**.
8. Copy the API key and paste it.`
}

func (l *Logfire) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Logfire API key with write_channel, project:read and read_token scopes",
			Required:    true,
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Description: "Optional override for region or self-hosted Logfire API base URL",
			Placeholder: "https://logfire-us.pydantic.dev",
			Required:    false,
		},
	}
}

func (l *Logfire) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Logfire) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("invalid Logfire API key - please verify your key and try again: %w", err)
	}

	metadata := Metadata{
		SupportsQueryAPI: true,
	}

	decodedMetadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &decodedMetadata); err == nil {
		if decodedMetadata.SupportsWebhookSetup {
			metadata.SupportsWebhookSetup = true
		}
	}

	if len(projects) > 0 {
		metadata.ExternalOrganizationID = projects[0].OrganizationName
	}

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()
	return nil
}

func (l *Logfire) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "project" && resourceType != "alert" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	if resourceType == "project" {
		projects, err := client.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to list Logfire projects: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(projects))
		for _, project := range projects {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: project.ProjectName,
				ID:   project.ID,
			})
		}

		return resources, nil
	}

	// resourceType == "alert"
	projectID := ctx.Parameters["projectId"]
	if isUnsetProjectID(projectID) {
		// Project is selected in a separate dropdown. When it isn't selected yet,
		// populate an empty alert dropdown instead of failing the resource load.
		return []core.IntegrationResource{}, nil
	}

	projectID = strings.TrimSpace(projectID)
	alerts, err := client.ListAlerts(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Logfire alerts: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(alerts))
	for _, alert := range alerts {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: alert.Name,
			ID:   alert.ID,
		})
	}

	return resources, nil
}

func isUnsetProjectID(projectID string) bool {
	trimmed := strings.TrimSpace(projectID)
	if trimmed == "" {
		return true
	}

	switch strings.ToLower(trimmed) {
	case "undefined", "null":
		return true
	default:
		return false
	}
}

func (l *Logfire) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (l *Logfire) Actions() []core.Action {
	return []core.Action{}
}

func (l *Logfire) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (l *Logfire) Components() []core.Component {
	return []core.Component{
		&QueryLogfire{},
	}
}

func (l *Logfire) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertReceived{},
	}
}
