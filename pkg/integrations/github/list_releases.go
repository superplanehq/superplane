package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListReleases struct{}

type ListReleasesConfiguration struct {
	Repository string `mapstructure:"repository"`
	PerPage    string `mapstructure:"perPage"`
	Page       string `mapstructure:"page"`
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

- **Changelog generation**: List releases for reporting or changelog generation
- **Notifications**: Fetch the latest N releases for notifications or status pages
- **Sync to external systems**: Sync release list to external systems like Jira or Slack
- **Release management**: Monitor releases across repositories

## Configuration

- **Repository**: Select the GitHub repository to list releases from
- **Per Page**: Number of releases per page (default: 30, max: 100)
- **Page**: Page number for pagination (default: 1)

## Output

Returns a list of releases, each containing:
- Release ID and node ID
- Tag name and release name
- Release body/description
- Published date
- Draft and prerelease flags
- Download URLs (tarball, zipball)
- Assets with download URLs and counts
- Author information`
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
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "30",
			Placeholder: "30",
			Description: "Number of releases per page (max 100)",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "1",
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
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Parse pagination parameters
	perPage := 30
	if config.PerPage != "" {
		parsed, err := strconv.Atoi(config.PerPage)
		if err != nil {
			return fmt.Errorf("perPage must be a valid number: %w", err)
		}
		if parsed < 1 {
			return fmt.Errorf("perPage must be at least 1")
		}
		if parsed > 100 {
			parsed = 100 // GitHub API max
		}
		perPage = parsed
	}

	page := 1
	if config.Page != "" {
		parsed, err := strconv.Atoi(config.Page)
		if err != nil {
			return fmt.Errorf("page must be a valid number: %w", err)
		}
		if parsed < 1 {
			return fmt.Errorf("page must be at least 1")
		}
		page = parsed
	}

	// List releases
	releases, resp, err := client.Repositories.ListReleases(
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

	// Transform releases to output format
	releaseData := make([]any, 0, len(releases))
	for _, release := range releases {
		// Transform assets
		assets := make([]map[string]any, 0, len(release.Assets))
		for _, asset := range release.Assets {
			assets = append(assets, map[string]any{
				"id":                   asset.GetID(),
				"name":                 asset.GetName(),
				"label":                asset.GetLabel(),
				"state":                asset.GetState(),
				"content_type":         asset.GetContentType(),
				"size":                 asset.GetSize(),
				"download_count":       asset.GetDownloadCount(),
				"browser_download_url": asset.GetBrowserDownloadURL(),
				"created_at":           asset.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
				"updated_at":           asset.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
			})
		}

		releaseItem := map[string]any{
			"id":           release.GetID(),
			"node_id":      release.GetNodeID(),
			"tag_name":     release.GetTagName(),
			"name":         release.GetName(),
			"body":         release.GetBody(),
			"draft":        release.GetDraft(),
			"prerelease":   release.GetPrerelease(),
			"html_url":     release.GetHTMLURL(),
			"tarball_url":  release.GetTarballURL(),
			"zipball_url":  release.GetZipballURL(),
			"assets":       assets,
			"assets_count": len(release.Assets),
		}

		if release.PublishedAt != nil {
			releaseItem["published_at"] = release.PublishedAt.Format("2006-01-02T15:04:05Z")
		}
		if release.CreatedAt != nil {
			releaseItem["created_at"] = release.CreatedAt.Format("2006-01-02T15:04:05Z")
		}
		if release.Author != nil {
			releaseItem["author"] = map[string]any{
				"login":      release.Author.GetLogin(),
				"id":         release.Author.GetID(),
				"avatar_url": release.Author.GetAvatarURL(),
				"html_url":   release.Author.GetHTMLURL(),
			}
		}

		releaseData = append(releaseData, releaseItem)
	}

	// Add pagination metadata
	output := map[string]any{
		"releases":    releaseData,
		"count":       len(releaseData),
		"page":        page,
		"per_page":    perPage,
		"has_more":    resp.NextPage != 0,
		"next_page":   resp.NextPage,
		"last_page":   resp.LastPage,
		"total_count": len(releaseData), // Approximate since GitHub doesn't return total
		"repository":  config.Repository,
		"owner":       appMetadata.Owner,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.releases",
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
