package ssh

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const ExecuteScriptSuccessChannel = "success"
const ExecuteScriptFailedChannel = "failed"

type ExecuteScript struct{}

type ExecuteScriptSpec struct {
	Host             string `json:"host"`
	Script           string `json:"script"`
	Interpreter      string `json:"interpreter,omitempty"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	Timeout          int    `json:"timeout,omitempty"`
}

type ExecuteScriptExecutionMetadata struct {
	Result *CommandResult `json:"result" mapstructure:"result"`
}

func (e *ExecuteScript) Name() string {
	return "ssh.executeScript"
}

func (e *ExecuteScript) Label() string {
	return "Execute Script"
}

func (e *ExecuteScript) Description() string {
	return "Execute a script on a remote host via SSH"
}

func (c *ExecuteScript) Documentation() string {
	return `Execute a multi-line script on a remote host via SSH.

## Use Cases

- **Deployment**: Run deployment scripts on remote servers
- **Configuration**: Execute configuration scripts across infrastructure
- **Maintenance**: Run maintenance and cleanup scripts
- **Automation**: Automate complex multi-step operations on remote hosts

## Configuration

- **Host**: Select a host resource from the SSH integration (format: user@host:port)
- **Script**: The script content to execute (multi-line supported)
- **Interpreter**: The script interpreter to use (bash, sh, python, python3, etc.). Defaults to "bash"
- **Working Directory**: Optional directory to change to before executing the script
- **Timeout (seconds)**: Maximum execution time. 0 means no timeout

## Output

The component emits to one of two channels based on the script's exit code:

- **success**: Exit code is 0 - script completed successfully
- **failed**: Exit code is non-zero - script failed

Output includes:
- **stdout**: Standard output from the script
- **stderr**: Standard error output from the script
- **exitCode**: The exit code returned by the script`
}

func (e *ExecuteScript) Icon() string {
	return "file-code"
}

func (e *ExecuteScript) Color() string {
	return "purple"
}

func (e *ExecuteScript) ExampleOutput() map[string]any {
	return map[string]any{
		"result": map[string]any{
			"stdout":   "Script output line 1\nScript output line 2\n",
			"stderr":   "",
			"exitCode": 0,
		},
	}
}

func (e *ExecuteScript) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  ExecuteScriptSuccessChannel,
			Label: "Success",
		},
		{
			Name:  ExecuteScriptFailedChannel,
			Label: "Failed",
		},
	}
}

func (e *ExecuteScript) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "host",
			Label:    "Host",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "host",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "script",
			Label:       "Script",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Script content to execute (multi-line supported)",
			Placeholder: "#!/bin/bash\necho 'Hello, World!'",
		},
		{
			Name:        "interpreter",
			Label:       "Interpreter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Script interpreter (bash, sh, python, python3, etc.)",
			Placeholder: "bash",
			Default:     "bash",
		},
		{
			Name:        "workingDirectory",
			Label:       "Working Directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Working directory for script execution",
			Placeholder: "e.g. /home/user",
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Script timeout in seconds (0 for no timeout)",
			Default:     0,
		},
	}
}

func (e *ExecuteScript) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExecuteScript) Setup(ctx core.SetupContext) error {
	spec := ExecuteScriptSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Host == "" {
		return fmt.Errorf("host is required")
	}

	if spec.Script == "" {
		return fmt.Errorf("script is required")
	}

	return nil
}

func (e *ExecuteScript) Execute(ctx core.ExecutionContext) error {
	spec := ExecuteScriptSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("error decoding configuration: %v", err),
		)
	}

	username, host, port, err := parseHostIdentifier(spec.Host)
	if err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("invalid host format (expected user@host:port): %v", err),
		)
	}

	privateKey, err := ctx.Integration.GetConfig("privateKey")
	if err != nil || len(privateKey) == 0 {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			"privateKey is required in SSH integration configuration",
		)
	}

	passphrase, _ := ctx.Integration.GetConfig("passphrase")

	sshCfg := Configuration{
		Host:       host,
		Port:       port,
		Username:   username,
		PrivateKey: string(privateKey),
		Passphrase: string(passphrase),
	}

	client, err := NewClientFromConfig(sshCfg)
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to create SSH client: %v", err))
	}
	defer client.Close()

	// Set default interpreter
	interpreter := spec.Interpreter
	if interpreter == "" {
		interpreter = "bash"
	}

	// Build script command with working directory if specified
	script := spec.Script
	if spec.WorkingDirectory != "" {
		script = fmt.Sprintf("cd %s\n%s", spec.WorkingDirectory, script)
	}

	// Set timeout
	timeout := time.Duration(0)
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}

	ctx.Logger.Infof("Executing script on SSH host using %s", interpreter)

	result, err := client.ExecuteScript(script, interpreter, timeout)
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to execute script: %v", err))
	}

	// Store result in metadata
	err = ctx.Metadata.Set(ExecuteScriptExecutionMetadata{
		Result: result,
	})
	if err != nil {
		return ctx.ExecutionState.Fail(models.WorkflowNodeExecutionResultReasonError, fmt.Sprintf("failed to set metadata: %v", err))
	}

	// Emit to appropriate channel based on exit code
	if result.ExitCode == 0 {
		return ctx.ExecutionState.Emit(ExecuteScriptSuccessChannel, "ssh.script.executed", []any{result})
	}

	return ctx.ExecutionState.Emit(ExecuteScriptFailedChannel, "ssh.script.failed", []any{result})
}

func (e *ExecuteScript) Cancel(ctx core.ExecutionContext) error {
	// SSH scripts can't be easily cancelled, but we can close the connection
	return nil
}

func (e *ExecuteScript) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// SSH doesn't handle webhooks
	return 404, fmt.Errorf("SSH executeScript does not handle webhooks")
}

func (e *ExecuteScript) Actions() []core.Action {
	return []core.Action{}
}

func (e *ExecuteScript) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for executeScript")
}
