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

type DeleteRepository struct{}

type DeleteRepositoryConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Domain     string `json:"domain" mapstructure:"domain"`
	Repository string `json:"repository" mapstructure:"repository"`
}

func (c *DeleteRepository) Name() string {
	return "aws.codeArtifact.deleteRepository"
}

func (c *DeleteRepository) Label() string {
	return "CodeArtifact • Delete Repository"
}

func (c *DeleteRepository) Description() string {
	return "Delete an AWS CodeArtifact repository from a domain"
}

func (c *DeleteRepository) Documentation() string {
	return `The Delete Repository component deletes a repository from an AWS CodeArtifact domain.

## Use Cases

- **Cleanup**: Remove repositories after migration or deprecation
- **Environment teardown**: Delete temporary repositories created by workflows
- **Lifecycle management**: Enforce retention by deleting old repositories
`
}

func (c *DeleteRepository) Icon() string {
	return "aws"
}

func (c *DeleteRepository) Color() string {
	return "gray"
}

func (c *DeleteRepository) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRepository) Configuration() []configuration.Field {
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
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
				{
					Field:  "domain",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.repository",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "domain",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "domain",
							},
						},
					},
				},
			},
		},
	}
}

func (c *DeleteRepository) Setup(ctx core.SetupContext) error {
	var config DeleteRepositoryConfiguration
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
		return fmt.Errorf("repository is required")
	}

	return nil
}

func (c *DeleteRepository) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteRepository) Execute(ctx core.ExecutionContext) error {
	var config DeleteRepositoryConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	repo, err := client.DeleteRepository(DeleteRepositoryInput{
		Domain:     config.Domain,
		Repository: config.Repository,
	})
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
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

func (c *DeleteRepository) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteRepository) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteRepository) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteRepository) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteRepository) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteRepository) normalizeConfig(config DeleteRepositoryConfiguration) DeleteRepositoryConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.Repository = strings.TrimSpace(config.Repository)
	return config
}
