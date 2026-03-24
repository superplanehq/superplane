package logfire

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
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
	ExternalProjectID      string `json:"externalProjectId,omitempty"`
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
5. Enable these scopes:
   - **Organization scopes**: ` + "`organization:write_channel`" + ` (required for auto-creating webhook channels)
   - **Project scopes**: ` + "`project:read`" + ` and ` + "`project:read_token`" + `
6. Choose project access:
   - **All projects**, or
   - Select a specific project from the dropdown.
7. Click **Create API Key**.
8. Copy the API key and paste it into SuperPlane integration settings.`
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
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	bootstrap, err := client.BootstrapIntegration("superplane-query-token")
	if err != nil {
		return err
	}

	if err := ctx.Integration.SetSecret(readTokenSecretName, []byte(bootstrap.ReadToken)); err != nil {
		return fmt.Errorf("failed to store Logfire read token: %w", err)
	}

	metadata := Metadata{
		SupportsWebhookSetup: false,
		SupportsQueryAPI:     true,
	}

	decodedMetadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &decodedMetadata); err == nil {
		if decodedMetadata.ExternalOrganizationID != "" {
			metadata.ExternalOrganizationID = decodedMetadata.ExternalOrganizationID
		}
		if decodedMetadata.ExternalProjectID != "" {
			metadata.ExternalProjectID = decodedMetadata.ExternalProjectID
		}
		if decodedMetadata.SupportsWebhookSetup {
			metadata.SupportsWebhookSetup = true
		}
	}
	metadata.ExternalOrganizationID = bootstrap.Project.OrganizationName
	metadata.ExternalProjectID = bootstrap.Project.ID

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()
	return nil
}

func (l *Logfire) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
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
