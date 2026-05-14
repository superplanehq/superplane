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
	AccountID    string `json:"accountId"`
	WorkerScript string `json:"workerScript"`
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

- **Worker Script**: The Worker script in your Cloudflare account (picker lists scripts for the account).

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
			Name:        "workerScript",
			Label:       "Worker Script",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Worker Script to describe",
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

	if spec.WorkerScript == "" {
		return errors.New("workerScript is required")
	}

	return resolveWorkerScriptMetadata(ctx, accountID, spec.WorkerScript)
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

	settings, err := client.GetWorkerSettings(accountID, spec.WorkerScript)
	if err != nil {
		return fmt.Errorf("failed to get worker settings: %w", err)
	}

	deployments, err := client.ListWorkerDeployments(accountID, spec.WorkerScript)
	if err != nil {
		return fmt.Errorf("failed to list worker deployments: %w", err)
	}

	result := map[string]any{
		"accountId":    accountID,
		"workerScript": spec.WorkerScript,
		"settings":     settings,
		"deployments":  deployments,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.worker.fetched",
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
