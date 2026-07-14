package ssh

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	channelSuccess = "success"
	channelFailed  = "failed"
)

const (
	CommandSourceInline = "inline"
	CommandSourceFile   = "file"
)

// Cap the size of a command file we are willing to load over SSH so that a
// runaway script cannot blow up worker memory or push past shell argv limits.
const maxCommandFileSize = 256 * 1024

var environmentVariableNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func init() {
	registry.RegisterAction("ssh", &SSHCommand{})
}

type SSHCommand struct{}

type AuthSpec struct {
	Method     string                     `json:"authMethod" mapstructure:"authMethod"`
	PrivateKey configuration.SecretKeyRef `json:"privateKey" mapstructure:"privateKey"`
	Passphrase configuration.SecretKeyRef `json:"passphrase" mapstructure:"passphrase"`
	Password   configuration.SecretKeyRef `json:"password" mapstructure:"password"`
}

type RetrySpec struct {
	Enabled         bool `json:"enabled" mapstructure:"enabled"`
	Retries         int  `json:"retries" mapstructure:"retries"`
	IntervalSeconds int  `json:"intervalSeconds" mapstructure:"intervalSeconds"`
}

type ConnectionRetrySpec = RetrySpec
type ExecutionRetrySpec = RetrySpec

type EnvironmentVariable struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type Spec struct {
	Host             string                `json:"host" mapstructure:"host"`
	Port             int                   `json:"port" mapstructure:"port"`
	User             string                `json:"username" mapstructure:"username"`
	Authentication   AuthSpec              `json:"authentication" mapstructure:"authentication"`
	CommandSource    string                `json:"commandSource,omitempty" mapstructure:"commandSource"`
	Commands         string                `json:"commands" mapstructure:"commands"`
	CommandFile      string                `json:"commandFile,omitempty" mapstructure:"commandFile"`
	WorkingDirectory string                `json:"workingDirectory,omitempty" mapstructure:"workingDirectory"`
	Environment      []EnvironmentVariable `json:"environment,omitempty" mapstructure:"environment"`
	Timeout          int                   `json:"timeout" mapstructure:"timeout"`
	ConnectionRetry  *ConnectionRetrySpec  `json:"connectionRetry,omitempty" mapstructure:"connectionRetry"`
	ExecutionRetry   *ExecutionRetrySpec   `json:"executionRetry,omitempty" mapstructure:"executionRetry"`
}

// commandSourceOrDefault returns the command source for the spec. A truly
// unset (empty or whitespace-only) value defaults to inline so nodes saved
// before the file feature keep working. Any other value is returned verbatim
// and must match a known source exactly. We must NOT trim a non-empty value:
// the UI evaluates the commandFile/commands visibility and required conditions
// with an exact string comparison, so a padded value like "\tfile\n" would be
// treated as hidden there (dropping commandFile from the saved payload) while a
// trimmed value here would still run in file mode on the worker. Returning the
// value verbatim keeps both sides in agreement and makes a padded value fail
// loudly as an invalid command source instead of silently losing the path.
func (s Spec) commandSourceOrDefault() string {
	if strings.TrimSpace(s.CommandSource) == "" {
		return CommandSourceInline
	}
	return s.CommandSource
}

type ExecutionMetadata struct {
	Result           *CommandResult        `json:"result" mapstructure:"result"`
	Host             string                `json:"host" mapstructure:"host"`
	Port             int                   `json:"port" mapstructure:"port"`
	User             string                `json:"user" mapstructure:"user"`
	CommandSource    string                `json:"commandSource" mapstructure:"commandSource"`
	Commands         string                `json:"commands" mapstructure:"commands"`
	CommandFile      string                `json:"commandFile" mapstructure:"commandFile"`
	WorkingDirectory string                `json:"workingDirectory" mapstructure:"workingDirectory"`
	Environment      []EnvironmentVariable `json:"environment" mapstructure:"environment"`
	Timeout          int                   `json:"timeout" mapstructure:"timeout"`
	ConnectionRetry  *ConnectionRetrySpec  `json:"connectionRetry" mapstructure:"connectionRetry"`
	ExecutionRetry   *ExecutionRetrySpec   `json:"executionRetry" mapstructure:"executionRetry"`
	Attempt          int                   `json:"attempt" mapstructure:"attempt"`
	ExecutionAttempt int                   `json:"executionAttempt" mapstructure:"executionAttempt"`
	MaxRetries       int                   `json:"maxRetries" mapstructure:"maxRetries"`
	IntervalSeconds  int                   `json:"intervalSeconds" mapstructure:"intervalSeconds"`
	Authentication   AuthSpec              `json:"authentication" mapstructure:"authentication"`
}

