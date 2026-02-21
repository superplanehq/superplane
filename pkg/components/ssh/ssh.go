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

const (
	channelSuccess = "success"
	channelFailed  = "failed"
)

func init() {
	registry.RegisterComponent("ssh", &SSHCommand{})
}

type SSHCommand struct{}

type SecretKeyRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

func (r SecretKeyRef) IsSet() bool {
	return r.Secret != "" && r.Key != ""
}

type AuthSpec struct {
	Method     string       `json:"authMethod" mapstructure:"authMethod"`
	PrivateKey SecretKeyRef `json:"privateKey" mapstructure:"privateKey"`
	Passphrase SecretKeyRef `json:"passphrase" mapstructure:"passphrase"`
	Password   SecretKeyRef `json:"password" mapstructure:"password"`
}

type ConnectionRetrySpec struct {
	Enabled         bool `json:"enabled" mapstructure:"enabled"`
	Retries         int  `json:"retries" mapstructure:"retries"`
	IntervalSeconds int  `json:"intervalSeconds" mapstructure:"intervalSeconds"`
}

type Spec struct {
	Host             string               `json:"host" mapstructure:"host"`
	Port             int                  `json:"port" mapstructure:"port"`
	User             string               `json:"username" mapstructure:"username"`
	Authentication   AuthSpec             `json:"authentication" mapstructure:"authentication"`
	Command          string               `json:"command" mapstructure:"command"`
	WorkingDirectory string               `json:"workingDirectory,omitempty" mapstructure:"workingDirectory"`
	Timeout          int                  `json:"timeout" mapstructure:"timeout"`
	ConnectionRetry  *ConnectionRetrySpec `json:"connectionRetry,omitempty" mapstructure:"connectionRetry"`
}

type ExecutionMetadata struct {
	Result           *CommandResult       `json:"result" mapstructure:"result"`
	Host             string               `json:"host" mapstructure:"host"`
	Port             int                  `json:"port" mapstructure:"port"`
	User             string               `json:"user" mapstructure:"user"`
	Command          string               `json:"command" mapstructure:"command"`
	WorkingDirectory string               `json:"workingDirectory" mapstructure:"workingDirectory"`
	Timeout          int                  `json:"timeout" mapstructure:"timeout"`
	ConnectionRetry  *ConnectionRetrySpec `json:"connectionRetry" mapstructure:"connectionRetry"`
	Attempt          int                  `json:"attempt" mapstructure:"attempt"`
	MaxRetries       int                  `json:"maxRetries" mapstructure:"maxRetries"`
	IntervalSeconds  int                  `json:"intervalSeconds" mapstructure:"intervalSeconds"`
	Authentication   AuthSpec             `json:"authentication" mapstructure:"authentication"`
}

type ConnectionRetryState struct {
	Attempt         int `json:"attempt" mapstructure:"attempt"`                 // retries done so far (1 = first retry)
	MaxRetries      int `json:"maxRetries" mapstructure:"maxRetries"`           // max retries from config
	IntervalSeconds int `json:"intervalSeconds" mapstructure:"intervalSeconds"` // seconds between attempts
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
- **Working directory**: Optional; Changes to this directory before running the command.
- **Timeout (seconds)**: How long the command may run (default 60).
- **Connection retry** (optional): Enable to retry connecting when the host is not reachable yet (e.g. server still booting). Set number of retries and interval between attempts.

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
		{
			Name:        "connectionRetry",
			Label:       "Connection retry",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optionally retry connecting when the host is unreachable (e.g. server still booting).",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "enabled",
							Label:       "Enable connection retry",
							Type:        configuration.FieldTypeBool,
							Required:    false,
							Default:     false,
							Description: "Retry connecting if the host is not reachable yet.",
						},
						{
							Name:        "retries",
							Label:       "Retries",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     5,
							Description: "Number of retry attempts.",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
						{
							Name:        "intervalSeconds",
							Label:       "Retry interval (seconds)",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Default:     15,
							Description: "Seconds to wait between connect attempts.",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "enabled", Values: []string{"true"}},
							},
						},
					},
				},
			},
		},
	}
}

