package github

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRelease struct{}

type GetReleaseConfiguration struct {
	Repository      string  `mapstructure:"repository"`
	ReleaseStrategy string  `mapstructure:"releaseStrategy"`
	TagName         *string `mapstructure:"tagName,omitempty"`
	ReleaseID       *int64  `mapstructure:"releaseId,omitempty"`
}

func (c *GetRelease) Name() string {
	return "github.getRelease"
}

func (c *GetRelease) Label() string {
	return "Get Release"
}

func (c *GetRelease) Description() string {
	return "Get a release from a GitHub repository"
}

func (c *GetRelease) Documentation() string {
	return `The Get Release component retrieves release information from a GitHub repository.

## Use Cases

- **Release Monitoring**: Get details about a specific release
- **Deployment Pipelines**: Fetch release assets and metadata for deployment
- **Version Tracking**: Monitor release status and changelog
- **CI/CD Integration**: Retrieve release information for build processes

## Configuration

- **Repository**: Select the GitHub repository
- **Release Strategy**: How to find the release (by tag name, by ID, or latest)
- **Tag Name**: Git tag name of the release (if using tag strategy)
- **Release ID**: Numeric release ID (if using ID strategy)

## Output

Returns release information including:
- Release ID, name, and tag name
- Release body/description
- Draft and prerelease status
- Created and published timestamps
- Author information
- Asset URLs`
}

func (c *GetRelease) Icon() string {
	return "github"
}

func (c *GetRelease) Color() string {
	return "gray"
}

func (c *GetRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRelease) Configuration() []configuration.Field {
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
			Name:     "releaseStrategy",
			Label:    "Release to Get",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "specific",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Specific tag",
							Value: "specific",
						},
						{
							Label: "By release ID",
							Value: "byId",
						},
						{
							Label: "Latest release",
							Value: "latest",
						},
						{
							Label: "Latest draft",
							Value: "latestDraft",
						},
						{
							Label: "Latest prerelease",
							Value: "latestPrerelease",
						},
					},
				},
			},
			Description: "How to identify which release to retrieve",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., v1.0.0 or {{$.data.tag_name}}",
			Description: "Git tag identifying the release. Supports template variables from previous steps.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "releaseStrategy",
					Values: []string{"specific"},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "releaseStrategy",
					Values: []string{"specific"},
				},
			},
		},
		{
			Name:        "releaseId",
			Label:       "Release ID",
			Type:        configuration.FieldTypeNumber,
			Placeholder: "e.g., 12345678",
			Description: "Numeric release ID. Supports template variables from previous steps.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "releaseStrategy",
					Values: []string{"byId"},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "releaseStrategy",
					Values: []string{"byId"},
				},
			},
		},
	}
}

func (c *GetRelease) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *GetRelease) Execute(ctx core.ExecutionContext) error {
	var config GetReleaseConfiguration
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

	//
	// Fetch the release based on the selected strategy
	//
	var releaseData map[string]any

	if config.ReleaseStrategy == "byId" {
		if config.ReleaseID == nil {
			return fmt.Errorf("release ID is required when using byId strategy")
		}
		// Fetch by release ID
		release, _, err := client.Repositories.GetRelease(
			context.Background(),
			appMetadata.Owner,
			config.Repository,
			*config.ReleaseID,
		)
		if err != nil {
			return fmt.Errorf("failed to get release with ID %d: %w", *config.ReleaseID, err)
		}
		releaseData = buildReleaseData(release)
	} else {
		// Use the common helper for tag-based strategies
		tagName := ""
		if config.TagName != nil {
			tagName = *config.TagName
		}
		release, err := fetchReleaseByStrategy(client, appMetadata.Owner, config.Repository, config.ReleaseStrategy, tagName)
		if err != nil {
			return err
		}
		releaseData = buildReleaseData(release)
	}

	//
	// Emit output with release data
	//
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.release",
		[]any{releaseData},
	)
}

func (c *GetRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetRelease) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRelease) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}
