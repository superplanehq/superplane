package terraform

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("terraform", &TerraformIntegration{}, &WebhookHandler{})
}

type TerraformIntegration struct{}

func (i *TerraformIntegration) Name() string {
	return "terraform"
}

func (i *TerraformIntegration) Label() string {
	return "Terraform"
}

func (i *TerraformIntegration) Icon() string {
	return "terraform"
}

func (i *TerraformIntegration) Description() string {
	return "HashiCorp HCP Terraform & Enterprise Integration"
}

func (i *TerraformIntegration) Instructions() string {
	return "Generate an API token from your HCP Terraform Team settings."
}

func (i *TerraformIntegration) Configuration() []configuration.Field {
	return getConfigurationFields()
}

func (i *TerraformIntegration) Sync(ctx core.SyncContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Validate(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func getClientFromIntegration(integration core.IntegrationContext) (*Client, error) {
	configAPI, err := integration.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}
	configAddr, err := integration.GetConfig("address")
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	return NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
}

func (i *TerraformIntegration) Cleanup(ctx core.IntegrationCleanupContext) error     { return nil }
func (i *TerraformIntegration) Actions() []core.Action                               { return nil }
func (i *TerraformIntegration) HandleAction(ctx core.IntegrationActionContext) error { return nil }
func (i *TerraformIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}

func (i *TerraformIntegration) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *TerraformIntegration) Triggers() []core.Trigger {
	return []core.Trigger{
		&RunEvent{},
		&TerraformNeedsAttention{},
	}
}

func (i *TerraformIntegration) Components() []core.Component {
	return []core.Component{
		&QueueRun{},
		&ApplyRun{},
		&DiscardRun{},
		&OverridePolicy{},
		&ReadRun{},
		&WaitForApproval{},
		&TrackRun{},
	}
}
