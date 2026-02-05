package ssh

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	channelSuccess = "success"
	channelFailed  = "failed"
)

func init() {
	registry.RegisterComponent("ssh", &SSHCommand{})
}

type SSHCommand struct{}

type Spec struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"username"`
	Method string `json:"authMethod"`

	PrivateKeySecretRef  string `json:"privateKeySecretRef,omitempty"`
	PrivateKeyKeyName    string `json:"privateKeyKeyName,omitempty"`
	PassphraseSecretRef  string `json:"passphraseSecretRef,omitempty"`
	PassphraseKeyName    string `json:"passphraseKeyName,omitempty"`
	PasswordSecretRef    string `json:"passwordSecretRef,omitempty"`
	PasswordKeyName      string `json:"passwordKeyName,omitempty"`
	Command              string `json:"command"`
	WorkingDirectory     string `json:"workingDirectory,omitempty"`
	Timeout              int    `json:"timeout,omitempty"`
}

type ExecutionMetadata struct {
	Result *CommandResult `json:"result" mapstructure:"result"`
}

func (c *SSHCommand) Name() string  { return "ssh" }
func (c *SSHCommand) Label() string { return "SSH Command" }
func (c *SSHCommand) Description() string {
	return "Run a command on a remote host via SSH. Authenticate using an organization Secret (SSH key or password)."
}
func (c *SSHCommand) Documentation() string {
	return `Run a single command on a remote host via SSH.

## Authentication

Choose **SSH key** or **Password**, then select the organization Secret and the key name within that secret that holds the credential.

- **SSH key**: Secret key containing the private key (PEM/OpenSSH). Optionally a second secret+key for passphrase if the key is encrypted.
- **Password**: Secret key containing the password.

## Configuration

- **Host**, **Port** (default 22), **Username**: Connection details.
- **Command**: The command to run (supports expressions).
- **Working directory**: Optional; runs \`cd <dir> && <command>\`.
- **Timeout (seconds)**: 0 = no timeout.

## Output

- **success**: Exit code 0
- **failed**: Non-zero exit code
`
}
func (c *SSHCommand) Icon() string  { return "terminal" }
func (c *SSHCommand) Color() string { return "blue" }

func (c *SSHCommand) ExampleOutput() map[string]any {
	return map[string]any{
		"result": map[string]any{
			"stdout":   "Hello, World!\n",
			"stderr":   "",
			"exitCode": 0,
		},
	}
}

func (c *SSHCommand) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelSuccess, Label: "Success"},
		{Name: channelFailed, Label: "Failed"},
	}
}

func (c *SSHCommand) Configuration() []configuration.Field {
	sshKeyOnly := []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{AuthMethodSSHKey}}}
	passwordOnly := []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{AuthMethodPassword}}}

	return []configuration.Field{
		{
			Name:        "host",
			Label:       "Host",
			Type:        configuration.FieldTypeString,
			Description: "SSH hostname or IP address",
			Placeholder: "e.g. example.com or 192.168.1.100",
			Required:    true,
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeNumber,
			Description: "SSH port",
			Placeholder: "22",
			Default:     22,
			Required:    false,
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Description: "SSH username",
			Placeholder: "e.g. root, ubuntu",
			Required:    true,
		},
		{
			Name:        "authMethod",
			Label:       "Authentication",
			Type:        configuration.FieldTypeSelect,
			Description: "How to authenticate to the host",
			Required:    true,
			Default:     AuthMethodSSHKey,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "SSH key", Value: AuthMethodSSHKey},
						{Label: "Password", Value: AuthMethodPassword},
					},
				},
			},
		},
		{
			Name:                 "privateKeySecretRef",
			Label:                "Secret (private key)",
			Type:                 configuration.FieldTypeString,
			Description:          "Organization Secret name or ID that contains the private key",
			Placeholder:          "e.g. prod-ssh",
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodSSHKey}}},
			VisibilityConditions: sshKeyOnly,
		},
		{
			Name:                 "privateKeyKeyName",
			Label:                "Key name (private key)",
			Type:                 configuration.FieldTypeString,
			Description:          "Key name within the secret that holds the private key value",
			Placeholder:          "e.g. private_key",
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodSSHKey}}},
			VisibilityConditions: sshKeyOnly,
		},
		{
			Name:                 "passphraseSecretRef",
			Label:                "Secret (passphrase, optional)",
			Type:                 configuration.FieldTypeString,
			Description:          "Secret containing passphrase for encrypted key",
			Required:             false,
			VisibilityConditions: sshKeyOnly,
		},
		{
			Name:                 "passphraseKeyName",
			Label:                "Key name (passphrase)",
			Type:                 configuration.FieldTypeString,
			Description:          "Key name for passphrase value",
			Required:             false,
			VisibilityConditions: sshKeyOnly,
		},
		{
			Name:                 "passwordSecretRef",
			Label:                "Secret (password)",
			Type:                 configuration.FieldTypeString,
			Description:          "Organization Secret that contains the password",
			Placeholder:          "e.g. prod-ssh",
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodPassword}}},
			VisibilityConditions: passwordOnly,
		},
		{
			Name:                 "passwordKeyName",
			Label:                "Key name (password)",
			Type:                 configuration.FieldTypeString,
			Description:          "Key name within the secret that holds the password",
			Placeholder:          "e.g. password",
			Required:             false,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodPassword}}},
			VisibilityConditions: passwordOnly,
		},
		{
			Name:        "command",
			Label:       "Command",
			Type:        configuration.FieldTypeString,
			Description: "Command to run on the remote host",
			Placeholder: "e.g. ls -la /tmp",
			Required:    true,
		},
		{
			Name:        "workingDirectory",
			Label:       "Working directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Change to this directory before running the command",
			Placeholder: "e.g. /home/user",
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Command timeout; 0 = no timeout",
			Default:     0,
		},
	}
}

