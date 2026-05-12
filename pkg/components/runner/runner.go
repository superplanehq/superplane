package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const (
	channelSuccess = "success"
	channelFailed  = "failed"
	hookPoll       = "poll"
	payloadType    = "runner.result"

	defaultTimeoutSeconds = 600
	defaultPollInterval   = 1 * time.Second
	maxCapturedLogBytes   = 64 * 1024
)

var environmentVariableNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// cloneRepoFieldVisibility hides repository sub-fields unless "Clone repository" is on (matches JS String(true) === "true").
var cloneRepoFieldVisibility = []configuration.VisibilityCondition{
	{Field: "enabled", Values: []string{"true"}},
}

func init() {
	registry.RegisterAction("runner", &Runner{})
}

type Runner struct{}

type Spec struct {
	Source       *SourceSpec           `json:"source,omitempty" mapstructure:"source"`
	Commands     string                `json:"commands" mapstructure:"commands"`
	Environment  []EnvironmentVariable `json:"environment,omitempty" mapstructure:"environment"`
	Secrets      []SecretVariable      `json:"secrets,omitempty" mapstructure:"secrets"`
	Timeout      int                   `json:"timeout" mapstructure:"timeout"`
	RuntimeImage string                `json:"runtimeImage,omitempty" mapstructure:"runtimeImage"`
	ComputeSize  string                `json:"computeSize,omitempty" mapstructure:"computeSize"`
}

type SourceSpec struct {
	Enabled    bool                       `json:"enabled,omitempty" mapstructure:"enabled"`
	Repository string                     `json:"repository" mapstructure:"repository"`
	Ref        string                     `json:"ref" mapstructure:"ref"`
	Depth      int                        `json:"depth,omitempty" mapstructure:"depth"`
	Username   string                     `json:"username,omitempty" mapstructure:"username"`
	Token      configuration.SecretKeyRef `json:"token,omitempty" mapstructure:"token"`
}

type EnvironmentVariable struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type SecretVariable struct {
	Name  string                     `json:"name" mapstructure:"name"`
	Value configuration.SecretKeyRef `json:"value" mapstructure:"value"`
}

type ExecutionMetadata struct {
	BuildID           string                 `json:"buildId" mapstructure:"buildId"`
	BuildARN          string                 `json:"buildArn" mapstructure:"buildArn"`
	Status            string                 `json:"status" mapstructure:"status"`
	StartedAt         string                 `json:"startedAt,omitempty" mapstructure:"startedAt"`
	FinishedAt        string                 `json:"finishedAt,omitempty" mapstructure:"finishedAt"`
	ExitCode          *int                   `json:"exitCode,omitempty" mapstructure:"exitCode"`
	Source            *SourceMetadata        `json:"source,omitempty" mapstructure:"source"`
	RuntimeImage      string                 `json:"runtimeImage,omitempty" mapstructure:"runtimeImage"`
	ComputeSize       string                 `json:"computeSize,omitempty" mapstructure:"computeSize"`
	DockerEnabled     bool                   `json:"dockerEnabled" mapstructure:"dockerEnabled"`
	Artifacts         []ArtifactMetadata     `json:"artifacts,omitempty" mapstructure:"artifacts"`
	Logs              LogMetadata            `json:"logs" mapstructure:"logs"`
	Output            OutputMetadata         `json:"output" mapstructure:"output"`
	ConfigurationHash map[string]interface{} `json:"-" mapstructure:"-"`
}

type SourceMetadata struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Ref        string `json:"ref,omitempty" mapstructure:"ref"`
	CommitSHA  string `json:"commitSha,omitempty" mapstructure:"commitSha"`
}

type ArtifactMetadata struct {
	Name string `json:"name" mapstructure:"name"`
	Path string `json:"path" mapstructure:"path"`
}

type LogMetadata struct {
	GroupName  string `json:"groupName,omitempty" mapstructure:"groupName"`
	StreamName string `json:"streamName,omitempty" mapstructure:"streamName"`
	DeepLink   string `json:"deepLink,omitempty" mapstructure:"deepLink"`
}

type OutputMetadata struct {
	Stdout    string `json:"stdout" mapstructure:"stdout"`
	Stderr    string `json:"stderr" mapstructure:"stderr"`
	Truncated bool   `json:"truncated" mapstructure:"truncated"`
}

