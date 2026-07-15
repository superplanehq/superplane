package ssh

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	channelSuccess = "success"
	channelFailed  = "failed"

	FinishedEventType = "ssh.command.executed"
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
	MachineType      string                `json:"machineType" mapstructure:"machineType"`
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

func (c *SSHCommand) Name() string  { return "ssh" }
func (c *SSHCommand) Label() string { return "SSH Command" }
func (c *SSHCommand) Description() string {
	return "Run one or more commands on a remote host via SSH, executed on a fleet runner. Authenticate using an organization Secret (SSH key or password)."
}
func (c *SSHCommand) Documentation() string {
	return `Run one or more commands on a remote host via SSH.

The connection and commands run on a fleet **runner** rather than in the control
plane, so command output streams to **View logs** and long-running scripts are
not bound by a control-plane time limit.

## Authentication

Choose **SSH key** or **Password**, then select the organization Secret and the key name within that secret that holds the credential.

- **SSH key**: Secret key containing the private key (PEM/OpenSSH). Optionally a second secret+key for passphrase if the key is encrypted.
- **Password**: Secret key containing the password.

## Configuration

- **Machine type**: Runner fleet that opens the SSH connection (required). Its host must provide the ` + "`ssh`" + ` client (and ` + "`sshpass`" + ` for password or passphrase auth).
- **Host**, **Port** (default 22), **Username**: Connection details.
- **Command source**: Choose **Inline** to type commands directly, or **From file** to load them from a file in the app's repository (e.g. scripts/deploy.sh).
- **Commands** (inline mode): One or more commands to run, one per line (supports expressions). They run with ` + "`set -e`" + ` so a failing command aborts the run.
- **Command file** (file mode): Path to a file in the app's repository (e.g. ` + "`scripts/deploy.sh`" + `).
- **Working directory**: Optional; changes to this directory before running the command.
- **Environment variables**: Optional list of key/value pairs exported on the remote host during command execution.
- **Execution timeout (seconds)**: Wall-clock limit for the whole runner task (default 3600, max 86400).
- **Connection retry** (optional): Retry connecting when the host is not reachable yet (e.g. server still booting).
- **Execution retry** (optional): Retry running the command when the exit status is not 0.

## Output channels

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

func (c *SSHCommand) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      FinishedEventType,
		"timestamp": "2026-01-19T12:00:00Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
		}},
	}
}

func (c *SSHCommand) Configuration() []configuration.Field {
	sshKeyOnly := []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{AuthMethodSSHKey}}}
	passwordOnly := []configuration.VisibilityCondition{{Field: "authMethod", Values: []string{AuthMethodPassword}}}

	return []configuration.Field{
		{
			Name:        "machineType",
			Label:       "Machine type",
			Type:        configuration.FieldTypeSelect,
			Description: "Runner fleet that opens the SSH connection",
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: runner.MachineTypeSelectOptions(),
				},
			},
		},
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
			Label:       "Execution timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultTimeoutSeconds,
			Description: "Wall-clock limit for the whole runner task. Defaults to 3600 seconds (1 hour); max 86400.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(maxTimeoutSeconds),
				},
			},
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

const (
	defaultTimeoutSeconds = 3600
	maxTimeoutSeconds     = 86400
)

func intPtr(v int) *int { return &v }

func (c *SSHCommand) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateSpec(spec); err != nil {
		return err
	}
	if err := validateCommandSource(ctx, spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

// validateSpec checks the configuration values that do not require repository
// access. Command-source validation (which reads repo files) is handled
// separately by validateCommandSource.
func validateSpec(spec Spec) error {
	if spec.Host == "" {
		return errors.New("host is required")
	}
	if spec.User == "" {
		return errors.New("username is required")
	}
	if strings.TrimSpace(spec.MachineType) == "" {
		return errors.New("machine type is required")
	}
	if spec.Port != 0 && (spec.Port < 1 || spec.Port > 65535) {
		return fmt.Errorf("invalid port: %d", spec.Port)
	}
	if spec.Timeout != 0 && (spec.Timeout < 1 || spec.Timeout > maxTimeoutSeconds) {
		return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", maxTimeoutSeconds, defaultTimeoutSeconds)
	}
	if err := validateEnvironment(spec.Environment); err != nil {
		return err
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

	if err := validateRetry("connection retry", spec.ConnectionRetry); err != nil {
		return err
	}
	return validateRetry("execution retry", spec.ExecutionRetry)
}

func validateEnvironment(environment []EnvironmentVariable) error {
	for _, variable := range environment {
		if variable.Name == "" {
			return errors.New("environment variable name is required")
		}
		if !environmentVariableNameRegex.MatchString(variable.Name) {
			return fmt.Errorf("invalid environment variable name: %s", variable.Name)
		}
	}
	return nil
}

func validateRetry(label string, spec *RetrySpec) error {
	if spec == nil || !spec.Enabled {
		return nil
	}
	if spec.Retries < 0 {
		return fmt.Errorf("%s: retries must be 0 or greater", label)
	}
	if spec.IntervalSeconds < 1 {
		return fmt.Errorf("%s: interval must be at least 1 second", label)
	}
	return nil
}

func decodeSpec(raw any) (Spec, error) {
	config, ok := raw.(map[string]any)
	if !ok || config == nil {
		return Spec{}, fmt.Errorf("decode configuration: invalid configuration type")
	}
	var spec Spec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return Spec{}, fmt.Errorf("decode configuration: %w", err)
	}
	return spec, nil
}

func (c *SSHCommand) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	body, err := c.commandBody(ctx.Files, spec)
	if err != nil {
		return err
	}

	environment, err := c.authEnvironment(ctx.Secrets, spec)
	if err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	broker, err := runner.NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	remoteScript := buildRemoteScript(spec.Environment, spec.WorkingDirectory, body)
	taskID, err := broker.CreateTask(runner.CreateTaskParams{
		MachineType:    spec.MachineType,
		RunMode:        runner.RunModeBash,
		Script:         buildRunnerScript(spec, remoteScript),
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: spec.Timeout,
	})
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return runner.AfterTaskCreated(ctx, taskID)
}

// commandBody resolves the commands to run on the remote host. Inline mode uses
// the (already template-resolved) commands verbatim; file mode reads the script
// from the app repository on every attempt so the script never lives in the
// database.
func (c *SSHCommand) commandBody(files core.RepositoryFilesContext, spec Spec) (string, error) {
	if spec.commandSourceOrDefault() == CommandSourceFile {
		return loadCommandFile(files, spec.CommandFile)
	}
	if strings.TrimSpace(spec.Commands) == "" {
		return "", errors.New("commands is required")
	}
	return spec.Commands, nil
}

// authEnvironment resolves the SSH credentials from secrets and returns them as
// broker task environment variables. Passing them this way keeps the secrets out
// of persisted execution metadata; the generated runner script consumes them.
func (c *SSHCommand) authEnvironment(secrets core.SecretsContext, spec Spec) ([]runner.BrokerEnvironmentVariable, error) {
	if secrets == nil {
		return nil, errors.New("secrets context is unavailable")
	}

	switch spec.Authentication.Method {
	case AuthMethodSSHKey:
		privateKey, err := secrets.GetKey(spec.Authentication.PrivateKey.Secret, spec.Authentication.PrivateKey.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot get private key: %w", err)
		}
		env := []runner.BrokerEnvironmentVariable{
			{Name: envPrivateKey, Value: string(normalizePrivateKey(privateKey))},
		}
		if spec.Authentication.Passphrase.IsSet() {
			passphrase, err := secrets.GetKey(spec.Authentication.Passphrase.Secret, spec.Authentication.Passphrase.Key)
			if err != nil {
				return nil, fmt.Errorf("cannot get passphrase: %w", err)
			}
			env = append(env, runner.BrokerEnvironmentVariable{Name: envPassphrase, Value: string(passphrase)})
		}
		return env, nil

	case AuthMethodPassword:
		password, err := secrets.GetKey(spec.Authentication.Password.Secret, spec.Authentication.Password.Key)
		if err != nil {
			return nil, fmt.Errorf("cannot get password: %w", err)
		}
		return []runner.BrokerEnvironmentVariable{{Name: envPassword, Value: string(password)}}, nil

	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", spec.Authentication.Method)
	}
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

func (c *SSHCommand) taskOutcome() runner.TaskOutcome {
	return runner.TaskOutcome{
		FinishedEventType: FinishedEventType,
		PassedChannel:     channelSuccess,
		FailedChannel:     channelFailed,
	}
}

func (c *SSHCommand) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookActionPoll, Type: core.HookTypeInternal}}
}

func (c *SSHCommand) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case runner.HookActionPoll:
		return runner.PollTask(ctx, c.taskOutcome())
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *SSHCommand) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleTaskWebhook(ctx, c.taskOutcome())
}

func (c *SSHCommand) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *SSHCommand) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SSHCommand) Cleanup(ctx core.SetupContext) error {
	return nil
}
