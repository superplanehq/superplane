package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueConfiguration struct {
	BaseRepositoryConfig `mapstructure:",squash"`

	IssueNumber int      `mapstructure:"issueNumber"`
	Title       string   `mapstructure:"title"`
	Body        string   `mapstructure:"body"`
	State       string   `mapstructure:"state"`
	Assignees   []string `mapstructure:"assignees"`
	Labels      []string `mapstructure:"labels"`
}

func (c *UpdateIssue) Name() string {
	return "github.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update a GitHub issue"
}

func (c *UpdateIssue) Icon() string {
	return "github"
}

func (c *UpdateIssue) Color() string {
	return "gray"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
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
			Description: "The issue number to update",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
		},
		{
			Name:        "title",
			Label:       "Title",
			Description: "The new title for the issue",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "body",
			Label:       "Body",
			Description: "The new body/description for the issue",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
		{
			Name:        "state",
			Label:       "State",
			Description: "The state of the issue",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "open",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Open",
							Value: "open",
						},
						{
							Label: "Closed",
							Value: "closed",
						},
					},
				},
			},
		},
		{
			Name:        "assignees",
			Label:       "Assignees",
			Description: "GitHub usernames to assign to the issue",
			Type:        configuration.FieldTypeList,
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignee",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Description: "Labels for the issue",
			Type:        configuration.FieldTypeList,
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
		ctx.Configuration,
	)
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadataContext.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallationContext.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallationContext, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Prepare the update request based on configuration
	//
	issueRequest := &github.IssueRequest{}

	if config.Title != "" {
		issueRequest.Title = &config.Title
	}

	if config.Body != "" {
		issueRequest.Body = &config.Body
	}

	if config.State != "" {
		issueRequest.State = &config.State
	}

	if len(config.Assignees) > 0 {
		issueRequest.Assignees = &config.Assignees
	}

	if len(config.Labels) > 0 {
		issueRequest.Labels = &config.Labels
	}

	// Update the issue
	issue, _, err := client.Issues.Edit(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		config.IssueNumber,
		issueRequest,
	)

	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
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