type backendConfig struct {
	Region      string
	Project     string
	Credentials *aws.Credentials
}

func (c *Runner) Name() string  { return "runner" }
func (c *Runner) Label() string { return "Runner" }
func (c *Runner) Description() string {
	return "Run shell commands on provisioned compute with full logs, managed by your SuperPlane deployment."
}
func (c *Runner) Documentation() string {
	return `Run shell commands in a managed, ephemeral environment on dedicated runners.

## Use Cases

- Clone a repository and run project scripts
- Build and push Docker images
- Run Terraform plan/apply/destroy commands
- Execute release, deployment, migration, diagnostic, or remediation scripts

## Server configuration

Self-hosted SuperPlane must configure dedicated RUNNER_* environment variables for the execution backend. See your installation or operations guide for the required variables and how to set them.

## Output

- **success**: Command exits with code 0
- **failed**: Command runs and exits with a non-zero code

Submission, checkout, credential injection, or infrastructure issues are shown as execution errors.`
}
func (c *Runner) Icon() string  { return "terminal" }
func (c *Runner) Color() string { return "blue" }

func (c *Runner) ExampleOutput() map[string]any {
	return map[string]any{
		"command": map[string]any{
			"exitCode":        0,
			"status":          "SUCCEEDED",
			"durationSeconds": 42,
			"stdout":          "Successfully built image\n",
			"stderr":          "",
			"source": map[string]any{
				"repository": "github.com/example/app",
				"ref":        "main",
				"commitSha":  "abc123",
			},
			"buildId":  "superplane-runner:example",
			"buildArn": "arn:example:superplane-run/abc123",
			"logUrl":   "https://example.com/logs/...",
		},
	}
}

func (c *Runner) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelSuccess, Label: "Success"},
		{Name: channelFailed, Label: "Failed"},
	}
}

func (c *Runner) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:    "computeSize",
			Label:   "Compute size",
			Type:    configuration.FieldTypeSelect,
			Default: "small",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				// Matches AWS CodeBuild Linux general1.small | medium | large (standard compute types).
				{Label: "2 vCPU · 3 GB", Value: "small"},
				{Label: "4 vCPU · 7 GB", Value: "medium"},
				{Label: "8 vCPU · 15 GB", Value: "large"},
			}}},
		},
		sourceField(),
		{
			Name:        "commands",
			Label:       "Commands",
			Type:        configuration.FieldTypeText,
			Description: "Bash commands or script body to execute. Supports expressions.",
			Placeholder: "docker build -t registry.example.com/app:$TAG .\ndocker push registry.example.com/app:$TAG",
			Required:    true,
		},
		listField("environment", "Environment variables", "Variable", []configuration.Field{
			{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true, Placeholder: "e.g. IMAGE_TAG"},
			{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true, Description: "Supports expressions.", Placeholder: "e.g. {{ $.version }}"},
		}),
		listField("secrets", "Secret environment variables", "Secret", []configuration.Field{
			{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true, Placeholder: "e.g. DOCKER_PASSWORD"},
			{Name: "value", Label: "Secret key", Type: configuration.FieldTypeSecretKey, Required: true},
		}),
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Default:     defaultTimeoutSeconds,
			Description: "Maximum execution time.",
		},
		{
			Name:        "runtimeImage",
			Label:       "Environment preset",
			Type:        configuration.FieldTypeSelect,
			Default:     "default",
			Description: "Server-defined tooling preset (not a custom image URL). Match Docker or Terraform to the CLIs your commands need.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Default · general-purpose CLI", Value: "default"},
				{Label: "Docker · includes Docker CLI", Value: "docker"},
				{Label: "Terraform · includes Terraform CLI", Value: "terraform"},
			}}},
		},
	}
}

