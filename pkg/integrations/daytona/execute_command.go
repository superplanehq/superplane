package daytona

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ExecuteCommandPayloadType          = "daytona.command.response"
	ExecuteCommandPollInterval         = 5 * time.Second
	ExecuteCommandOutputChannelSuccess = "success"
	ExecuteCommandOutputChannelFailed  = "failed"
)

var envVariableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type ExecuteCommand struct{}

type ExecuteCommandSpec struct {
	Sandbox string        `json:"sandbox"`
	Command string        `json:"command"`
	Cwd     string        `json:"cwd,omitempty"`
	Env     []EnvVariable `json:"env,omitempty"`
	Timeout int           `json:"timeout,omitempty"`
}

type ExecuteCommandMetadata struct {
	SandboxID string `json:"sandboxId" mapstructure:"sandboxId"`
	SessionID string `json:"sessionId" mapstructure:"sessionId"`
	CmdID     string `json:"cmdId" mapstructure:"cmdId"`
	StartedAt int64  `json:"startedAt" mapstructure:"startedAt"`
	Timeout   int    `json:"timeout" mapstructure:"timeout"`
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

- **Sandbox**: The sandbox ID to run commands in (from createSandbox output). Supports expressions, e.g. ` + "`" + `{{ $["daytona.createSandbox"].data.id }}` + "`" + `
- **Command**: The shell command to execute
- **Working Directory**: Optional working directory for the command
- **Environment Variables**: Optional key-value pairs exported before command execution
- **Timeout**: Optional execution timeout in seconds

## Output

Routes to one of two channels:
- **success**: Exit code is 0
- **failed**: Exit code is non-zero

The payload includes:
- **exitCode**: The process exit code (0 for success)
- **timeout**: Whether the command timed out
- **result**: The stdout/output from the command execution

## Notes

- The sandbox must be created first using createSandbox
- Commands run in a shell environment
- Non-zero exit codes indicate command failures`
}

func (e *ExecuteCommand) Icon() string {
	return "daytona"
}

func (e *ExecuteCommand) Color() string {
	return "orange"
}

func (e *ExecuteCommand) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ExecuteCommandOutputChannelSuccess, Label: "Success"},
		{Name: ExecuteCommandOutputChannelFailed, Label: "Failed"},
	}
}

func (e *ExecuteCommand) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sandbox",
			Label:       "Sandbox",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Sandbox to run the command in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sandbox",
				},
			},
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
			Description: "Environment variables to export before running the command",
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Execution timeout in seconds",
			Default:     60,
		},
	}
}

func (e *ExecuteCommand) Setup(ctx core.SetupContext) error {
	spec := ExecuteCommandSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Sandbox == "" {
		return fmt.Errorf("sandbox is required")
	}

	if spec.Command == "" {
		return fmt.Errorf("command is required")
	}

	for _, env := range spec.Env {
		name := strings.TrimSpace(env.Name)
		if name == "" {
			return fmt.Errorf("env variable name is required")
		}

		if !envVariableNamePattern.MatchString(name) {
			return fmt.Errorf("invalid env variable name: %s", env.Name)
		}
	}

	return nil
}

func (e *ExecuteCommand) Execute(ctx core.ExecutionContext) error {
	spec := ExecuteCommandSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	sessionID := uuid.New().String()
	if err := client.CreateSession(spec.Sandbox, sessionID); err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	command := spec.Command
	if spec.Cwd != "" {
		command = fmt.Sprintf("cd %s && %s", spec.Cwd, spec.Command)
	}

	if len(spec.Env) > 0 {
		envExports := make([]string, 0, len(spec.Env))
		for _, env := range spec.Env {
			name := strings.TrimSpace(env.Name)
			if name == "" {
				continue
			}

			envExports = append(envExports, fmt.Sprintf("%s=%s", name, shellQuote(env.Value)))
		}

		if len(envExports) > 0 {
			command = fmt.Sprintf("export %s && %s", strings.Join(envExports, " "), command)
		}
	}

	response, err := client.ExecuteSessionCommand(spec.Sandbox, sessionID, command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}

	timeout := spec.Timeout
	if timeout == 0 {
		timeout = 60
	}

	metadata := ExecuteCommandMetadata{
		SandboxID: spec.Sandbox,
		SessionID: sessionID,
		CmdID:     response.CmdID,
		StartedAt: time.Now().Unix(),
		Timeout:   timeout,
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCommandPollInterval)
}

func (e *ExecuteCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *ExecuteCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExecuteCommand) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (e *ExecuteCommand) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return e.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (e *ExecuteCommand) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata ExecuteCommandMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if time.Now().Unix()-metadata.StartedAt > int64(metadata.Timeout) {
		return ctx.ExecutionState.Emit(ExecuteCommandOutputChannelFailed, ExecuteCommandPayloadType, []any{map[string]any{
			"exitCode": nil,
			"timeout":  true,
			"result":   fmt.Sprintf("command timed out after %d seconds", metadata.Timeout),
		}})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	session, err := client.GetSession(metadata.SandboxID, metadata.SessionID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCommandPollInterval)
	}

	cmd := session.FindCommand(metadata.CmdID)
	if cmd == nil || cmd.ExitCode == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCommandPollInterval)
	}

	logs, err := client.GetSessionCommandLogs(metadata.SandboxID, metadata.SessionID, metadata.CmdID)
	if err != nil {
		logs = ""
	}

	result := &ExecuteCommandResponse{
		ExitCode: *cmd.ExitCode,
		Timeout:  false,
		Result:   logs,
	}

	channel := ExecuteCommandOutputChannelFailed
	if *cmd.ExitCode == 0 {
		channel = ExecuteCommandOutputChannelSuccess
	}

	return ctx.ExecutionState.Emit(
		channel,
		ExecuteCommandPayloadType,
		[]any{result},
	)
}

func (e *ExecuteCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func (e *ExecuteCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}
