package github

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetIssue struct{}

type GetIssueConfiguration struct {
	BaseRepositoryConfig `mapstructure:",squash"`

	IssueNumber int `mapstructure:"issueNumber"`
}

func (c *GetIssue) Name() string {
	return "github.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Get a GitHub issue by number"
}

func (c *GetIssue) Icon() string {
	return "github"
}

func (c *GetIssue) Color() string {
	return "gray"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Description: "The repository containing the issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "issueNumber",
			Label:       "Issue Number",
			Description: "The issue number to retrieve",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
		ctx.Configuration,
	)
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.AppInstallationContext, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Get the issue
	issue, _, err := client.Issues.Get(
		context.Background(),
		appMetadata.Owner,
		config.BaseRepositoryConfig.Repository,
		config.IssueNumber,
	)

	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
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
