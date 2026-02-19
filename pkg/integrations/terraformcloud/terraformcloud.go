package terraformcloud

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("terraformcloud", &TerraformCloud{}, &TerraformCloudWebhookHandler{})
}

type TerraformCloud struct{}

type Configuration struct {
	Hostname string `json:"hostname"`
	APIToken string `json:"apiToken"`
}

type Metadata struct{}

func (t *TerraformCloud) Name() string {
	return "terraformcloud"
}

func (t *TerraformCloud) Label() string {
	return "Terraform Cloud"
}

func (t *TerraformCloud) Icon() string {
	return "cloud"
}

func (t *TerraformCloud) Description() string {
	return "Trigger and monitor Terraform Cloud runs"
}

func (t *TerraformCloud) Instructions() string {
	return "Create an API token in Terraform Cloud → User Settings → Tokens. The token must have permissions to read and create runs in the target workspaces."
}

func (t *TerraformCloud) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "hostname",
			Label:       "Hostname",
			Type:        configuration.FieldTypeString,
			Description: "Terraform Cloud hostname",
			Placeholder: "app.terraform.io",
			Default:     "app.terraform.io",
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Terraform Cloud API token (user or team token)",
			Placeholder: "Your Terraform Cloud API token",
			Required:    true,
		},
	}
}

func (t *TerraformCloud) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (t *TerraformCloud) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (t *TerraformCloud) HandleRequest(ctx core.HTTPRequestContext) {}

func (t *TerraformCloud) Actions() []core.Action {
	return []core.Action{}
}

func (t *TerraformCloud) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (t *TerraformCloud) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != ResourceTypeWorkspace {
		return []core.IntegrationResource{}, nil
	}

	organization := ctx.Parameters["organization"]
	if organization == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	workspaces, err := client.ListWorkspaces(organization)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(workspaces))
	for _, ws := range workspaces {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeWorkspace,
			Name: ws.Attributes.Name,
			ID:   ws.ID,
		})
	}

	return resources, nil
}

func (t *TerraformCloud) Components() []core.Component {
	return []core.Component{
		&TriggerRun{},
	}
}

func (t *TerraformCloud) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnRunCompleted{},
	}
}

const ResourceTypeWorkspace = "workspace"