func (c *SSHCommand) Setup(ctx core.SetupContext) error {
	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if spec.Host == "" {
		return errors.New("host is required")
	}
	if spec.User == "" {
		return errors.New("username is required")
	}
	if spec.Command == "" {
		return errors.New("command is required")
	}
	if spec.Port != 0 && (spec.Port < 1 || spec.Port > 65535) {
		return fmt.Errorf("invalid port: %d", spec.Port)
	}

	switch spec.Method {
	case AuthMethodSSHKey:
		if spec.PrivateKeySecretRef == "" || spec.PrivateKeyKeyName == "" {
			return errors.New("for SSH key auth, secret and key name for the private key are required")
		}
	case AuthMethodPassword:
		if spec.PasswordSecretRef == "" || spec.PasswordKeyName == "" {
			return errors.New("for password auth, secret and key name for the password are required")
		}
	default:
		return fmt.Errorf("invalid auth method: %s", spec.Method)
	}

	return nil
}

func (c *SSHCommand) Execute(ctx core.ExecutionContext) error {
	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("decode configuration: %v", err),
		)
	}

	if spec.Host == "" || spec.User == "" || spec.Command == "" {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			"host, username, and command are required",
		)
	}

	if ctx.Secrets == nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			"secrets context not available",
		)
	}

	port := spec.Port
	if port == 0 {
		port = 22
	}

	var client *Client
	switch spec.Method {
	case AuthMethodSSHKey:
		privateKey, err := ctx.Secrets.GetKey(spec.PrivateKeySecretRef, spec.PrivateKeyKeyName)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return ctx.ExecutionState.Fail(
					models.WorkflowNodeExecutionResultReasonError,
					"private key could not be resolved from the selected secret and key",
				)
			}
			return ctx.ExecutionState.Fail(
				models.WorkflowNodeExecutionResultReasonError,
				fmt.Sprintf("resolve private key: %v", err),
			)
		}

		var passphrase []byte
		if spec.PassphraseSecretRef != "" && spec.PassphraseKeyName != "" {
			passphrase, _ = ctx.Secrets.GetKey(spec.PassphraseSecretRef, spec.PassphraseKeyName)
		}

		client = NewClientKey(spec.Host, port, spec.User, privateKey, passphrase)

	case AuthMethodPassword:
		password, err := ctx.Secrets.GetKey(spec.PasswordSecretRef, spec.PasswordKeyName)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return ctx.ExecutionState.Fail(
					models.WorkflowNodeExecutionResultReasonError,
					"password could not be resolved from the selected secret and key",
				)
			}
			return ctx.ExecutionState.Fail(
				models.WorkflowNodeExecutionResultReasonError,
				fmt.Sprintf("resolve password: %v", err),
			)
		}

		client = NewClientPassword(spec.Host, port, spec.User, password)

	default:
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("invalid auth method: %s", spec.Method),
		)
	}

	defer func() { _ = client.Close() }()

	command := spec.Command
	if spec.WorkingDirectory != "" {
		command = fmt.Sprintf("cd %s && %s", spec.WorkingDirectory, command)
	}

	var timeout time.Duration
	if spec.Timeout > 0 {
		timeout = time.Duration(spec.Timeout) * time.Second
	}

	ctx.Logger.Infof("Executing SSH command on %s@%s:%d: %s", spec.User, spec.Host, port, command)

	result, err := client.ExecuteCommand(command, timeout)
	if err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("SSH execution failed: %v", err),
		)
	}

	if err := ctx.Metadata.Set(ExecutionMetadata{Result: result}); err != nil {
		return ctx.ExecutionState.Fail(
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("set metadata: %v", err),
		)
	}

	if result.ExitCode == 0 {
		return ctx.ExecutionState.Emit(channelSuccess, "ssh.command.executed", []any{result})
	}
	return ctx.ExecutionState.Emit(channelFailed, "ssh.command.failed", []any{result})
}

func (c *SSHCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SSHCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SSHCommand) Actions() []core.Action {
	return nil
}

func (c *SSHCommand) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined for ssh")
}

func (c *SSHCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("SSH component does not handle webhooks")
}

func (c *SSHCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}
