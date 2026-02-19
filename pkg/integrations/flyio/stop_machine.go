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
	StopMachinePayloadType          = "flyio.machine"
	StopMachineSuccessOutputChannel = "success"
	StopMachineFailedOutputChannel  = "failed"
	stopMachinePollInterval         = 30 * time.Second
)

type StopMachine struct{}

type StopMachineSpec struct {
	App     string `json:"app" mapstructure:"app"`
	Machine string `json:"machine" mapstructure:"machine"`
	Signal  string `json:"signal" mapstructure:"signal"`
}

func (c *StopMachine) Name() string {
	return "flyio.stopMachine"
}

func (c *StopMachine) Label() string {
	return "Stop Machine"
}

func (c *StopMachine) Description() string {
	return "Stop a running Fly.io Machine and wait for it to be stopped"
}

func (c *StopMachine) Documentation() string {
	return `The Stop Machine component stops a running Fly.io Machine and waits for it to reach the ` + "`stopped`" + ` state.

## Use Cases

- **Scheduled scale-down**: Stop machines to save costs outside business hours
- **Lifecycle management**: Shut down machines after a task completes
- **Workflow teardown**: Stop a machine as the final step in a deployment pipeline

## How It Works

1. Calls the Fly.io Machines API to stop the selected Machine
2. Polls until the machine state reaches ` + "`stopped`" + `
3. Routes execution based on outcome:
   - **Success channel**: Machine successfully stopped
   - **Failed channel**: Machine could not be stopped

## Configuration

- **App**: The Fly.io application that owns the Machine
- **Machine**: The specific Machine to stop (filtered by the selected App)
- **Signal** (optional): Signal to send to the Machine process (e.g., SIGTERM, SIGKILL)

## Output Channels

- **Success**: Machine reached stopped state
- **Failed**: Machine could not be stopped`
}

func (c *StopMachine) Icon() string {
	return "square"
}

func (c *StopMachine) Color() string {
	return "red"
}

func (c *StopMachine) ExampleOutput() map[string]any {
	return map[string]any{
		"machineId": "148ed726c17589",
		"appName":   "my-fly-app",
		"state":     "stopped",
		"region":    "iad",
	}
}

func (c *StopMachine) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: StopMachineSuccessOutputChannel, Label: "Success"},
		{Name: StopMachineFailedOutputChannel, Label: "Failed"},
	}
}

func (c *StopMachine) Configuration() []configuration.Field {
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
			Description: "Machine to stop",
		},
		{
			Name:     "signal",
			Label:    "Signal",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "SIGTERM (graceful)", Value: "SIGTERM"},
						{Label: "SIGKILL (immediate)", Value: "SIGKILL"},
					},
				},
			},
			Description: "Signal to send to the machine process (defaults to SIGTERM)",
		},
	}
}

func decodeStopMachineSpec(configuration any) (StopMachineSpec, error) {
	spec := StopMachineSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return StopMachineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.App = strings.TrimSpace(spec.App)
	if spec.App == "" {
		return StopMachineSpec{}, fmt.Errorf("app is required")
	}

	spec.Machine = strings.TrimSpace(spec.Machine)
	if spec.Machine == "" {
		return StopMachineSpec{}, fmt.Errorf("machine is required")
	}

	return spec, nil
}

func (c *StopMachine) Setup(ctx core.SetupContext) error {
	_, err := decodeStopMachineSpec(ctx.Configuration)
	return err
}

func (c *StopMachine) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StopMachine) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeStopMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := machineIDFromStopSpec(spec)

	var req *StopMachineRequest
	if spec.Signal != "" {
		req = &StopMachineRequest{Signal: spec.Signal}
	}

	if err := client.StopMachine(spec.App, machineID, req); err != nil {
		return fmt.Errorf("failed to stop machine: %w", err)
	}

	ctx.Logger.Infof("Stop requested for machine %s/%s, polling for state", spec.App, machineID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, stopMachinePollInterval)
}

func (c *StopMachine) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *StopMachine) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *StopMachine) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeStopMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := machineIDFromStopSpec(spec)
	machine, err := client.GetMachine(spec.App, machineID)
	if err != nil {
		ctx.Logger.Warnf("Failed to get machine state, will retry: %v", err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, stopMachinePollInterval)
	}

	ctx.Logger.Infof("Machine %s state: %s", machineID, machine.State)

	switch machine.State {
	case "stopped":
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(StopMachineSuccessOutputChannel, StopMachinePayloadType, []any{payload})
	case "starting", "started":
		// something went wrong - it's running when it should be stopping
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(StopMachineFailedOutputChannel, StopMachinePayloadType, []any{payload})
	default:
		// still transitioning (stopping, etc.) — keep polling
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, stopMachinePollInterval)
	}
}

func (c *StopMachine) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *StopMachine) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *StopMachine) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func machineIDFromStopSpec(spec StopMachineSpec) string {
	parts := strings.SplitN(spec.Machine, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return spec.Machine
}
