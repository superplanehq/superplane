package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteRelease struct{}

type DeleteReleaseConfiguration struct {
	Repository      string `mapstructure:"repository"`
	ReleaseStrategy string `mapstructure:"releaseStrategy"`
	TagName         string `mapstructure:"tagName"`
	DeleteTag       bool   `mapstructure:"deleteTag"`
}

func (c *DeleteRelease) Name() string {
	return "github.deleteRelease"
}

func (c *DeleteRelease) Label() string {
	return "Delete Release"
}

func (c *DeleteRelease) Description() string {
	return "Delete a release from a GitHub repository"
}

func (c *DeleteRelease) Icon() string {
	return "github"
}

func (c *DeleteRelease) Color() string {
	return "gray"
}

func (c *DeleteRelease) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRelease) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "releaseStrategy",
			Label:    "Release to Delete",
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
			Description: "How to identify which release to delete",
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., v1.0.0 or {{$.data.tag_name}}",
			Description: "Git tag identifying the release to delete. Supports template variables from previous steps.",
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
			Name:        "deleteTag",
			Label:       "Also delete Git tag",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "When enabled, also deletes the associated Git tag from the repository",
		},
	}
}

func (c *DeleteRelease) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)
}

func (c *DeleteRelease) Execute(ctx core.ExecutionContext) error {
	var config DeleteReleaseConfiguration
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
	// Fetch the release based on the selected strategy
	//
	release, err := fetchReleaseByStrategy(client, appMetadata.Owner, config.Repository, config.ReleaseStrategy, config.TagName)
	if err != nil {
		return err
	}

	//
	// Store pre-deletion state for output
	//
	deletedReleaseData := map[string]any{
		"id":          release.GetID(),
		"tag_name":    release.GetTagName(),
		"name":        release.GetName(),
		"html_url":    release.GetHTMLURL(),
		"draft":       release.GetDraft(),
		"prerelease":  release.GetPrerelease(),
		"deleted_at":  time.Now().Format(time.RFC3339),
		"tag_deleted": false,
	}

	//
	// Delete the release
	//
	_, err = client.Repositories.DeleteRelease(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		release.GetID(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	//
	// Optionally delete the Git tag
	//
	if config.DeleteTag {
		_, err = client.Git.DeleteRef(
			context.Background(),
			appMetadata.Owner,
			config.Repository,
			fmt.Sprintf("tags/%s", release.GetTagName()),
		)
		if err != nil {
			// Log warning but don't fail the operation since release deletion succeeded
			ctx.Logger.Warnf("Release deleted successfully, but failed to delete Git tag %s: %v", release.GetTagName(), err)
		} else {
			deletedReleaseData["tag_deleted"] = true
		}
	}

	//
	// Emit output with deleted release data
	//
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.release",
		[]any{deletedReleaseData},
	)
}

func (c *DeleteRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *DeleteRelease) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteRelease) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}
