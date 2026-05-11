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

type GetWorker struct{}

type GetWorkerSpec struct {
	AccountID  string `json:"accountId"`
	ScriptName string `json:"scriptName"`
}

func (g *GetWorker) Name() string {
	return "cloudflare.getWorker"
}

func (g *GetWorker) Label() string {
	return "Get Worker"
}

func (g *GetWorker) Description() string {
	return "Retrieve metadata and settings for a deployed Worker script"
}

func (g *GetWorker) Documentation() string {
	return `The Get Worker component loads Workers script **settings** (bindings, compatibility, usage model, etc.) and the list of **deployments** (newest first per Cloudflare).

## Configuration

- **Script name**: The Worker script name in your Cloudflare account.

## Output

Emits ` + "`settings`" + ` and ` + "`deployments`" + ` for use in later workflow steps.`
}

func (g *GetWorker) Icon() string {
	return "cloud"
}

func (g *GetWorker) Color() string {
	return "orange"
}

func (g *GetWorker) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetWorker) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "scriptName",
			Label:       "Worker script name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Worker script to describe",
			Placeholder: "my-worker",
		},
	}
}

func (g *GetWorker) Setup(ctx core.SetupContext) error {
	spec := GetWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.ScriptName == "" {
		return errors.New("scriptName is required")
	}

	return nil
}

func (g *GetWorker) Execute(ctx core.ExecutionContext) error {
	spec := GetWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	settings, err := client.GetWorkerSettings(accountID, spec.ScriptName)
	if err != nil {
		return fmt.Errorf("failed to get worker settings: %w", err)
	}

	deployments, err := client.ListWorkerDeployments(accountID, spec.ScriptName)
	if err != nil {
		return fmt.Errorf("failed to list worker deployments: %w", err)
	}

	result := map[string]any{
		"accountId":   accountID,
		"scriptName":  spec.ScriptName,
		"settings":    settings,
		"deployments": deployments,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.worker.metadata",
		[]any{result},
	)
}

func (g *GetWorker) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetWorker) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetWorker) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetWorker) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetWorker) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetWorker) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
