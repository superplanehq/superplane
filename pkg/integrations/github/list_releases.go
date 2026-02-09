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
	Repository string  `mapstructure:"repository"`
	PerPage    *string `mapstructure:"perPage,omitempty"`
	Page       *string `mapstructure:"page,omitempty"`
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
	return `The List Releases component returns releases from a GitHub repository with optional pagination.

## Use Cases

- **Release auditing**: Review recent releases for compliance or visibility
- **Deployment inputs**: Feed release metadata into deployment workflows
- **Changelog automation**: Aggregate release notes across versions
- **Reporting**: Build release dashboards or summaries

## Configuration

- **Repository**: Select the GitHub repository
- **Per Page**: Number of releases per page (max 100)
- **Page**: Page number for pagination

## Output

Returns a list of releases and their details including:
- Release id, tag, and name
- Body and published/created timestamps
- Author information
- Assets, tarball URL, and zipball URL
- Other metadata fields`
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
				Resource: &configuration.ResourceTypeOptions{Type: "repository", UseNameAsValue: true},
			},
		},
		{
			Name:        "perPage",
			Label:       "Per Page",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g. 30",
			Description: "Number of releases per page (max 100)",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g. 1",
			Description: "Page number for pagination",
		},
	}
}

func (c *ListReleases) Setup(ctx core.SetupContext) error {
	if err := ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	); err != nil {
		return err
	}

	var config ListReleasesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return validateListReleasesPagination(config)
}

func (c *ListReleases) Execute(ctx core.ExecutionContext) error {
	var config ListReleasesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateListReleasesPagination(config); err != nil {
		return err
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

	perPage := 30
	if config.PerPage != nil && *config.PerPage != "" {
		perPageValue, _ := strconv.Atoi(*config.PerPage)
		if perPageValue > 0 {
			perPage = perPageValue
		}
	}

	page := 1
	if config.Page != nil && *config.Page != "" {
		pageValue, _ := strconv.Atoi(*config.Page)
		if pageValue > 0 {
			page = pageValue
		}
	}

	releases, _, err := client.Repositories.ListReleases(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		&github.ListOptions{PerPage: perPage, Page: page},
	)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.releases.list",
		[]any{releases},
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

func validateListReleasesPagination(config ListReleasesConfiguration) error {
	if config.PerPage != nil && *config.PerPage != "" {
		perPageValue, err := strconv.Atoi(*config.PerPage)
		if err != nil {
			return fmt.Errorf("invalid perPage value '%s': must be a valid number", *config.PerPage)
		}

		if perPageValue <= 0 {
			return fmt.Errorf("perPage must be greater than 0")
		}

		if perPageValue > 100 {
			return fmt.Errorf("perPage must not exceed 100")
		}
	}

	if config.Page != nil && *config.Page != "" {
		pageValue, err := strconv.Atoi(*config.Page)
		if err != nil {
			return fmt.Errorf("invalid page value '%s': must be a valid number", *config.Page)
		}

		if pageValue <= 0 {
			return fmt.Errorf("page must be greater than 0")
		}
	}

	return nil
}
