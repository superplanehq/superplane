package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRepositoryPermission struct{}

type GetRepositoryPermissionConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Username   string `json:"username" mapstructure:"username"`
}

func (c *GetRepositoryPermission) Name() string {
	return "github.getRepositoryPermission"
}

func (c *GetRepositoryPermission) Label() string {
	return "Get Repository Permission"
}

func (c *GetRepositoryPermission) Description() string {
	return "Get the role and permission level for a user on a GitHub repository"
}

func (c *GetRepositoryPermission) Documentation() string {
	return `The Get Repository Permission component retrieves a user's effective permission level for a GitHub repository.

## Use Cases

- **Access checks**: Verify if a user has expected repository access
- **Automation gates**: Branch workflow behavior by repository permissions
- **Auditing**: Inspect repository roles in automated compliance checks
- **Triage routing**: Route incidents based on whether a user can push/administer
`
}

func (c *GetRepositoryPermission) Icon() string {
	return "github"
}

func (c *GetRepositoryPermission) Color() string {
	return "gray"
}

func (c *GetRepositoryPermission) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRepositoryPermission) Configuration() []configuration.Field {
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
			Name:     "username",
			Label:    "Username",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (c *GetRepositoryPermission) Setup(ctx core.SetupContext) error {
	var config GetRepositoryPermissionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Username == "" {
		return errors.New("username is required")
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *GetRepositoryPermission) Execute(ctx core.ExecutionContext) error {
	var config GetRepositoryPermissionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var integrationMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, integrationMetadata.GitHubApp.ID, integrationMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	username := strings.TrimPrefix(config.Username, "@")
	permission, _, err := client.Repositories.GetPermissionLevel(
		context.Background(),
		integrationMetadata.Owner,
		config.Repository,
		username,
	)
	if err != nil {
		return fmt.Errorf("failed to get repository permission level: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.repositoryPermission",
		[]any{permission},
	)
}

func (c *GetRepositoryPermission) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRepositoryPermission) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetRepositoryPermission) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRepositoryPermission) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRepositoryPermission) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRepositoryPermission) Cleanup(ctx core.SetupContext) error {
	return nil
}
