package github

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type PublishCommitStatus struct{}

type PublishCommitStatusConfiguration struct {
	Repository  string `mapstructure:"repository"`
	SHA         string `mapstructure:"sha"`
	State       string `mapstructure:"state"`
	Context     string `mapstructure:"context"`
	Description string `mapstructure:"description"`
	TargetURL   string `mapstructure:"targetUrl"`
}

var shaRegex = regexp.MustCompile(`^[a-f0-9]{40}$`)

func (c *PublishCommitStatus) Name() string {
	return "github.publishCommitStatus"
}

func (c *PublishCommitStatus) Label() string {
	return "Publish Commit Status"
}

func (c *PublishCommitStatus) Description() string {
	return "Publish a status check to a GitHub commit"
}

func (c *PublishCommitStatus) Icon() string {
	return "github"
}

func (c *PublishCommitStatus) Color() string {
	return "gray"
}

func (c *PublishCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PublishCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "repository",
				},
			},
		},
		{
			Name:        "sha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., abc123def456... or {{event.data.after}}",
			Description: "The full SHA of the commit to attach the status to",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Pending",
							Value: "pending",
						},
						{
							Label: "Success",
							Value: "success",
						},
						{
							Label: "Failure",
							Value: "failure",
						},
						{
							Label: "Error",
							Value: "error",
						},
					},
				},
			},
		},
		{
			Name:        "context",
			Label:       "Context",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., ci/build, test/unit, deploy/production",
			Description: "A label to identify this status check",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "e.g., Build completed successfully",
			Description: "Short description of the status (max ~140 characters)",
		},
		{
			Name:        "targetUrl",
			Label:       "Target URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "https://...",
			Description: "e.g. Link to build logs, test results, ...",
		},
	}
}

func (c *PublishCommitStatus) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)
}

func (c *PublishCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config PublishCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate SHA format
	if !shaRegex.MatchString(config.SHA) {
		return fmt.Errorf("invalid commit SHA format: expected 40-character hexadecimal string, got %q", config.SHA)
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallation, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Prepare the status request based on the configuration
	//
	repoStatus := &github.RepoStatus{
		State:   &config.State,
		Context: &config.Context,
	}

	if config.Description != "" {
		repoStatus.Description = &config.Description
	}

	if config.TargetURL != "" {
		repoStatus.TargetURL = &config.TargetURL
	}

	// Create the commit status
	status, _, err := client.Repositories.CreateStatus(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		config.SHA,
		repoStatus,
	)

	if err != nil {
		return fmt.Errorf("failed to create commit status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.commitStatus",
		[]any{status},
	)
}

func (c *PublishCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PublishCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *PublishCommitStatus) Actions() []core.Action {
	return []core.Action{}
}

func (c *PublishCommitStatus) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PublishCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}
