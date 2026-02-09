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
	return "List releases for a GitHub repository with optional pagination"
}

func (c *ListReleases) Documentation() string {
	return `The List Releases component retrieves a list of releases from a GitHub repository.

## Use Cases

- **Changelog/Reporting**: List releases for changelog or reporting from SuperPlane
- **Notifications**: Fetch the latest N releases for notifications or status pages
- **Sync to External Systems**: Sync release list to external systems (Jira, Slack)
- **Release Monitoring**: Monitor all releases in a repository
- **Deployment Tracking**: Track deployment history through releases

## Configuration

- **Repository**: Select the GitHub repository (required)
- **Per Page**: Number of releases per page (default: 30, max: 100)
- **Page**: Page number for pagination (default: 1)

## Output

Returns a list of releases, each containing:
- Release ID, name, and tag name
- Release body/description
- Draft and prerelease status
- Created and published timestamps
- Author information
- Asset URLs (tarball_url, zipball_url)
- Release assets with download URLs

## Pagination

Use the 'page' and 'perPage' options to paginate through large lists of releases.
The maximum value for 'perPage' is 100 (GitHub API limit).`
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
	minOne := 1
	maxHundred := 100

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
			Placeholder: "30",
			Description: "Number of releases per page (default: 30, max: 100). Supports template variables.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minOne,
					Max: &maxHundred,
				},
			},
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeNumber,
			Placeholder: "1",
			Description: "Page number for pagination (default: 1). Supports template variables.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minOne,
				},
			},
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

	// Set default pagination values
	perPage := 30
	page := 1

	if config.PerPage != nil && *config.PerPage > 0 {
		perPage = *config.PerPage
		if perPage > 100 {
			perPage = 100 // GitHub API max limit
		}
	}

	if config.Page != nil && *config.Page > 0 {
		page = *config.Page
	}

	// Fetch releases from GitHub API
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
		return fmt.Errorf("failed to list releases for repository %s: %w", config.Repository, err)
	}

	// Convert releases to a format suitable for output
	releaseList := make([]any, 0, len(releases))
	for _, release := range releases {
		releaseData := map[string]any{
			"id":           release.GetID(),
			"tag_name":     release.GetTagName(),
			"name":         release.GetName(),
			"body":         release.GetBody(),
			"html_url":     release.GetHTMLURL(),
			"draft":        release.GetDraft(),
			"prerelease":   release.GetPrerelease(),
			"created_at":   release.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			"published_at": formatTimestamp(release.PublishedAt),
			"tarball_url":  release.GetTarballURL(),
			"zipball_url":  release.GetZipballURL(),
		}

		// Add author information if available
		if author := release.GetAuthor(); author != nil {
			releaseData["author"] = map[string]any{
				"login":      author.GetLogin(),
				"id":         author.GetID(),
				"avatar_url": author.GetAvatarURL(),
				"html_url":   author.GetHTMLURL(),
			}
		}

		// Add assets if available
		if len(release.Assets) > 0 {
			assets := make([]map[string]any, 0, len(release.Assets))
			for _, asset := range release.Assets {
				assets = append(assets, map[string]any{
					"id":                   asset.GetID(),
					"name":                 asset.GetName(),
					"label":                asset.GetLabel(),
					"content_type":         asset.GetContentType(),
					"size":                 asset.GetSize(),
					"download_count":       asset.GetDownloadCount(),
					"browser_download_url": asset.GetBrowserDownloadURL(),
				})
			}
			releaseData["assets"] = assets
		}

		releaseList = append(releaseList, releaseData)
	}

	// Emit output with releases list
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.releases",
		[]any{map[string]any{
			"releases":   releaseList,
			"count":      len(releaseList),
			"page":       page,
			"per_page":   perPage,
			"repository": config.Repository,
		}},
	)
}

// formatTimestamp safely formats a GitHub Timestamp to string
func formatTimestamp(t *github.Timestamp) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z")
}

// parseIntFromConfig attempts to parse an int from various config value types
func parseIntFromConfig(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot parse int from type %T", value)
	}
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
