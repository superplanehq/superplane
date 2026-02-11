package codeartifact

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CreateRepository struct{}

type CreateRepositoryConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Domain      string `json:"domain" mapstructure:"domain"`
	Repository  string `json:"repository" mapstructure:"repository"`
	Description string `json:"description" mapstructure:"description"`
}

func (c *CreateRepository) Name() string {
	return "aws.codeArtifact.createRepository"
}

func (c *CreateRepository) Label() string {
	return "CodeArtifact • Create Repository"
}

func (c *CreateRepository) Description() string {
	return "Create an AWS CodeArtifact repository in a domain"
}

func (c *CreateRepository) Documentation() string {
	return `The Create Repository component creates a new repository in an AWS CodeArtifact domain.

## Use Cases

- **Automated setup**: Create repositories as part of onboarding or pipeline setup
- **Environment replication**: Mirror repository structure across domains
- **Workflow provisioning**: Create a destination repository before copying packages
`
}

func (c *CreateRepository) Icon() string {
	return "aws"
}

func (c *CreateRepository) Color() string {
	return "gray"
}

func (c *CreateRepository) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRepository) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: RegionsForCodeArtifact,
				},
			},
		},
		{
			Name:     "domain",
			Label:    "Domain",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.domain",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "my-repo",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "domain",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional repository description",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "repository",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *CreateRepository) Setup(ctx core.SetupContext) error {
	var config CreateRepositoryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if config.Repository == "" {
		return fmt.Errorf("repository name is required")
	}

	return nil
}

func (c *CreateRepository) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRepository) Execute(ctx core.ExecutionContext) error {
	var config CreateRepositoryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	repo, err := client.CreateRepository(CreateRepositoryInput{
		Domain:      config.Domain,
		Repository:  config.Repository,
		Description: config.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	output := map[string]any{
		"repository": repo,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codeartifact.repository",
		[]any{output},
	)
}

func (c *CreateRepository) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateRepository) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateRepository) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateRepository) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRepository) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateRepository) normalizeConfig(config CreateRepositoryConfiguration) CreateRepositoryConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Description = strings.TrimSpace(config.Description)
	return config
}
