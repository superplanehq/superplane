package harness

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("harness", &Harness{})
}

type Harness struct{}

type Configuration struct {
	AccountID string `json:"accountId" mapstructure:"accountId"`
	APIToken  string `json:"apiToken" mapstructure:"apiToken"`
}

type Metadata struct{}

func (h *Harness) Name() string {
	return "harness"
}

func (h *Harness) Label() string {
	return "Harness"
}

func (h *Harness) Icon() string {
	return "workflow"
}

func (h *Harness) Description() string {
	return "Run and react to your Harness pipelines"
}

func (h *Harness) Instructions() string {
	return ""
}

func (h *Harness) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Description: "Your Harness account identifier. Found in any Harness URL: https://app.harness.io/ng/#/account/<ACCOUNT_ID>/...",
			Placeholder: "e.g. abc123xyz",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Harness API token (personal or service account token)",
			Required:    true,
		},
	}
}

func (h *Harness) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Harness) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	//
	// Verify credentials by listing pipelines.
	// Harness doesn't have a dedicated whoami endpoint,
	// so we use the account-level user aggregation endpoint.
	//
	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("error validating credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (h *Harness) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (h *Harness) Actions() []core.Action {
	return []core.Action{}
}

func (h *Harness) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (h *Harness) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (h *Harness) Components() []core.Component {
	return []core.Component{
		&RunPipeline{},
	}
}

func (h *Harness) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineCompleted{},
	}
}
