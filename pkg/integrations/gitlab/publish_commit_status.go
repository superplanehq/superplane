package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
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
	Project     string  `mapstructure:"project"`
	SHA         string  `mapstructure:"sha"`
	State       string  `mapstructure:"state"`
	Name        string  `mapstructure:"name"`
	TargetURL   string  `mapstructure:"targetUrl"`
	Description string  `mapstructure:"description"`
	Ref         string  `mapstructure:"ref"`
	Coverage    float64 `mapstructure:"coverage"`
	PipelineID  string  `mapstructure:"pipelineId"`
}

// publishCommitStatusToggles tracks which optional fields were turned on via
// their UI toggle. Coverage in particular needs this: 0 is a valid value, so a
// toggled-on zero must be sent while an untoggled field is omitted.
type publishCommitStatusToggles struct {
	Name        bool
	TargetURL   bool
	Description bool
	Ref         bool
	Coverage    bool
	PipelineID  bool
}

func newPublishCommitStatusToggles(raw map[string]any) publishCommitStatusToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}
	return publishCommitStatusToggles{
		Name:        enabled("name"),
		TargetURL:   enabled("targetUrl"),
		Description: enabled("description"),
		Ref:         enabled("ref"),
		Coverage:    enabled("coverage"),
		PipelineID:  enabled("pipelineId"),
	}
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
- **Name** (toggle): Label distinguishing this status from others on the same commit (GitLab defaults to ` + "`default`" + `)
- **Target URL** (toggle): URL the status links to, e.g. the build or job page
- **Description** (toggle): Short description shown alongside the status
- **Ref** (toggle): Branch or tag the status is reported for
- **Coverage** (toggle): Total code coverage percentage
- **Pipeline ID** (toggle): Attach the status to a specific pipeline when several ran on the commit

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
			Togglable:   true,
			Description: "Label distinguishing this status from others on the same commit",
		},
		{
			Name:        "targetUrl",
			Label:       "Target URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "URL the status links to, e.g. the build or job page",
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Branch or tag the status is reported for",
		},
		{
			Name:        "coverage",
			Label:       "Coverage",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Total code coverage percentage",
		},
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
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

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newPublishCommitStatusToggles(raw)
	req := &CreateCommitStatusRequest{State: config.State}

	if toggles.Name {
		req.Name = config.Name
	}

	if toggles.TargetURL {
		req.TargetURL = config.TargetURL
	}

	if toggles.Description {
		req.Description = config.Description
	}

	if toggles.Ref {
		req.Ref = config.Ref
	}

	if toggles.Coverage {
		coverage := config.Coverage
		req.Coverage = &coverage
	}

	if toggles.PipelineID && strings.TrimSpace(config.PipelineID) != "" {
		pipelineID, err := parseWholeNumberID(config.PipelineID, "pipeline ID")
		if err != nil {
			return err
		}
		req.PipelineID = &pipelineID
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

func (c *PublishCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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
