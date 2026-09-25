package productive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Productive to work with SuperPlane:

1. **Get an API token**: In Productive, go to **Settings > API Integrations**. Create a token with read and write access. SuperPlane needs that access to create webhooks.
2. **Get the organization id**: The organization id is the numeric id in your Productive URL, e.g. ` + "`app.productive.io/<organization-id>-name`" + `.
3. **Enter credentials**: Provide the API token and organization id in the integration configuration.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("productive", &Productive{}, &ProductiveWebhookHandler{})
}

type Productive struct{}

func (p *Productive) Name() string {
	return "productive"
}

func (p *Productive) Label() string {
	return "Productive"
}

func (p *Productive) Icon() string {
	return "productive"
}

func (p *Productive) Description() string {
	return "Create backlog items from tasks in Productive"
}

func (p *Productive) Instructions() string {
	return installationInstructions
}

func (p *Productive) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Personal access token from Productive Settings > API Integrations. SuperPlane needs read and write access to create webhooks.",
		},
		{
			Name:        "organizationId",
			Label:       "Organization ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The numeric organization id in your Productive URL",
		},
		{
			Name:        "region",
			Label:       "API Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: fmt.Sprintf("Override the API base URL. Defaults to %s", BaseURL),
		},
	}
}

func (p *Productive) Actions() []core.Action {
	return []core.Action{}
}

func (p *Productive) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnTask{},
	}
}

func (p *Productive) Sync(ctx core.SyncContext) error {
	config := struct {
		APIToken       string `mapstructure:"apiToken"`
		OrganizationID string `mapstructure:"organizationId"`
	}{}

	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}

	if strings.TrimSpace(config.OrganizationID) == "" {
		return fmt.Errorf("organizationId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.ValidateCredentials(); err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	if err := deleteRetainedProbeWebhook(client, ctx.Integration); err != nil {
		return fmt.Errorf("%w. The next sync will try again", err)
	}

	if err := client.ValidateWebhookPermission(); err != nil {
		var cleanupErr *probeWebhookCleanupError
		if errors.As(err, &cleanupErr) {
			retainProbeWebhook(ctx.Integration, cleanupErr.webhookID)
			return fmt.Errorf("SuperPlane could not delete webhook %s from the permission check. SuperPlane will try to delete that webhook again: %w", cleanupErr.webhookID, cleanupErr.err)
		}
		if errors.Is(err, ErrMissingWritePermission) {
			return ErrMissingWritePermission
		}
		if errors.Is(err, ErrWebhooksLimitExceeded) {
			return webhooksUnavailableError(err)
		}
		return fmt.Errorf("error checking webhook permission: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

const probeWebhookMetadataKey = "probeWebhookId"

func deleteRetainedProbeWebhook(client *Client, integration core.IntegrationContext) error {
	webhookID := retainedProbeWebhookID(integration)
	if webhookID == "" {
		return nil
	}

	if err := client.DeleteWebhook(webhookID); err != nil && !IsNotFoundError(err) {
		return fmt.Errorf("could not delete webhook %s from the previous permission check: %w", webhookID, err)
	}

	metadata := integrationMetadata(integration)
	delete(metadata, probeWebhookMetadataKey)
	integration.SetMetadata(metadata)
	return nil
}

func retainedProbeWebhookID(integration core.IntegrationContext) string {
	webhookID, _ := integrationMetadata(integration)[probeWebhookMetadataKey].(string)
	return strings.TrimSpace(webhookID)
}

func retainProbeWebhook(integration core.IntegrationContext, webhookID string) {
	metadata := integrationMetadata(integration)
	metadata[probeWebhookMetadataKey] = webhookID
	integration.SetMetadata(metadata)
}

func integrationMetadata(integration core.IntegrationContext) map[string]any {
	metadata := map[string]any{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func webhooksUnavailableError(err error) error {
	return fmt.Errorf("Productive does not offer webhooks on this plan, so the On Task trigger cannot be set up: %w", err)
}

func (p *Productive) Cleanup(ctx core.IntegrationCleanupContext) error {
	if retainedProbeWebhookID(ctx.Integration) == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	return deleteRetainedProbeWebhook(client, ctx.Integration)
}

func (p *Productive) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *Productive) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

func (p *Productive) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - Productive.io calls the per-node webhook URL directly, not the
	// integration's own HTTP endpoint.
}

func (p *Productive) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeProject:
		return listProjectResources(ctx)
	case ResourceTypeTaskList:
		return listTaskListResources(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listProjectResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProject,
			Name: project.Name,
			ID:   project.ID,
		})
	}

	return resources, nil
}

func listTaskListResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	projectID := ""
	if ctx.Parameters != nil {
		projectID = ctx.Parameters["project"]
	}

	lists, err := client.ListTaskLists(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task lists: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(lists))
	for _, list := range lists {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeTaskList,
			Name: list.Name,
			ID:   list.ID,
		})
	}

	return resources, nil
}
