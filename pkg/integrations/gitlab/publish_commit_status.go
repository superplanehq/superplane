package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_publish_commit_status.json
var exampleOutputPublishCommitStatus []byte

const (
	CommitStatusPayloadType = "gitlab.commitStatus"

	CommitStatusStatePending  = "pending"
	CommitStatusStateRunning  = "running"
	CommitStatusStateSuccess  = "success"
	CommitStatusStateFailed   = "failed"
	CommitStatusStateCanceled = "canceled"
	CommitStatusStateSkipped  = "skipped"
)

// commitStatusStates lists the states accepted by GitLab's commit status API.
var commitStatusStates = []string{
	CommitStatusStatePending,
	CommitStatusStateRunning,
	CommitStatusStateSuccess,
	CommitStatusStateFailed,
	CommitStatusStateCanceled,
	CommitStatusStateSkipped,
}

func commitStatusStateOptions() []configuration.FieldOption {
	return []configuration.FieldOption{
		{Label: "Pending", Value: CommitStatusStatePending},
		{Label: "Running", Value: CommitStatusStateRunning},
		{Label: "Success", Value: CommitStatusStateSuccess},
		{Label: "Failed", Value: CommitStatusStateFailed},
		{Label: "Canceled", Value: CommitStatusStateCanceled},
		{Label: "Skipped", Value: CommitStatusStateSkipped},
	}
}

type PublishCommitStatus struct{}

type PublishCommitStatusConfiguration struct {
	Project     string   `mapstructure:"project"`
	SHA         string   `mapstructure:"sha"`
	State       string   `mapstructure:"state"`
	Name        string   `mapstructure:"name"`
	TargetURL   string   `mapstructure:"targetUrl"`
	Description string   `mapstructure:"description"`
	Ref         string   `mapstructure:"ref"`
	Coverage    *float64 `mapstructure:"coverage"`
	PipelineID  string   `mapstructure:"pipelineId"`
}

func (c *PublishCommitStatus) Name() string {
	return "gitlab.publishCommitStatus"
}

func (c *PublishCommitStatus) Label() string {
	return "Publish Commit Status"
}

func (c *PublishCommitStatus) Description() string {
	return "Publish a build/CI status on a GitLab commit"
}

func (c *PublishCommitStatus) Documentation() string {
	return `The Publish Commit Status component sets a build/CI status on a specific commit, the way GitLab CI and external CI systems report per-commit check results.

## Use Cases

- **External CI reporting**: Report the outcome of a build, test, or scan run outside GitLab back onto the commit
- **Gate signalling**: Mark a commit pending while a workflow runs, then success or failed when it finishes
- **Custom checks**: Surface a workflow's own pass/fail state on the commit and merge request

## Configuration

- **Project** (required): The GitLab project containing the commit
- **Commit SHA** (required): The commit to attach the status to (supports expressions)
- **State** (required): One of pending, running, success, failed, canceled, skipped
- **Name** (optional): Label distinguishing this status from others on the same commit (GitLab defaults to ` + "`default`" + `)
- **Target URL** (optional): URL the status links to, e.g. the build or job page
- **Description** (optional): Short description shown alongside the status
- **Ref** (optional): Branch or tag the status is reported for
- **Coverage** (optional): Total code coverage percentage
- **Pipeline ID** (optional): Attach the status to a specific pipeline when several ran on the commit

## Permissions

The connected token needs the ` + "`api`" + ` scope and at least the **Developer** role on the project.

## Output

Returns the created commit status object.`
}

func (c *PublishCommitStatus) Icon() string {
	return "gitlab"
}

func (c *PublishCommitStatus) Color() string {
	return "orange"
}

func (c *PublishCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputPublishCommitStatus, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *PublishCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "sha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "{{ event.data.checkout_sha }}",
			Description: "The commit SHA to attach the status to. Supports expressions.",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  CommitStatusStateSuccess,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: commitStatusStateOptions(),
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Label distinguishing this status from others on the same commit",
		},
		{
			Name:        "targetUrl",
			Label:       "Target URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "URL the status links to, e.g. the build or job page",
		},
		{
			Name:     "description",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Branch or tag the status is reported for",
		},
		{
			Name:        "coverage",
			Label:       "Coverage",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Total code coverage percentage",
		},
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Attach the status to a specific pipeline when several ran on the commit",
		},
	}
}

func (c *PublishCommitStatus) Setup(ctx core.SetupContext) error {
	var config PublishCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if strings.TrimSpace(config.SHA) == "" {
		return errors.New("commit SHA is required")
	}

	if strings.TrimSpace(config.State) == "" {
		return errors.New("state is required")
	}

	if !slices.Contains(commitStatusStates, config.State) {
		return fmt.Errorf("invalid state %q: must be one of pending, running, success, failed, canceled, skipped", config.State)
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *PublishCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config PublishCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.SHA) == "" {
		return errors.New("commit SHA is required")
	}

	if !slices.Contains(commitStatusStates, config.State) {
		return fmt.Errorf("invalid state %q: must be one of pending, running, success, failed, canceled, skipped", config.State)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	// Optional fields are omitted when empty (the request struct uses omitempty).
	// Name and Ref are context identifiers, so trim whitespace - otherwise a padded value
	// publishes under a context that won't match on read-back.
	// Coverage is a pointer so an explicit 0 is still sent while an unset value is dropped.
	req := &CreateCommitStatusRequest{
		State:       config.State,
		Name:        strings.TrimSpace(config.Name),
		TargetURL:   config.TargetURL,
		Description: config.Description,
		Ref:         strings.TrimSpace(config.Ref),
		Coverage:    config.Coverage,
	}

	if pipelineID := strings.TrimSpace(config.PipelineID); pipelineID != "" {
		id, err := parseWholeNumberID(pipelineID, "pipeline ID")
		if err != nil {
			return err
		}
		req.PipelineID = &id
	}

	status, err := client.CreateCommitStatus(context.Background(), config.Project, strings.TrimSpace(config.SHA), req)
	if err != nil {
		return fmt.Errorf("failed to publish commit status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CommitStatusPayloadType,
		[]any{status},
	)
}

func (c *PublishCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *PublishCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PublishCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PublishCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *PublishCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
