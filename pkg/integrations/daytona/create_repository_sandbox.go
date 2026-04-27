package daytona

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CreateRepositorySandboxPayloadType           = "daytona.repository.sandbox"
	CreateRepositorySandboxPollInterval          = 5 * time.Second
	CreateRepositorySandboxBootstrapPollInterval = 2 * time.Second
	CreateRepositorySandboxDefaultTimeout        = 5 * time.Minute
	CreateRepositorySandboxMaxTimeout            = time.Hour
	CreateRepositorySandboxBootstrapLogMaxBytes  = 64 * 1024

	SandboxBootstrapFromInline = "inline"
	SandboxBootstrapFromFile   = "file"

	repositorySandboxStagePreparingSandbox = "preparingSandbox"
	repositorySandboxStageBootstrapping    = "bootstrapping"
	repositorySandboxStageDone             = "done"

	repositorySandboxInlineBootstrapPath = SandboxBaseDir + "/bootstrap.sh"
)

type CreateRepositorySandbox struct{}

type CreateRepositorySandboxSpec struct {
	Snapshot         string                                `json:"snapshot,omitempty"`
	Target           string                                `json:"target,omitempty"`
	AutoStopInterval int                                   `json:"autoStopInterval,omitempty"`
	Env              []EnvVariable                         `json:"env,omitempty"`
	Secrets          []SandboxSecret                       `json:"secrets,omitempty"`
	Repository       string                                `json:"repository"`
	Bootstrap        *CreateRepositorySandboxBootstrapSpec `json:"bootstrap"`
}

type CreateRepositorySandboxBootstrapSpec struct {
	From    string `json:"from,omitempty"`
	Script  string `json:"script,omitempty"`
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type CreateRepositorySandboxMetadata struct {
	Stage            string             `json:"stage" mapstructure:"stage"`
	SandboxID        string             `json:"sandboxId" mapstructure:"sandboxId"`
	SandboxStartedAt string             `json:"sandboxStartedAt" mapstructure:"sandboxStartedAt"`
	SessionID        string             `json:"sessionId" mapstructure:"sessionId"`
	Timeout          int                `json:"timeout" mapstructure:"timeout"`
	Repository       string             `json:"repository" mapstructure:"repository"`
	Directory        string             `json:"directory" mapstructure:"directory"`
	Secrets          []SandboxSecret    `json:"secrets,omitempty" mapstructure:"secrets,omitempty"`
	Clone            *CloneMetadata     `json:"clone,omitempty" mapstructure:"clone,omitempty"`
	Bootstrap        *BootstrapMetadata `json:"bootstrap,omitempty" mapstructure:"bootstrap,omitempty"`
}

type CloneMetadata struct {
	StartedAt  string  `json:"startedAt" mapstructure:"startedAt"`
	FinishedAt string  `json:"finishedAt" mapstructure:"finishedAt"`
	Error      *string `json:"error,omitempty" mapstructure:"error,omitempty"`
}

type BootstrapMetadata struct {
	CmdID      string  `json:"cmdId" mapstructure:"cmdId"`
	StartedAt  string  `json:"startedAt" mapstructure:"startedAt"`
	FinishedAt string  `json:"finishedAt" mapstructure:"finishedAt"`
	ExitCode   int     `json:"exitCode" mapstructure:"exitCode"`
	Result     string  `json:"result" mapstructure:"result"`
	Log        string  `json:"log,omitempty" mapstructure:"log,omitempty"`
	From       string  `json:"from" mapstructure:"from"`
	Script     *string `json:"script,omitempty" mapstructure:"script,omitempty"`
	Path       *string `json:"path,omitempty" mapstructure:"path,omitempty"`
	URL        *string `json:"url,omitempty" mapstructure:"url,omitempty"`
}

func (c *CreateRepositorySandbox) Name() string {
	return "daytona.createRepositorySandbox"
}

func (c *CreateRepositorySandbox) Label() string {
	return "Create Repository Sandbox"
}

func (c *CreateRepositorySandbox) Description() string {
	return "Create a sandbox, clone a repository, and run a bootstrap script"
}

func (c *CreateRepositorySandbox) Documentation() string {
	return `The Create Repository Sandbox component creates a new Daytona sandbox, clones a repository, and runs a bootstrap script.

## Use Cases

- **Ephemeral dev environments**: Spin up a fresh environment for a repository on demand
- **CI-like workflows**: Clone code and run setup scripts before downstream tasks
- **Automated validation**: Prepare repository state before executing tests or commands

## Notes

- The component waits for the sandbox to reach the "started" state
- Clone and bootstrap run sequentially in the same session
- If clone or bootstrap fails, the component returns an error`
}

func (c *CreateRepositorySandbox) Icon() string {
	return "daytona"
}

func (c *CreateRepositorySandbox) Color() string {
	return "orange"
}

func (c *CreateRepositorySandbox) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRepositorySandbox) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "snapshot",
			Label:    "Snapshot",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "snapshot",
					UseNameAsValue: true,
				},
			},
			Description: "Base environment snapshot for the sandbox",
			Default:     "default",
		},
		{
			Name:        "target",
			Label:       "Target Region",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. us, eu, local",
			Description: "Target region for the sandbox",
			Default:     "us",
		},
		{
			Name:        "autoStopInterval",
			Label:       "Auto Stop Interval",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Time in minutes before the sandbox auto-stops",
			Default:     15,
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Repository URL to clone",
			Placeholder: "https://github.com/owner/repository.git",
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
			Description: "Environment variables to set in the sandbox",
		},
		sandboxSecretsConfigurationField(),
		{
			Name:        "bootstrap",
			Label:       "Bootstrap",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Execute script after the sandbox is running, and the repository is cloned",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "from",
							Label:    "From",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							Default:  SandboxBootstrapFromInline,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Inline Script", Value: SandboxBootstrapFromInline},
										{Label: "Repository File", Value: SandboxBootstrapFromFile},
									},
								},
							},
						},
						{
							Name:        "script",
							Label:       "Script",
							Type:        configuration.FieldTypeText,
							Required:    false,
							Placeholder: "npm ci && npm test",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "from", Values: []string{SandboxBootstrapFromInline}},
							},
						},
						{
							Name:        "path",
							Label:       "Path",
							Type:        configuration.FieldTypeString,
							Required:    false,
							Placeholder: "scripts/bootstrap.sh",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "from", Values: []string{SandboxBootstrapFromFile}},
							},
						},
						{
							Name:        "timeout",
							Label:       "Timeout",
							Type:        configuration.FieldTypeNumber,
							Required:    false,
							Description: "Time in minutes before the bootstrap fails",
							Default:     int(CreateRepositorySandboxDefaultTimeout.Minutes()),
						},
					},
				},
			},
		},
	}
}

