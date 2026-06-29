package cloudsql

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetInstance struct{}

type GetInstanceSpec struct {
	Instance string `json:"instance" mapstructure:"instance"`
}

func (g *GetInstance) Name() string {
	return "gcp.cloudsql.getInstance"
}

func (g *GetInstance) Label() string {
	return "Cloud SQL • Get Instance"
}

func (g *GetInstance) Description() string {
	return "Fetch a Cloud SQL instance's configuration and status"
}

func (g *GetInstance) Documentation() string {
	return `The Get Instance component retrieves a Cloud SQL instance's configuration and current status.

## Use Cases

- **Readiness polling**: After Create Instance, poll until the instance reaches ` + "`RUNNABLE`" + `
- **Enrichment**: Read the connection name or IP address to feed a downstream step
- **Auditing**: Capture instance details as part of a workflow

## Configuration

- **Instance**: The Cloud SQL instance to fetch (required)

## Output

Emits a ` + "`gcp.cloudsql.instance`" + ` payload with the instance ` + "`name`" + `, ` + "`state`" + `, ` + "`databaseVersion`" + `, ` + "`region`" + `, ` + "`tier`" + `, ` + "`connectionName`" + `, ` + "`ipAddress`" + `, and ` + "`selfLink`" + `.

## Important Notes

- Requires the ` + "`roles/cloudsql.viewer`" + ` (or ` + "`roles/cloudsql.admin`" + `) IAM role, and the **Cloud SQL Admin API** enabled.`
}

func (g *GetInstance) Icon() string {
	return "database"
}

func (g *GetInstance) Color() string {
	return "blue"
}

func (g *GetInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud SQL instance to fetch",
			Placeholder: "Select an instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
	}
}

func (g *GetInstance) Setup(ctx core.SetupContext) error {
	spec := GetInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Instance) == "" {
		return fmt.Errorf("instance is required")
	}
	return ctx.Metadata.Set(InstanceNodeMetadata{Instance: strings.TrimSpace(spec.Instance)})
}

func (g *GetInstance) Execute(ctx core.ExecutionContext) error {
	spec := GetInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}
	instance := strings.TrimSpace(spec.Instance)
	if instance == "" {
		return ctx.ExecutionState.Fail("error", "instance is required")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	inst, err := getInstance(context.Background(), client, client.ProjectID(), instance)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to get instance", err, roleHintViewer))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{instancePayload(inst)})
}

func (g *GetInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
