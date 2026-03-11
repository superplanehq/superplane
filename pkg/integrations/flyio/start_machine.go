package flyio

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	StartMachinePayloadType          = "flyio.machine"
	StartMachineSuccessOutputChannel = "success"
	StartMachineFailedOutputChannel  = "failed"
)

type StartMachine struct{}

type StartMachineSpec struct {
	App     string `json:"app" mapstructure:"app"`
	Machine string `json:"machine" mapstructure:"machine"`
}

func (c *StartMachine) Name() string {
	return "flyio.startMachine"
}

func (c *StartMachine) Label() string {
	return "Start Machine"
}

func (c *StartMachine) Description() string {
	return "Start a stopped Fly.io Machine and wait for it to be running"
}

func (c *StartMachine) Documentation() string {
	return `The Start Machine component starts a stopped Fly.io Machine and waits for it to reach the ` + "`started`" + ` state.

## Use Cases

- **Scheduled scale-up**: Start machines on a schedule or in response to upstream events
- **On-demand environments**: Boot a staging machine when a deployment is triggered
- **Machine lifecycle**: Control machine uptime as part of a workflow

## How It Works

1. Calls the Fly.io Machines API to start the selected Machine
2. Polls until the machine state reaches ` + "`started`" + `
3. Routes execution based on outcome:
   - **Success channel**: Machine successfully started
   - **Failed channel**: Machine failed to start

## Configuration

- **App**: The Fly.io application that owns the Machine
- **Machine**: The specific Machine to start (filtered by the selected App)

## Output Channels

- **Success**: Machine reached started state
- **Failed**: Machine could not be started`
}

func (c *StartMachine) Icon() string {
	return "play"
}

func (c *StartMachine) Color() string {
	return "green"
}

func (c *StartMachine) ExampleOutput() map[string]any {
	return map[string]any{
		"machineId": "148ed726c17589",
		"appName":   "my-fly-app",
		"state":     "started",
		"region":    "iad",
	}
}

func (c *StartMachine) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: StartMachineSuccessOutputChannel, Label: "Success"},
		{Name: StartMachineFailedOutputChannel, Label: "Failed"},
	}
}

func (c *StartMachine) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "app",
			Label:    "App",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "app",
					UseNameAsValue: true,
				},
			},
			Description: "Fly.io application that owns the Machine",
		},
		{
			Name:     "machine",
			Label:    "Machine",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "machine",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "app",
							ValueFrom: &configuration.ParameterValueFrom{Field: "app"},
						},
					},
				},
			},
			Description: "Machine to start",
		},
	}
}

func decodeStartMachineSpec(configuration any) (StartMachineSpec, error) {
	spec := StartMachineSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return StartMachineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.App = strings.TrimSpace(spec.App)
	if spec.App == "" {
		return StartMachineSpec{}, fmt.Errorf("app is required")
	}

	spec.Machine = strings.TrimSpace(spec.Machine)
	if spec.Machine == "" {
		return StartMachineSpec{}, fmt.Errorf("machine is required")
	}

	return spec, nil
}

func (c *StartMachine) Setup(ctx core.SetupContext) error {
	_, err := decodeStartMachineSpec(ctx.Configuration)
	return err
}

func (c *StartMachine) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StartMachine) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeStartMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := parseMachineID(spec.Machine)
	if err := client.StartMachine(spec.App, machineID); err != nil {
		return fmt.Errorf("failed to start machine: %w", err)
	}

	ctx.Logger.Infof("Start requested for machine %s/%s, polling for state", spec.App, machineID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1}, machinePollInterval*time.Second)
}

func (c *StartMachine) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *StartMachine) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *StartMachine) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	attempt := 1
	if v, ok := ctx.Parameters["attempt"].(float64); ok {
		attempt = int(v)
	}

	if attempt > maxMachinePollAttempts {
		ctx.Logger.Errorf("Machine start exceeded maximum poll attempts (%d), giving up", maxMachinePollAttempts)
		// Best-effort: fetch current state for the payload; ignore error.
		spec, _ := decodeStartMachineSpec(ctx.Configuration)
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			if machine, err := client.GetMachine(spec.App, parseMachineID(spec.Machine)); err == nil {
				payload := machinePayload(spec.App, machine)
				return ctx.ExecutionState.Emit(StartMachineFailedOutputChannel, StartMachinePayloadType, []any{payload})
			}
		}
		return ctx.ExecutionState.Emit(StartMachineFailedOutputChannel, StartMachinePayloadType, []any{map[string]any{"reason": "timeout"}})
	}

	spec, err := decodeStartMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := parseMachineID(spec.Machine)
	machine, err := client.GetMachine(spec.App, machineID)
	if err != nil {
		ctx.Logger.Warnf("Failed to get machine state (attempt %d), will retry: %v", attempt, err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": attempt + 1}, machinePollInterval*time.Second)
	}

	ctx.Logger.Infof("Machine %s state: %s (attempt %d/%d)", machineID, machine.State, attempt, maxMachinePollAttempts)

	switch machine.State {
	case "started":
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(StartMachineSuccessOutputChannel, StartMachinePayloadType, []any{payload})
	case "stopping", "stopped", "destroying", "destroyed":
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(StartMachineFailedOutputChannel, StartMachinePayloadType, []any{payload})
	default:
		// still transitioning — keep polling
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": attempt + 1}, machinePollInterval*time.Second)
	}
}

func (c *StartMachine) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *StartMachine) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *StartMachine) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