func sourceField() configuration.Field {
	return objectField("source", "Repository source", "", []configuration.Field{
		{Name: "enabled", Label: "Clone repository", Type: configuration.FieldTypeBool, Default: false, Description: "When enabled, clones the repository before your commands run."},
		{Name: "repository", Label: "Repository URL", Type: configuration.FieldTypeString, Placeholder: "https://github.com/org/repo.git", VisibilityConditions: cloneRepoFieldVisibility},
		{Name: "ref", Label: "Ref", Type: configuration.FieldTypeString, Placeholder: "main", VisibilityConditions: cloneRepoFieldVisibility},
		{Name: "depth", Label: "Checkout depth", Type: configuration.FieldTypeNumber, Default: 1, VisibilityConditions: cloneRepoFieldVisibility},
		{Name: "username", Label: "Username", Type: configuration.FieldTypeString, Placeholder: "git", VisibilityConditions: cloneRepoFieldVisibility},
		{Name: "token", Label: "Token", Type: configuration.FieldTypeSecretKey, VisibilityConditions: cloneRepoFieldVisibility},
	})
}

func objectField(name, label, description string, schema []configuration.Field) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeObject,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			Object: &configuration.ObjectTypeOptions{Schema: schema},
		},
	}
}

func listField(name, label, itemLabel string, schema []configuration.Field) configuration.Field {
	return configuration.Field{
		Name:  name,
		Label: label,
		Type:  configuration.FieldTypeList,
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: itemLabel,
				ItemDefinition: &configuration.ListItemDefinition{
					Type:   configuration.FieldTypeObject,
					Schema: schema,
				},
			},
		},
	}
}

func (c *Runner) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
}

func (c *Runner) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Runner) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	backend, err := loadBackendConfig()
	if err != nil {
		return err
	}

	env, err := buildEnvironment(ctx.Secrets, spec)
	if err != nil {
		return err
	}

	client := newCodeBuildClient(ctx.HTTP, backend.Credentials, backend.Region)
	build, err := client.startBuild(startBuildInput{
		ProjectName:                  backend.Project,
		BuildspecOverride:            buildspec(spec),
		SourceTypeOverride:           "NO_SOURCE",
		EnvironmentVariablesOverride: env,
		TimeoutInMinutesOverride:     timeoutMinutes(spec.Timeout),
	})
	if err != nil {
		return fmt.Errorf("failed to start run: %w", err)
	}

	metadata := metadataFromBuild(build, spec, nil)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}
	if build.ID != "" {
		if err := ctx.ExecutionState.SetKV("codebuild_build_id", build.ID); err != nil {
			return err
		}
	}

	if isTerminalStatus(build.BuildStatus) {
		return c.complete(ctx.HTTP, ctx.ExecutionState, ctx.Metadata, client, metadata)
	}

	return ctx.Requests.ScheduleActionCall(hookPoll, map[string]any{}, defaultPollInterval)
}

func (c *Runner) Hooks() []core.Hook {
	return []core.Hook{{Type: core.HookTypeInternal, Name: hookPoll}}
}

func (c *Runner) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != hookPoll {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	backend, err := loadBackendConfig()
	if err != nil {
		return err
	}

	metadata, err := decodeMetadata(ctx.Metadata.Get())
	if err != nil {
		return err
	}
	if metadata.BuildID == "" {
		return errors.New("run identifier is missing from execution metadata")
	}

	client := newCodeBuildClient(ctx.HTTP, backend.Credentials, backend.Region)
	build, err := client.getBuild(metadata.BuildID)
	if err != nil {
		return err
	}

	metadata = metadataFromBuild(build, spec, &metadata)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	if !isTerminalStatus(build.BuildStatus) {
		return ctx.Requests.ScheduleActionCall(hookPoll, map[string]any{}, defaultPollInterval)
	}

	return c.complete(ctx.HTTP, ctx.ExecutionState, ctx.Metadata, client, metadata)
}

func (c *Runner) complete(
	httpCtx core.HTTPContext,
	state core.ExecutionStateContext,
	metadataWriter core.MetadataWriter,
	client *codeBuildClient,
	metadata ExecutionMetadata,
) error {
	events, err := client.getLogEvents(metadata.Logs.GroupName, metadata.Logs.StreamName)
	if err == nil {
		metadata.Output.Stdout, metadata.Output.Truncated = captureLogOutput(events)
		if exitCode, ok := parseExitCode(metadata.Output.Stdout); ok {
			metadata.ExitCode = &exitCode
		}
	}

	if err := metadataWriter.Set(metadata); err != nil {
		return err
	}

	if isSuccessfulStatus(metadata.Status) && (metadata.ExitCode == nil || *metadata.ExitCode == 0) {
		payload := fitRunnerEmitPayload(payloadFromMetadata(metadata))
		return state.Emit(channelSuccess, payloadType, []any{payload})
	}

	if isFailedCommandStatus(metadata.Status) {
		payload := fitRunnerEmitPayload(payloadFromMetadata(metadata))
		return state.Emit(channelFailed, payloadType, []any{payload})
	}

	return state.Fail("RESULT_REASON_ERROR", fmt.Sprintf("run ended with unexpected status: %s", metadata.Status))
}

