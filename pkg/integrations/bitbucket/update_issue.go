package bitbucket

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber int    `json:"issueNumber" mapstructure:"issueNumber"`
	Title       string `json:"title" mapstructure:"title"`
	Body        string `json:"body" mapstructure:"body"`
	State       string `json:"state" mapstructure:"state"`
}

func (c *UpdateIssue) Name() string {
	return "bitbucket.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update an existing Bitbucket issue"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component updates fields on an existing Bitbucket issue.

## Use Cases

- **Status transitions**: Move issues to resolved or closed based on workflow outcomes
- **Issue maintenance**: Update titles or descriptions with fresh context
- **Automation**: Keep issue state synchronized with external systems

## Configuration

- **Repository**: Select the Bitbucket repository containing the issue
- **Issue Number**: Numeric issue ID to update
- **Title**: Optional new title
- **Body**: Optional new description
- **State**: Optional state update

## Output

Returns the updated issue payload from Bitbucket.`
}

func (c *UpdateIssue) Icon() string {
	return "bitbucket"
}

func (c *UpdateIssue) Color() string {
	return "blue"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "issueNumber",
			Label:    "Issue Number",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "New", Value: "new"},
						{Label: "Open", Value: "open"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "On Hold", Value: "on hold"},
						{Label: "Invalid", Value: "invalid"},
						{Label: "Duplicate", Value: "duplicate"},
						{Label: "Won't Fix", Value: "wontfix"},
						{Label: "Closed", Value: "closed"},
					},
				},
			},
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if config.IssueNumber <= 0 {
		return fmt.Errorf("issue number is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadataCtx := metadataContextForExecution(ctx)
	if metadataCtx == nil {
		return fmt.Errorf("metadata context is required")
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, metadataCtx, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	client, integrationMetadata, err := newClientFromIntegration(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	request := map[string]any{}
	if config.Title != "" {
		request["title"] = config.Title
	}
	if config.Body != "" {
		request["content"] = map[string]any{
			"raw": config.Body,
		}
	}
	if config.State != "" {
		request["state"] = config.State
	}

	if len(request) == 0 {
		return fmt.Errorf("at least one field to update is required")
	}

	issue, err := client.UpdateIssue(integrationMetadata.Workspace.Slug, repo.Slug, config.IssueNumber, request)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		issuePayloadType,
		[]any{issue},
	)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
