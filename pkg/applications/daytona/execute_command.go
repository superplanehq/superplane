package daytona

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ExecuteCommandPayloadType = "daytona.command.response"

type ExecuteCommand struct{}

type ExecuteCommandSpec struct {
	SandboxID string `json:"sandboxId"`
	Command   string `json:"command"`
	Cwd       string `json:"cwd,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

type ExecuteCommandPayload struct {
	ExitCode int    `json:"exitCode"`
	Result   string `json:"result"`
}

func (e *ExecuteCommand) Name() string {
	return "daytona.executeCommand"
}

func (e *ExecuteCommand) Label() string {
	return "Execute Command"
}

func (e *ExecuteCommand) Description() string {
	return "Run a shell command in a sandbox environment"
}

func (e *ExecuteCommand) Documentation() string {
	return `The Execute Command component runs shell commands in an existing Daytona sandbox.

## Use Cases

- **Package installation**: Install dependencies (pip install, npm install)
- **File operations**: Create, move, or delete files in the sandbox
- **System commands**: Run any shell command in the isolated environment
- **Build processes**: Execute build scripts or compilation commands

## Configuration

- **Sandbox ID**: The ID of the sandbox (from createSandbox output)
- **Command**: The shell command to execute
- **Working Directory**: Optional working directory for the command
- **Timeout**: Optional execution timeout in seconds

## Output

Returns the command result including:
- **exitCode**: The process exit code (0 for success)
- **result**: The stdout/output from the command execution

## Notes

- The sandbox must be created first using createSandbox
- Commands run in a shell environment
- Non-zero exit codes indicate command failures`
}

func (e *ExecuteCommand) Icon() string {
	return "terminal"
}

func (e *ExecuteCommand) Color() string {
	return "orange"
}

func (e *ExecuteCommand) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (e *ExecuteCommand) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sandboxId",
			Label:       "Sandbox ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the sandbox to run the command in",
			Placeholder: "{{ $.createSandbox.data.id }}",
		},
		{
			Name:        "command",
			Label:       "Command",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The shell command to execute",
			Placeholder: "pip install requests",
		},
		{
			Name:        "cwd",
			Label:       "Working Directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Working directory for the command",
			Placeholder: "/home/daytona",
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Execution timeout in seconds",
			Placeholder: "60",
		},
	}
}

func (e *ExecuteCommand) Setup(ctx core.SetupContext) error {
	spec := ExecuteCommandSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.SandboxID == "" {
		return fmt.Errorf("sandboxId is required")
	}

	if spec.Command == "" {
		return fmt.Errorf("command is required")
	}

	return nil
}

func (e *ExecuteCommand) Execute(ctx core.ExecutionContext) error {
	spec := ExecuteCommandSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	req := &ExecuteCommandRequest{
		Command: spec.Command,
		Cwd:     spec.Cwd,
		Timeout: spec.Timeout,
	}

	response, err := client.ExecuteCommand(spec.SandboxID, req)
	if err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}

	payload := ExecuteCommandPayload{
		ExitCode: response.ExitCode,
		Result:   response.Result,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ExecuteCommandPayloadType,
		[]any{payload},
	)
}

func (e *ExecuteCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *ExecuteCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExecuteCommand) Actions() []core.Action {
	return []core.Action{}
}

func (e *ExecuteCommand) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (e *ExecuteCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
