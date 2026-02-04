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
	Repository string `json:"repository" mapstructure:"repository"`
	PerPage    int    `json:"perPage" mapstructure:"perPage"`
	Page       int    `json:"page" mapstructure:"page"`
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
	return `The List Releases component retrieves releases from a GitHub repository with optional pagination.

## Use Cases

- **Changelog generation**: List releases for changelog or reporting from SuperPlane
- **Latest releases**: Fetch the latest N releases for notifications or status pages
- **External sync**: Sync release list to external systems (Jira, Slack)
- **Release monitoring**: Track releases across repositories

## Configuration

- **Repository**: Select the GitHub repository to list releases from
- **Per Page** (optional): Number of releases per page (default 30, max 100)
- **Page** (optional): Page number for pagination (default 1)

## Output

Returns a list of release objects, each containing:
- Release ID and tag name
- Release name and body (description)
- Published timestamp
- Draft and prerelease flags
- Author information
- Assets (downloadable files)
- Tarball and zipball URLs`
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
			Description: "Number of releases per page (max 100)",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Description: "Page number for pagination",
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
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Set defaults for pagination
	perPage := config.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	page := config.Page
	if page <= 0 {
		page = 1
	}

	// List releases
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
