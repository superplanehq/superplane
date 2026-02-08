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

type CopyPackageVersions struct{}

type CopyPackageVersionsConfiguration struct {
	Region                string `json:"region" mapstructure:"region"`
	Domain                string `json:"domain" mapstructure:"domain"`
	SourceRepository      string `json:"sourceRepository" mapstructure:"sourceRepository"`
	DestinationRepository string `json:"destinationRepository" mapstructure:"destinationRepository"`
	Format                string `json:"format" mapstructure:"format"`
	Package               string `json:"package" mapstructure:"package"`
	Namespace             string `json:"namespace" mapstructure:"namespace"`
	Versions              string `json:"versions" mapstructure:"versions"` // comma or newline separated
	AllowOverwrite        bool   `json:"allowOverwrite" mapstructure:"allowOverwrite"`
	IncludeFromUpstream   bool   `json:"includeFromUpstream" mapstructure:"includeFromUpstream"`
}

func (c *CopyPackageVersions) Name() string {
	return "aws.codeArtifact.copyPackageVersions"
}

func (c *CopyPackageVersions) Label() string {
	return "CodeArtifact • Copy Package Versions"
}

func (c *CopyPackageVersions) Description() string {
	return "Copy package versions from one repository to another in the same domain"
}

func (c *CopyPackageVersions) Documentation() string {
	return `The Copy Package Versions component copies one or more package versions from a source repository to a destination repository in the same domain.

## Use Cases

- **Promotion**: Copy approved versions from staging to production
- **Replication**: Mirror packages across repositories
- **Migration**: Move versions between repos in the same domain
`
}

func (c *CopyPackageVersions) Icon() string {
	return "aws"
}

func (c *CopyPackageVersions) Color() string {
	return "gray"
}

func (c *CopyPackageVersions) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CopyPackageVersions) Configuration() []configuration.Field {
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
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.domain",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
					},
				},
			},
		},
		{
			Name:     "sourceRepository",
			Label:    "Source repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "domain", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.repository",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
						{Name: "domain", ValueFrom: &configuration.ParameterValueFrom{Field: "domain"}},
					},
				},
			},
		},
		{
			Name:     "destinationRepository",
			Label:    "Destination repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
				{Field: "domain", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.repository",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}},
						{Name: "domain", ValueFrom: &configuration.ParameterValueFrom{Field: "domain"}},
					},
				},
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
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
		{
			Name:        "package",
			Label:       "Package name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. lodash (npm), my-package (pypi)",
			Description: "Name of the package to copy (must exist in source repository)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
		{
			Name:        "versions",
			Label:       "Versions",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "1.0.0, 1.0.1 or one per line",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
		{
			Name:     "allowOverwrite",
			Label:    "Allow overwrite",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
		{
			Name:     "includeFromUpstream",
			Label:    "Include from upstream",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
		{
			Name:     "namespace",
			Label:    "Namespace",
			Type:     configuration.FieldTypeString,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceRepository", Values: []string{"*"}},
			},
		},
	}
}

func (c *CopyPackageVersions) Setup(ctx core.SetupContext) error {
	var config CopyPackageVersionsConfiguration
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
	if config.SourceRepository == "" {
		return fmt.Errorf("source repository is required")
	}
	if config.DestinationRepository == "" {
		return fmt.Errorf("destination repository is required")
	}
	if config.Format == "" {
		return fmt.Errorf("format is required")
	}
	if config.Package == "" {
		return fmt.Errorf("package is required")
	}
	versions := parseVersionsList(config.Versions)
	if len(versions) == 0 {
		return fmt.Errorf("at least one version is required")
	}
	return nil
}

func (c *CopyPackageVersions) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CopyPackageVersions) Execute(ctx core.ExecutionContext) error {
	var config CopyPackageVersionsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}
	versions := parseVersionsList(config.Versions)
	client := NewClient(ctx.HTTP, creds, config.Region)
	resp, err := client.CopyPackageVersions(CopyPackageVersionsInput{
		Domain:                config.Domain,
		SourceRepository:      config.SourceRepository,
		DestinationRepository: config.DestinationRepository,
		Format:                config.Format,
		Namespace:             config.Namespace,
		Package:               config.Package,
		Versions:              versions,
		AllowOverwrite:        config.AllowOverwrite,
		IncludeFromUpstream:   config.IncludeFromUpstream,
	})
	if err != nil {
		return fmt.Errorf("failed to copy package versions: %w", err)
	}
	output := map[string]any{
		"successfulVersions": resp.SuccessfulVersions,
		"failedVersions":     resp.FailedVersions,
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codeartifact.package.versions.copied",
		[]any{output},
	)
}

func (c *CopyPackageVersions) Actions() []core.Action {
	return nil
}

func (c *CopyPackageVersions) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CopyPackageVersions) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CopyPackageVersions) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CopyPackageVersions) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CopyPackageVersions) normalizeConfig(config CopyPackageVersionsConfiguration) CopyPackageVersionsConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.SourceRepository = strings.TrimSpace(config.SourceRepository)
	config.DestinationRepository = strings.TrimSpace(config.DestinationRepository)
	config.Format = strings.ToLower(strings.TrimSpace(config.Format))
	config.Package = strings.TrimSpace(config.Package)
	config.Namespace = strings.TrimSpace(config.Namespace)
	config.Versions = strings.TrimSpace(config.Versions)
	return config
}