func (c *Runner) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Runner) Cancel(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	backend, err := loadBackendConfig()
	if err != nil {
		return err
	}
	metadata, err := decodeMetadata(ctx.Metadata.Get())
	if err != nil {
		return err
	}
	if metadata.BuildID == "" || isTerminalStatus(metadata.Status) {
		return nil
	}

	client := newCodeBuildClient(ctx.HTTP, backend.Credentials, backend.Region)
	build, err := client.stopBuild(metadata.BuildID)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(metadataFromBuild(build, spec, &metadata))
}

func (c *Runner) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeSpec(raw any) (Spec, error) {
	spec := Spec{}
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeSourceEnabledFromRaw(raw, &spec)
	if spec.Timeout == 0 {
		spec.Timeout = defaultTimeoutSeconds
	}
	return spec, nil
}

// normalizeSourceEnabledFromRaw sets Source.Enabled for workflows saved before the toggle existed:
// if "enabled" was never stored and a repository URL is present, treat clone as on.
func normalizeSourceEnabledFromRaw(raw any, spec *Spec) {
	if spec.Source == nil {
		return
	}
	srcMap, ok := sourceConfigMap(raw)
	if !ok {
		return
	}
	if _, hasEnabled := srcMap["enabled"]; hasEnabled {
		return
	}
	if strings.TrimSpace(spec.Source.Repository) != "" {
		spec.Source.Enabled = true
	}
}

func sourceConfigMap(raw any) (map[string]any, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	src, ok := m["source"].(map[string]any)
	if !ok || src == nil {
		return nil, false
	}
	return src, true
}

func sourceCloneRequested(spec Spec) bool {
	return spec.Source != nil && spec.Source.Enabled && strings.TrimSpace(spec.Source.Repository) != ""
}

func validateSpec(spec Spec) error {
	if strings.TrimSpace(spec.Commands) == "" {
		return errors.New("commands is required")
	}
	if spec.Timeout < 1 {
		return errors.New("timeout must be at least 1 second")
	}
	for _, variable := range spec.Environment {
		if err := validateEnvironmentName(variable.Name); err != nil {
			return err
		}
	}
	for _, secret := range spec.Secrets {
		if err := validateEnvironmentName(secret.Name); err != nil {
			return err
		}
		if !secret.Value.IsSet() {
			return fmt.Errorf("secret value is required for %s", secret.Name)
		}
	}
	if spec.Source != nil {
		if spec.Source.Depth < 0 {
			return errors.New("source checkout depth must be 0 or greater")
		}
		if spec.Source.Enabled {
			if strings.TrimSpace(spec.Source.Repository) == "" {
				return errors.New("source repository is required when clone repository is enabled")
			}
		} else {
			if strings.TrimSpace(spec.Source.Repository) == "" && (strings.TrimSpace(spec.Source.Ref) != "" || spec.Source.Token.IsSet()) {
				return errors.New("enable clone repository or clear source ref and token")
			}
		}
	}
	return nil
}

func validateEnvironmentName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("environment variable name is required")
	}
	if !environmentVariableNameRegex.MatchString(name) {
		return fmt.Errorf("invalid environment variable name: %s", name)
	}
	return nil
}

