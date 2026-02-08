package github

import (
	"context"
	"errors"
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
	Repository string  `mapstructure:"repository" json:"repository"`
	PerPage    *string `mapstructure:"perPage,omitempty" json:"perPage,omitempty"`
	Page       *string `mapstructure:"page,omitempty" json:"page,omitempty"`
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
	return `The List Releases component fetches releases from a GitHub repository.

## Use Cases

- **Changelog/reporting**: Fetch recent releases for dashboards or reports
- **Notifications**: Notify on the latest N releases
- **Sync**: Mirror releases to external systems

## Configuration

- **Repository**: Select the GitHub repository
- **Per page**: Optional page size (max 100)
- **Page**: Optional page number for pagination (1-based)

## Output

Emits one payload per release with fields like:
- id, tag_name, name, body
- created_at, published_at
- assets, tarball_url, zipball_url`
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
			Label:       "Per page",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., 30 (max 100)",
			Description: "Number of releases to return per page (max 100).",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., 1",
			Description: "Page number (1-based).",
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

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	perPage := 30
	if config.PerPage != nil && *config.PerPage != "" {
		v, err := strconv.Atoi(*config.PerPage)
		if err != nil {
			return fmt.Errorf("per page is not a number: %v", err)
		}
		if v <= 0 {
			return errors.New("per page must be greater than 0")
		}
		if v > 100 {
			return errors.New("per page must be <= 100")
		}
		perPage = v
	}

	page := 0
	if config.Page != nil && *config.Page != "" {
		v, err := strconv.Atoi(*config.Page)
		if err != nil {
			return fmt.Errorf("page is not a number: %v", err)
		}
		if v <= 0 {
			return errors.New("page must be greater than 0")
		}
		page = v
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

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

	events := make([]any, 0, len(releases))
	for _, r := range releases {
		events = append(events, r)
	}

	if len(events) == 0 {
		return nil
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.release",
		events,
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
