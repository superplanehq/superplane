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
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
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

Emits a ` + "`gcp.cloudsql.instance`" + ` payload with the instance ` + "`name`" + ` and ` + "`deleted: true`" + `.

## Important Notes

- **This permanently deletes the instance and all its databases and data — it is irreversible.**
- Instance deletion is asynchronous and takes several minutes; this component polls until the instance is fully deleted (or times out) before emitting.
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

	if _, err := deleteInstance(context.Background(), client, client.ProjectID(), instance); err != nil {
		if gcpcommon.IsNotFoundError(err) {
			// The instance is already gone — confirm the deletion instead of
			// failing, mirroring how the poll path treats a 404. This keeps the
			// component idempotent when the instance was removed out of band.
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{
				map[string]any{"name": instance, "deleted": true},
			})
		}
		return ctx.ExecutionState.Fail("error", apiErrorMessage("failed to delete instance", err, roleHintAdmin))
	}

	// Instance deletion takes minutes, so poll until the instance is gone instead
	// of blocking this execution. Failures here are terminal: the GCP operation
	// is already running, and a plain error would roll back the request and
	// re-run Execute against an instance that is already being deleted.
	if err := ctx.Metadata.Set(instanceExecMetadata{Instance: instance}); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("instance deletion started but failed to record poll state: %v", err))
	}
	if err := ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("instance deletion started but failed to schedule the status poll: %v", err))
	}
	return nil
}

// poll re-checks the instance until it no longer exists (then emits a deletion
// confirmation), or the attempt/error budget is exhausted; otherwise it
// re-schedules itself. Terminal conditions fail the execution via
// ExecutionState.Fail — returning a plain error would roll back the request and
// leave the run in progress forever.
func (d *DeleteInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var md instanceExecMetadata
	if err := mapstructure.WeakDecode(ctx.Metadata.Get(), &md); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode poll metadata: %v", err))
	}
	if md.Instance == "" {
		return ctx.ExecutionState.Fail("error", "poll metadata is missing the instance name")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	_, err = getInstance(context.Background(), client, client.ProjectID(), md.Instance)
	if err != nil {
		if gcpcommon.IsNotFoundError(err) {
			// The instance is gone — deletion is complete.
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, instancePayloadType, []any{
				map[string]any{"name": md.Instance, "deleted": true},
			})
		}
		md.PollErrors++
		ctx.Logger.Warnf("failed to get instance %s (attempt %d/%d): %v", md.Instance, md.PollErrors, maxPollErrors, err)
		if md.PollErrors >= maxPollErrors {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("giving up polling instance %s after %d consecutive errors: %v", md.Instance, maxPollErrors, err))
		}
		if err := ctx.Metadata.Set(md); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval)
	}

	// Still present — keep waiting until it's gone.
	md.PollErrors = 0
	md.PollAttempts++
	if err := ctx.Metadata.Set(md); err != nil {
		return err
	}
	if md.PollAttempts >= instanceMaxPollAttempts {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("timed out waiting for instance %s to be deleted after %d polls", md.Instance, md.PollAttempts))
	}
	return ctx.Requests.ScheduleActionCall(pollHookName, map[string]any{}, instancePollInterval)
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
	return []core.Hook{{Name: pollHookName, Type: core.HookTypeInternal}}
}

func (d *DeleteInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != pollHookName {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	return d.poll(ctx)
}
