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

const (
	ManageInstancePowerPayloadType = "oci.manageInstancePower"
	instanceStateStopped           = "STOPPED"
)

var instancePowerTargetStates = map[string]string{
	"START":     instanceStateRunning,
	"STOP":      instanceStateStopped,
	"SOFTSTOP":  instanceStateStopped,
	"RESET":     instanceStateRunning,
	"SOFTRESET": instanceStateRunning,
}

type ManageInstancePower struct{}

type ManageInstancePowerSpec struct {
	Instance   string `json:"instance" mapstructure:"instance"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
	Action     string `json:"action" mapstructure:"action"`
}

type ManageInstancePowerMetadata struct {
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	Action       string `json:"action" mapstructure:"action"`
	TargetState  string `json:"targetState" mapstructure:"targetState"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *ManageInstancePower) Name() string {
	return "oci.manageInstancePower"
}

func (c *ManageInstancePower) Label() string {
	return "Manage Instance Power"
}

func (c *ManageInstancePower) Description() string {
	return "Start, stop, or reset an OCI Compute instance and wait for the target state"
}

func (c *ManageInstancePower) Documentation() string {
	return `The Manage Instance Power component performs a power lifecycle action on an Oracle Cloud Infrastructure Compute instance and waits for the target state.

## Use Cases

- **Scheduled shutdowns**: Stop instances outside business hours to reduce cost
- **Recovery workflows**: Reset or soft-reset an instance after an incident
- **Startup automation**: Start instances before a deployment, test, or maintenance workflow runs

## Configuration

- **Instance**: The OCI Compute instance.
- **Action**: One of ` + "`START`" + `, ` + "`STOP`" + `, ` + "`SOFTSTOP`" + `, ` + "`RESET`" + `, or ` + "`SOFTRESET`" + `.

## Output

Emits the instance details on the default output channel once the instance reaches the expected lifecycle state:
- ` + "`instanceId`" + ` — instance OCID
- ` + "`displayName`" + ` — instance display name
- ` + "`lifecycleState`" + ` — ` + "`RUNNING`" + ` for start/reset actions, or ` + "`STOPPED`" + ` for stop actions
- ` + "`shape`" + ` — the instance shape
- ` + "`availabilityDomain`" + ` — the availability domain
- ` + "`compartmentId`" + ` — the compartment OCID
- ` + "`region`" + ` — the region
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *ManageInstancePower) Icon() string {
	return "oci"
}

func (c *ManageInstancePower) Color() string {
	return "red"
}

func (c *ManageInstancePower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ManageInstancePower) ExampleOutput() map[string]any {
	return exampleOutputManageInstancePower()
}

func (c *ManageInstancePower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Compute instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Power action to perform on the instance",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Start", Value: "START"},
						{Label: "Stop", Value: "STOP"},
						{Label: "Soft Stop", Value: "SOFTSTOP"},
						{Label: "Reset", Value: "RESET"},
						{Label: "Soft Reset", Value: "SOFTRESET"},
					},
				},
			},
		},
	}
}

func (s ManageInstancePowerSpec) selectedInstance() string {
	if strings.TrimSpace(s.Instance) != "" {
		return s.Instance
	}
	return s.InstanceID
}

func (c *ManageInstancePower) Setup(ctx core.SetupContext) error {
	spec := ManageInstancePowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.selectedInstance()) == "" {
		return errors.New("instance is required")
	}
	if strings.TrimSpace(spec.Action) == "" {
		return errors.New("action is required")
	}
	if _, ok := instancePowerTargetStates[spec.Action]; !ok {
		return fmt.Errorf("unsupported action: %s", spec.Action)
	}
	return nil
}

func (c *ManageInstancePower) Execute(ctx core.ExecutionContext) error {
	spec := ManageInstancePowerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	instance := spec.selectedInstance()

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if _, err := client.InstanceAction(instance, spec.Action); err != nil {
		return fmt.Errorf("failed to run instance action: %w", err)
	}

	targetState := instancePowerTargetStates[spec.Action]
	if err := ctx.Metadata.Set(ManageInstancePowerMetadata{
		InstanceID:  instance,
		Action:      spec.Action,
		TargetState: targetState,
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
}

func (c *ManageInstancePower) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *ManageInstancePower) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *ManageInstancePower) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata ManageInstancePowerMetadata
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

	if instance.LifecycleState == metadata.TargetState {
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, ManageInstancePowerPayloadType, []any{instanceToMap(instance)})
	}
	if metadata.PollAttempts >= maxPollAttempts {
		return fmt.Errorf("timed out waiting for instance %s to reach %s after %d poll attempts (state: %s)", metadata.InstanceID, metadata.TargetState, metadata.PollAttempts, instance.LifecycleState)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createInstancePollInterval)
}

func (c *ManageInstancePower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ManageInstancePower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ManageInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ManageInstancePower) Cleanup(ctx core.SetupContext) error {
	return nil
}
