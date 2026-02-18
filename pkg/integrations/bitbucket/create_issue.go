package bitbucket

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssue struct{}

type CreateIssueConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Title      string `json:"title" mapstructure:"title"`
	Body       string `json:"body" mapstructure:"body"`
}

func (c *CreateIssue) Name() string {
	return "bitbucket.createIssue"
}

func (c *CreateIssue) Label() string {
	return "Create Issue"
}

func (c *CreateIssue) Description() string {
	return "Create a new issue in a Bitbucket repository"
}

func (c *CreateIssue) Documentation() string {
	return `The Create Issue component creates a new issue in a Bitbucket repository.

## Use Cases

- **Incident follow-ups**: Open issues automatically from alerts
- **Task creation**: Turn workflow outputs into trackable work
- **Operational logging**: Capture remediation steps as issues

## Configuration

- **Repository**: Select the Bitbucket repository where the issue will be created
- **Title**: The issue title (supports expressions)
- **Body**: Optional issue description

## Output

Returns the created issue payload from Bitbucket.`
}

func (c *CreateIssue) Icon() string {
	return "bitbucket"
}

func (c *CreateIssue) Color() string {
	return "blue"
}

func (c *CreateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssue) Configuration() []configuration.Field {
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
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	var config CreateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if config.Title == "" {
		return fmt.Errorf("title is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueConfiguration
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

	request := map[string]any{
		"title": config.Title,
	}
	if config.Body != "" {
		request["content"] = map[string]any{
			"raw": config.Body,
		}
	}

	issue, err := client.CreateIssue(integrationMetadata.Workspace.Slug, repo.Slug, request)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		issuePayloadType,
		[]any{issue},
	)
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
