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

const (
	targetStatusArchived  = "Archived"
	targetStatusPublished = "Published"
	targetStatusUnlisted  = "Unlisted"
)

var updatePackageVersionsStatusTargetOptions = []configuration.FieldOption{
	{Value: targetStatusArchived, Label: "Archived"},
	{Value: targetStatusPublished, Label: "Published"},
	{Value: targetStatusUnlisted, Label: "Unlisted"},
}

type UpdatePackageVersionsStatus struct{}

type UpdatePackageVersionsStatusConfiguration struct {
	Region         string `json:"region" mapstructure:"region"`
	Domain         string `json:"domain" mapstructure:"domain"`
	Repository     string `json:"repository" mapstructure:"repository"`
	Format         string `json:"format" mapstructure:"format"`
	Package        string `json:"package" mapstructure:"package"`
	Namespace      string `json:"namespace" mapstructure:"namespace"`
	Versions       string `json:"versions" mapstructure:"versions"` // comma separated
	TargetStatus   string `json:"targetStatus" mapstructure:"targetStatus"`
	ExpectedStatus string `json:"expectedStatus" mapstructure:"expectedStatus"`
}

func (c *UpdatePackageVersionsStatus) Name() string {
	return "aws.codeArtifact.updatePackageVersionsStatus"
}

func (c *UpdatePackageVersionsStatus) Label() string {
	return "CodeArtifact • Update Package Versions Status"
}

func (c *UpdatePackageVersionsStatus) Description() string {
	return "Update the status of one or more package versions (Archived, Published, Unlisted)"
}

func (c *UpdatePackageVersionsStatus) Documentation() string {
	return `The Update Package Versions Status component sets the status of package versions to Archived, Published, or Unlisted.

## Use Cases

- **Lifecycle management**: Archive old versions or publish after validation
- **Visibility**: Unlist versions without deleting them
- **Compliance**: Align version status with release policies
`
}

func (c *UpdatePackageVersionsStatus) Icon() string {
	return "aws"
}

func (c *UpdatePackageVersionsStatus) Color() string {
	return "gray"
}

func (c *UpdatePackageVersionsStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdatePackageVersionsStatus) Configuration() []configuration.Field {
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
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:        "package",
			Label:       "Package name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. lodash (npm), my-package (pypi)",
			Description: "Name of the package in the repository",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:        "versions",
			Label:       "Versions",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "1.0.0, 1.0.1",
			Description: "Comma separated list of versions",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
		{
			Name:     "targetStatus",
			Label:    "Target status",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: updatePackageVersionsStatusTargetOptions,
				},
			},
		},
		{
			Name:     "expectedStatus",
			Label:    "Expected status (optional)",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: PackageVersionStatusOptions},
			},
		},
		{
			Name:     "namespace",
			Label:    "Namespace",
			Type:     configuration.FieldTypeString,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "repository", Values: []string{"*"}},
			},
		},
	}
}

func (c *UpdatePackageVersionsStatus) Setup(ctx core.SetupContext) error {
	var config UpdatePackageVersionsStatusConfiguration
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

	versions := parseVersionsList(config.Versions)
	if len(versions) == 0 {
		return fmt.Errorf("at least one version is required")
	}

	if config.TargetStatus != targetStatusArchived && config.TargetStatus != targetStatusPublished && config.TargetStatus != targetStatusUnlisted {
		return fmt.Errorf("targetStatus must be Archived, Published, or Unlisted")
	}

	return nil
}

func (c *UpdatePackageVersionsStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdatePackageVersionsStatus) Execute(ctx core.ExecutionContext) error {
	var config UpdatePackageVersionsStatusConfiguration
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
	resp, err := client.UpdatePackageVersionsStatus(UpdatePackageVersionsStatusInput{
		Domain:         config.Domain,
		Repository:     config.Repository,
		Format:         config.Format,
		Namespace:      config.Namespace,
		Package:        config.Package,
		Versions:       versions,
		TargetStatus:   config.TargetStatus,
		ExpectedStatus: config.ExpectedStatus,
	})

	if err != nil {
		return fmt.Errorf("failed to update package versions status: %w", err)
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

func (c *UpdatePackageVersionsStatus) Actions() []core.Action {
	return nil
}

func (c *UpdatePackageVersionsStatus) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdatePackageVersionsStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdatePackageVersionsStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdatePackageVersionsStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdatePackageVersionsStatus) normalizeConfig(config UpdatePackageVersionsStatusConfiguration) UpdatePackageVersionsStatusConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.Domain = strings.TrimSpace(config.Domain)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Format = strings.ToLower(strings.TrimSpace(config.Format))
	config.Package = strings.TrimSpace(config.Package)
	config.Namespace = strings.TrimSpace(config.Namespace)
	config.Versions = strings.TrimSpace(config.Versions)
	config.TargetStatus = strings.TrimSpace(config.TargetStatus)
	config.ExpectedStatus = strings.TrimSpace(config.ExpectedStatus)
	return config
}