func (c *CreateRepositorySandbox) Setup(ctx core.SetupContext) error {
	spec := CreateRepositorySandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Snapshot != "" && strings.TrimSpace(spec.Snapshot) == "" {
		return fmt.Errorf("snapshot must not be empty if provided")
	}

	if spec.AutoStopInterval < 0 {
		return fmt.Errorf("autoStopInterval cannot be negative")
	}

	if spec.Bootstrap != nil {
		if spec.Bootstrap.Timeout < 0 {
			return fmt.Errorf("bootstrap.timeout cannot be negative")
		}
		if spec.Bootstrap.Timeout > int(CreateRepositorySandboxMaxTimeout.Minutes()) {
			return fmt.Errorf("bootstrap.timeout cannot exceed %d minutes", int(CreateRepositorySandboxMaxTimeout.Minutes()))
		}
	}

	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
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

	if err := validateSandboxSecrets(spec.Secrets); err != nil {
		return err
	}

	_, err := c.bootstrapMetadataFromSpec(spec)
	if err != nil {
		return fmt.Errorf("failed to validate bootstrap configuration: %v", err)
	}

	return nil
}

func (c *CreateRepositorySandbox) bootstrapMetadataFromSpec(spec CreateRepositorySandboxSpec) (*BootstrapMetadata, error) {

	//
	// Having no bootstrap configuration is valid, and will result in no bootstrap being executed.
	//
	if spec.Bootstrap == nil {
		return nil, nil
	}

	if spec.Bootstrap.From == "" {
		return nil, fmt.Errorf("bootstrap.from is required")
	}

	metadata := BootstrapMetadata{
		From: spec.Bootstrap.From,
	}

	switch spec.Bootstrap.From {
	case SandboxBootstrapFromInline:
		if strings.TrimSpace(spec.Bootstrap.Script) == "" {
			return nil, fmt.Errorf("bootstrap.script is required when bootstrap.from is inline")
		}

		metadata.Script = &spec.Bootstrap.Script
		return &metadata, nil

	case SandboxBootstrapFromFile:
		if strings.TrimSpace(spec.Bootstrap.Path) == "" {
			return nil, fmt.Errorf("bootstrap.path is required when bootstrap.from is file")
		}

		metadata.Path = &spec.Bootstrap.Path
		return &metadata, nil

	default:
		return nil, fmt.Errorf("invalid bootstrap.from: %s", spec.Bootstrap.From)
	}
}