func (c *SSHCommand) Setup(ctx core.SetupContext) error {
	var spec Spec
	config, ok := ctx.Configuration.(map[string]any)
	if !ok || config == nil {
		return fmt.Errorf("decode configuration: invalid configuration type")
	}
	if err := mapstructure.Decode(config, &spec); err != nil {
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
		if !spec.Authentication.PrivateKey.IsSet() {
			return errors.New("for SSH key auth, private key credential is required")
		}
	case AuthMethodPassword:
		if !spec.Authentication.Password.IsSet() {
			return errors.New("for password auth, password credential is required")
		}
	default:
		return fmt.Errorf("invalid auth method: %s", spec.Authentication.Method)
	}
	if spec.ConnectionRetry != nil && spec.ConnectionRetry.Enabled {
		if spec.ConnectionRetry.Retries < 0 {
			return errors.New("connection retry: retries must be 0 or greater")
		}
		if spec.ConnectionRetry.IntervalSeconds < 1 {
			return errors.New("connection retry: interval must be at least 1 second")
		}
	}

	return nil
}

func (c *SSHCommand) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	metadata := ExecutionMetadata{
		Host:             spec.Host,
		Port:             spec.Port,
		User:             spec.User,
		Command:          spec.Command,
		WorkingDirectory: spec.WorkingDirectory,
		Timeout:          spec.Timeout,
		ConnectionRetry:  spec.ConnectionRetry,
		Attempt:          0,
		MaxRetries:       spec.ConnectionRetry.Retries,
		IntervalSeconds:  spec.ConnectionRetry.IntervalSeconds,
		Authentication:   spec.Authentication,
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	execCtx := ExecuteSSHContext{
		secretsCtx:   ctx.Secrets,
		requestsCtx:  ctx.Requests,
		stateCtx:     ctx.ExecutionState,
		metadataCtx:  ctx.Metadata,
		execMetadata: metadata,
	}

	return c.executeSSH(execCtx)
}

