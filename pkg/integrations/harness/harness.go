package harness

import (
	"fmt"
	"strings"

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
	APIToken      string `json:"apiToken" mapstructure:"apiToken"`
	AccountID     string `json:"accountId" mapstructure:"accountId"`
	OrgID         string `json:"orgId" mapstructure:"orgId"`
	ProjectID     string `json:"projectId" mapstructure:"projectId"`
	BaseURL       string `json:"baseURL" mapstructure:"baseURL"`
	WebhookSecret string `json:"webhookSecret" mapstructure:"webhookSecret"`
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
	return "Run and monitor Harness pipelines from SuperPlane workflows"
}

func (h *Harness) Instructions() string {
	return `1. **Create API key:** In Harness, create a service-account API key with permission to run and read pipeline executions.
2. **Scope fields:** Enter **Account ID**, and optionally **Org ID** and **Project ID** for scoped APIs (**Project ID** requires **Org ID**).
3. **Webhook Secret (optional but recommended):** Set a secret token and configure Harness webhook notifications to send it in an Authorization Bearer header.
4. **Trigger setup:** After adding the On Pipeline Completed trigger, copy the generated SuperPlane webhook URL from trigger metadata into a Harness notification webhook.
5. **Auth method:** SuperPlane calls Harness APIs with ` + "`x-api-key: <token>`" + ` against ` + "`https://app.harness.io/gateway`" + ` unless overridden by Base URL.`
}

func (h *Harness) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Harness API key used for API calls",
		},
		{
			Name:        "accountId",
			Label:       "Account ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Harness account identifier",
		},
		{
			Name:        "orgId",
			Label:       "Org ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Harness organization identifier",
		},
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Harness project identifier (requires Org ID)",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     DefaultBaseURL,
			Placeholder: "https://app.harness.io/gateway",
			Description: "Harness API gateway base URL",
		},
		{
			Name:        "webhookSecret",
			Label:       "Webhook Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    false,
			Description: "Optional shared secret expected in incoming Harness webhook requests",
		},
	}
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

func (h *Harness) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Harness) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.AccountID = strings.TrimSpace(config.AccountID)
	if config.AccountID == "" {
		return fmt.Errorf("accountId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Harness credentials: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func (h *Harness) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (h *Harness) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != ResourceTypePipeline {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	pipelines, err := client.ListPipelines()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(pipelines))
	for _, pipeline := range pipelines {
		if strings.TrimSpace(pipeline.Identifier) == "" {
			continue
		}

		name := strings.TrimSpace(pipeline.Name)
		if name == "" {
			name = pipeline.Identifier
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypePipeline,
			Name: name,
			ID:   pipeline.Identifier,
		})
	}

	return resources, nil
}

func (h *Harness) Actions() []core.Action {
	return []core.Action{}
}

func (h *Harness) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