func (c *CreateRepositorySandbox) Execute(ctx core.ExecutionContext) error {
	spec := CreateRepositorySandboxSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	var envMap map[string]string
	if len(spec.Env) > 0 {
		envMap = make(map[string]string, len(spec.Env))
		for _, env := range spec.Env {
			envMap[strings.TrimSpace(env.Name)] = env.Value
		}
	}

	repositoryDirectory, err := c.getDirectoryName(spec.Repository)
	if err != nil {
		return fmt.Errorf("failed to determine repository directory name: %v", err)
	}

	bootstrapMetadata, err := c.bootstrapMetadataFromSpec(spec)
	if err != nil {
		return err
	}

	sandbox, err := client.CreateSandbox(&CreateSandboxRequest{
		Snapshot:         spec.Snapshot,
		Target:           spec.Target,
		AutoStopInterval: spec.AutoStopInterval,
		Env:              envMap,
	})

	if err != nil {
		return fmt.Errorf("failed to create sandbox: %v", err)
	}

	ctx.Logger.Infof("Created sandbox %s", sandbox.ID)

	metadata := CreateRepositorySandboxMetadata{
		Stage:            repositorySandboxStagePreparingSandbox,
		SandboxID:        sandbox.ID,
		SandboxStartedAt: time.Now().Format(time.RFC3339),
		Timeout:          resolveTimeoutSeconds(spec.Bootstrap),
		Repository:       strings.TrimSpace(spec.Repository),
		Directory:        path.Join(SandboxHomeDir, repositoryDirectory),
		Secrets:          spec.Secrets,
		Bootstrap:        bootstrapMetadata,
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
}

func (c *CreateRepositorySandbox) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRepositorySandbox) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRepositorySandbox) Hooks() []core.Hook {
	return []core.Hook{
		{Name: "poll", Type: core.HookTypeInternal},
	}
}

func (c *CreateRepositorySandbox) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateRepositorySandbox) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateRepositorySandboxMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	startedAt, err := time.Parse(time.RFC3339, metadata.SandboxStartedAt)
	if err != nil {
		return fmt.Errorf("failed to parse sandbox started at: %v", err)
	}

	timeout := time.Duration(metadata.Timeout) * time.Second
	if time.Since(startedAt) > timeout {
		c.markBootstrapTimedOut(ctx, &metadata)
		ctx.Logger.Errorf("sandbox creation failed on stage %s after %v", metadata.Stage, timeout)
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("sandbox creation failed on stage %s after %v", metadata.Stage, timeout))
	}

	switch metadata.Stage {
	case repositorySandboxStagePreparingSandbox:
		return c.pollWaitingSandbox(ctx, &metadata)

	case repositorySandboxStageBootstrapping:
		return c.pollBootstrapping(ctx, &metadata)

	default:
		return fmt.Errorf("unknown create repository sandbox stage: %s", metadata.Stage)
	}
}

