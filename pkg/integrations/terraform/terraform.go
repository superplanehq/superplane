package terraform

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("terraform", &TerraformIntegration{}, &TerraformWebhookHandler{})
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
	configAPI, err := ctx.Integration.GetConfig("apiToken")
	if err != nil {
		return err
	}
	configAddr, err := ctx.Integration.GetConfig("address")
	if err != nil {
		return err
	}

	client, err := NewClient(map[string]any{
		"apiToken": string(configAPI),
		"address":  string(configAddr),
	})
	if err := client.Validate(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
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
		&TerraformRunEvent{},
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
