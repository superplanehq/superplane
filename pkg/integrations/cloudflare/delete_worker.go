package cloudflare

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteWorker struct{}

type DeleteWorkerSpec struct {
	AccountID    string `json:"accountId"`
	WorkerScript string `json:"workerScript"`
	Force        *bool  `json:"force"`
}

func (d *DeleteWorker) Name() string {
	return "cloudflare.deleteWorker"
}

func (d *DeleteWorker) Label() string {
	return "Delete Worker"
}

func (d *DeleteWorker) Description() string {
	return "Delete a Worker script from your Cloudflare account"
}

func (d *DeleteWorker) Documentation() string {
	return `The Delete Worker component removes a Worker script.

## Configuration

- **Worker Script**: Worker to delete (picker lists scripts for the account).
- **Force**: When enabled, Cloudflare deletes the script even when blocked by bindings (see Cloudflare API ` + "`force`" + ` query parameter).

## Output

Emits the account ID and Worker Script that were deleted.

> **Warning**: This operation is irreversible for the Worker script.`
}

func (d *DeleteWorker) Icon() string {
	return "cloud"
}

func (d *DeleteWorker) Color() string {
	return "orange"
}

func (d *DeleteWorker) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteWorker) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "workerScript",
			Label:       "Worker Script",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Worker Script to delete",
			Placeholder: "Select a Worker script",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "workerScript",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "accountId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "accountId"},
						},
					},
				},
			},
		},
		{
			Name:        "force",
			Label:       "Force delete",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Pass Cloudflare force=true when deleting (see Cloudflare Workers API)",
		},
	}
}

func (d *DeleteWorker) Setup(ctx core.SetupContext) error {
	spec := DeleteWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.WorkerScript == "" {
		return errors.New("workerScript is required")
	}

	return resolveWorkerScriptMetadata(ctx, accountID, spec.WorkerScript)
}

func (d *DeleteWorker) Execute(ctx core.ExecutionContext) error {
	spec := DeleteWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	force := false
	if spec.Force != nil {
		force = *spec.Force
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeleteWorkerScript(accountID, spec.WorkerScript, force); err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}

	result := map[string]any{
		"accountId":    accountID,
		"workerScript": spec.WorkerScript,
		"deleted":      true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.worker.deleted",
		[]any{result},
	)
}

func (d *DeleteWorker) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteWorker) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteWorker) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteWorker) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteWorker) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteWorker) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