func (c *CreateRepositorySandbox) pollWaitingSandbox(ctx core.ActionHookContext, metadata *CreateRepositorySandboxMetadata) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	sandbox, err := client.GetSandbox(metadata.SandboxID)
	if err != nil {
		ctx.Logger.Errorf("failed to get sandbox %s: %v", metadata.SandboxID, err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
	}

	switch sandbox.State {
	case "started":
		if err := injectSandboxSecrets(client, metadata.SandboxID, ctx.Secrets, metadata.Secrets); err != nil {
			return fmt.Errorf("failed to inject sandbox secrets: %v", err)
		}

		return c.startClone(ctx, client, metadata)
	case "error":
		return fmt.Errorf("sandbox %s failed to start", metadata.SandboxID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
	}
}

func (c *CreateRepositorySandbox) startClone(ctx core.ActionHookContext, client *Client, metadata *CreateRepositorySandboxMetadata) error {
	cloneRequest, err := c.cloneRepositoryRequest(ctx.Secrets, metadata)
	if err != nil {
		return err
	}

	cloneStartedAt := time.Now().Format(time.RFC3339)
	if err := client.CloneRepository(metadata.SandboxID, cloneRequest); err != nil {
		errorMessage := err.Error()
		metadata.Clone = &CloneMetadata{
			StartedAt:  cloneStartedAt,
			FinishedAt: time.Now().Format(time.RFC3339),
			Error:      &errorMessage,
		}

		if err := ctx.Metadata.Set(*metadata); err != nil {
			return err
		}

		ctx.Logger.Errorf("repository clone failed: %v", err)
		ctx.ExecutionState.Fail("error", fmt.Sprintf("repository clone failed: %v", err))
		return nil
	}

	metadata.Clone = &CloneMetadata{
		StartedAt:  cloneStartedAt,
		FinishedAt: time.Now().Format(time.RFC3339),
	}

	//
	// If no bootstrap is required, we can finish after cloning.
	//
	if metadata.Bootstrap == nil {
		return c.finish(ctx, metadata)
	}

	if err := c.prepareInlineBootstrapScript(client, metadata); err != nil {
		return err
	}

	sessionID := uuid.New().String()
	if err := client.CreateSession(metadata.SandboxID, sessionID); err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	bootstrapCommand := c.bootstrapCommand(metadata)
	bootstrapCommand = wrapCommandWithSandboxSecretEnv(bootstrapCommand)
	response, err := client.ExecuteSessionCommand(metadata.SandboxID, sessionID, bootstrapCommand)
	if err != nil {
		return fmt.Errorf("failed to execute bootstrap script: %v", err)
	}

	metadata.Stage = repositorySandboxStageBootstrapping
	metadata.SessionID = sessionID
	metadata.Bootstrap.CmdID = response.CmdID
	metadata.Bootstrap.StartedAt = time.Now().Format(time.RFC3339)

	if err := ctx.Metadata.Set(*metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxBootstrapPollInterval)
}

func (c *CreateRepositorySandbox) cloneRepositoryRequest(secretsContext core.SecretsContext, metadata *CreateRepositorySandboxMetadata) (*CloneRepositoryRequest, error) {
	request := &CloneRepositoryRequest{
		URL:  metadata.Repository,
		Path: metadata.Directory,
	}

	token, err := c.findCloneToken(secretsContext, metadata)
	if err != nil {
		return nil, err
	}

	if token != "" {
		request.Username = "x-access-token"
		request.Password = token
	}

	return request, nil
}

/*
 * If a "GITHUB_TOKEN" secret is found in the spec,
 * we use it to clone the repository with the toolbox Git API.
 */
func (c *CreateRepositorySandbox) findCloneToken(secretsContext core.SecretsContext, metadata *CreateRepositorySandboxMetadata) (string, error) {
	for _, secret := range metadata.Secrets {
		if strings.TrimSpace(secret.Type) != SandboxSecretTypeEnvVar {
			continue
		}

		if strings.TrimSpace(secret.Name) != "GITHUB_TOKEN" {
			continue
		}

		if !secret.Value.IsSet() {
			continue
		}

		value, err := secretsContext.GetKey(secret.Value.Secret, secret.Value.Key)
		if err != nil {
			return "", fmt.Errorf("failed to resolve secret %s/%s: %w", secret.Value.Secret, secret.Value.Key, err)
		}

		token := string(value)
		if token != "" {
			return token, nil
		}
	}

	return "", nil
}

/*
 * If an inline bootstrap script is provided,
 * we upload it to the sandbox base directory.
 */
func (c *CreateRepositorySandbox) prepareInlineBootstrapScript(client *Client, metadata *CreateRepositorySandboxMetadata) error {
	if metadata.Bootstrap == nil || metadata.Bootstrap.From != SandboxBootstrapFromInline {
		return nil
	}

	if metadata.Bootstrap.Script == nil {
		return fmt.Errorf("bootstrap.script is required when bootstrap.from is inline")
	}

	if err := ensureFolderExists(client, metadata.SandboxID, SandboxBaseDir); err != nil {
		return err
	}

	inlineScriptPath := repositorySandboxInlineBootstrapPath
	//
	// Normalize line endings to LF before upload. The configuration form may
	// produce CRLF (Windows-style) line endings, which dash interprets
	// literally — turning `sleep 2` into `sleep "2\r"` and breaking shell
	// keywords like `done\r`. Strip carriage returns so the script the user
	// sees is the script the sandbox runs.
	//
	script := strings.ReplaceAll(*metadata.Bootstrap.Script, "\r\n", "\n")
	script = strings.ReplaceAll(script, "\r", "\n")
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}

	if err := client.UploadFile(metadata.SandboxID, inlineScriptPath, []byte(script)); err != nil {
		return fmt.Errorf("failed to upload inline bootstrap script: %v", err)
	}

	metadata.Bootstrap.Path = &inlineScriptPath
	return nil
}

