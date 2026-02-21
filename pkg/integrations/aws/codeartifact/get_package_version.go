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

type GetPackageVersion struct{}

type GetPackageVersionConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Domain     string `json:"domain" mapstructure:"domain"`
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
	Format     string `json:"format" mapstructure:"format"`
	Namespace  string `json:"namespace" mapstructure:"namespace"`
	Version    string `json:"version" mapstructure:"version"`
}

func (c *GetPackageVersion) Name() string {
	return "aws.codeArtifact.getPackageVersion"
}

func (c *GetPackageVersion) Label() string {
	return "CodeArtifact • Get Package Version"
}

func (c *GetPackageVersion) Description() string {
	return "Describe an AWS CodeArtifact package version"
}

func (c *GetPackageVersion) Documentation() string {
	return `The Get Package Version component retrieves metadata for a specific package version in AWS CodeArtifact.

## Use Cases

- **Release automation**: Resolve package metadata before promotion
- **Audit trails**: Capture version details for reporting
- **Dependency checks**: Validate status and origin of package versions
`
}

func (c *GetPackageVersion) Icon() string {
	return "aws"
}

func (c *GetPackageVersion) Color() string {
	return "gray"
}

func (c *GetPackageVersion) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPackageVersion) Configuration() []configuration.Field {
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
		{
			Name:        "package",
			Label:       "Package name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. lodash, @my-scope/package (npm), my-python-package (pypi)",
			Description: "Name of the package in the repository (format-specific, e.g. npm scope/name)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:     "version",
			Label:    "Version",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:     "format",
			Label:    "Package format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: PackageFormatOptions},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:     "namespace",
			Label:    "Namespace",
			Type:     configuration.FieldTypeString,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "repository",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *GetPackageVersion) Setup(ctx core.SetupContext) error {
	var config GetPackageVersionConfiguration
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

	if config.Format == "" {
		return fmt.Errorf("format is required")
	}

	if config.Package == "" {
		return fmt.Errorf("package is required")
	}

	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	return nil
}

func (c *GetPackageVersion) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPackageVersion) Execute(ctx core.ExecutionContext) error {
	var config GetPackageVersionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.DescribePackageVersion(DescribePackageVersionInput{
		Domain:         config.Domain,
		Repository:     config.Repository,
		Format:         config.Format,
		Namespace:      config.Namespace,
		Package:        config.Package,
		PackageVersion: config.Version,
	})

	if err != nil {
		return fmt.Errorf("failed to describe package version: %w", err)
	}

	assets, err := client.ListPackageVersionAssets(ListPackageVersionAssetsInput{
		Domain:         config.Domain,
		Repository:     config.Repository,
		Format:         config.Format,
		Namespace:      config.Namespace,
		Package:        config.Package,
		PackageVersion: config.Version,
	})
	if err != nil {
		return fmt.Errorf("failed to list package version assets: %w", err)
	}

	output := map[string]any{
		"package": result,
		"assets":  assets,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codeartifact.package.version",
		[]any{output},
	)
}

func (c *GetPackageVersion) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPackageVersion) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPackageVersion) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPackageVersion) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPackageVersion) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetPackageVersion) normalizeConfig(config GetPackageVersionConfiguration) GetPackageVersionConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	config.Format = strings.ToLower(strings.TrimSpace(config.Format))
	config.Namespace = strings.TrimSpace(config.Namespace)
	config.Version = strings.TrimSpace(config.Version)
	return config
}
