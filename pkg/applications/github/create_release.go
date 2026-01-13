package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateRelease struct{}

type CreateReleaseConfiguration struct {
	Repository           string `mapstructure:"repository"`
	VersionStrategy      string `mapstructure:"versionStrategy"`
	TagName              string `mapstructure:"tagName"`
	Name                 string `mapstructure:"name"`
	Draft                bool   `mapstructure:"draft"`
	Prerelease           bool   `mapstructure:"prerelease"`
	GenerateReleaseNotes bool   `mapstructure:"generateReleaseNotes"`
	Body                 string `mapstructure:"body"`
}

// Semantic version regex pattern: captures optional prefix, major, minor, patch
var semverRegex = regexp.MustCompile(`^([a-zA-Z-]*)(\d+)\.(\d+)\.(\d+)`)

func (c *CreateRelease) Name() string {
	return "github.createRelease"
}

func (c *CreateRelease) Label() string {
	return "Create Release"
}

func (c *CreateRelease) Description() string {
	return "Create a new release in a GitHub repository"
}

func (c *CreateRelease) Icon() string {
	return "github"
}

func (c *CreateRelease) Color() string {
	return "gray"
}

func (c *CreateRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRelease) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "versionStrategy",
			Label:    "Version Strategy",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "manual",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Manual (specify tag)",
							Value: "manual",
						},
						{
							Label: "Auto-increment patch (x.x.X)",
							Value: "patch",
						},
						{
							Label: "Auto-increment minor (x.X.0)",
							Value: "minor",
						},
						{
							Label: "Auto-increment major (X.0.0)",
							Value: "major",
						},
					},
				},
			},
			Description: "How to determine the release version",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "v1.0.0",
			Description: "The name of the tag to create the release for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "versionStrategy",
					Values: []string{"manual"},
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "versionStrategy",
					Values: []string{"manual"},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Release Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Release 1.0.0",
			Description: "The title of the release",
		},
		{
			Name:        "draft",
			Label:       "Draft",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Mark this release as a draft",
		},
		{
			Name:        "prerelease",
			Label:       "Prerelease",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Mark this release as a prerelease",
		},
		{
			Name:        "generateReleaseNotes",
			Label:       "Generate release notes",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Automatically generate release notes from commits since the last release",
		},
		{
			Name:        "body",
			Label:       "Additional notes",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "## Important Notes\n\nPlease review the breaking changes...",
			Description: "Optional text to append after auto-generated release notes. If auto-generation is off, this becomes the entire release description.",
		},
	}
}

func (c *CreateRelease) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)
}