func loadBackendConfig() (backendConfig, error) {
	region := strings.TrimSpace(os.Getenv("RUNNER_CODEBUILD_REGION"))
	if region == "" {
		return backendConfig{}, errors.New("RUNNER_CODEBUILD_REGION is required")
	}

	project := os.Getenv("RUNNER_CODEBUILD_PROJECT")
	if strings.TrimSpace(project) == "" {
		return backendConfig{}, errors.New("RUNNER_CODEBUILD_PROJECT is required")
	}

	accessKey := strings.TrimSpace(os.Getenv("RUNNER_AWS_ACCESS_KEY_ID"))
	secretKey := strings.TrimSpace(os.Getenv("RUNNER_AWS_SECRET_ACCESS_KEY"))
	sessionToken := strings.TrimSpace(os.Getenv("RUNNER_AWS_SESSION_TOKEN"))
	if accessKey == "" || secretKey == "" {
		return backendConfig{}, errors.New("RUNNER_AWS_ACCESS_KEY_ID and RUNNER_AWS_SECRET_ACCESS_KEY are required")
	}

	return backendConfig{
		Region:  region,
		Project: project,
		Credentials: &aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    sessionToken,
			Source:          "runner-env",
		},
	}, nil
}

func buildEnvironment(secrets core.SecretsContext, spec Spec) ([]environmentVariableOverride, error) {
	env := []environmentVariableOverride{
		{Name: "SUPERPLANE_RUNTIME_IMAGE", Value: firstNonEmpty(spec.RuntimeImage, "default"), Type: "PLAINTEXT"},
		{Name: "SUPERPLANE_COMPUTE_SIZE", Value: firstNonEmpty(spec.ComputeSize, "small"), Type: "PLAINTEXT"},
	}
	for _, variable := range spec.Environment {
		env = append(env, environmentVariableOverride{Name: variable.Name, Value: variable.Value, Type: "PLAINTEXT"})
	}
	for _, secret := range spec.Secrets {
		value, err := secrets.GetKey(secret.Value.Secret, secret.Value.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret for %s: %w", secret.Name, err)
		}
		env = append(env, environmentVariableOverride{Name: secret.Name, Value: string(value), Type: "PLAINTEXT"})
	}
	if sourceCloneRequested(spec) {
		env = append(env,
			environmentVariableOverride{Name: "SUPERPLANE_SOURCE_REPOSITORY", Value: spec.Source.Repository, Type: "PLAINTEXT"},
			environmentVariableOverride{Name: "SUPERPLANE_SOURCE_REF", Value: spec.Source.Ref, Type: "PLAINTEXT"},
			environmentVariableOverride{Name: "SUPERPLANE_SOURCE_DEPTH", Value: strconv.Itoa(spec.Source.Depth), Type: "PLAINTEXT"},
			environmentVariableOverride{Name: "SUPERPLANE_SOURCE_USERNAME", Value: firstNonEmpty(spec.Source.Username, "git"), Type: "PLAINTEXT"},
		)
		if spec.Source.Token.IsSet() {
			value, err := secrets.GetKey(spec.Source.Token.Secret, spec.Source.Token.Key)
			if err != nil {
				return nil, fmt.Errorf("failed to read source token: %w", err)
			}
			env = append(env, environmentVariableOverride{Name: "SUPERPLANE_SOURCE_TOKEN", Value: string(value), Type: "PLAINTEXT"})
		}
	}
	return env, nil
}

func buildspec(spec Spec) string {
	script := renderScript(spec)
	return fmt.Sprintf(`version: 0.2
env:
  shell: bash
phases:
  build:
    commands:
      - |
        cat > /tmp/superplane-run.sh <<'SUPERPLANE_SCRIPT'
%s
        SUPERPLANE_SCRIPT
        chmod +x /tmp/superplane-run.sh
        /tmp/superplane-run.sh
`, indent(script, 8))
}

