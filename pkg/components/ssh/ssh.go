package ssh

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// parseSecretKeyValue splits "secretRef::keyName" into (secretRef, keyName). Returns empty strings if invalid.
func parseSecretKeyValue(v string) (secretRef, keyName string) {
	v = strings.TrimSpace(v)
	idx := strings.Index(v, secretKeyDelimiter)
	if idx < 0 {
		return "", ""
	}
	return strings.TrimSpace(v[:idx]), strings.TrimSpace(v[idx+len(secretKeyDelimiter):])
}

const (
	channelSuccess = "success"
	channelFailed  = "failed"
)

func init() {
	registry.RegisterComponent("ssh", &SSHCommand{})
}

const secretKeyDelimiter = "::"

type SSHCommand struct{}

// AuthSpec is the authentication config group (SSH key or password, credential references).
type AuthSpec struct {
	Method     string `json:"authMethod" mapstructure:"authMethod"`
	PrivateKey string `json:"privateKey" mapstructure:"privateKey"` // stores "secretRef::keyName" for SSH key
	Passphrase string `json:"passphrase" mapstructure:"passphrase"`   // optional, "secretRef::keyName"
	Password   string `json:"password" mapstructure:"password"`       // stores "secretRef::keyName" for password auth
}

type Spec struct {
	Host             string   `json:"host" mapstructure:"host"`
	Port             int      `json:"port" mapstructure:"port"`
	User             string   `json:"username" mapstructure:"username"`
	Authentication   AuthSpec `json:"authentication" mapstructure:"authentication"`
	Command          string   `json:"command" mapstructure:"command"`
	WorkingDirectory string   `json:"workingDirectory,omitempty" mapstructure:"workingDirectory"`
	Timeout          int      `json:"timeout" mapstructure:"timeout"` // command timeout in seconds (default 60)
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
- **Working directory**: Optional; runs "cd <dir> && <command>".
- **Timeout (seconds)**: How long the command may run (default 60).

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
			Description: "Hostname or IP address of the SSH server",
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
			Description: "User to log in as on the remote host",
			Placeholder: "e.g. root, ubuntu",
			Required:    true,
		},
		{
			Name:        "authentication",
			Label:       "Authentication",
			Type:        configuration.FieldTypeObject,
			Description: "How to authenticate to the host and which credentials to use",
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "authMethod",
							Label:       "Method",
							Type:        configuration.FieldTypeSelect,
							Description: "Authentication method",
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
							Name:                 "privateKey",
							Label:                "Private key",
							Type:                 configuration.FieldTypeSecretKey,
							Description:          "Stored credential that holds the SSH private key (PEM/OpenSSH)",
							Required:             false,
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodSSHKey}}},
							VisibilityConditions: sshKeyOnly,
						},
						{
							Name:                 "passphrase",
							Label:                "Passphrase",
							Type:                 configuration.FieldTypeSecretKey,
							Description:          "Stored credential for the key passphrase, if the key is encrypted",
							Required:             false,
							VisibilityConditions: sshKeyOnly,
						},
						{
							Name:                 "password",
							Label:                "Password",
							Type:                 configuration.FieldTypeSecretKey,
							Description:          "Stored credential that holds the login password",
							Required:             false,
							RequiredConditions:   []configuration.RequiredCondition{{Field: "authMethod", Values: []string{AuthMethodPassword}}},
							VisibilityConditions: passwordOnly,
						},
					},
				},
			},
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
			Required:    true,
			Default:     60,
			Description: "Limit how long the command may run (seconds).",
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
	if spec.Timeout < 1 {
		return errors.New("timeout is required and must be at least 1 second")
	}

	switch spec.Authentication.Method {
	case AuthMethodSSHKey:
		ref, key := parseSecretKeyValue(spec.Authentication.PrivateKey)
		if ref == "" || key == "" {
			return errors.New("for SSH key auth, private key credential is required")
		}
	case AuthMethodPassword:
		ref, key := parseSecretKeyValue(spec.Authentication.Password)
		if ref == "" || key == "" {
			return errors.New("for password auth, password credential is required")
		}
	default:
		return fmt.Errorf("invalid auth method: %s", spec.Authentication.Method)
	}

	return nil
}

func (c *SSHCommand) Execute(ctx core.ExecutionContext) error {
	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if spec.Host == "" || spec.User == "" || spec.Command == "" {
		return fmt.Errorf("host, username, and command are required")
	}

	if ctx.Secrets == nil {
		return fmt.Errorf("secrets context not available")
	}

	port := spec.Port
	if port == 0 {
		port = 22
	}

	var client *Client
	switch spec.Authentication.Method {
	case AuthMethodSSHKey:
		secretRef, keyName := parseSecretKeyValue(spec.Authentication.PrivateKey)
		if secretRef == "" || keyName == "" {
			return fmt.Errorf("private key credential is required")
		}
		privateKey, err := ctx.Secrets.GetKey(secretRef, keyName)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return fmt.Errorf("private key could not be resolved from the selected credential")
			}
			return fmt.Errorf("resolve private key: %w", err)
		}

		var passphrase []byte
		if spec.Authentication.Passphrase != "" {
			pref, pkey := parseSecretKeyValue(spec.Authentication.Passphrase)
			if pref != "" && pkey != "" {
				passphrase, _ = ctx.Secrets.GetKey(pref, pkey)
			}
		}

		client = NewClientKey(spec.Host, port, spec.User, privateKey, passphrase)

	case AuthMethodPassword:
		secretRef, keyName := parseSecretKeyValue(spec.Authentication.Password)
		if secretRef == "" || keyName == "" {
			return fmt.Errorf("password credential is required")
		}
		password, err := ctx.Secrets.GetKey(secretRef, keyName)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return fmt.Errorf("password could not be resolved from the selected credential")
			}
			return fmt.Errorf("resolve password: %w", err)
		}

		client = NewClientPassword(spec.Host, port, spec.User, password)

	default:
		return fmt.Errorf("invalid auth method: %s", spec.Authentication.Method)
	}

	defer func() { _ = client.Close() }()

	command := spec.Command
	if spec.WorkingDirectory != "" {
		command = fmt.Sprintf("cd %s && %s", spec.WorkingDirectory, command)
	}

	timeoutSec := spec.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	timeout := time.Duration(timeoutSec) * time.Second

	ctx.Logger.Infof("Executing SSH command on %s@%s:%d: %s", spec.User, spec.Host, port, command)

	result, err := client.ExecuteCommand(command, timeout)
	if err != nil {
		return fmt.Errorf("SSH execution failed: %w", err)
	}

	if err := ctx.Metadata.Set(ExecutionMetadata{Result: result}); err != nil {
		return fmt.Errorf("set metadata: %w", err)
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
