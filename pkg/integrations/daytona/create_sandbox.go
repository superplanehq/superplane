package daytona

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
	SandboxPayloadType        = "daytona.sandbox"
	CreateSandboxPollInterval = 5 * time.Second
	CreateSandboxTimeout      = 5 * time.Minute
)

type CreateSandbox struct{}

type CreateSandboxSpec struct {
	Snapshot         string        `json:"snapshot,omitempty"`
	Target           string        `json:"target,omitempty"`
	AutoStopInterval int           `json:"autoStopInterval,omitempty"`
	Env              []EnvVariable `json:"env,omitempty"`
}

type CreateSandboxMetadata struct {
	SandboxID string `json:"sandboxId" mapstructure:"sandboxId"`
	StartedAt int64  `json:"startedAt" mapstructure:"startedAt"`
}

type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *CreateSandbox) Name() string {
	return "daytona.createSandbox"
}

func (c *CreateSandbox) Label() string {
	return "Create Sandbox"
}

func (c *CreateSandbox) Description() string {
	return "Create an isolated sandbox environment for code execution"
}

func (c *CreateSandbox) Documentation() string {
	return `The Create Sandbox component creates an isolated environment for executing code safely.

## Use Cases

- **AI code execution**: Run AI-generated code in a secure sandbox
- **Code testing**: Test untrusted code without affecting your infrastructure
- **Development environments**: Create ephemeral development environments

## Configuration

- **Snapshot**: Base environment snapshot (optional, uses default if not specified)
- **Target**: Target region for the sandbox (optional)
- **Auto Stop Interval**: Time in minutes before the sandbox auto-stops
- **Environment Variables**: Key-value pairs to set as environment variables in the sandbox

## Output

Returns the sandbox information including:
- **id**: The unique sandbox identifier (use this in subsequent execute/delete operations)
- **state**: The current state of the sandbox (e.g., "started")

## Notes

- The component polls the sandbox state until it reaches "started"
- Each sandbox is fully isolated
- Remember to delete sandboxes when done to free resources`
}

func (c *CreateSandbox) Icon() string {
	return "daytona"
}

func (c *CreateSandbox) Color() string {
	return "orange"
}

func (c *CreateSandbox) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSandbox) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "snapshot",
			Label:    "Snapshot",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "snapshot",
					UseNameAsValue: true,
				},
			},
			Description: "Base environment snapshot for the sandbox",
			Default:     "default",
		},
		{
			Name:        "target",
			Label:       "Target Region",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. us, eu, local",
			Description: "Target region for the sandbox",
		},
		{
			Name:        "autoStopInterval",
			Label:       "Auto Stop Interval",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Time in minutes before the sandbox auto-stops",
			Default:     15,
		},
		{
			Name:  "env",
			Label: "Environment Variables",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "name",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
			Required:    false,
			Description: "Environment variables to set in the sandbox",
		},
	}
}

func (c *CreateSandbox) Setup(ctx core.SetupContext) error {
	spec := CreateSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Snapshot != "" {
		trimmed := strings.TrimSpace(spec.Snapshot)
		if trimmed == "" {
			return fmt.Errorf("snapshot must not be empty if provided")
		}
	}

	if spec.AutoStopInterval < 0 {
		return fmt.Errorf("autoStopInterval cannot be negative")
	}

	return nil
}

func (c *CreateSandbox) Execute(ctx core.ExecutionContext) error {
	spec := CreateSandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	var envMap map[string]string
	if len(spec.Env) > 0 {
		envMap = make(map[string]string, len(spec.Env))
		for _, e := range spec.Env {
			envMap[e.Name] = e.Value
		}
	}

	req := &CreateSandboxRequest{
		Snapshot:         spec.Snapshot,
		Target:           spec.Target,
		AutoStopInterval: spec.AutoStopInterval,
		Env:              envMap,
	}

	sandbox, err := client.CreateSandbox(req)
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %v", err)
	}

	metadata := CreateSandboxMetadata{
		SandboxID: sandbox.ID,
		StartedAt: time.Now().UnixNano(),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSandboxPollInterval)
}

func (c *CreateSandbox) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSandbox) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSandbox) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateSandbox) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateSandbox) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateSandboxMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if time.Since(time.Unix(0, metadata.StartedAt)) > CreateSandboxTimeout {
		return fmt.Errorf("sandbox %s timed out waiting to start", metadata.SandboxID)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	sandbox, err := client.GetSandbox(metadata.SandboxID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSandboxPollInterval)
	}

	switch sandbox.State {
	case "started":
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			SandboxPayloadType,
			[]any{sandbox},
		)
	case "error":
		return fmt.Errorf("sandbox %s failed to start", metadata.SandboxID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateSandboxPollInterval)
	}
}

func (c *CreateSandbox) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateSandbox) Cleanup(ctx core.SetupContext) error {
	return nil
}