func (c *CreateRelease) Execute(ctx core.ExecutionContext) error {
	var config CreateReleaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallation, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Determine the tag name based on version strategy
	//
	tagName, err := c.determineTagName(ctx, client, appMetadata.Owner, config)
	if err != nil {
		return fmt.Errorf("failed to determine tag name: %w", err)
	}

	//
	// Determine release body
	//
	var body string
	if config.GenerateReleaseNotes {
		generatedNotes, err := c.generateReleaseNotes(ctx, client, appMetadata.Owner, config.Repository, tagName)
		if err != nil {
			return fmt.Errorf("failed to generate release notes: %w", err)
		}
		body = generatedNotes

		// Append custom notes if provided
		if config.Body != "" {
			body = body + "\n\n" + config.Body
		}
	} else {
		body = config.Body
	}

	//
	// Prepare the release request
	//
	releaseRequest := &github.RepositoryRelease{
		TagName:    &tagName,
		Draft:      &config.Draft,
		Prerelease: &config.Prerelease,
	}

	if config.Name != "" {
		releaseRequest.Name = &config.Name
	}

	if body != "" {
		releaseRequest.Body = &body
	}

	//
	// Create the release
	//
	release, _, err := client.Repositories.CreateRelease(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		releaseRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	//
	// Emit output with release data
	//
	output := map[string]interface{}{
		"id":           release.GetID(),
		"tag_name":     release.GetTagName(),
		"name":         release.GetName(),
		"html_url":     release.GetHTMLURL(),
		"draft":        release.GetDraft(),
		"prerelease":   release.GetPrerelease(),
		"created_at":   release.GetCreatedAt().String(),
		"published_at": release.GetPublishedAt().String(),
		"author":       release.GetAuthor(),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.release",
		[]any{output},
	)
}

func (c *CreateRelease) determineTagName(ctx core.ExecutionContext, client *github.Client, owner string, config CreateReleaseConfiguration) (string, error) {
	if config.VersionStrategy == "manual" {
		return config.TagName, nil
	}

	//
	// Get latest release for auto-increment
	//
	latestRelease, _, err := client.Repositories.GetLatestRelease(
		context.Background(),
		owner,
		config.Repository,
	)
	if err != nil {
		return "", fmt.Errorf("no previous release found for auto-increment. Please create the first release manually: %w", err)
	}

	currentTag := latestRelease.GetTagName()

	//
	// Try to find an available version (handle case where tag might exist from draft/prerelease)
	//
	return c.findAvailableVersion(ctx, client, owner, config.Repository, currentTag, config.VersionStrategy)
}

func (c *CreateRelease) findAvailableVersion(ctx core.ExecutionContext, client *github.Client, owner, repo, currentTag, strategy string) (string, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		newTag, err := c.incrementVersion(currentTag, strategy)
		if err != nil {
			return "", err
		}

		// Check if this tag already exists
		exists, err := c.tagExists(client, owner, repo, newTag)
		if err != nil {
			// If we can't check, just try to create anyway
			ctx.Logger.Warnf("Failed to check if tag %s exists: %v. Will attempt to create.", newTag, err)
			return newTag, nil
		}

		if !exists {
			// Found an available version!
			if attempt > 0 {
				ctx.Logger.Infof("Tag %s already existed. Using next available version: %s", currentTag, newTag)
			}
			return newTag, nil
		}

		// Tag exists, try the next version
		ctx.Logger.Infof("Tag %s already exists, trying next version...", newTag)
		currentTag = newTag
	}

	// Exhausted all attempts
	lastAttempt, _ := c.incrementVersion(currentTag, strategy)
	return "", fmt.Errorf("failed to find available version after %d attempts. Last tried: %s. This may indicate many draft/prerelease versions exist. Please manually specify a version or clean up existing tags", maxAttempts, lastAttempt)
}

func (c *CreateRelease) tagExists(client *github.Client, owner, repo, tag string) (bool, error) {
	_, resp, err := client.Git.GetRef(
		context.Background(),
		owner,
		repo,
		fmt.Sprintf("tags/%s", tag),
	)

	if err != nil {
		// 404 means tag doesn't exist
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (c *CreateRelease) incrementVersion(currentTag string, strategy string) (string, error) {
	//
	// Parse the current version using regex
	//
	matches := semverRegex.FindStringSubmatch(currentTag)
	if matches == nil || len(matches) < 5 {
		return "", fmt.Errorf("invalid version format: %s. Expected semantic version like v1.2.3", currentTag)
	}

	prefix := matches[1]       // e.g., "v" or "version-" or ""
	major, _ := strconv.Atoi(matches[2])
	minor, _ := strconv.Atoi(matches[3])
	patch, _ := strconv.Atoi(matches[4])

	//
	// Increment based on strategy
	//
	switch strategy {
	case "patch":
		patch++
	case "minor":
		minor++
		patch = 0
	case "major":
		major++
		minor = 0
		patch = 0
	default:
		return "", fmt.Errorf("invalid version strategy: %s", strategy)
	}

	//
	// Reconstruct the tag with the same prefix
	//
	newTag := fmt.Sprintf("%s%d.%d.%d", prefix, major, minor, patch)
	return newTag, nil
}

func (c *CreateRelease) generateReleaseNotes(ctx core.ExecutionContext, client *github.Client, owner, repo, tagName string) (string, error) {
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

func (c *CreateRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateRelease) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateRelease) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}