func (c *CreateRepositorySandbox) pollBootstrapping(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		// Treat as transient — the previous getCommandResult-based
		// implementation also retried on client-creation failures via
		// the caller's error-to-reschedule path. Keep that behavior so
		// a flaky integration context doesn't fail an in-flight bootstrap.
		ctx.Logger.Errorf("failed to create client: %v", err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxBootstrapPollInterval)
	}

	//
	// Fetch the latest bootstrap logs on every poll so the UI has a live view
	// even while the command is still running. Transient log-fetch errors are
	// logged but never fail the execution; the top-of-poll timeout check is
	// the single authoritative exit for long hangs.
	//
	logs, logsFetched := fetchBootstrapLogs(ctx, client, metadata)
	if logsFetched {
		metadata.Bootstrap.Log = tailBytes(logs, CreateRepositorySandboxBootstrapLogMaxBytes)
		if err := ctx.Metadata.Set(*metadata); err != nil {
			return err
		}
	}

	session, err := client.GetSession(metadata.SandboxID, metadata.SessionID)
	if err != nil {
		ctx.Logger.Errorf("failed to get session %s: %v", metadata.SessionID, err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxBootstrapPollInterval)
	}

	command := session.FindCommand(metadata.Bootstrap.CmdID)
	if command == nil || command.ExitCode == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxBootstrapPollInterval)
	}

	//
	// Re-fetch logs once more now that the command has exited. The earlier
	// fetch happened before we confirmed the command was done, so any output
	// produced between the two calls (commonly the final "done" echo) would
	// otherwise be dropped from the captured result.
	//
	if finalLogs, finalFetched := fetchBootstrapLogs(ctx, client, metadata); finalFetched {
		logs = finalLogs
		logsFetched = true
	}

	//
	// Only overwrite the persisted log/result when we actually captured
	// fresh content this poll. Otherwise keep whatever a prior successful
	// poll persisted — back-to-back fetch failures must not erase the log
	// snapshot the user can already see in the UI.
	//
	if logsFetched {
		metadata.Bootstrap.Result = logs
		metadata.Bootstrap.Log = tailBytes(logs, CreateRepositorySandboxBootstrapLogMaxBytes)
	}
	metadata.Bootstrap.FinishedAt = time.Now().Format(time.RFC3339)
	metadata.Bootstrap.ExitCode = *command.ExitCode

	result := &ExecuteCommandResponse{
		ExitCode: *command.ExitCode,
		Result:   metadata.Bootstrap.Result,
	}

	if result.ExitCode != 0 {
		if err := ctx.Metadata.Set(*metadata); err != nil {
			return err
		}

		ctx.Logger.Errorf("bootstrap script failed with exit code %d: %s", result.ExitCode, result.ShortResult())
		ctx.ExecutionState.Fail("error", fmt.Sprintf("bootstrap script failed with exit code %d: %s", result.ExitCode, result.ShortResult()))
		return nil
	}

	return c.finish(ctx, metadata)
}

// markBootstrapTimedOut records the best-effort terminal state for the
// bootstrap phase when the deadline fires, so the UI has a definite
// FinishedAt timestamp and the last captured log.
func (c *CreateRepositorySandbox) markBootstrapTimedOut(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) {
	if metadata.Bootstrap == nil || metadata.Stage != repositorySandboxStageBootstrapping {
		return
	}

	if metadata.Bootstrap.FinishedAt == "" {
		metadata.Bootstrap.FinishedAt = time.Now().Format(time.RFC3339)
	}

	if err := ctx.Metadata.Set(*metadata); err != nil {
		ctx.Logger.Errorf("failed to persist bootstrap metadata on timeout: %v", err)
	}
}

