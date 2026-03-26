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

type CreateDeploy struct{}

type CreateDeployNodeMetadata struct {
	Project *ProjectSummary `json:"project,omitempty" mapstructure:"project"`
}

type CreateDeployConfiguration struct {
	Project        string `json:"project" mapstructure:"project"`
	ReleaseVersion string `json:"releaseVersion" mapstructure:"releaseVersion"`
	Environment    string `json:"environment" mapstructure:"environment"`
	Name           string `json:"name" mapstructure:"name"`
	URL            string `json:"url" mapstructure:"url"`
	DateStarted    string `json:"dateStarted" mapstructure:"dateStarted"`
	DateFinished   string `json:"dateFinished" mapstructure:"dateFinished"`
}

func (c *CreateDeploy) Name() string {
	return "sentry.createDeploy"
}

func (c *CreateDeploy) Label() string {
	return "Create Deploy"
}

func (c *CreateDeploy) Description() string {
	return "Mark a deploy against an existing Sentry release"
}

func (c *CreateDeploy) Documentation() string {
	return `The Create Deploy component marks a deploy against an existing Sentry release.

## Use Cases

- **Deployment tracking**: record when a release reaches staging or production
- **Release automation**: pair with Create Release in CI/CD canvases
- **Operational context**: give Sentry deployment timing and environment information for later triage

## Configuration

- **Project**: Optional Sentry project to associate with the deploy
- **Release**: Select the existing Sentry release to deploy
- **Environment**: The target environment, such as staging or production
- **Name**: Optional deploy name
- **Deploy URL**: Optional URL for the deployment
- **Started At**: Optional deployment start time
- **Finished At**: Optional deployment finish time

## Output

Returns the created deploy record, including environment, timestamps, URL, and release version.`
}

func (c *CreateDeploy) Icon() string {
	return "bug"
}

func (c *CreateDeploy) Color() string {
	return "gray"
}

func (c *CreateDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateDeploy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional Sentry project to associate with the deploy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "releaseVersion",
			Label:       "Release",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Existing Sentry release to deploy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRelease,
					Parameters: []configuration.ParameterRef{
						{
							Name: "project",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "project",
							},
						},
					},
				},
			},
		},
		{
			Name:        "environment",
			Label:       "Environment",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Deployment environment, such as staging or production",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional deploy name",
		},
		{
			Name:        "url",
			Label:       "Deploy URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional URL for the deploy or release dashboard",
		},
		{
			Name:        "dateStarted",
			Label:       "Started At",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional deployment start time",
		},
		{
			Name:        "dateFinished",
			Label:       "Finished At",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "Optional deployment finish time",
		},
	}
}

func (c *CreateDeploy) Setup(ctx core.SetupContext) error {
	config, err := decodeCreateDeployConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateDeployConfiguration(config); err != nil {
		return err
	}

	var metadata CreateDeployNodeMetadata
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	if config.Project == "" {
		if err := client.ValidateReleaseAccess(); err != nil {
			return fmt.Errorf("failed to validate sentry release access: %w", err)
		}

		return ctx.Metadata.Set(metadata)
	}

	project := findProject(ctx.Integration, config.Project)
	if project == nil {
		return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
	}

	if err := client.ValidateReleaseAccess(); err != nil {
		return fmt.Errorf("failed to validate sentry release access: %w", err)
	}

	metadata = CreateDeployNodeMetadata{
		Project: project,
	}

	return ctx.Metadata.Set(metadata)
}

func (c *CreateDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateDeploy) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateDeployConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateCreateDeployConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	projects := []string(nil)
	if config.Project != "" {
		projects = []string{config.Project}
	}

	deploy, err := client.CreateDeploy(config.ReleaseVersion, CreateDeployRequest{
		Environment:  config.Environment,
		Name:         config.Name,
		URL:          config.URL,
		Projects:     projects,
		DateStarted:  config.DateStarted,
		DateFinished: config.DateFinished,
	})
	if err != nil {
		return fmt.Errorf("failed to create sentry deploy: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.deploy", []any{deploy})
}

func (c *CreateDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeCreateDeployConfiguration(input any) (CreateDeployConfiguration, error) {
	config := CreateDeployConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return CreateDeployConfiguration{}, err
	}

	config.Project = strings.TrimSpace(config.Project)
	config.ReleaseVersion = strings.TrimSpace(config.ReleaseVersion)
	config.Environment = strings.TrimSpace(config.Environment)
	config.Name = strings.TrimSpace(config.Name)
	config.URL = strings.TrimSpace(config.URL)
	config.DateStarted = strings.TrimSpace(config.DateStarted)
	config.DateFinished = strings.TrimSpace(config.DateFinished)

	return config, nil
}

func validateCreateDeployConfiguration(config CreateDeployConfiguration) error {
	if config.ReleaseVersion == "" {
		return fmt.Errorf("releaseVersion is required")
	}
	if config.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	return nil
}
