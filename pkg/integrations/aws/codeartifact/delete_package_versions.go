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

type DeletePackageVersions struct{}

type DeletePackageVersionsConfiguration struct {
	Region         string `json:"region" mapstructure:"region"`
	Domain         string `json:"domain" mapstructure:"domain"`
	Repository     string `json:"repository" mapstructure:"repository"`
	Format         string `json:"format" mapstructure:"format"`
	Package        string `json:"package" mapstructure:"package"`
	Namespace      string `json:"namespace" mapstructure:"namespace"`
	Versions       string `json:"versions" mapstructure:"versions"`
	ExpectedStatus string `json:"expectedStatus" mapstructure:"expectedStatus"`
}

func (c *DeletePackageVersions) Name() string {
	return "aws.codeArtifact.deletePackageVersions"
}

func (c *DeletePackageVersions) Label() string {
	return "CodeArtifact • Delete Package Versions"
}

func (c *DeletePackageVersions) Description() string {
	return "Permanently delete one or more package versions from a repository"
}

func (c *DeletePackageVersions) Documentation() string {
	return `The Delete Package Versions component permanently removes package versions and their assets. Deleted versions cannot be restored. To remove from view but keep the option to restore later, use Update Package Versions Status to set status to Archived instead.

## Use Cases

- **Cleanup**: Remove obsolete or invalid versions
- **Compliance**: Permanently remove versions that must not be retained
- **Storage**: Free space by deleting unused versions
`
}

func (c *DeletePackageVersions) Icon() string {
	return "aws"
}

func (c *DeletePackageVersions) Color() string {
	return "gray"
}

func (c *DeletePackageVersions) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeletePackageVersions) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: RegionsForCodeArtifact},
			},
		},
		{
			Name:                 "domain",
			Label:                "Domain",
			Type:                 configuration.FieldTypeIntegrationResource,
			Required:             true,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "region", Values: []string{"*"}}},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "codeartifact.domain",
					UseNameAsValue: true,
					Parameters:     []configuration.ParameterRef{{Name: "region", ValueFrom: &configuration.ParameterValueFrom{Field: "region"}}},
				},
			},
		},
		{
			Name:     "repository",
			Label:    "Repository",
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
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "repository", Values: []string{"*"}}},
		},
		{
			Name:                 "package",
			Label:                "Package name",
			Type:                 configuration.FieldTypeString,
			Required:             true,
			Placeholder:          "e.g. lodash (npm), my-package (pypi)",
			Description:          "Name of the package whose versions to delete",
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "repository", Values: []string{"*"}}},
		},
		{
			Name:                 "versions",
			Label:                "Versions",
			Type:                 configuration.FieldTypeString,
			Required:             true,
			Placeholder:          "1.0.0, 1.0.1",
			Description:          "Comma separated list of versions",
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "repository", Values: []string{"*"}}},
		},
		{
			Name:                 "expectedStatus",
			Label:                "Expected status (optional)",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "repository", Values: []string{"*"}}},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: PackageVersionStatusOptions},
			},
		},
		{
			Name:                 "namespace",
			Label:                "Namespace",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "repository", Values: []string{"*"}}},
		},
	}
}

func (c *DeletePackageVersions) Setup(ctx core.SetupContext) error {
	var config DeletePackageVersionsConfiguration
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

	if len(parseVersionsList(config.Versions)) == 0 {
		return fmt.Errorf("at least one version is required")
	}

	return nil
}

func (c *DeletePackageVersions) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeletePackageVersions) Execute(ctx core.ExecutionContext) error {
	var config DeletePackageVersionsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	resp, err := client.DeletePackageVersions(DeletePackageVersionsInput{
		Domain:         config.Domain,
		Repository:     config.Repository,
		Format:         config.Format,
		Namespace:      config.Namespace,
		Package:        config.Package,
		Versions:       parseVersionsList(config.Versions),
		ExpectedStatus: config.ExpectedStatus,
	})

	if err != nil {
		return fmt.Errorf("failed to delete package versions: %w", err)
	}

	data := map[string]any{
		"successfulVersions": resp.SuccessfulVersions,
		"failedVersions":     resp.FailedVersions,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.codeartifact.packageVersions",
		[]any{data},
	)
}

func (c *DeletePackageVersions) Actions() []core.Action {
	return nil
}

func (c *DeletePackageVersions) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeletePackageVersions) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeletePackageVersions) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeletePackageVersions) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeletePackageVersions) normalizeConfig(config DeletePackageVersionsConfiguration) DeletePackageVersionsConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Format = strings.ToLower(strings.TrimSpace(config.Format))
	config.Package = strings.TrimSpace(config.Package)
	config.Namespace = strings.TrimSpace(config.Namespace)
	config.Versions = strings.TrimSpace(config.Versions)
	config.ExpectedStatus = strings.TrimSpace(config.ExpectedStatus)
	return config
}
