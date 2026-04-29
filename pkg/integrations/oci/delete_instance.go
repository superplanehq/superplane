package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DeleteInstancePayloadType = "oci.deleteInstance"

type DeleteInstance struct{}

type DeleteInstanceSpec struct {
	InstanceID         string `json:"instanceId" mapstructure:"instanceId"`
	PreserveBootVolume bool   `json:"preserveBootVolume" mapstructure:"preserveBootVolume"`
}

type DeleteInstanceMetadata struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *DeleteInstance) Name() string {
	return "oci.deleteInstance"
}

func (c *DeleteInstance) Label() string {
	return "Delete Instance"
}

func (c *DeleteInstance) Description() string {
	return "Terminate an OCI Compute instance and wait for termination"
}

func (c *DeleteInstance) Documentation() string {
	return `The Delete Instance component terminates an Oracle Cloud Infrastructure Compute instance and waits until OCI reports it as terminated.

## Use Cases

- **Cleanup workflows**: Tear down temporary instances after a test or deployment finishes
- **Cost controls**: Delete unused Compute instances from automation
- **Incident remediation**: Terminate compromised or unhealthy instances after replacement capacity exists

## Configuration

- **Instance**: The OCI Compute instance to terminate.
- **Preserve Boot Volume**: When enabled, OCI preserves the boot volume after instance termination.

## Output

Emits a minimal deletion payload on the default output channel:
- ` + "`instanceId`" + ` — instance OCID
- ` + "`lifecycleState`" + ` — ` + "`TERMINATED`" + `
`
}

func (c *DeleteInstance) Icon() string {
	return "oci"
}

func (c *DeleteInstance) Color() string {
	return "red"
}

func (c *DeleteInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteInstance) ExampleOutput() map[string]any {
	return exampleOutputDeleteInstance()
}

func (c *DeleteInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Compute instance to terminate",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "preserveBootVolume",
			Label:       "Preserve Boot Volume",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     false,
			Description: "Preserve the boot volume after terminating the instance",
		},
	}
}

func (c *DeleteInstance) Setup(ctx core.SetupContext) error {
	spec := DeleteInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.InstanceID) == "" {
		return errors.New("instanceId is required")
	}
	return nil
}

func (c *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	spec := DeleteInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := client.TerminateInstance(spec.InstanceID, spec.PreserveBootVolume); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	if err := ctx.Metadata.Set(DeleteInstanceMetadata{InstanceID: spec.InstanceID}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
}

func (c *DeleteInstance) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *DeleteInstance) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *DeleteInstance) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata DeleteInstanceMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.InstanceID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	instance, err := client.GetInstance(metadata.InstanceID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return c.emitTerminated(ctx, metadata.InstanceID)
		}

		metadata.PollErrors++
		ctx.Logger.Warnf("failed to get instance %s (attempt %d/%d): %v", metadata.InstanceID, metadata.PollErrors, maxPollErrors, err)
		if metadata.PollErrors >= maxPollErrors {
			return fmt.Errorf("giving up polling instance %s after %d consecutive errors: %w", metadata.InstanceID, maxPollErrors, err)
		}
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
	}

	metadata.PollErrors = 0
	metadata.PollAttempts++
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if instance.LifecycleState == instanceStateTerminated {
		return c.emitTerminated(ctx, metadata.InstanceID)
	}
	if metadata.PollAttempts >= maxPollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to terminate after %d poll attempts (state: %s)", metadata.InstanceID, metadata.PollAttempts, instance.LifecycleState)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
}

func (c *DeleteInstance) emitTerminated(ctx core.ActionHookContext, instanceID string) error {
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteInstancePayloadType, []any{
		map[string]any{
			"instanceId":     instanceID,
			"lifecycleState": instanceStateTerminated,
		},
	})
}

func (c *DeleteInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
