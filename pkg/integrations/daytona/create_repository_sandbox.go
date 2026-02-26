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
	CreateRepositorySandboxPayloadType    = "daytona.repository.sandbox"
	CreateRepositorySandboxPollInterval   = 5 * time.Second
	CreateRepositorySandboxDefaultTimeout = 5 * time.Minute
	CreateRepositorySandboxCloneBasePath  = "/home/daytona"
)

const (
	SandboxBootstrapFromInline = "inline"
	SandboxBootstrapFromFile   = "file"
)

const (
	repositorySandboxStageWaitingSandbox = "waitingSandbox"
	repositorySandboxStageCloningRepo    = "cloningRepository"
	repositorySandboxStageBootstrapping  = "bootstrapping"
	repositorySandboxStageDone           = "done"
)

type CreateRepositorySandbox struct{}

type CreateRepositorySandboxSpec struct {
	Snapshot         string                                `json:"snapshot,omitempty"`
	Target           string                                `json:"target,omitempty"`
	AutoStopInterval int                                   `json:"autoStopInterval,omitempty"`
	Env              []EnvVariable                         `json:"env,omitempty"`
	Repository       string                                `json:"repository"`
	Bootstrap        *CreateRepositorySandboxBootstrapSpec `json:"bootstrap"`
}

type CreateRepositorySandboxBootstrapSpec struct {
	From   string `json:"from,omitempty"`
	Script string `json:"script,omitempty"`
	Path   string `json:"path,omitempty"`
	URL    string `json:"url,omitempty"`
}

type CreateRepositorySandboxMetadata struct {
	Stage            string             `json:"stage" mapstructure:"stage"`
	SandboxID        string             `json:"sandboxId" mapstructure:"sandboxId"`
	SandboxStartedAt string             `json:"sandboxStartedAt" mapstructure:"sandboxStartedAt"`
	SessionID        string             `json:"sessionId" mapstructure:"sessionId"`
	Timeout          int                `json:"timeout" mapstructure:"timeout"`
	Repository       string             `json:"repository" mapstructure:"repository"`
	Directory        string             `json:"directory" mapstructure:"directory"`
	Clone            *CloneMetadata     `json:"clone,omitempty" mapstructure:"clone,omitempty"`
	Bootstrap        *BootstrapMetadata `json:"bootstrap,omitempty" mapstructure:"bootstrap,omitempty"`
}

type CloneMetadata struct {
	CmdID      string `json:"cmdId" mapstructure:"cmdId"`
	StartedAt  string `json:"startedAt" mapstructure:"startedAt"`
	FinishedAt string `json:"finishedAt" mapstructure:"finishedAt"`
	ExitCode   int    `json:"exitCode" mapstructure:"exitCode"`
	Result     string `json:"result" mapstructure:"result"`
}

type BootstrapMetadata struct {
	CmdID      string  `json:"cmdId" mapstructure:"cmdId"`
	StartedAt  string  `json:"startedAt" mapstructure:"startedAt"`
	FinishedAt string  `json:"finishedAt" mapstructure:"finishedAt"`
	ExitCode   int     `json:"exitCode" mapstructure:"exitCode"`
	Result     string  `json:"result" mapstructure:"result"`
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
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Repository URL to clone",
			Placeholder: "https://github.com/owner/repository.git",
		},
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

	sandbox, err := client.CreateSandbox(&CreateSandboxRequest{
		Snapshot:         spec.Snapshot,
		Target:           spec.Target,
		AutoStopInterval: spec.AutoStopInterval,
		Env:              envMap,
	})
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %v", err)
	}

	repositoryDirectory, err := c.getDirectoryName(spec.Repository)
	if err != nil {
		return fmt.Errorf("failed to determine repository directory name: %v", err)
	}

	bootstrapMetadata, err := c.bootstrapMetadataFromSpec(spec)
	if err != nil {
		return err
	}

	metadata := CreateRepositorySandboxMetadata{
		Stage:            repositorySandboxStageWaitingSandbox,
		SandboxID:        sandbox.ID,
		SandboxStartedAt: time.Now().Format(time.RFC3339),
		Timeout:          int(CreateRepositorySandboxDefaultTimeout.Seconds()),
		Repository:       strings.TrimSpace(spec.Repository),
		Directory:        path.Join(CreateRepositorySandboxCloneBasePath, repositoryDirectory),
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

func (c *CreateRepositorySandbox) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateRepositorySandbox) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateRepositorySandbox) poll(ctx core.ActionContext) error {
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
		return fmt.Errorf("bootstrapping repository sandbox %s timed out after %v", metadata.SandboxID, metadata.Timeout)
	}

	switch metadata.Stage {
	case repositorySandboxStageWaitingSandbox:
		return c.pollWaitingSandbox(ctx, &metadata)

	case repositorySandboxStageCloningRepo:
		return c.pollCloningRepository(ctx, &metadata)

	case repositorySandboxStageBootstrapping:
		return c.pollBootstrapping(ctx, &metadata)

	default:
		return fmt.Errorf("unknown create repository sandbox stage: %s", metadata.Stage)
	}
}