type ConnectionRetryState struct {
	Attempt         int `json:"attempt" mapstructure:"attempt"`                 // retries done so far (1 = first retry)
	MaxRetries      int `json:"maxRetries" mapstructure:"maxRetries"`           // max retries from config
	IntervalSeconds int `json:"intervalSeconds" mapstructure:"intervalSeconds"` // seconds between attempts
}

func (c *SSHCommand) Name() string  { return "ssh" }
func (c *SSHCommand) Label() string { return "SSH Command" }
func (c *SSHCommand) Description() string {
	return "Run one or more commands on a remote host via SSH. Authenticate using an organization Secret (SSH key or password)."
}
func (c *SSHCommand) Documentation() string {
	return `Run one or more commands on a remote host via SSH.

## Authentication

Choose **SSH key** or **Password**, then select the organization Secret and the key name within that secret that holds the credential.

- **SSH key**: Secret key containing the private key (PEM/OpenSSH). Optionally a second secret+key for passphrase if the key is encrypted.
- **Password**: Secret key containing the password.

## Configuration

- **Host**, **Port** (default 22), **Username**: Connection details.
- **Command source**: Choose **Inline** to type commands directly, or **From file** to load them from a file in the app's repository (e.g. scripts/deploy.sh).
- **Commands** (inline mode): One or more commands to run, one per line (supports expressions). Each non-empty line becomes a command joined with &&. The output payload is based on the last command.
- **Command file** (file mode): Path to a file in the app's repository (e.g. ` + "`scripts/deploy.sh`" + `).
- **Working directory**: Optional; Changes to this directory before running the command.
- **Environment variables**: Optional list of key/value pairs available during command execution.
- **Timeout (seconds)**: How long the command may run (default 60).
- **Connection retry** (optional): Enable to retry connecting when the host is not reachable yet (e.g. server still booting). Set number of retries and interval between attempts.
- **Execution retry** (optional): Enable to retry running the command when the exit status is not 0. Set number of retries and interval between attempts.

## Output

- **success**: Exit code 0
- **failed**: Non-zero exit code
`
}
func (c *SSHCommand) Icon() string  { return "terminal" }
func (c *SSHCommand) Color() string { return "blue" }

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
			Name:        "commandSource",
			Label:       "Command source",
			Type:        configuration.FieldTypeSelect,
			Description: "Where the commands come from",
			// Optional at the schema level even though it has a default:
			// configuration.ValidateConfiguration does not apply Field.Default, so
			// requiring it would reject legacy SSH nodes saved before this field
			// existed when their configuration is re-validated or patched. The
			// worker defaults a missing/blank value to inline via
			// commandSourceOrDefault, and the UI uses Default to pre-fill new nodes.
			Required: false,
			Default:  CommandSourceInline,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Inline", Value: CommandSourceInline, Description: "Type commands directly into the node"},
						{Label: "From file", Value: CommandSourceFile, Description: "Load commands from a file in the app's repository"},
					},
				},
			},
		},
		{
			Name:                 "commands",
			Label:                "Commands",
			Type:                 configuration.FieldTypeText,
			Description:          "One or more commands to run on the remote host, one per line",
			Placeholder:          "e.g. echo hello\nls -la /tmp",
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "commandSource", Values: []string{CommandSourceInline}}},
			RequiredConditions:   []configuration.RequiredCondition{{Field: "commandSource", Values: []string{CommandSourceInline}}},
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "shell",
				},
			},
		},
		{
			Name:                 "commandFile",
			Label:                "Command file",
			Type:                 configuration.FieldTypeRepositoryFile,
			Description:          "Path to a file in the app's repository (e.g. scripts/deploy.sh).",
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "commandSource", Values: []string{CommandSourceFile}}},
			RequiredConditions:   []configuration.RequiredCondition{{Field: "commandSource", Values: []string{CommandSourceFile}}},
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
			Name:        "environment",
			Label:       "Environment variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional key/value pairs available to the command",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Description: "Environment variable name (letters, numbers, underscore)",
								Placeholder: "e.g. ENVIRONMENT",
								Required:    true,
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Description: "Environment variable value",
								Placeholder: "e.g. production",
								Required:    true,
							},
						},
					},
				},
			},
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
		{
			Name:        "executionRetry",
			Label:       "Execution retry",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optionally retry running the command when the exit status is not 0.",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "enabled",
							Label:       "Enable execution retry",
							Type:        configuration.FieldTypeBool,
							Required:    false,
							Default:     false,
							Description: "Retry running the command if the exit status is not 0.",
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
							Description: "Seconds to wait between command attempts.",
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
	if err := validateCommandSource(ctx, spec); err != nil {
		return err
	}
	if spec.Port != 0 && (spec.Port < 1 || spec.Port > 65535) {
		return fmt.Errorf("invalid port: %d", spec.Port)
	}
	if spec.Timeout < 1 {
		return errors.New("timeout is required and must be at least 1 second")
	}
	for _, variable := range spec.Environment {
		if variable.Name == "" {
			return errors.New("environment variable name is required")
		}
		if !environmentVariableNameRegex.MatchString(variable.Name) {
			return fmt.Errorf("invalid environment variable name: %s", variable.Name)
		}
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
	if spec.ExecutionRetry != nil && spec.ExecutionRetry.Enabled {
		if spec.ExecutionRetry.Retries < 0 {
			return errors.New("execution retry: retries must be 0 or greater")
		}
		if spec.ExecutionRetry.IntervalSeconds < 1 {
			return errors.New("execution retry: interval must be at least 1 second")
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

	source := spec.commandSourceOrDefault()

	// Inline mode: the (already template-resolved) commands are stored in
	// metadata so retries don't have to re-evaluate expressions. File mode:
	// metadata only carries the path; the file is re-read from the repo on
	// every attempt so the script content never lives in the database.
	resolvedCommands := ""
	if source == CommandSourceInline {
		if strings.TrimSpace(spec.Commands) == "" {
			return errors.New("commands is required")
		}
		resolvedCommands = spec.Commands
	}

	for _, variable := range spec.Environment {
		if variable.Name == "" {
			return errors.New("environment variable name is required")
		}
		if !environmentVariableNameRegex.MatchString(variable.Name) {
			return fmt.Errorf("invalid environment variable name: %s", variable.Name)
		}
	}

	metadata := ExecutionMetadata{
		Host:             spec.Host,
		Port:             spec.Port,
		User:             spec.User,
		CommandSource:    source,
		Commands:         resolvedCommands,
		CommandFile:      spec.CommandFile,
		WorkingDirectory: spec.WorkingDirectory,
		Environment:      spec.Environment,
		Timeout:          spec.Timeout,
		ConnectionRetry:  spec.ConnectionRetry,
		ExecutionRetry:   spec.ExecutionRetry,
		Attempt:          0,
		ExecutionAttempt: 0,
		MaxRetries:       0,
		IntervalSeconds:  0,
		Authentication:   spec.Authentication,
	}
	if spec.ConnectionRetry != nil {
		metadata.MaxRetries = spec.ConnectionRetry.Retries
		metadata.IntervalSeconds = spec.ConnectionRetry.IntervalSeconds
	}

	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	// File mode: load the script body now so file errors surface before
	// we try to open an SSH session. This re-checks the same size and
	// empty-file guards Setup ran at publish time, in case the file
	// changed in the repo between publish and run.
	scriptBody := ""
	if source == CommandSourceFile {
		scriptBody, err = loadCommandFile(ctx.Files, spec.CommandFile)
		if err != nil {
			return err
		}
	}

	execCtx := ExecuteSSHContext{
		secretsCtx:   ctx.Secrets,
		requestsCtx:  ctx.Requests,
		stateCtx:     ctx.ExecutionState,
		metadataCtx:  ctx.Metadata,
		execMetadata: metadata,
		scriptBody:   scriptBody,
	}

	return c.executeSSH(execCtx)
}

// loadCommandFile reads, size-limits, and validates the command file referenced
// by rawPath, returning its content with shell-safe line endings. It rejects
// empty or whitespace-only files so both Setup (publish time) and Execute (run
// time) agree on what counts as a runnable script.
func loadCommandFile(files core.RepositoryFilesContext, rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", errors.New("command file is required")
	}
	normalized, err := gitprovider.ValidateUserPath(path)
	if err != nil {
		return "", fmt.Errorf("invalid command file %q: %w", path, err)
	}
	if files == nil {
		return "", errors.New("command file configured but file access is not available")
	}
	reader, err := files.Read(normalized)
	if err != nil {
		return "", fmt.Errorf("read command file %q: %w", path, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, maxCommandFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read command file %q: %w", path, err)
	}
	if len(data) > maxCommandFileSize {
		return "", fmt.Errorf("command file %q exceeds maximum size of %d bytes", path, maxCommandFileSize)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("command file %q is empty", path)
	}
	return normalizeScriptLineEndings(string(data)), nil
}

func (c *SSHCommand) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "connectionRetry", "executionRetry":
		if ctx.ExecutionState.IsFinished() {
			return nil
		}

		metadata := ExecutionMetadata{}
		err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
		if err != nil {
			return err
		}

		// Re-read the command file on every retry so the script body
		// never lives in stored metadata. Same guards Execute applies on
		// the initial attempt: a missing file or empty content fails
		// fast before we try to open an SSH session.
		scriptBody := ""
		if metadata.CommandSource == CommandSourceFile {
			scriptBody, err = loadCommandFile(ctx.Files, metadata.CommandFile)
			if err != nil {
				return err
			}
		}

		execCtx := ExecuteSSHContext{
			secretsCtx:   ctx.Secrets,
			requestsCtx:  ctx.Requests,
			stateCtx:     ctx.ExecutionState,
			metadataCtx:  ctx.Metadata,
			execMetadata: metadata,
			scriptBody:   scriptBody,
		}

		return c.executeSSH(execCtx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

type ExecuteSSHContext struct {
	secretsCtx  core.SecretsContext
	requestsCtx core.RequestContext
	stateCtx    core.ExecutionStateContext
	metadataCtx core.MetadataWriter

	execMetadata ExecutionMetadata

	// File-mode script body to stream as stdin. Loaded by the caller so
	// file-system errors surface before the SSH client is created. Empty
	// for inline mode.
	scriptBody string
}

func (c *SSHCommand) executeSSH(ctx ExecuteSSHContext) error {
	client, err := c.createClient(ctx.secretsCtx, ctx.execMetadata)
	if err != nil {
		return err
	}
	defer client.Close()

	command, stdin, err := c.buildExecutionCommand(ctx.execMetadata, ctx.scriptBody)
	if err != nil {
		return err
	}
	result, err := client.ExecuteScript(command, stdin, time.Duration(ctx.execMetadata.Timeout)*time.Second)
	if c.isConnectError(err) {
		if c.shouldRetry(ctx.execMetadata.ConnectionRetry, c.getRetryAttempt(ctx.metadataCtx)) {
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

	if result.ExitCode != 0 {
		if c.shouldRetry(ctx.execMetadata.ExecutionRetry, c.getExecutionAttempt(ctx.metadataCtx)) {
			err = c.incrementExecutionRetryCount(ctx.metadataCtx)
			if err != nil {
				return err
			}

			return ctx.requestsCtx.ScheduleActionCall(
				"executionRetry",
				map[string]any{},
				time.Duration(ctx.execMetadata.ExecutionRetry.IntervalSeconds)*time.Second,
			)
		}
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

func (c *SSHCommand) shouldRetry(retrySpec *RetrySpec, attempt int) bool {
	if retrySpec == nil || !retrySpec.Enabled {
		return false
	}

	return attempt < retrySpec.Retries
}

func (c *SSHCommand) incrementRetryCount(metadata core.MetadataWriter) error {
	current := c.getMetadataMap(metadata)
	current["attempt"] = c.getRetryAttempt(metadata) + 1

	return metadata.Set(current)
}

func (c *SSHCommand) incrementExecutionRetryCount(metadata core.MetadataWriter) error {
	current := c.getMetadataMap(metadata)
	current["executionAttempt"] = c.getExecutionAttempt(metadata) + 1

	return metadata.Set(current)
}

func (c *SSHCommand) getExecutionAttempt(metadata core.MetadataWriter) int {
	return c.getMetadataInt(metadata, "executionAttempt")
}

func (c *SSHCommand) getRetryAttempt(metadata core.MetadataWriter) int {
	return c.getMetadataInt(metadata, "attempt")
}

func (c *SSHCommand) getMetadataInt(metadata core.MetadataWriter, key string) int {
	meta := c.getMetadataMap(metadata)

	value, ok := meta[key]
	if !ok {
		return 0
	}

	// JSON numbers deserialize as float64
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func (c *SSHCommand) getMetadataMap(metadata core.MetadataWriter) map[string]any {
	current := metadata.Get()
	if current == nil {
		return map[string]any{}
	}

	if metadataMap, ok := current.(map[string]any); ok {
		return metadataMap
	}

	data, err := json.Marshal(current)
	if err != nil {
		return map[string]any{}
	}

	metadataMap := map[string]any{}
	if err := json.Unmarshal(data, &metadataMap); err != nil {
		return map[string]any{}
	}

	return metadataMap
}

func (c *SSHCommand) setResultMetadata(metadata core.MetadataWriter, result *CommandResult) error {
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

func (c *SSHCommand) buildRemoteCommand(workingDirectory string, environment []EnvironmentVariable, command string) string {
	finalCommand := command

	if workingDirectory != "" {
		finalCommand = fmt.Sprintf("cd %s && %s", shellQuote(workingDirectory), finalCommand)
	}

	if len(environment) == 0 {
		return finalCommand
	}

	envAssignments := make([]string, 0, len(environment))
	for _, variable := range environment {
		envAssignments = append(envAssignments, fmt.Sprintf("%s=%s", variable.Name, shellQuote(variable.Value)))
	}

	return fmt.Sprintf("env %s sh -lc %s", strings.Join(envAssignments, " "), shellQuote(finalCommand))
}

// buildExecutionCommand assembles the final command sent to the remote host
// and returns a stdin payload when the command should be streamed as a script.
//
// Inline mode keeps one-line commands on the command line, preserving the
// long-standing behavior for simple invocations. Multi-line inline scripts are
// streamed to `bash -s` so shell syntax that depends on real newlines (shebangs,
// comments, conditionals, here-docs) is not flattened into invalid `&&` chains.
//
// File mode pipes the (already-loaded) script body to `bash -s` over stdin.
// This avoids embedding the whole script in the command line — argv has size
// limits and nested quoting is fragile — while preserving multi-line
// constructs, comments, and bash-only features (pipefail, declare -A, here-docs,
// process substitution) except for normalizing CRLF/CR line endings to LF. The
// stream is always interpreted by bash, so any leading `#!` line is just a
// comment: non-bash shebangs are not honored.
func (c *SSHCommand) buildExecutionCommand(metadata ExecutionMetadata, scriptBody string) (string, io.Reader, error) {
	if metadata.CommandSource == CommandSourceFile {
		if scriptBody == "" {
			return "", nil, errors.New("command file body is required")
		}
		command, payload := c.buildScriptCommand(metadata.WorkingDirectory, metadata.Environment, scriptBody)
		return command, strings.NewReader(payload), nil
	}

	if isMultilineInlineScript(metadata.Commands) {
		command, payload := c.buildInlineScriptCommand(metadata.WorkingDirectory, metadata.Environment, metadata.Commands)
		return command, strings.NewReader(payload), nil
	}

	combined := buildCombinedCommands(metadata.Commands)
	if combined == "" {
		return "", nil, errors.New("commands is required")
	}
	return c.buildRemoteCommand(metadata.WorkingDirectory, metadata.Environment, combined), nil, nil
}

// buildScriptCommand returns the remote command (`env VAR=v ... bash -s`) and
// the script body to stream over stdin. The working-directory change is
// prepended on its own line (followed by `|| exit 1`) so a leading shebang or
// comment in the script cannot swallow the `cd` via `#`-to-end-of-line.
func (c *SSHCommand) buildScriptCommand(workingDirectory string, environment []EnvironmentVariable, script string) (string, string) {
	return c.buildScriptCommandWithShell("bash -s", workingDirectory, environment, script)
}

func (c *SSHCommand) buildInlineScriptCommand(workingDirectory string, environment []EnvironmentVariable, script string) (string, string) {
	return c.buildScriptCommandWithShell("bash -e -s", workingDirectory, environment, script)
}

func (c *SSHCommand) buildScriptCommandWithShell(shellCommand string, workingDirectory string, environment []EnvironmentVariable, script string) (string, string) {
	payload := normalizeScriptLineEndings(script)
	if workingDirectory != "" {
		payload = fmt.Sprintf("cd %s || exit 1\n%s", shellQuote(workingDirectory), payload)
	}

	command := shellCommand
	if len(environment) > 0 {
		envAssignments := make([]string, 0, len(environment))
		for _, variable := range environment {
			envAssignments = append(envAssignments, fmt.Sprintf("%s=%s", variable.Name, shellQuote(variable.Value)))
		}
		command = fmt.Sprintf("env %s %s", strings.Join(envAssignments, " "), command)
	}

	return command, payload
}

// validateCommandSource enforces the configured command-source variant.
// Inline mode requires a non-empty commands string. File mode requires a path
// that resolves to a real file in the canvas's git repository so Setup catches
// typos at publish time instead of at every execution.
func validateCommandSource(ctx core.SetupContext, spec Spec) error {
	switch spec.commandSourceOrDefault() {
	case CommandSourceInline:
		if strings.TrimSpace(spec.Commands) == "" {
			return errors.New("commands is required")
		}
		return nil

	case CommandSourceFile:
		path := strings.TrimSpace(spec.CommandFile)
		if path == "" {
			return errors.New("command file is required")
		}
		normalized, err := gitprovider.ValidateUserPath(path)
		if err != nil {
			return fmt.Errorf("invalid command file %q: %w", path, err)
		}
		if ctx.Files == nil {
			return errors.New("command file configured but file access is not available")
		}
		available, err := ctx.Files.List()
		if err != nil {
			return fmt.Errorf("failed to list repository files: %w", err)
		}
		found := false
		for _, candidate := range available {
			if norm, normErr := gitprovider.NormalizePath(candidate); normErr == nil && norm == normalized {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("command file %q not found in app repository", path)
		}
		// The file exists; read it now so publish fails fast on empty or
		// whitespace-only scripts instead of letting every execution fail
		// with an empty-file error.
		if _, err := loadCommandFile(ctx.Files, spec.CommandFile); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("invalid command source: %s", spec.CommandSource)
	}
}

func buildCombinedCommands(commands string) string {
	lines := strings.Split(normalizeScriptLineEndings(commands), "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		parts = append(parts, l)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " && ")
}

func isMultilineInlineScript(commands string) bool {
	lines := strings.Split(normalizeScriptLineEndings(commands), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmptyLines++
		if nonEmptyLines > 1 {
			return true
		}
	}

	return false
}

func normalizeScriptLineEndings(script string) string {
	script = strings.ReplaceAll(script, "\r\n", "\n")
	return strings.ReplaceAll(script, "\r", "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func (c *SSHCommand) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SSHCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SSHCommand) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "connectionRetry",
			Type: core.HookTypeInternal,
		},
		{
			Name: "executionRetry",
			Type: core.HookTypeInternal,
		},
	}
}

func (c *SSHCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 404, nil, fmt.Errorf("SSH component does not handle webhooks")
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
