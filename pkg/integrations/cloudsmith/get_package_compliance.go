package cloudsmith

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetPackageCompliance struct{}

type GetPackageComplianceSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

// PackageCompliance is the curated, license/policy/governance view of a package
// emitted by this component. Vulnerability data is intentionally excluded.
type PackageCompliance struct {
	Name           string              `json:"name"`
	Version        string              `json:"version"`
	SlugPerm       string              `json:"slug_perm"`
	Format         string              `json:"format"`
	License        string              `json:"license"`
	SPDXLicense    string              `json:"spdx_license"`
	OSIApproved    bool                `json:"osi_approved"`
	PolicyViolated bool                `json:"policy_violated"`
	IsQuarantined  bool                `json:"is_quarantined"`
	Status         string              `json:"status"`
	StatusReason   string              `json:"status_reason"`
	Stage          string              `json:"stage"`
	Tags           map[string][]string `json:"tags"`
	URL            string              `json:"url"`
}

// PackageComplianceNodeMetadata caches the package display name so the collapsed
// UI can show it without re-fetching on every render.
type PackageComplianceNodeMetadata struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	PackageID   string `json:"packageId" mapstructure:"packageId"`
	PackageName string `json:"packageName" mapstructure:"packageName"`
	Version     string `json:"version" mapstructure:"version"`
}

func (g *GetPackageCompliance) Name() string {
	return "cloudsmith.getPackageCompliance"
}

func (g *GetPackageCompliance) Label() string {
	return "Get Package Compliance"
}

func (g *GetPackageCompliance) Description() string {
	return "Fetch license, policy, and governance compliance details for a Cloudsmith package"
}

func (g *GetPackageCompliance) Documentation() string {
	return `The Get Package Compliance component retrieves the license, policy, and governance metadata for a specific Cloudsmith package — useful for gating promotions or downstream actions on a package's compliance state.

> Vulnerability and security-scan data are handled by a separate component and are not included here.

## Use Cases

- **Promotion gates**: Block promotion of packages that are quarantined or violate a policy
- **License governance**: Check a package's detected license (and whether it is OSI-approved) before consuming it
- **Audit**: Record the compliance state of a package as part of a workflow

## Configuration

- **Repository**: The repository the package belongs to, in the form ` + "`owner/repository`" + ` (required, supports expressions)
- **Package**: The package to inspect, identified by its permanent slug (required, supports expressions). The picker lists packages in the selected repository.

## Output

Returns the package's compliance details including:
- **name** / **version** / **slug_perm** / **format**: Package identity
- **license** / **spdx_license**: The detected license
- **osi_approved**: Whether the detected license is OSI-approved
- **policy_violated**: Whether the package violates a configured policy
- **is_quarantined**: Whether the package is currently quarantined
- **status** / **status_reason** / **stage**: Processing and compliance status
- **tags**: Package tags
- **url**: Link to the package in the Cloudsmith web app`
}

func (g *GetPackageCompliance) Icon() string {
	return "shield-check"
}

func (g *GetPackageCompliance) Color() string {
	return "blue"
}

func (g *GetPackageCompliance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetPackageCompliance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository the package belongs to",
			Placeholder: "Select repository",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "package",
			Label:       "Package",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The package to inspect",
			Placeholder: "Select package",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "package",
					UseNameAsValue: false,
					Parameters: []configuration.ParameterRef{
						{
							Name: "repository",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "repository",
							},
						},
					},
				},
			},
		},
	}
}

func (g *GetPackageCompliance) Setup(ctx core.SetupContext) error {
	spec := GetPackageComplianceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Repository) == "" {
		return errors.New("repository is required")
	}
	if strings.TrimSpace(spec.Package) == "" {
		return errors.New("package is required")
	}

	return resolvePackageComplianceMetadata(ctx, spec.Repository, spec.Package)
}

func (g *GetPackageCompliance) Execute(ctx core.ExecutionContext) error {
	spec := GetPackageComplianceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	owner, identifier, err := parseRepositoryID(spec.Repository)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", spec.Repository, err)
	}

	slugPerm := strings.TrimSpace(spec.Package)
	if slugPerm == "" {
		return errors.New("package is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	pkg, err := client.GetPackage(owner, identifier, slugPerm)
	if err != nil {
		return fmt.Errorf("failed to get package: %v", err)
	}

	compliance := PackageCompliance{
		Name:           pkg.Name,
		Version:        pkg.Version,
		SlugPerm:       pkg.SlugPerm,
		Format:         pkg.Format,
		License:        pkg.License,
		SPDXLicense:    pkg.SPDXLicense,
		OSIApproved:    pkg.OSIApproved,
		PolicyViolated: pkg.PolicyViolated,
		IsQuarantined:  pkg.IsQuarantined,
		Status:         pkg.Status,
		StatusReason:   pkg.StatusReason,
		Stage:          pkg.Stage,
		Tags:           pkg.Tags,
		URL:            pkg.SelfHTMLURL,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.complianceFetched",
		[]any{compliance},
	)
}

// resolvePackageComplianceMetadata stores display metadata for the selected
// package. Expressions are stored verbatim because they can only be resolved at
// execution time.
func resolvePackageComplianceMetadata(ctx core.SetupContext, repositoryID, packageID string) error {
	if strings.Contains(repositoryID, "{{") || strings.Contains(packageID, "{{") {
		return ctx.Metadata.Set(PackageComplianceNodeMetadata{
			Repository:  repositoryID,
			PackageID:   packageID,
			PackageName: packageID,
		})
	}

	owner, identifier, err := parseRepositoryID(repositoryID)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	var existing PackageComplianceNodeMetadata
	if decodeErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decodeErr == nil &&
		existing.PackageID == packageID && existing.Repository == repositoryID && existing.PackageName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pkg, err := client.GetPackage(owner, identifier, packageID)
	if err != nil {
		return fmt.Errorf("failed to fetch package %q: %w", packageID, err)
	}

	name := pkg.Name
	if name == "" {
		name = packageID
	}

	return ctx.Metadata.Set(PackageComplianceNodeMetadata{
		Repository:  repositoryID,
		PackageID:   packageID,
		PackageName: name,
		Version:     pkg.Version,
	})
}

func (g *GetPackageCompliance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPackageCompliance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPackageCompliance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetPackageCompliance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetPackageCompliance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetPackageCompliance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
