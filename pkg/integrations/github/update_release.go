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

type UpdateRelease struct{}

type UpdateReleaseConfiguration struct {
	Repository           string `mapstructure:"repository"`
	ReleaseStrategy      string `mapstructure:"releaseStrategy"`
	TagName              string `mapstructure:"tagName"`
	Name                 string `mapstructure:"name"`
	Body                 string `mapstructure:"body"`
	Draft                bool   `mapstructure:"draft"`
	Prerelease           bool   `mapstructure:"prerelease"`
	GenerateReleaseNotes bool   `mapstructure:"generateReleaseNotes"`
}

func (c *UpdateRelease) Name() string {
	return "github.updateRelease"
}

func (c *UpdateRelease) Label() string {
	return "Update Release"
}

func (c *UpdateRelease) Description() string {
	return "Update an existing release in a GitHub repository"
}

func (c *UpdateRelease) Documentation() string {
	return `The Update Release component modifies an existing GitHub release.

## Use Cases

- **Release updates**: Update release notes or metadata after creation
- **Draft to published**: Convert draft releases to published releases
- **Metadata updates**: Update release name, description, or tags
- **Prerelease management**: Change prerelease status

## Configuration

- **Repository**: Select the GitHub repository
- **Release Strategy**: How to find the release (by tag name or latest)
- **Tag Name**: Git tag name of the release to update (if using tag strategy)
- **Name**: New release title/name (optional, supports expressions)
- **Body**: New release notes/description (optional, supports markdown and expressions)
- **Draft**: Update draft status
- **Prerelease**: Update prerelease status
- **Generate Release Notes**: Regenerate release notes from commits

## Output

Returns the updated release object with all current information.`
}

func (c *UpdateRelease) Icon() string {
	return "github"
}

func (c *UpdateRelease) Color() string {
	return "gray"
}

func (c *UpdateRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateRelease) Configuration() []configuration.Field {
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
			Label:    "Release Strategy",
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
			Description: "How to identify which release to update",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., v1.0.0 or {{$.data.tag_name}}",
			Description: "Git tag identifying the release to update. Supports template variables from previous steps.",
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
			Name:        "name",
			Label:       "Release Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Release 1.0.0",
			Description: "Update the release title (leave empty to keep current)",
		},
		{
			Name:        "generateReleaseNotes",
			Label:       "Generate release notes",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Automatically generate release notes from commits since the last release. If body is also provided, custom text is appended.",
		},
		{
			Name:        "body",
			Label:       "Release Notes",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "## What's Changed\n\n...",
			Description: "Update release description (leave empty to keep current)",
		},
		{
			Name:        "draft",
			Label:       "Draft",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Mark release as draft or publish it",
		},
		{
			Name:        "prerelease",
			Label:       "Prerelease",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Mark as prerelease or stable release",
		},
	}
}

func (c *UpdateRelease) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *UpdateRelease) Execute(ctx core.ExecutionContext) error {
	var config UpdateReleaseConfiguration
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
	// Fetch the existing release based on the selected strategy
	//
	release, err := fetchReleaseByStrategy(client, appMetadata.Owner, config.Repository, config.ReleaseStrategy, config.TagName)
	if err != nil {
		return err
	}

	//
	// Build the update request with partial updates
	//
	releaseRequest := &github.RepositoryRelease{}

	// Only set name if provided
	if config.Name != "" {
		releaseRequest.Name = &config.Name
	}

	// Handle body/notes logic
	if config.GenerateReleaseNotes {
		generatedNotes, err := c.generateReleaseNotes(ctx, client, appMetadata.Owner, config.Repository, release.GetTagName())
		if err != nil {
			return fmt.Errorf("failed to generate release notes: %w", err)
		}
		body := generatedNotes

		// Append custom notes if provided
		if config.Body != "" {
			body = body + "\n\n" + config.Body
		}
		releaseRequest.Body = &body
	} else if config.Body != "" {
		releaseRequest.Body = &config.Body
	}

	// Handle boolean fields - compare against current state
	// to distinguish "keep current" vs "change to false"
	currentDraft := release.GetDraft()
	currentPrerelease := release.GetPrerelease()

	// Only set if value changed from current state
	if config.Draft != currentDraft {
		releaseRequest.Draft = &config.Draft
	}
	if config.Prerelease != currentPrerelease {
		releaseRequest.Prerelease = &config.Prerelease
	}

	//
	// Update the release
	//
	updatedRelease, _, err := client.Repositories.EditRelease(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		release.GetID(),
		releaseRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to update release: %w", err)
	}

	//
	// Emit output with updated release data
	//
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.release",
		[]any{updatedRelease},
	)
}

func (c *UpdateRelease) generateReleaseNotes(_ core.ExecutionContext, client *github.Client, owner, repo, tagName string) (string, error) {
	opts := &github.GenerateNotesOptions{
		TagName: tagName,
	}

	notes, _, err := client.Repositories.GenerateReleaseNotes(
		context.Background(),
		owner,
		repo,
		opts,
	)
	if err != nil {
		return "", err
	}

	return notes.Body, nil
}

func (c *UpdateRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *UpdateRelease) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateRelease) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}