// resolveTimeoutSeconds returns the effective timeout in seconds. The user
// configures it in minutes under the bootstrap block; when bootstrap is not
// configured or the value is unset, the default applies.
func resolveTimeoutSeconds(bootstrap *CreateRepositorySandboxBootstrapSpec) int {
	if bootstrap != nil && bootstrap.Timeout > 0 {
		return bootstrap.Timeout * 60
	}
	return int(CreateRepositorySandboxDefaultTimeout.Seconds())
}

// fetchBootstrapLogs retrieves the current bootstrap command logs.
// Returns the log content and whether the fetch succeeded; on failure
// the error is logged and the caller is expected to keep any previously
// persisted log content intact rather than overwriting it with empty.
func fetchBootstrapLogs(ctx core.ActionContext, client *Client, metadata *CreateRepositorySandboxMetadata) (string, bool) {
	logs, err := client.GetSessionCommandLogs(metadata.SandboxID, metadata.SessionID, metadata.Bootstrap.CmdID)
	if err != nil {
		ctx.Logger.Errorf("failed to get bootstrap command logs for %s: %v", metadata.Bootstrap.CmdID, err)
		return "", false
	}
	return logs, true
}

// tailBytes returns the tail of s capped at max bytes total, including
// a truncation marker when clipping occurs. The returned string length
// never exceeds max, so callers can rely on the cap as a hard upper
// bound when storing the result (e.g. in execution metadata).
func tailBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}

	marker := fmt.Sprintf("…[truncated %d bytes]\n", len(s)-max)
	if len(marker) >= max {
		return marker[:max]
	}

	tail := s[len(s)-(max-len(marker)):]
	return marker + tail
}

func (c *CreateRepositorySandbox) finish(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) error {
	metadata.Stage = repositorySandboxStageDone
	err := ctx.Metadata.Set(*metadata)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateRepositorySandboxPayloadType,
		[]any{*metadata},
	)
}

/*
 * Git remotes may be URI-style (https://..., ssh://...) or SCP-style (git@host:org/repo.git).
 * Handle only those two formats.
 */
func (c *CreateRepositorySandbox) getDirectoryName(repository string) (string, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return "", fmt.Errorf("failed to resolve repository directory from %q", repository)
	}

	if isURIStyleRepository(repository) {
		return getDirectoryFromURI(repository)
	}

	if isSCPStyleRepository(repository) {
		return getDirectoryFromSCP(repository)
	}

	return "", fmt.Errorf("repository must be URI or SCP format: %q", repository)
}

func isURIStyleRepository(repository string) bool {
	return strings.Contains(repository, "://")
}

func isSCPStyleRepository(repository string) bool {
	return strings.Contains(repository, "@") && strings.Contains(repository, ":") && !isURIStyleRepository(repository)
}

func getDirectoryFromURI(repository string) (string, error) {
	parsed, err := url.Parse(repository)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %v", err)
	}

	return directoryFromPath(parsed.Path, repository)
}

func getDirectoryFromSCP(repository string) (string, error) {
	parts := strings.SplitN(repository, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("failed to resolve repository directory from %q", repository)
	}

	return directoryFromPath(parts[1], repository)
}

func directoryFromPath(candidate, original string) (string, error) {
	candidate = strings.TrimSuffix(candidate, "/")
	if candidate == "" {
		return "", fmt.Errorf("failed to resolve repository directory from %q", original)
	}

	parts := strings.Split(candidate, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	if name == "" {
		return "", fmt.Errorf("failed to resolve repository directory from %q", original)
	}

	return name, nil
}

func (c *CreateRepositorySandbox) bootstrapCommand(metadata *CreateRepositorySandboxMetadata) string {
	return strings.Join(
		[]string{
			fmt.Sprintf("cd %s", shellQuote(metadata.Directory)),
			fmt.Sprintf("sh %s", shellQuote(*metadata.Bootstrap.Path)),
		},
		" && ",
	)
}

func (c *CreateRepositorySandbox) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateRepositorySandbox) Cleanup(ctx core.SetupContext) error {
	return nil
}