func (c *CreateRepositorySandbox) pollWaitingSandbox(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) error {
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
		return c.startClone(ctx, client, metadata)
	case "error":
		return fmt.Errorf("sandbox %s failed to start", metadata.SandboxID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
	}
}

func (c *CreateRepositorySandbox) startClone(ctx core.ActionContext, client *Client, metadata *CreateRepositorySandboxMetadata) error {
	sessionID := uuid.New().String()
	if err := client.CreateSession(metadata.SandboxID, sessionID); err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	command := fmt.Sprintf(
		"git clone --depth 1 %s %s",
		shellQuote(metadata.Repository),
		shellQuote(metadata.Directory),
	)

	response, err := client.ExecuteSessionCommand(metadata.SandboxID, sessionID, command)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %v", err)
	}

	metadata.Stage = repositorySandboxStageCloningRepo
	metadata.SessionID = sessionID
	metadata.Clone = &CloneMetadata{
		CmdID:     response.CmdID,
		StartedAt: time.Now().Format(time.RFC3339),
	}

	if err := ctx.Metadata.Set(*metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
}

func (c *CreateRepositorySandbox) pollCloningRepository(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) error {
	result, err := c.getCommandResult(ctx, metadata, metadata.Clone.CmdID)
	if err != nil {
		ctx.Logger.Errorf("failed to get clone command result for %s: %v", metadata.Clone.CmdID, err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
	}

	metadata.Clone.Result = result.Result
	metadata.Clone.FinishedAt = time.Now().Format(time.RFC3339)
	metadata.Clone.ExitCode = result.ExitCode

	if result.ExitCode != 0 {
		if err := ctx.Metadata.Set(*metadata); err != nil {
			return err
		}

		return fmt.Errorf(
			"repository clone failed with exit code %d: %s",
			result.ExitCode,
			result.ShortResult(),
		)
	}

	//
	// If no bootstrap is required, we can skip the bootstrap step.
	//
	if metadata.Bootstrap == nil {
		return c.finish(ctx, metadata)
	}

	//
	// Otherwise, we need to execute the bootstrap script.
	//
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	bootstrapCommand, err := c.bootstrapCommand(metadata)
	if err != nil {
		return err
	}

	response, err := client.ExecuteSessionCommand(metadata.SandboxID, metadata.SessionID, bootstrapCommand)
	if err != nil {
		return fmt.Errorf("failed to execute bootstrap script: %v", err)
	}

	metadata.Stage = repositorySandboxStageBootstrapping
	metadata.Bootstrap.CmdID = response.CmdID
	metadata.Bootstrap.StartedAt = time.Now().Format(time.RFC3339)
	if err := ctx.Metadata.Set(*metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
}

func (c *CreateRepositorySandbox) pollBootstrapping(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata) error {
	result, err := c.getCommandResult(ctx, metadata, metadata.Bootstrap.CmdID)
	if err != nil {
		ctx.Logger.Errorf("failed to get bootstrap command result for %s: %v", metadata.Bootstrap.CmdID, err)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CreateRepositorySandboxPollInterval)
	}

	metadata.Bootstrap.Result = result.Result
	metadata.Bootstrap.FinishedAt = time.Now().Format(time.RFC3339)
	metadata.Bootstrap.ExitCode = result.ExitCode

	if result.ExitCode != 0 {
		if err := ctx.Metadata.Set(*metadata); err != nil {
			return err
		}

		return fmt.Errorf(
			"bootstrap script failed with exit code %d: %s",
			result.ExitCode,
			result.ShortResult(),
		)
	}

	return c.finish(ctx, metadata)
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

func (c *CreateRepositorySandbox) getCommandResult(ctx core.ActionContext, metadata *CreateRepositorySandboxMetadata, cmdID string) (*ExecuteCommandResponse, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	session, err := client.GetSession(metadata.SandboxID, metadata.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session %s: %v", metadata.SessionID, err)
	}

	command := session.FindCommand(cmdID)
	if command == nil || command.ExitCode == nil {
		return nil, fmt.Errorf("command %s not found in session %s", cmdID, metadata.SessionID)
	}

	logs, err := client.GetSessionCommandLogs(metadata.SandboxID, metadata.SessionID, cmdID)
	if err != nil {
		ctx.Logger.Errorf("failed to get command logs for %s: %v", cmdID, err)
	}

	return &ExecuteCommandResponse{
		ExitCode: *command.ExitCode,
		Result:   logs,
	}, nil
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

func (c *CreateRepositorySandbox) bootstrapCommand(metadata *CreateRepositorySandboxMetadata) (string, error) {
	base := fmt.Sprintf("cd %s && ", shellQuote(metadata.Directory))

	switch metadata.Bootstrap.From {
	case SandboxBootstrapFromInline:
		return base + *metadata.Bootstrap.Script, nil
	case SandboxBootstrapFromFile:
		return fmt.Sprintf("%ssh %s", base, shellQuote(*metadata.Bootstrap.Path)), nil
	default:
		return "", fmt.Errorf("invalid bootstrap.from: %s", metadata.Bootstrap.From)
	}
}

func (c *CreateRepositorySandbox) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateRepositorySandbox) Cleanup(ctx core.SetupContext) error {
	return nil
}
