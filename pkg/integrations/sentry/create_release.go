package sentry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateRelease struct{}

type CreateReleaseNodeMetadata struct {
	Project *ProjectSummary `json:"project,omitempty" mapstructure:"project"`
}

type CreateReleaseConfiguration struct {
	Project string                             `json:"project" mapstructure:"project"`
	Version string                             `json:"version" mapstructure:"version"`
	Ref     string                             `json:"ref" mapstructure:"ref"`
	URL     string                             `json:"url" mapstructure:"url"`
	Commits []CreateReleaseCommitConfiguration `json:"commits" mapstructure:"commits"`
	Refs    []CreateReleaseRefConfiguration    `json:"refs" mapstructure:"refs"`
}

type CreateReleaseCommitConfiguration struct {
	ID          string `json:"id" mapstructure:"id"`
	Repository  string `json:"repository" mapstructure:"repository"`
	Message     string `json:"message" mapstructure:"message"`
	AuthorName  string `json:"authorName" mapstructure:"authorName"`
	AuthorEmail string `json:"authorEmail" mapstructure:"authorEmail"`
	Timestamp   string `json:"timestamp" mapstructure:"timestamp"`
}

type CreateReleaseRefConfiguration struct {
	Repository     string `json:"repository" mapstructure:"repository"`
	Commit         string `json:"commit" mapstructure:"commit"`
	PreviousCommit string `json:"previousCommit" mapstructure:"previousCommit"`
}

func (c *CreateRelease) Name() string {
	return "sentry.createRelease"
}

func (c *CreateRelease) Label() string {
	return "Create Release"
}

func (c *CreateRelease) Description() string {
	return "Register a new release in Sentry after a deploy or build step"
}

func (c *CreateRelease) Documentation() string {
	return `The Create Release component registers a new release in Sentry for a selected project.

## Use Cases

- **Release tracking**: create a release after a build or deploy succeeds
- **Commit association**: attach commits and refs so Sentry can correlate new issues with code changes
- **Post-deploy automation**: feed the created release into downstream deployment and monitoring steps

## Configuration

- **Project**: Select the Sentry project this release applies to
- **Version**: The release version identifier
- **Ref**: Optional commit or tag reference for the release
- **Release URL**: Optional URL for the release, build, or changelog
- **Commits**: Optional commit metadata to associate with the release
- **Refs**: Optional repository head/previous commit refs for release comparison

## Output

Returns the created Sentry release object, including version, associated projects, deploy count, and release metadata.`
}

func (c *CreateRelease) Icon() string {
	return "bug"
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
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Sentry project for this release",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "version",
			Label:       "Version",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Release version identifier, such as 2026.03.25 or a commit SHA",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional tag or commit reference for the release",
		},
		{
			Name:        "url",
			Label:       "Release URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional URL to the release, build, or changelog",
		},
		{
			Name:        "commits",
			Label:       "Commits",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional commits to associate with the release",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Commit",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "id",
								Label:    "Commit SHA",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "repository",
								Label:    "Repository",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "message",
								Label:    "Message",
								Type:     configuration.FieldTypeText,
								Required: false,
							},
							{
								Name:     "authorName",
								Label:    "Author Name",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "authorEmail",
								Label:    "Author Email",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
							{
								Name:     "timestamp",
								Label:    "Timestamp",
								Type:     configuration.FieldTypeDateTime,
								Required: false,
							},
						},
					},
				},
			},
		},
		{
			Name:        "refs",
			Label:       "Refs",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional repository refs to compare this release against previous commits",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Ref",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "repository",
								Label:    "Repository",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "commit",
								Label:    "Commit",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "previousCommit",
								Label:    "Previous Commit",
								Type:     configuration.FieldTypeString,
								Required: false,
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateRelease) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateReleaseConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateReleaseConfiguration(config); err != nil {
		return err
	}

	project := findProject(ctx.Integration, config.Project)
	if project == nil {
		return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	if err := client.ValidateReleaseAccess(); err != nil {
		return fmt.Errorf("failed to validate sentry release access: %w", err)
	}

	return ctx.Metadata.Set(CreateReleaseNodeMetadata{
		Project: project,
	})
}

func (c *CreateRelease) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRelease) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateReleaseConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateReleaseConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	release, err := client.CreateRelease(CreateReleaseRequest{
		Version:  config.Version,
		Projects: []string{config.Project},
		Ref:      config.Ref,
		URL:      config.URL,
		Commits:  buildReleaseCommitPayload(config.Commits),
		Refs:     buildReleaseRefPayload(config.Refs),
	})
	if err != nil {
		return fmt.Errorf("failed to create sentry release: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.release", []any{release})
}

func (c *CreateRelease) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateRelease) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateRelease) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateRelease) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRelease) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateReleaseConfiguration(input any) (CreateReleaseConfiguration, error) {
	config := CreateReleaseConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return CreateReleaseConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	config.Version = strings.TrimSpace(config.Version)
	config.Ref = strings.TrimSpace(config.Ref)
	config.URL = strings.TrimSpace(config.URL)

	for index := range config.Commits {
		config.Commits[index].ID = strings.TrimSpace(config.Commits[index].ID)
		config.Commits[index].Repository = strings.TrimSpace(config.Commits[index].Repository)
		config.Commits[index].Message = strings.TrimSpace(config.Commits[index].Message)
		config.Commits[index].AuthorName = strings.TrimSpace(config.Commits[index].AuthorName)
		config.Commits[index].AuthorEmail = strings.TrimSpace(config.Commits[index].AuthorEmail)
		config.Commits[index].Timestamp = strings.TrimSpace(config.Commits[index].Timestamp)
	}

	for index := range config.Refs {
		config.Refs[index].Repository = strings.TrimSpace(config.Refs[index].Repository)
		config.Refs[index].Commit = strings.TrimSpace(config.Refs[index].Commit)
		config.Refs[index].PreviousCommit = strings.TrimSpace(config.Refs[index].PreviousCommit)
	}

	return config, nil
}

func validateCreateReleaseConfiguration(config CreateReleaseConfiguration) error {
	if config.Project == "" {
		return fmt.Errorf("project is required")
	}
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	for index, commit := range config.Commits {
		if commit.ID == "" {
			return fmt.Errorf("commits[%d].id is required", index)
		}
	}

	for index, ref := range config.Refs {
		if ref.Repository == "" {
			return fmt.Errorf("refs[%d].repository is required", index)
		}
		if ref.Commit == "" {
			return fmt.Errorf("refs[%d].commit is required", index)
		}
	}

	return nil
}

func buildReleaseCommitPayload(commits []CreateReleaseCommitConfiguration) []ReleaseCommit {
	if len(commits) == 0 {
		return nil
	}

	payload := make([]ReleaseCommit, 0, len(commits))
	for _, commit := range commits {
		payload = append(payload, ReleaseCommit{
			ID:          commit.ID,
			Repository:  commit.Repository,
			Message:     commit.Message,
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
			Timestamp:   commit.Timestamp,
		})
	}

	return payload
}

func buildReleaseRefPayload(refs []CreateReleaseRefConfiguration) []ReleaseRef {
	if len(refs) == 0 {
		return nil
	}

	payload := make([]ReleaseRef, 0, len(refs))
	for _, ref := range refs {
		payload = append(payload, ReleaseRef{
			Repository:     ref.Repository,
			Commit:         ref.Commit,
			PreviousCommit: ref.PreviousCommit,
		})
	}

	return payload
}