func renderScript(spec Spec) string {
	commands := strings.TrimRight(spec.Commands, "\n")
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -u\n")
	b.WriteString("workspace=\"$CODEBUILD_SRC_DIR\"\n")
	b.WriteString("cd \"$workspace\"\n")
	if sourceCloneRequested(spec) {
		b.WriteString(`repo="$SUPERPLANE_SOURCE_REPOSITORY"
clone_url="$repo"
if [[ -n "${SUPERPLANE_SOURCE_TOKEN:-}" && "$repo" == https://* ]]; then
  clone_url="https://${SUPERPLANE_SOURCE_USERNAME:-git}:${SUPERPLANE_SOURCE_TOKEN}@${repo#https://}"
fi
clone_args=()
if [[ -n "${SUPERPLANE_SOURCE_DEPTH:-}" && "$SUPERPLANE_SOURCE_DEPTH" != "0" ]]; then
  clone_args+=(--depth "$SUPERPLANE_SOURCE_DEPTH")
fi
if [[ -n "${SUPERPLANE_SOURCE_REF:-}" ]]; then
  clone_args+=(--branch "$SUPERPLANE_SOURCE_REF")
fi
git clone "${clone_args[@]}" "$clone_url" source
cd source
resolved_sha="$(git rev-parse HEAD || true)"
echo "SUPERPLANE_SOURCE_COMMIT_SHA=$resolved_sha"
`)
	}
	b.WriteString("set +e\n")
	b.WriteString("bash <<'SUPERPLANE_USER_COMMANDS'\n")
	b.WriteString(commands)
	b.WriteString("\nSUPERPLANE_USER_COMMANDS\n")
	b.WriteString("exit_code=$?\n")
	b.WriteString("echo \"SUPERPLANE_EXIT_CODE=$exit_code\"\n")
	b.WriteString("exit \"$exit_code\"\n")
	return b.String()
}

func indent(value string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func timeoutMinutes(seconds int) int {
	if seconds < 1 {
		seconds = defaultTimeoutSeconds
	}
	return (seconds + 59) / 60
}

func metadataFromBuild(build *build, spec Spec, existing *ExecutionMetadata) ExecutionMetadata {
	metadata := ExecutionMetadata{}
	if existing != nil {
		metadata = *existing
	}
	if build != nil {
		metadata.BuildID = build.ID
		metadata.BuildARN = build.ARN
		metadata.Status = build.BuildStatus
		metadata.Logs = LogMetadata{
			GroupName:  build.Logs.GroupName,
			StreamName: build.Logs.StreamName,
			DeepLink:   build.Logs.DeepLink,
		}
		if !build.StartTime.IsZero() {
			metadata.StartedAt = build.StartTime.Format(time.RFC3339)
		}
		if !build.EndTime.IsZero() {
			metadata.FinishedAt = build.EndTime.Format(time.RFC3339)
		}
	}
	metadata.RuntimeImage = firstNonEmpty(spec.RuntimeImage, "default")
	metadata.ComputeSize = firstNonEmpty(spec.ComputeSize, "small")
	if sourceCloneRequested(spec) {
		source := SourceMetadata{Repository: sanitizeRepository(spec.Source.Repository), Ref: spec.Source.Ref}
		if existing != nil && existing.Source != nil {
			source.CommitSHA = existing.Source.CommitSHA
		}
		metadata.Source = &source
	}
	return metadata
}

func decodeMetadata(raw any) (ExecutionMetadata, error) {
	metadata := ExecutionMetadata{}
	if raw == nil {
		return metadata, nil
	}
	if err := mapstructure.Decode(raw, &metadata); err != nil {
		return ExecutionMetadata{}, fmt.Errorf("failed to decode execution metadata: %w", err)
	}
	return metadata, nil
}

// workflowEventEnvelope matches pkg/workers/contexts.ExecutionStateContext.Emit JSON shape.
func workflowEventEnvelope(payloadTypeName string, data map[string]any) map[string]any {
	return map[string]any{
		"type":      payloadTypeName,
		"timestamp": time.Now(),
		"data":      data,
	}
}

// fitRunnerEmitPayload trims stdout/stderr in the emitted payload so the marshaled workflow event
// stays within DefaultMaxPayloadSize (the limit includes type, timestamp, and wrapper JSON).
func fitRunnerEmitPayload(data map[string]any) map[string]any {
	max := contexts.DefaultMaxPayloadSize
	for range 10000 {
		raw, err := json.Marshal(workflowEventEnvelope(payloadType, data))
		if err != nil {
			return data
		}
		if len(raw) <= max {
			return data
		}
		cmd, ok := data["command"].(map[string]any)
		if !ok {
			return data
		}
		if shrinkRunnerCommandStreams(cmd) {
			cmd["outputTruncated"] = true
			continue
		}
		if deleteRunnerOptionalCommandFields(cmd) {
			cmd["outputTruncated"] = true
			continue
		}
		return data
	}

	return data
}

func deleteRunnerOptionalCommandFields(cmd map[string]any) bool {
	removed := false
	if _, ok := cmd["artifacts"]; ok {
		delete(cmd, "artifacts")
		removed = true
	}
	if _, ok := cmd["source"]; ok {
		delete(cmd, "source")
		removed = true
	}
	return removed
}

func shrinkRunnerCommandStreams(cmd map[string]any) bool {
	if s, ok := cmd["stdout"].(string); ok && len(s) > 0 {
		step := runnerStreamShrinkStep(len(s))
		newLen := len(s) - step
		if newLen < 0 {
			newLen = 0
		}
		cmd["stdout"] = truncateStringByBytes(s, newLen)
		return true
	}
	if s, ok := cmd["stderr"].(string); ok && len(s) > 0 {
		step := runnerStreamShrinkStep(len(s))
		newLen := len(s) - step
		if newLen < 0 {
			newLen = 0
		}
		cmd["stderr"] = truncateStringByBytes(s, newLen)
		return true
	}
	return false
}

func runnerStreamShrinkStep(n int) int {
	if n <= 1 {
		return 1
	}
	step := n / 4
	if step < 512 {
		step = 512
	}
	return step
}

// truncateStringByBytes cuts s to at most maxBytes UTF-8 octets without splitting a code point.
func truncateStringByBytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	b := []byte(s)
	if len(b) <= maxBytes {
		return s
	}
	b = b[:maxBytes]
	for len(b) > 0 && (b[len(b)-1]&0xc0) == 0x80 {
		b = b[:len(b)-1]
	}
	return string(b)
}

