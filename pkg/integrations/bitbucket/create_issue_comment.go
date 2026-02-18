package bitbucket

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssueComment struct{}

type CreateIssueCommentConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber string `json:"issueNumber" mapstructure:"issueNumber"`
	Body        string `json:"body" mapstructure:"body"`
}

func (c *CreateIssueComment) Name() string {
	return "bitbucket.createIssueComment"
}

func (c *CreateIssueComment) Label() string {
	return "Create Issue Comment"
}

func (c *CreateIssueComment) Description() string {
	return "Add a comment to a Bitbucket issue"
}

func (c *CreateIssueComment) Documentation() string {
	return `The Create Issue Comment component adds a comment to a Bitbucket issue.

## Use Cases

- **Incident updates**: Post status updates on linked issues
- **Automation notes**: Record workflow decisions directly on issues
- **Cross-system sync**: Mirror notes from external systems into Bitbucket

## Configuration

- **Repository**: Select the Bitbucket repository containing the issue
- **Issue Number**: Numeric issue ID to comment on (supports expressions)
- **Body**: Comment text

## Output

Returns the created issue comment payload from Bitbucket.`
}

func (c *CreateIssueComment) Icon() string {
	return "bitbucket"
}

func (c *CreateIssueComment) Color() string {
	return "blue"
}

func (c *CreateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssueComment) Configuration() []configuration.Field {
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
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: true,
		},
	}
}

func (c *CreateIssueComment) Setup(ctx core.SetupContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if config.IssueNumber == "" {
		return fmt.Errorf("issue number is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *CreateIssueComment) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	issueNumber, err := strconv.Atoi(config.IssueNumber)
	if err != nil {
		return fmt.Errorf("issue number is not a number: %v", err)
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

	comment, err := client.CreateIssueComment(integrationMetadata.Workspace.Slug, repo.Slug, issueNumber, config.Body)
	if err != nil {
		return fmt.Errorf("failed to create issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		issueCommentPayloadType,
		[]any{comment},
	)
}

func (c *CreateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateIssueComment) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssueComment) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}
