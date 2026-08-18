package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_commit_status.json
var exampleOutputGetCommitStatus []byte

const CommitPayloadType = "gitlab.commit"

type GetCommitStatus struct{}

type GetCommitStatusConfiguration struct {
	Project string `mapstructure:"project"`
	Ref     string `mapstructure:"ref"`
}

func (c *GetCommitStatus) Name() string {
	return "gitlab.getCommitStatus"
}

func (c *GetCommitStatus) Label() string {
	return "Get Commit Status"
}

func (c *GetCommitStatus) Description() string {
	return "Get the overall CI status of a GitLab commit, branch, or tag"
}

func (c *GetCommitStatus) Documentation() string {
	return `The Get Commit Status component reads a commit's overall CI status.

In GitLab a commit's statuses are jobs grouped under pipelines, and GitLab rolls them up for you: every commit carries a single overall **status** and a **last_pipeline**. This component returns that native result - one meaningful state plus a link to the pipeline - rather than a list of every job.

## Use Cases

- **Status gates**: Check whether a commit is green before continuing
- **Follow-up to Publish Commit Status**: Read back a commit's overall state
- **Notifications**: Report a commit's CI outcome with a link to the pipeline

## Configuration

- **Project** (required): The GitLab project containing the commit
- **Ref** (required): Commit SHA, branch name, or tag name to inspect (supports expressions)

## Permissions

The connected token needs the ` + "`read_api`" + ` (or ` + "`api`" + `) scope and at least the **Reporter** role on the project.

## Output

Emits the commit object, including its overall **status** and **last_pipeline** (id, ref, sha, status, web_url).`
}

func (c *GetCommitStatus) Icon() string {
	return "gitlab"
}

func (c *GetCommitStatus) Color() string {
	return "orange"
}

func (c *GetCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetCommitStatus, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetCommitStatus) Configuration() []configuration.Field {
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
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "main, v1.2.3, or {{ event.data.checkout_sha }}",
			Description: "Commit SHA, branch name, or tag name to inspect. Supports expressions.",
		},
	}
}

func (c *GetCommitStatus) Setup(ctx core.SetupContext) error {
	var config GetCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if strings.TrimSpace(config.Ref) == "" {
		return errors.New("ref is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *GetCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config GetCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ref := strings.TrimSpace(config.Ref)
	if ref == "" {
		return errors.New("ref is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	commit, err := client.GetCommit(context.Background(), config.Project, ref)
	if err != nil {
		return fmt.Errorf("failed to get commit status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CommitPayloadType,
		[]any{commit},
	)
}

func (c *GetCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
