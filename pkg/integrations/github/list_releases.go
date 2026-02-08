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
	PerPage    *int   `mapstructure:"perPage,omitempty"`
	Page       *int   `mapstructure:"page,omitempty"`
}

func (c *ListReleases) Name() string {
	return "github.listReleases"
}

func (c *ListReleases) Label() string {
	return "List Releases"
}

func (c *ListReleases) Description() string {
	return "List releases from a GitHub repository"
}

func (c *ListReleases) Documentation() string {
	return `The List Releases component retrieves releases from a GitHub repository.

## Use Cases

- **Release monitoring**: List all releases and their metadata
- **Deployment pipelines**: Discover available versions for deployment
- **Release dashboards**: Build dashboards showing release history
- **CI/CD integration**: Iterate over releases for automated actions

## Configuration

- **Repository**: Select the GitHub repository (required)
- **Per Page**: Number of releases per page, max 100 (optional, defaults to 30)
- **Page**: Page number for pagination (optional, defaults to 1)

## Output

Returns a list of releases, each containing:
- Release ID, tag name, and name
- Release body/description
- Published date
- Asset information (download URLs, sizes)
- Source archive URLs (tarball, zipball)`
}

func (c *ListReleases) Icon() string {
	return "github"
}

func (c *ListReleases) Color() string {
	return "gray"
}

func (c *ListReleases) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
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
			Placeholder: "e.g., 30",
			Description: "Number of releases to return per page (max 100). Defaults to 30.",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Placeholder: "e.g., 1",
			Description: "Page number for pagination. Defaults to 1.",
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

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Build list options from configuration
	listOpts := &github.ListOptions{
		PerPage: 30,
		Page:    1,
	}

	if config.PerPage != nil && *config.PerPage > 0 {
		perPage := *config.PerPage
		if perPage > 100 {
			perPage = 100
		}
		listOpts.PerPage = perPage
	}

	if config.Page != nil && *config.Page > 0 {
		listOpts.Page = *config.Page
	}

	// Fetch releases from the repository
	releases, _, err := client.Repositories.ListReleases(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		listOpts,
	)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Convert releases to a slice of any for emission
	releaseList := make([]any, len(releases))
	for i, release := range releases {
		releaseList[i] = release
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.releases",
		releaseList,
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
