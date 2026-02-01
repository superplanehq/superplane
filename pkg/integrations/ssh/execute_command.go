package ssh

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const ExecuteCommandSuccessChannel = "success"
const ExecuteCommandFailedChannel = "failed"

type ExecuteCommand struct{}

type ExecuteCommandSpec struct {
	Host             string `json:"host"`
	Command          string `json:"command"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	Timeout          int    `json:"timeout,omitempty"`
}

type ExecuteCommandExecutionMetadata struct {
	Result *CommandResult `json:"result" mapstructure:"result"`
}

func (c *ExecuteCommand) Name() string {
	return "ssh.executeCommand"
}

func (c *ExecuteCommand) Label() string {
	return "Execute Command"
}

func (c *ExecuteCommand) Description() string {
	return "Execute a single command on a remote host via SSH"
}

func (c *ExecuteCommand) Documentation() string {
	return `Executes a single command on a remote host via SSH.

## Configuration
- **Host**: Select a host resource (format: user@host:port)
- **Command**: Command to run
- **Working Directory**: Optional directory (runs "cd <dir> && <command>")
- **Timeout (seconds)**: 0 means no timeout

## Output
- **success**: exitCode == 0
- **failed**: exitCode != 0
`
}

func (c *ExecuteCommand) Icon() string  { return "terminal" }
func (c *ExecuteCommand) Color() string { return "blue" }

func (c *ExecuteCommand) ExampleOutput() map[string]any {
	return map[string]any{
		"result": map[string]any{
			"stdout":   "Hello, World!\n",
			"stderr":   "",
			"exitCode": 0,
		},
	}
}

func (c *ExecuteCommand) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ExecuteCommandSuccessChannel, Label: "Success"},
		{Name: ExecuteCommandFailedChannel, Label: "Failed"},
	}
}

func (c *ExecuteCommand) Configuration() []configuration.Field {
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
			Name:        "command",
			Label:       "Command",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Command to execute on the remote host",
			Placeholder: "e.g. ls -la /tmp",
		},
		{
			Name:        "workingDirectory",
			Label:       "Working Directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Working directory for command execution",
			Placeholder: "e.g. /home/user",
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Command timeout in seconds (0 for no timeout)",
			Default:     0,
		},
	}
}

func (c *ExecuteCommand) Setup(ctx core.SetupContext) error {
	spec := ExecuteCommandSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Host == "" {
		return errors.New("host is required")
	}
	if spec.Command == "" {
		return errors.New("command is required")
	}

	// Validate that host is parseable (user@host:port)
	if _, _, _, err := parseHostIdentifier(spec.Host); err != nil {
		return fmt.Errorf("invalid host format (expected user@host:port): %v", err)
	}

	return nil
}

func (c *ExecuteCommand) Execute(ctx core.ExecutionContext) error {
	spec := ExecuteCommandSpec{}
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
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("error creating SSH client: %v", err),
		)
	}
	defer func() { _ = client.Close() }()

	// Build command with working directory if specified
	command := spec.Command
	if spec.WorkingDirectory != "" {
		command = fmt.Sprintf("cd %s && %s", spec.WorkingDirectory, command)
	}

	// Timeout
	var timeout time.Duration
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}

	ctx.Logger.Infof("Executing SSH command on %s: %s", spec.Host, command)

	result, err := client.ExecuteCommand(command, timeout)
	if err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("failed to execute command: %v", err),
		)
	}

	if err := ctx.Metadata.Set(ExecuteCommandExecutionMetadata{Result: result}); err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("failed to set metadata: %v", err),
		)
	}

	if result.ExitCode == 0 {
		return ctx.ExecutionState.Emit(ExecuteCommandSuccessChannel, "ssh.command.executed", []any{result})
	}
	return ctx.ExecutionState.Emit(ExecuteCommandFailedChannel, "ssh.command.failed", []any{result})
}

func (c *ExecuteCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ExecuteCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ExecuteCommand) Actions() []core.Action {
	return []core.Action{}
}

func (c *ExecuteCommand) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for executeCommand")
}

func (c *ExecuteCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("SSH executeCommand does not handle webhooks")
}

// parseHostIdentifier expects "user@host:port" (port optional -> defaults to 22).
// Supports IPv6 as "user@[::1]:2222".
func parseHostIdentifier(v string) (username, host string, port int, err error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", "", 0, errors.New("empty host value")
	}

	at := strings.Index(v, "@")
	if at <= 0 {
		return "", "", 0, errors.New("missing username (expected user@host:port)")
	}
	username = v[:at]
	rest := v[at+1:]

	// Default port
	port = 22

	// IPv6 in brackets: [::1]:2222 or [::1] (without port)
	if strings.HasPrefix(rest, "[") {
		// Check if there's a port after the closing bracket
		closeBracket := strings.Index(rest, "]")
		if closeBracket == -1 {
			return "", "", 0, errors.New("invalid IPv6 format: missing closing bracket")
		}

		host = rest[1:closeBracket] // Extract IPv6 address without brackets
		afterBracket := rest[closeBracket+1:]

		if afterBracket == "" {
			// No port specified, use default
			return username, host, port, nil
		}

		if !strings.HasPrefix(afterBracket, ":") {
			return "", "", 0, errors.New("invalid format after IPv6 address")
		}

		portStr := afterBracket[1:]
		if portStr == "" {
			return "", "", 0, errors.New("empty port after colon")
		}

		pi, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			return "", "", 0, fmt.Errorf("invalid port: %v", convErr)
		}
		return username, host, pi, nil
	}

	// Try host:port by last colon
	lastColon := strings.LastIndex(rest, ":")
	if lastColon > 0 && lastColon < len(rest)-1 {
		maybePort := rest[lastColon+1:]
		if pi, convErr := strconv.Atoi(maybePort); convErr == nil {
			host = rest[:lastColon]
			port = pi
			if host == "" {
				return "", "", 0, errors.New("empty host")
			}
			return username, host, port, nil
		}
	}

	// No port provided
	host = rest
	if host == "" {
		return "", "", 0, errors.New("empty host")
	}
	return username, host, port, nil
}
