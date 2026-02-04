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
	return "List releases for a GitHub repository with optional pagination"
}

func (c *ListReleases) Documentation() string {
	return `The List Releases component retrieves a list of releases from a GitHub repository.

## Use Cases

- **Changelog generation**: List releases for reporting from SuperPlane
- **Release monitoring**: Fetch the latest N releases for notifications or status pages
- **Data sync**: Sync release list to external systems (Jira, Slack)
- **Version tracking**: Track all releases for version management

## Configuration

- **Repository**: Select the GitHub repository (owner/repo format)
- **Per Page**: Number of releases per page (default 30, max 100)
- **Page**: Page number for pagination (default 1)

## Output

Returns a list of releases with the following information for each release:
- Release ID
- Tag name
- Release name
- Release body/notes
- Published date
- Assets (download links)
- Tarball and zipball URLs

If the repository is not found or the API fails, an error is returned.`
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
			Description: "Number of releases per page (max 100)",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1,
			Placeholder: "1",
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
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Set up pagination options
	opts := &github.ListOptions{
		Page:    config.Page,
		PerPage: config.PerPage,
	}

	// Enforce max per page limit
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.PerPage <= 0 {
		opts.PerPage = 30
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	// Call GitHub API to list releases
	releases, resp, err := client.Repositories.ListReleases(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Map releases to output format
	releaseData := make([]map[string]any, 0, len(releases))
	for _, release := range releases {
		releaseItem := map[string]any{
			"id":          release.GetID(),
			"tag_name":    release.GetTagName(),
			"name":        release.GetName(),
			"body":        release.GetBody(),
			"tarball_url": release.GetTarballURL(),
			"zipball_url": release.GetZipballURL(),
		}

		// Add published date if available
		if release.PublishedAt != nil {
			releaseItem["published_at"] = release.PublishedAt.Format("2006-01-02T15:04:05Z")
		} else {
			releaseItem["published_at"] = ""
		}

		// Map assets
		assets := make([]map[string]any, 0, len(release.Assets))
		for _, asset := range release.Assets {
			assets = append(assets, map[string]any{
				"id":                   asset.GetID(),
				"name":                 asset.GetName(),
				"browser_download_url": asset.GetBrowserDownloadURL(),
			})
		}
		releaseItem["assets"] = assets

		releaseData = append(releaseData, releaseItem)
	}

	// Emit output
releaseDataAny := make([]any, len(releaseData))
for i, v := range releaseData {
    releaseDataAny[i] = v
}
return ctx.ExecutionState.Emit(
    core.DefaultOutputChannel.Name,
    "github.releases",
    releaseDataAny,
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
func (c *ListReleases) ExampleOutput() map[string]any {
	return map[string]any{
		"default": []any{
			map[string]any{
				"id":          int64(12345678),
				"tag_name":    "v1.0.0",
				"name":        "Release 1.0.0",
				"body":        "## What's Changed\n\n- Initial release",
				"published_at": "2024-01-15T10:30:00Z",
				"tarball_url": "https://api.github.com/repos/owner/repo/tarball/v1.0.0",
				"zipball_url": "https://api.github.com/repos/owner/repo/zipball/v1.0.0",
				"assets": []any{
					map[string]any{
						"id":                   int64(87654321),
						"name":                 "release-binary.zip",
						"browser_download_url": "https://github.com/owner/repo/releases/download/v1.0.0/release-binary.zip",
					},
				},
			},
		},
	}
}
