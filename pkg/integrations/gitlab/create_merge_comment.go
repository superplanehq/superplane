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

//go:embed example_output_create_merge_comment.json
var exampleOutputCreateMergeComment []byte

type CreateMergeComment struct{}

type CreateMergeCommentConfiguration struct {
	Project         string `mapstructure:"project"`
	MergeRequestIID string `mapstructure:"mergeRequestIid"`
	Body            string `mapstructure:"body"`
}

func (c *CreateMergeComment) Name() string {
	return "gitlab.createMergeComment"
}

func (c *CreateMergeComment) Label() string {
	return "Create Merge Request Comment"
}

func (c *CreateMergeComment) Description() string {
	return "Add a comment to a GitLab merge request"
}

func (c *CreateMergeComment) Documentation() string {
	return `The Create Merge Request Comment component adds a comment (note) to an existing GitLab merge request.

## Use Cases

- **Deployment updates**: Post deployment status or remediation updates to a merge request
- **Automated feedback**: Add automated review notes based on pipeline or workflow results
- **Cross-platform sync**: Sync Slack or PagerDuty notes into GitLab as merge request comments

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request to comment on (supports expressions)
- **Body** (required): The comment text. Supports Markdown formatting.

## Output

Returns the created note object, including:
- Note ID
- Note body
- Author information
- Created timestamp`
}

func (c *CreateMergeComment) Icon() string {
	return "gitlab"
}

func (c *CreateMergeComment) Color() string {
	return "orange"
}

func (c *CreateMergeComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateMergeComment) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateMergeComment, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateMergeComment) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the merge request to comment on",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment text. Supports Markdown formatting.",
		},
	}
}

func (c *CreateMergeComment) Setup(ctx core.SetupContext) error {
	var config CreateMergeCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.MergeRequestIID == "" {
		return fmt.Errorf("merge request IID is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateMergeComment) Execute(ctx core.ExecutionContext) error {
	var config CreateMergeCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	note, err := client.CreateMergeRequestNote(context.Background(), config.Project, config.MergeRequestIID, &CreateNoteRequest{
		Body: config.Body,
	})
	if err != nil {
		return fmt.Errorf("failed to create merge request comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.createMergeComment",
		[]any{note},
	)
}

func (c *CreateMergeComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateMergeComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateMergeComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateMergeComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateMergeComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateMergeComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
