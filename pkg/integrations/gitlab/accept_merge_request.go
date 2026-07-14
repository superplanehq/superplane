package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_accept_merge_request.json
var exampleOutputAcceptMergeRequest []byte

type AcceptMergeRequest struct{}

type AcceptMergeRequestConfiguration struct {
	Project                  string  `mapstructure:"project"`
	MergeRequestIID          string  `mapstructure:"mergeRequestIid"`
	MergeCommitMessage       string  `mapstructure:"mergeCommitMessage"`
	Squash                   bool    `mapstructure:"squash"`
	SquashCommitMessage      *string `mapstructure:"squashCommitMessage,omitempty"`
	ShouldRemoveSourceBranch bool    `mapstructure:"shouldRemoveSourceBranch"`
	SHA                      string  `mapstructure:"sha"`
}

func (c *AcceptMergeRequest) Name() string {
	return "gitlab.acceptMergeRequest"
}

func (c *AcceptMergeRequest) Label() string {
	return "Accept Merge Request"
}

func (c *AcceptMergeRequest) Description() string {
	return "Merge an open GitLab merge request"
}

func (c *AcceptMergeRequest) Documentation() string {
	return `The Accept Merge Request component merges an open GitLab merge request.

## Use Cases

- **Automated merge gates**: Merge a merge request after agent checks, review pipelines, or policy checks pass
- **Release automation**: Merge approved promotion branches into a release branch
- **Queue workflows**: Merge merge requests from a controlled SuperPlane workflow

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request to merge (supports expressions)
- **Merge Commit Message**: Optional custom merge commit message
- **Squash**: If enabled, squash all commits into a single commit on merge
- **Squash Commit Message**: Optional custom squash commit message
- **Remove Source Branch**: If enabled, removes the source branch after merging
- **Expected SHA**: Optional head SHA guard. GitLab rejects the merge if the source branch head has changed.

## Permissions

The connected user must be allowed to merge into the target branch. Merging into
protected branches (e.g. the default branch) requires the **Maintainer** role by
default, unless the branch's **Allowed to merge** setting includes Developers.

## Output

Returns the merged merge request object, including state, merge commit SHA, source and target branches, and the merge request URL.`
}

func (c *AcceptMergeRequest) Icon() string {
	return "gitlab"
}

func (c *AcceptMergeRequest) Color() string {
	return "orange"
}

func (c *AcceptMergeRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AcceptMergeRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputAcceptMergeRequest, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *AcceptMergeRequest) Configuration() []configuration.Field {
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
			Name:        "mergeRequestIid",
			Label:       "Merge Request IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "42 or {{event.data.object_attributes.iid}}",
			Description: "The internal ID (IID) of the merge request to merge",
		},
		{
			Name:        "mergeCommitMessage",
			Label:       "Merge Commit Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional custom merge commit message",
		},
		{
			Name:        "squash",
			Label:       "Squash",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Squash all commits into a single commit on merge",
		},
		{
			Name:        "squashCommitMessage",
			Label:       "Squash Commit Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional custom squash commit message",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "squash", Values: []string{"true"}},
			},
		},
		{
			Name:        "shouldRemoveSourceBranch",
			Label:       "Remove Source Branch",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Remove the source branch after merging",
		},
		{
			Name:        "sha",
			Label:       "Expected SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "{{event.data.object_attributes.last_commit.id}}",
			Description: "Optional source branch head SHA. GitLab rejects the merge if the head changed.",
		},
	}
}

func (c *AcceptMergeRequest) Setup(ctx core.SetupContext) error {
	var config AcceptMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.MergeRequestIID == "" {
		return fmt.Errorf("merge request IID is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *AcceptMergeRequest) Execute(ctx core.ExecutionContext) error {
	var config AcceptMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	req := &AcceptMergeRequestRequest{
		MergeCommitMessage: config.MergeCommitMessage,
		SHA:                config.SHA,
	}

	if config.Squash {
		req.Squash = &config.Squash
		if config.SquashCommitMessage != nil {
			req.SquashCommitMessage = *config.SquashCommitMessage
		}
	}

	if config.ShouldRemoveSourceBranch {
		req.ShouldRemoveSourceBranch = &config.ShouldRemoveSourceBranch
	}

	mergeRequest, err := client.AcceptMergeRequest(context.Background(), config.Project, config.MergeRequestIID, req)
	if err != nil {
		return fmt.Errorf("failed to accept merge request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{mergeRequest},
	)
}

func (c *AcceptMergeRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AcceptMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AcceptMergeRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AcceptMergeRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AcceptMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AcceptMergeRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
