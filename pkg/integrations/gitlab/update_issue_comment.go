package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_update_issue_comment.json
var exampleOutputUpdateIssueComment []byte

type UpdateIssueComment struct{}

type UpdateIssueCommentConfiguration struct {
	Project   string `mapstructure:"project"`
	IssueIID  string `mapstructure:"issueIid"`
	CommentID string `mapstructure:"commentId"`
	Body      string `mapstructure:"body"`
}

// updateIssueCommentToggles tracks which optional fields were explicitly turned
// on via their UI toggle. Body is GitLab's only editable note field, so it is
// the single togglable field. See update_merge_request.go for the pattern.
type updateIssueCommentToggles struct {
	Body bool
}

func newUpdateIssueCommentToggles(raw map[string]any) updateIssueCommentToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}
	return updateIssueCommentToggles{
		Body: enabled("body"),
	}
}

func (t updateIssueCommentToggles) hasUpdates() bool {
	return t.Body
}

func (c *UpdateIssueComment) Name() string {
	return "gitlab.updateIssueComment"
}

func (c *UpdateIssueComment) Label() string {
	return "Update Issue Comment"
}

func (c *UpdateIssueComment) Description() string {
	return "Update an existing comment on a GitLab issue"
}

func (c *UpdateIssueComment) Documentation() string {
	return `The Update Issue Comment component edits an existing comment (note) on a GitLab issue.

## Use Cases

- **Status updates**: Update a summary comment on an issue instead of posting new comments on every run
- **Living reports**: Keep a single comment with the latest deployment status, test results, or remediation steps
- **Avoid spam**: Update one comment instead of flooding an issue with repeated bot comments

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue the comment belongs to (supports expressions)
- **Comment ID** (required): The numeric ID of the comment (note) to update (supports expressions)
- **Body** (toggle): The new comment text (supports Markdown and expressions). It is the only editable field on a GitLab note, so it must be enabled for the update to change anything, and cannot be empty when enabled.

## Permissions

The connected user must be the comment's author, or have at least the **Maintainer** role on the project, to edit it.

## Output

Returns the updated note object, including its ID, body, author, and timestamps.

## Notes

- The comment ID is returned in the output of ` + "`gitlab.createIssueComment`" + ` as ` + "`id`" + `
- You can store the comment ID in canvas memory on the first run, then reuse it for subsequent updates`
}

func (c *UpdateIssueComment) Icon() string {
	return "gitlab"
}

func (c *UpdateIssueComment) Color() string {
	return "orange"
}

func (c *UpdateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssueComment) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdateIssueComment, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *UpdateIssueComment) Configuration() []configuration.Field {
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
			Name:        "issueIid",
			Label:       "Issue IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The internal ID (IID) of the issue the comment belongs to",
		},
		{
			Name:        "commentId",
			Label:       "Comment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The numeric ID of the comment (note) to update",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "The new comment text. Supports Markdown formatting.",
		},
	}
}

func (c *UpdateIssueComment) Setup(ctx core.SetupContext) error {
	var config UpdateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	if config.CommentID == "" {
		return errors.New("comment ID is required")
	}

	if err := validateUpdateIssueComment(ctx.Configuration, config); err != nil {
		return err
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *UpdateIssueComment) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateUpdateIssueComment(ctx.Configuration, config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	note, err := client.UpdateIssueNote(context.Background(), config.Project, config.IssueIID, config.CommentID, &UpdateNoteRequest{Body: config.Body})
	if err != nil {
		return fmt.Errorf("failed to update issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.updateIssueComment",
		[]any{note},
	)
}

func validateUpdateIssueComment(rawConfig any, config UpdateIssueCommentConfiguration) error {
	raw, _ := rawConfig.(map[string]any)
	toggles := newUpdateIssueCommentToggles(raw)
	if !toggles.hasUpdates() {
		return errors.New("at least one field must be enabled to update")
	}

	if toggles.Body && config.Body == "" {
		return errors.New("body cannot be empty")
	}

	return nil
}

func (c *UpdateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssueComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
