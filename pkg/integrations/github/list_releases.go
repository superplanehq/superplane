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
	return "List releases for a GitHub repository"
}

func (c *ListReleases) Documentation() string {
	return `The List Releases component retrieves a list of releases from a GitHub repository.

## Use Cases

- **Changelog Generation**: List releases for changelog or reporting from SuperPlane
- **Release Notifications**: Fetch the latest N releases for notifications or status pages
- **System Synchronization**: Sync release list to external systems (Jira, Slack)
- **Release Monitoring**: Monitor all releases in a repository
- **Deployment Tracking**: Track release history for deployment pipelines

## Configuration

- **Repository**: Select the GitHub repository to list releases from
- **Per Page**: Number of releases to return per page (1-100, default: 30)
- **Page**: Page number for pagination (default: 1)

## Output

Returns a list of releases, each containing:
- Release ID, name, and tag name
- Release body/description
- Draft and prerelease status
- Created and published timestamps
- Author information
- Asset URLs (tarball, zipball, and uploaded assets)`
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
			Placeholder: "30",
			Description: "Number of releases to return per page (1-100). Default: 30",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(100),
				},
			},
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Placeholder: "1",
			Description: "Page number for pagination. Supports template variables (e.g., {{$.data.page}})",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
				},
			},
		},
	}
}

// intPtr returns a pointer to an int value
func intPtr(i int) *int {
	return &i
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

	// Set default pagination values
	perPage := 30
	if config.PerPage != nil && *config.PerPage > 0 {
		perPage = *config.PerPage
		if perPage > 100 {
			perPage = 100
		}
	}

	page := 1
	if config.Page != nil && *config.Page > 0 {
		page = *config.Page
	}

	// Fetch the list of releases
	releases, _, err := client.Repositories.ListReleases(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		&github.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Convert to []any for emission
	releaseList := make([]any, len(releases))
	for i, release := range releases {
		releaseList[i] = release
	}

	// Emit output with releases data
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
