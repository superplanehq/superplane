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

// SecretKeyRef is stored in YAML as: { secret: "name", key: "keyName" }.
type SecretKeyRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

func (r SecretKeyRef) IsSet() bool {
	return r.Secret != "" && r.Key != ""
}

// AuthSpec is the authentication config group (SSH key or password, credential references).
type AuthSpec struct {
	Method     string       `json:"authMethod" mapstructure:"authMethod"`
	PrivateKey SecretKeyRef `json:"privateKey" mapstructure:"privateKey"`
	Passphrase SecretKeyRef `json:"passphrase" mapstructure:"passphrase"`
	Password   SecretKeyRef `json:"password" mapstructure:"password"`
}

// ConnectionRetrySpec is the optional connection retry config group (togglable).
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
	ConnectionRetry *ConnectionRetrySpec `json:"connectionRetry,omitempty" mapstructure:"connectionRetry"`
}

type ExecutionMetadata struct {
	Result *CommandResult `json:"result" mapstructure:"result"`
}

// ConnectionRetryState is stored in metadata when scheduling a connection retry action (so we don't sit idle).
type ConnectionRetryState struct {
	Attempt         int `json:"attempt" mapstructure:"attempt"`                   // retries done so far (1 = first retry)
	MaxRetries      int `json:"maxRetries" mapstructure:"maxRetries"`              // max retries from config
	IntervalSeconds int `json:"intervalSeconds" mapstructure:"intervalSeconds"`     // seconds between attempts
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
	spec, err := c.decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if ctx.Secrets == nil {
		return fmt.Errorf("secrets context not available")
	}
	return c.executeSSH(ctx, spec)
}

// executeSSH runs one connect+command attempt. On connect error with retries enabled it schedules
// a connectionRetry action and returns nil (execution does not sit idle). On success or non-retryable error it sets state and returns.
func (c *SSHCommand) executeSSH(ctx core.ExecutionContext, spec Spec) error {
	port := spec.Port
	if port == 0 {
		port = 22
	}

	var currentAttempt, maxRetries, intervalSec int
	if meta := ctx.Metadata.Get(); meta != nil {
		var state ConnectionRetryState
		if err := mapstructure.Decode(meta, &state); err == nil && state.MaxRetries > 0 {
			currentAttempt = state.Attempt
			maxRetries = state.MaxRetries
			intervalSec = state.IntervalSeconds
			if intervalSec <= 0 {
				intervalSec = 15
			}
		}
	}
	if maxRetries == 0 && spec.ConnectionRetry != nil && spec.ConnectionRetry.Enabled && spec.ConnectionRetry.Retries > 0 {
		maxRetries = spec.ConnectionRetry.Retries
		intervalSec = spec.ConnectionRetry.IntervalSeconds
		if intervalSec <= 0 {
			intervalSec = 15
		}
	}

	result, err := c.runOneAttempt(ctx, spec, port)
	if err == nil {
		if err := ctx.Metadata.Set(ExecutionMetadata{Result: result}); err != nil {
			return fmt.Errorf("set metadata: %w", err)
		}
		if result.ExitCode == 0 {
			return ctx.ExecutionState.Emit(channelSuccess, "ssh.command.executed", []any{result})
		}
		return ctx.ExecutionState.Emit(channelFailed, "ssh.command.failed", []any{result})
	}

	if isConnectError(err) && currentAttempt < maxRetries {
		nextAttempt := currentAttempt + 1
		ctx.Logger.Infof("Connect attempt failed (%v), scheduling connection retry %d/%d in %ds", err, nextAttempt, maxRetries, intervalSec)
		state := ConnectionRetryState{Attempt: nextAttempt, MaxRetries: maxRetries, IntervalSeconds: intervalSec}
		if setErr := ctx.Metadata.Set(state); setErr != nil {
			return fmt.Errorf("set retry metadata: %w", setErr)
		}
		return ctx.Requests.ScheduleActionCall("connectionRetry", map[string]any{}, time.Duration(intervalSec)*time.Second)
	}

	return fmt.Errorf("SSH execution failed: %w", err)
}

// runOneAttempt builds the SSH client, runs the command once, and returns the result or error.
func (c *SSHCommand) runOneAttempt(ctx core.ExecutionContext, spec Spec, port int) (*CommandResult, error) {
	var client *Client
	switch spec.Authentication.Method {
	case AuthMethodSSHKey:
		if !spec.Authentication.PrivateKey.IsSet() {
			return nil, fmt.Errorf("private key credential is required")
		}
		privateKey, err := ctx.Secrets.GetKey(spec.Authentication.PrivateKey.Secret, spec.Authentication.PrivateKey.Key)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return nil, fmt.Errorf("private key could not be resolved from the selected credential")
			}
			return nil, fmt.Errorf("resolve private key: %w", err)
		}
		var passphrase []byte
		if spec.Authentication.Passphrase.IsSet() {
			passphrase, _ = ctx.Secrets.GetKey(spec.Authentication.Passphrase.Secret, spec.Authentication.Passphrase.Key)
		}
		client = NewClientKey(spec.Host, port, spec.User, privateKey, passphrase)
	case AuthMethodPassword:
		if !spec.Authentication.Password.IsSet() {
			return nil, fmt.Errorf("password credential is required")
		}
		password, err := ctx.Secrets.GetKey(spec.Authentication.Password.Secret, spec.Authentication.Password.Key)
		if err != nil {
			if errors.Is(err, core.ErrSecretKeyNotFound) {
				return nil, fmt.Errorf("password could not be resolved from the selected credential")
			}
			return nil, fmt.Errorf("resolve password: %w", err)
		}
		client = NewClientPassword(spec.Host, port, spec.User, password)
	default:
		return nil, fmt.Errorf("invalid auth method: %s", spec.Authentication.Method)
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
	return client.ExecuteCommand(command, timeout)
}

func (c *SSHCommand) decodeSpec(cfg any) (Spec, error) {
	var spec Spec
	config, ok := cfg.(map[string]any)
	if !ok || config == nil {
		return spec, fmt.Errorf("decode configuration: invalid configuration type")
	}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("decode configuration: %w", err)
	}
	if spec.Host == "" || spec.User == "" || spec.Command == "" {
		return spec, fmt.Errorf("host, username, and command are required")
	}
	return spec, nil
}

// isConnectError returns true for errors that indicate the SSH connection could not be established
// (e.g. host unreachable, timeout, connection refused). Used to decide whether to retry.
func isConnectError(err error) bool {
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

func (c *SSHCommand) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "connectionRetry" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}
	if ctx.Secrets == nil {
		return fmt.Errorf("secrets context not available for connection retry")
	}
	spec, err := c.decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	execCtx := core.ExecutionContext{
		Configuration:  ctx.Configuration,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Logger:         ctx.Logger,
		Secrets:        ctx.Secrets,
	}
	return c.executeSSH(execCtx, spec)
}

func (c *SSHCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("SSH component does not handle webhooks")
}

func (c *SSHCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}
