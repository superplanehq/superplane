package terraform

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("terraform", &Integration{}, &WebhookHandler{})
}

type Integration struct{}

func (i *Integration) Name() string {
	return "terraform"
}

func (i *Integration) Label() string {
	return "Terraform"
}

func (i *Integration) Icon() string {
	return "terraform"
}

func (i *Integration) Description() string {
	return "HashiCorp HCP Terraform & Enterprise Integration"
}

func (i *Integration) Instructions() string {
	return `## Terraform Configuration

To use the Terraform integration, you need an HCP Terraform API Token.

1. Go to your organization's settings in HCP Terraform.
2. Navigate to the **API Tokens** page.
3. Click **Create a team token** and provide a description.
4. Once generated, paste the token in the API Token field below.`
}

func (i *Integration) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "address",
			Label:       "Terraform Address",
			Type:        configuration.FieldTypeString,
			Default:     "https://app.terraform.io",
			Required:    true,
			Description: "The URL of the HCP Terraform or Terraform Enterprise instance.",
		},
		{
			Name:        "apiToken",
			Label:       "Team API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Your HCP Terraform Team API Token. This token must belong to a team with appropriate workspace permissions.",
		},
	}
}

type Configuration struct {
	Address  string `json:"address" mapstructure:"address"`
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

func (i *Integration) Sync(ctx core.SyncContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Validate(); err != nil {
		return err
	}

	// Auto-generate a webhook secret if one does not exist
	var webhookSecret []byte
	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return fmt.Errorf("failed to fetch secrets: %w", err)
	}

	for _, s := range secrets {
		if s.Name == "webhookSecret" {
			webhookSecret = s.Value
			break
		}
	}
	if len(webhookSecret) == 0 {
		newSecret, err := crypto.Base64String(32)
		if err != nil {
			return fmt.Errorf("failed to generate webhook secret: %w", err)
		}
		if err := ctx.Integration.SetSecret("webhookSecret", []byte(newSecret)); err != nil {
			return fmt.Errorf("failed to store generated webhook secret: %w", err)
		}
	}

	ctx.Integration.Ready()
	return nil
}

func getClientFromIntegration(integration core.IntegrationContext) (*Client, error) {
	configAPI, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}
	configAddr, _ := integration.GetConfig("address")

	return NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
}

func (i *Integration) Cleanup(ctx core.IntegrationCleanupContext) error     { return nil }
func (i *Integration) Actions() []core.Action                               { return nil }
func (i *Integration) HandleAction(ctx core.IntegrationActionContext) error { return nil }
func (i *Integration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "workspace" {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	orgs, err := client.ListOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	var results []core.IntegrationResource

	for _, org := range orgs {
		workspaces, err := client.ListWorkspaces(org.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces for org %s: %w", org.Name, err)
		}

		for _, ws := range workspaces {
			results = append(results, core.IntegrationResource{
				Type: "workspace",
				ID:   ws.ID,
				Name: fmt.Sprintf("%s / %s", org.Name, ws.Name),
			})
		}
	}

	return results, nil
}

func (i *Integration) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *Integration) Triggers() []core.Trigger {
	return []core.Trigger{
		&RunEvent{},
	}
}

func (i *Integration) Components() []core.Component {
	return []core.Component{
		&RunPlan{},
	}
}

type NodeMetadata struct {
	Workspace *Workspace `json:"workspace" mapstructure:"workspace"`
}

func ensureWorkspaceInMetadata(ctx core.MetadataContext, integration core.IntegrationContext, configuration any) error {
	if ctx == nil {
		return nil
	}

	var nodeMeta NodeMetadata

	wsID := getWorkspaceIDFromConfiguration(configuration)
	if wsID == "" {
		return nil
	}

	if err := mapstructure.Decode(ctx.Get(), &nodeMeta); err == nil && nodeMeta.Workspace != nil && nodeMeta.Workspace.Name != "" {
		if nodeMeta.Workspace.ID == wsID || nodeMeta.Workspace.Name == wsID {
			return nil
		}
	}

	client, err := getClientFromIntegration(integration)
	if err != nil {
		return err
	}

	resolvedID, err := client.ResolveWorkspaceID(wsID)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace id: %w", err)
	}

	ws, err := client.ReadWorkspace(resolvedID)
	if err != nil {
		return fmt.Errorf("failed to read workspace: %w", err)
	}

	return ctx.Set(NodeMetadata{
		Workspace: &Workspace{
			ID:   ws.ID,
			Name: ws.Attributes.Name,
		},
	})
}

func getWorkspaceIDFromConfiguration(c any) string {
	configMap, ok := c.(map[string]any)
	if !ok {
		return ""
	}

	r, ok := configMap["workspace"]
	if !ok {
		return ""
	}

	wsID, ok := r.(string)
	if !ok {
		return ""
	}

	return wsID
}
