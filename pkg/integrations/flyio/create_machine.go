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
	CreateMachinePayloadType          = "flyio.machine"
	CreateMachineSuccessOutputChannel = "success"
	CreateMachineFailedOutputChannel  = "failed"
	createMachinePollInterval         = 30 * time.Second
)

type CreateMachine struct{}

type CreateMachineSpec struct {
	App      string `json:"app" mapstructure:"app"`
	Image    string `json:"image" mapstructure:"image"`
	Region   string `json:"region" mapstructure:"region"`
	CPUKind  string `json:"cpuKind" mapstructure:"cpuKind"`
	CPUs     int    `json:"cpus" mapstructure:"cpus"`
	MemoryMB int    `json:"memoryMB" mapstructure:"memoryMB"`
	Name     string `json:"name" mapstructure:"name"`
}

type CreateMachineExecutionMetadata struct {
	MachineID string `json:"machineId" mapstructure:"machineId"`
}

func (c *CreateMachine) Name() string {
	return "flyio.createMachine"
}

func (c *CreateMachine) Label() string {
	return "Create Machine"
}

func (c *CreateMachine) Description() string {
	return "Create and start a new Fly.io Machine from a container image"
}

func (c *CreateMachine) Documentation() string {
	return `The Create Machine component launches a new Fly.io Machine from a container image and waits for it to start.

## Use Cases

- **Deploy new version**: Create a machine from a freshly-built container image as part of a CI/CD pipeline
- **Task runner**: Launch an ephemeral machine to run a one-off job
- **Scale out**: Add a new machine to an app on demand

## How It Works

1. Creates a new Machine in the selected Fly.io app via the Machines API
2. Polls until the machine reaches ` + "`started`" + ` state
3. Routes execution based on outcome:
   - **Success channel**: Machine started successfully
   - **Failed channel**: Machine could not be created or started

## Configuration

- **App**: The Fly.io application to create the Machine in
- **Image**: Container image to run (e.g., ` + "`registry.fly.io/my-app:latest`" + `)
- **Region** (optional): Region to deploy to (e.g., ` + "`iad`" + `, ` + "`lhr`" + `)
- **CPU Kind** (optional): CPU type — ` + "`shared`" + ` or ` + "`performance`" + `
- **CPUs** (optional): Number of CPUs
- **Memory (MB)** (optional): Memory in megabytes
- **Machine Name** (optional): Optional name for the new machine

## Output Channels

- **Success**: Machine started successfully; payload includes machineId, state, region, image
- **Failed**: Machine failed to start`
}

func (c *CreateMachine) Icon() string {
	return "plus-circle"
}

func (c *CreateMachine) Color() string {
	return "blue"
}

func (c *CreateMachine) ExampleOutput() map[string]any {
	return map[string]any{
		"machineId": "148ed726c17589",
		"appName":   "my-fly-app",
		"state":     "started",
		"region":    "iad",
		"image":     "registry.fly.io/my-fly-app:latest",
	}
}

func (c *CreateMachine) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: CreateMachineSuccessOutputChannel, Label: "Success"},
		{Name: CreateMachineFailedOutputChannel, Label: "Failed"},
	}
}

func (c *CreateMachine) Configuration() []configuration.Field {
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
			Description: "Fly.io application to create the Machine in",
		},
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "registry.fly.io/my-app:latest",
			Description: "Container image to run (supports expressions)",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "iad",
			Description: "Fly.io region to deploy to (e.g., iad, lhr, nrt). Defaults to the app's primary region.",
		},
		{
			Name:     "cpuKind",
			Label:    "CPU Kind",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "shared",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Shared", Value: "shared"},
						{Label: "Performance", Value: "performance"},
					},
				},
			},
			Description: "Type of CPU to allocate",
		},
		{
			Name:     "cpus",
			Label:    "CPUs",
			Type:     configuration.FieldTypeNumber,
			Required: false,
			Default:  1,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
					Max: func() *int { v := 64; return &v }(),
				},
			},
			Description: "Number of CPUs",
		},
		{
			Name:     "memoryMB",
			Label:    "Memory (MB)",
			Type:     configuration.FieldTypeNumber,
			Required: false,
			Default:  256,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 256; return &v }(),
					Max: func() *int { v := 131072; return &v }(),
				},
			},
			Description: "Memory in megabytes",
		},
		{
			Name:        "name",
			Label:       "Machine Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "my-machine",
			Description: "Optional name for the new machine",
		},
	}
}

func decodeCreateMachineSpec(configuration any) (CreateMachineSpec, error) {
	spec := CreateMachineSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return CreateMachineSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.App = strings.TrimSpace(spec.App)
	if spec.App == "" {
		return CreateMachineSpec{}, fmt.Errorf("app is required")
	}

	spec.Image = strings.TrimSpace(spec.Image)
	if spec.Image == "" {
		return CreateMachineSpec{}, fmt.Errorf("image is required")
	}

	return spec, nil
}

func (c *CreateMachine) Setup(ctx core.SetupContext) error {
	_, err := decodeCreateMachineSpec(ctx.Configuration)
	return err
}

func (c *CreateMachine) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateMachine) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	guest := &GuestConfig{
		CPUKind: spec.CPUKind,
		CPUs:    spec.CPUs,
	}
	if guest.CPUKind == "" {
		guest.CPUKind = "shared"
	}
	if guest.CPUs == 0 {
		guest.CPUs = 1
	}

	memoryMB := spec.MemoryMB
	if memoryMB == 0 {
		memoryMB = 256
	}
	guest.MemoryMB = memoryMB

	req := CreateMachineRequest{
		Name:   spec.Name,
		Region: spec.Region,
		Config: &MachineConfig{
			Image: spec.Image,
			Guest: guest,
		},
	}

	machine, err := client.CreateMachine(spec.App, req)
	if err != nil {
		return fmt.Errorf("failed to create machine: %w", err)
	}

	ctx.Logger.Infof("Machine %s created in app %s, polling for started state", machine.ID, spec.App)

	// Store machine ID in metadata so poll() can retrieve it
	if err := ctx.Metadata.Set(CreateMachineExecutionMetadata{MachineID: machine.ID}); err != nil {
		return fmt.Errorf("failed to store machine metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createMachinePollInterval)
}

func (c *CreateMachine) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateMachine) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateMachine) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeCreateMachineSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := CreateMachineExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.MachineID == "" {
		return fmt.Errorf("machine ID not found in metadata")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	machineID := metadata.MachineID

	machine, err := client.GetMachine(spec.App, machineID)
	if err != nil {
		ctx.Logger.Warnf("Failed to get machine state, will retry: %v", err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createMachinePollInterval)
	}

	ctx.Logger.Infof("Machine %s state: %s", machineID, machine.State)

	switch machine.State {
	case "started":
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(CreateMachineSuccessOutputChannel, CreateMachinePayloadType, []any{payload})
	case "stopping", "stopped", "destroying", "destroyed":
		payload := machinePayload(spec.App, machine)
		return ctx.ExecutionState.Emit(CreateMachineFailedOutputChannel, CreateMachinePayloadType, []any{payload})
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createMachinePollInterval)
	}
}

func (c *CreateMachine) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateMachine) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *CreateMachine) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
