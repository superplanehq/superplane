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

type DeleteInstance struct{}

type DeleteInstanceSpec struct {
	Instance string `json:"instance" mapstructure:"instance"`
}

func (d *DeleteInstance) Name() string {
	return "gcp.cloudsql.deleteInstance"
}

func (d *DeleteInstance) Label() string {
	return "Cloud SQL • Delete Instance"
}

func (d *DeleteInstance) Description() string {
	return "Delete a Cloud SQL instance"
}

func (d *DeleteInstance) Documentation() string {
	return `The Delete Instance component permanently deletes a Cloud SQL instance.

## Use Cases

- **Teardown**: Remove an instance when decommissioning an environment
- **Ephemeral cleanup**: Delete a preview/test instance when it is no longer needed

## Configuration

- **Instance**: The Cloud SQL instance to delete (required)

## Output

Emits a ` + "`gcp.cloudsql.instance`" + ` payload with the instance ` + "`name`" + `, the ` + "`operation`" + ` id, ` + "`status`" + `, and ` + "`deleting: true`" + `.

## Important Notes

- **This permanently deletes the instance and all its databases and data — it is irreversible.**
- Instance deletion is asynchronous; this component returns once the operation is accepted.
- Requires the ` + "`roles/cloudsql.admin`" + ` (or ` + "`roles/cloudsql.editor`" + `) IAM role, and the **Cloud SQL Admin API** enabled.`
}

func (d *DeleteInstance) Icon() string {
	return "database"
}

func (d *DeleteInstance) Color() string {
	return "red"
}

func (d *DeleteInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloud SQL instance to delete",
			Placeholder: "Select an instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
	}
}

func (d *DeleteInstance) Setup(ctx core.SetupContext) error {
	spec := DeleteInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Instance) == "" {
		return fmt.Errorf("instance is required")
	}
	return ctx.Metadata.Set(InstanceNodeMetadata{Instance: strings.TrimSpace(spec.Instance)})
}

func (d *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	spec := DeleteInstanceSpec{}
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

	op, err := deleteInstance(context.Background(), client, client.ProjectID(), instance)
	if err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to delete instance", err, roleHintAdmin))
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{
		map[string]any{
			"name":      instance,
			"operation": op.Name,
			"status":    op.Status,
			"deleting":  true,
		},
	})
}

func (d *DeleteInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
