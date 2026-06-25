package cloudsmith

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ScanPackage struct{}

type ScanPackageSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

type ScanPackageResult struct {
	Repository string `json:"repository"`
	Package    string `json:"package"`
	Name       string `json:"name"`
}

func (s *ScanPackage) Name() string {
	return "cloudsmith.scanPackage"
}

func (s *ScanPackage) Label() string {
	return "Scan Package"
}

func (s *ScanPackage) Description() string {
	return "Schedule a vulnerability scan for a specific Cloudsmith package"
}

func (s *ScanPackage) Documentation() string {
	return `The Scan Package component schedules a vulnerability scan for a specific Cloudsmith package.

## Use Cases

- **On-demand scanning**: Trigger a fresh scan after a package is uploaded or updated
- **Pre-promotion checks**: Scan a package before promoting it to a production repository
- **Scheduled audits**: Periodically re-scan packages to catch newly published CVEs

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions.

## Output

Emits the repository and package identifiers confirming the scan was scheduled. Scan results
are asynchronous — use the **On Vulnerability Scan Completed** trigger or the
**Get Package Vulnerabilities** action to retrieve results once the scan finishes.`
}

func (s *ScanPackage) Icon() string {
	return "scan"
}

func (s *ScanPackage) Color() string {
	return "gray"
}

func (s *ScanPackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (s *ScanPackage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository containing the package",
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
			Placeholder: "Select package",
			Description: "The package to scan",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "package",
					UseNameAsValue: false,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "repository",
							ValueFrom: &configuration.ParameterValueFrom{Field: "repository"},
						},
					},
				},
			},
		},
	}
}

func (s *ScanPackage) Setup(ctx core.SetupContext) error {
	spec := ScanPackageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	if spec.Package == "" {
		return errors.New("package is required")
	}

	return resolvePackageMetadata(ctx, spec.Repository, spec.Package)
}

func (s *ScanPackage) Execute(ctx core.ExecutionContext) error {
	spec := ScanPackageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	owner, repo, err := parseRepositoryID(spec.Repository)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", spec.Repository, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.ScanPackage(owner, repo, spec.Package); err != nil {
		return fmt.Errorf("failed to schedule scan: %v", err)
	}

	pkg, err := client.GetPackage(owner, repo, spec.Package)
	if err != nil {
		return fmt.Errorf("failed to get package details: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.scan_scheduled",
		[]any{ScanPackageResult{
			Repository: spec.Repository,
			Package:    spec.Package,
			Name:       pkg.Name,
		}},
	)
}

func (s *ScanPackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (s *ScanPackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *ScanPackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (s *ScanPackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (s *ScanPackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (s *ScanPackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