func (c *SSHCommand) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "connectionRetry" {
		if ctx.ExecutionState.IsFinished() {
			return nil
		}

		metadata := ExecutionMetadata{}
		err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		if err != nil {
			return err
		}

		execCtx := ExecuteSSHContext{
			secretsCtx:   ctx.Secrets,
			requestsCtx:  ctx.Requests,
			stateCtx:     ctx.ExecutionState,
			metadataCtx:  ctx.Metadata,
			execMetadata: metadata,
		}

		return c.executeSSH(execCtx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

type ExecuteSSHContext struct {
	secretsCtx  core.SecretsContext
	requestsCtx core.RequestContext
	stateCtx    core.ExecutionStateContext
	metadataCtx core.MetadataContext

	execMetadata ExecutionMetadata
}

func (c *SSHCommand) executeSSH(ctx ExecuteSSHContext) error {
	client, err := c.createClient(ctx.secretsCtx, ctx.execMetadata)
	if err != nil {
		return err
	}
	defer client.Close()

	result, err := client.ExecuteCommand(ctx.execMetadata.Command, time.Duration(ctx.execMetadata.Timeout)*time.Second)
	if c.isConnectError(err) {
		if c.shouldRetry(ctx.execMetadata.ConnectionRetry, ctx.metadataCtx) {
			err = c.incrementRetryCount(ctx.metadataCtx)
			if err != nil {
				return err
			}

			return ctx.requestsCtx.ScheduleActionCall("connectionRetry", map[string]any{}, time.Duration(ctx.execMetadata.ConnectionRetry.IntervalSeconds)*time.Second)
		}

		// Retries exhausted — emit on the failed channel with the connection error.
		attempt := c.getRetryAttempt(ctx.metadataCtx)
		failResult := &CommandResult{
			Stdout:   "",
			Stderr:   fmt.Sprintf("connection failed after %d retries: %s", attempt, err.Error()),
			ExitCode: -1,
		}

		err = c.setResultMetadata(ctx.metadataCtx, failResult)
		if err != nil {
			return err
		}

		return ctx.stateCtx.Emit(channelFailed, "ssh.connection.failed", []any{failResult})
	}

	if err != nil {
		return err
	}

	err = c.setResultMetadata(ctx.metadataCtx, result)
	if err != nil {
		return err
	}

	channel := channelFailed
	if result.ExitCode == 0 {
		channel = channelSuccess
	}

	return ctx.stateCtx.Emit(channel, "ssh.command.executed", []any{result})
}

func (c *SSHCommand) shouldRetry(retrySpec *ConnectionRetrySpec, metadata core.MetadataContext) bool {
	if retrySpec == nil || !retrySpec.Enabled {
		return false
	}

	return c.getRetryAttempt(metadata) < retrySpec.Retries
}

func (c *SSHCommand) incrementRetryCount(metadata core.MetadataContext) error {
	current := c.getMetadataMap(metadata)
	current["attempt"] = c.getRetryAttempt(metadata) + 1

	return metadata.Set(current)
}

func (c *SSHCommand) getRetryAttempt(metadata core.MetadataContext) int {
	meta := c.getMetadataMap(metadata)

	attempt, ok := meta["attempt"]
	if !ok {
		return 0
	}

	// JSON numbers deserialize as float64
	switch v := attempt.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func (c *SSHCommand) getMetadataMap(metadata core.MetadataContext) map[string]any {
	current, ok := metadata.Get().(map[string]any)
	if !ok || current == nil {
		return map[string]any{}
	}

	return current
}

func (c *SSHCommand) setResultMetadata(metadata core.MetadataContext, result *CommandResult) error {
	current := c.getMetadataMap(metadata)
	current["result"] = map[string]any{
		"exitCode": result.ExitCode,
		"stdout":   result.Stdout,
		"stderr":   result.Stderr,
	}

	return metadata.Set(current)
}

func (c *SSHCommand) isConnectError(err error) bool {
	if err == nil {
		return false
	}

	s := strings.ToLower(err.Error())

	return strings.Contains(s, "dial") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "i/o timeout") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "no route to host")
}

func (c *SSHCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SSHCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SSHCommand) Actions() []core.Action {
	return []core.Action{
		{Name: "connectionRetry"},
	}
}

func (c *SSHCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("SSH component does not handle webhooks")
}

func (c *SSHCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SSHCommand) createClient(secrets core.SecretsContext, metadata ExecutionMetadata) (*Client, error) {
	switch metadata.Authentication.Method {
	case AuthMethodSSHKey:
		return c.createClientSSHKey(secrets, metadata)
	case AuthMethodPassword:
		return c.createClientForPassword(secrets, metadata)
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", metadata.Authentication.Method)
	}
}

func (c *SSHCommand) createClientForPassword(secrets core.SecretsContext, metadata ExecutionMetadata) (*Client, error) {
	password, err := secrets.GetKey(metadata.Authentication.Password.Secret, metadata.Authentication.Password.Key)
	if err != nil {
		return nil, fmt.Errorf("cannot get password: %w", err)
	}
	return NewClientPassword(metadata.Host, metadata.Port, metadata.User, password), nil
}

func (c *SSHCommand) createClientSSHKey(secrets core.SecretsContext, metadata ExecutionMetadata) (*Client, error) {
	privateKey, err := secrets.GetKey(metadata.Authentication.PrivateKey.Secret, metadata.Authentication.PrivateKey.Key)
	if err != nil {
		return nil, fmt.Errorf("cannot get private key: %w", err)
	}

	return NewClientKey(metadata.Host, metadata.Port, metadata.User, privateKey, nil), nil
}
