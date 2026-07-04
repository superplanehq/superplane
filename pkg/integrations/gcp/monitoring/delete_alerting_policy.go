package monitoring

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteAlertingPolicy struct{}

type DeleteAlertingPolicySpec struct {
	AlertPolicy string `mapstructure:"alertPolicy"`
}

func (d *DeleteAlertingPolicy) Name() string {
	return "gcp.monitoring.deleteAlertingPolicy"
}

func (d *DeleteAlertingPolicy) Label() string {
	return "Monitoring • Delete Alerting Policy"
}

func (d *DeleteAlertingPolicy) Description() string {
	return "Permanently delete a Cloud Monitoring alerting policy"
}

func (d *DeleteAlertingPolicy) Documentation() string {
	return `The Delete Alerting Policy component permanently deletes a Cloud Monitoring alerting policy.

## Use Cases

- **Cleanup**: Remove policies for decommissioned services
- **Environment teardown**: Delete alerting as part of tearing down ephemeral environments

## Configuration

- **Alerting Policy**: Pick from the policies in your project, or pass an expression chained from an upstream node (e.g. the ` + "`name`" + ` emitted by ` + "`gcp.monitoring.createAlertingPolicy`" + `).

## Output

Returns the deleted policy reference:
- **name**: The resource name that was deleted
- **id**: The policy ID

## Important Notes

- This operation is **permanent** and cannot be undone
- Requires the ` + "`roles/monitoring.editor`" + ` IAM role on the integration's service account
- If the policy is not found, the action fails so stale expressions don't silently mask incomplete cleanup`
}

func (d *DeleteAlertingPolicy) Icon() string {
	return "trash-2"
}

func (d *DeleteAlertingPolicy) Color() string {
	return "red"
}

func (d *DeleteAlertingPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteAlertingPolicy) Configuration() []configuration.Field {
	return []configuration.Field{alertPolicySelectorField()}
}

func (d *DeleteAlertingPolicy) Setup(ctx core.SetupContext) error {
	spec := DeleteAlertingPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if err := validateAlertPolicySelection(spec.AlertPolicy); err != nil {
		return err
	}
	return resolveAlertPolicyMetadata(ctx, spec.AlertPolicy)
}

func (d *DeleteAlertingPolicy) Execute(ctx core.ExecutionContext) error {
	spec := DeleteAlertingPolicySpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	name, err := resolvePolicyName(spec.AlertPolicy, client.ProjectID())
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	if _, err := client.DeleteURL(context.Background(), fmt.Sprintf("%s/%s", monitoringBaseURL, name)); err != nil {
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to delete alerting policy", roleHintWrite, err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.monitoring.alertingPolicy.deleted",
		[]any{map[string]any{"name": name, "id": lastSegment(name)}},
	)
}

func (d *DeleteAlertingPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteAlertingPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteAlertingPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteAlertingPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeleteAlertingPolicy) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteAlertingPolicy) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
