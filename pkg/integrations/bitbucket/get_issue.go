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

type GetIssue struct{}

type GetIssueConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber string `json:"issueNumber" mapstructure:"issueNumber"`
}

func (c *GetIssue) Name() string {
	return "bitbucket.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Get a Bitbucket issue by ID"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component retrieves an issue from a Bitbucket repository.

## Use Cases

- **Issue lookups**: Fetch issue details for follow-up automation
- **State checks**: Read issue status before taking actions
- **Data enrichment**: Reuse issue fields in notifications or downstream systems

## Configuration

- **Repository**: Select the Bitbucket repository containing the issue
- **Issue Number**: Numeric issue ID to retrieve (supports expressions)

## Output

Returns the full issue payload returned by Bitbucket.`
}

func (c *GetIssue) Icon() string {
	return "bitbucket"
}

func (c *GetIssue) Color() string {
	return "blue"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
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
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if config.IssueNumber == "" {
		return fmt.Errorf("issue number is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	var config GetIssueConfiguration
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

	issue, err := client.GetIssue(integrationMetadata.Workspace.Slug, repo.Slug, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		issuePayloadType,
		[]any{issue},
	)
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
