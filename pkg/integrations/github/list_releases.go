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

type ListReleases struct{}

type ListReleasesConfiguration struct {
	Repository string `mapstructure:"repository"`
	PerPage    int    `mapstructure:"perPage"`
	Page       int    `mapstructure:"page"`
}

func (c *ListReleases) Name() string {
	return "github.listReleases"
}

func (c *ListReleases) Label() string {
	return "List Releases"
}

func (c *ListReleases) Description() string {
	return "List releases in a GitHub repository"
}

func (c *ListReleases) Documentation() string {
	return `The List Releases component fetches releases from a GitHub repository.

## Use Cases

- **Release monitoring**: Track new releases across repositories
- **Release history**: Get a list of past releases for reporting
- **Automation**: Trigger workflows based on release lists

## Configuration

- **Repository**: Select the GitHub repository
- **Per Page**: Number of releases per page (default: 30, max: 100)
- **Page**: Page number to fetch (default: 1)

## Output

Returns a list of release objects.`
}

func (c *ListReleases) Icon() string {
	return "github"
}

func (c *ListReleases) Color() string {
	return "gray"
}

func (c *ListReleases) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "releases",
			Label: "Releases",
		},
	}
}

func (c *ListReleases) Configuration() []configuration.Field {
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
			Name:        "perPage",
			Label:       "Per Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     30,
			Description: "Number of results per page (max 100)",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Description: "Page number to fetch",
		},
	}
}

func (c *ListReleases) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *ListReleases) Execute(ctx core.ExecutionContext) error {
	var config ListReleasesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClientWithTransport(
		ctx.Integration,
		ctx.HTTP,
		appMetadata.GitHubApp.ID,
		appMetadata.InstallationID,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Set defaults if 0
	if config.PerPage == 0 {
		config.PerPage = 30
	}
	if config.Page == 0 {
		config.Page = 1
	}

	opts := &github.ListOptions{
		PerPage: config.PerPage,
		Page:    config.Page,
	}

	releases, _, err := client.Repositories.ListReleases(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Convert []*github.RepositoryRelease to []any
	output := make([]any, len(releases))
	for i, r := range releases {
		output[i] = r
	}

	// Emit as a single payload containing the structured data
	return ctx.ExecutionState.Emit(
		"releases",
		"github.release",
		[]any{output},
	)
}

func (c *ListReleases) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListReleases) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *ListReleases) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListReleases) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListReleases) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListReleases) Cleanup(ctx core.SetupContext) error {
	return nil
}