func payloadFromMetadata(metadata ExecutionMetadata) map[string]any {
	command := map[string]any{
		"exitCode":        metadata.ExitCode,
		"status":          metadata.Status,
		"stdout":          metadata.Output.Stdout,
		"stderr":          metadata.Output.Stderr,
		"buildId":         metadata.BuildID,
		"buildArn":        metadata.BuildARN,
		"logUrl":          metadata.Logs.DeepLink,
		"outputTruncated": metadata.Output.Truncated,
	}
	if metadata.StartedAt != "" && metadata.FinishedAt != "" {
		if started, err := time.Parse(time.RFC3339, metadata.StartedAt); err == nil {
			if finished, err := time.Parse(time.RFC3339, metadata.FinishedAt); err == nil {
				command["durationSeconds"] = int(finished.Sub(started).Seconds())
			}
		}
	}
	if metadata.Source != nil {
		command["source"] = metadata.Source
	}
	if len(metadata.Artifacts) > 0 {
		command["artifacts"] = metadata.Artifacts
	}
	return map[string]any{"command": command}
}

func captureLogOutput(events []logEvent) (string, bool) {
	var b strings.Builder
	truncated := false
	for _, event := range events {
		if b.Len()+len(event.Message)+1 > maxCapturedLogBytes {
			remaining := maxCapturedLogBytes - b.Len()
			if remaining > 0 {
				b.WriteString(event.Message[:min(len(event.Message), remaining)])
			}
			truncated = true
			break
		}
		b.WriteString(event.Message)
		if !strings.HasSuffix(event.Message, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String(), truncated
}

func parseExitCode(output string) (int, bool) {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "SUPERPLANE_EXIT_CODE=") {
			continue
		}
		code, err := strconv.Atoi(strings.TrimPrefix(line, "SUPERPLANE_EXIT_CODE="))
		return code, err == nil
	}
	return 0, false
}

func isTerminalStatus(status string) bool {
	switch status {
	case "SUCCEEDED", "FAILED", "FAULT", "STOPPED", "TIMED_OUT":
		return true
	default:
		return false
	}
}

func isSuccessfulStatus(status string) bool {
	return status == "SUCCEEDED"
}

func isFailedCommandStatus(status string) bool {
	switch status {
	case "FAILED", "TIMED_OUT":
		return true
	default:
		return false
	}
}

func sanitizeRepository(repository string) string {
	repository = strings.TrimPrefix(repository, "https://")
	repository = strings.TrimPrefix(repository, "http://")
	if at := strings.LastIndex(repository, "@"); at >= 0 {
		repository = repository[at+1:]
	}
	return repository
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m ExecutionMetadata) MarshalJSON() ([]byte, error) {
	type Alias ExecutionMetadata
	return json.Marshal((Alias)(m))
}
