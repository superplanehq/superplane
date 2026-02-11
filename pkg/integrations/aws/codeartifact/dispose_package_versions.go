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

type DisposePackageVersions struct{}

type DisposePackageVersionsConfiguration struct {
	Region         string `json:"region" mapstructure:"region"`
	Domain         string `json:"domain" mapstructure:"domain"`
	Repository     string `json:"repository" mapstructure:"repository"`
	Format         string `json:"format" mapstructure:"format"`
	Package        string `json:"package" mapstructure:"package"`
	Namespace      string `json:"namespace" mapstructure:"namespace"`
	Versions       string `json:"versions" mapstructure:"versions"`
	ExpectedStatus string `json:"expectedStatus" mapstructure:"expectedStatus"`
}

func (c *DisposePackageVersions) Name() string {
	return "aws.codeArtifact.disposePackageVersions"
}

func (c *DisposePackageVersions) Label() string {
	return "CodeArtifact • Dispose Package Versions"
}

func (c *DisposePackageVersions) Description() string {
	return "Delete assets and set package version status to Disposed (record remains)"
}

func (c *DisposePackageVersions) Documentation() string {
	return `The Dispose Package Versions component deletes the assets of package versions and sets their status to Disposed. The version record remains so you can still see it in ListPackageVersions with status Disposed; assets cannot be restored.

## Use Cases

- **Retention**: Keep version metadata for audit while removing binary assets
- **Storage**: Free asset storage while preserving version history
- **Lifecycle**: Mark versions as disposed after a retention period
`
}

func (c *DisposePackageVersions) Icon() string {
	return "aws"
}

func (c *DisposePackageVersions) Color() string {
	return "gray"
}

func (c *DisposePackageVersions) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DisposePackageVersions) Configuration() []configuration.Field {
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
			Description:          "Name of the package whose versions to dispose",
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

func (c *DisposePackageVersions) Setup(ctx core.SetupContext) error {
	var config DisposePackageVersionsConfiguration
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

func (c *DisposePackageVersions) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DisposePackageVersions) Execute(ctx core.ExecutionContext) error {
	var config DisposePackageVersionsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config = c.normalizeConfig(config)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	resp, err := client.DisposePackageVersions(DisposePackageVersionsInput{
		Domain:         config.Domain,
		Repository:     config.Repository,
		Format:         config.Format,
		Namespace:      config.Namespace,
		Package:        config.Package,
		Versions:       parseVersionsList(config.Versions),
		ExpectedStatus: config.ExpectedStatus,
	})

	if err != nil {
		return fmt.Errorf("failed to dispose package versions: %w", err)
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

func (c *DisposePackageVersions) Actions() []core.Action                    { return nil }
func (c *DisposePackageVersions) HandleAction(ctx core.ActionContext) error { return nil }
func (c *DisposePackageVersions) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *DisposePackageVersions) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *DisposePackageVersions) Cleanup(ctx core.SetupContext) error    { return nil }

func (c *DisposePackageVersions) normalizeConfig(config DisposePackageVersionsConfiguration) DisposePackageVersionsConfiguration {
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
