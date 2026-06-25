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

type GetPackageVulnerabilities struct{}

type GetPackageVulnerabilitiesSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

func (g *GetPackageVulnerabilities) Name() string {
	return "cloudsmith.getPackageVulnerabilities"
}

func (g *GetPackageVulnerabilities) Label() string {
	return "Get Package Vulnerabilities"
}

func (g *GetPackageVulnerabilities) Description() string {
	return "Retrieve the latest vulnerability scan result for a specific Cloudsmith package"
}

func (g *GetPackageVulnerabilities) Documentation() string {
	return `The Get Package Vulnerabilities component retrieves the most recent vulnerability scan
result for a specific Cloudsmith package.

## Use Cases

- **Release gating**: Block promotion of packages that have known vulnerabilities
- **Audit reporting**: Record vulnerability counts and max severity for compliance dashboards
- **Automated response**: Route critical findings to an on-call team or trigger quarantine
- **Scan verification**: Confirm a scan has completed and check its outcome after **Scan Package**

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use
  ` + "`{{ $['On Vulnerability Scan Completed'].data.slug_perm }}`" + ` to reference an upstream trigger.

## Output

Returns the most recent scan result for the package:
- **identifier**: Unique ID for this scan result entry
- **created_at**: When the scan was completed (ISO 8601)
- **has_vulnerabilities**: Whether any vulnerabilities were detected
- **num_vulnerabilities**: Total number of vulnerabilities found
- **max_severity**: Highest severity level detected (e.g. ` + "`Critical`" + `, ` + "`High`" + `, ` + "`Medium`" + `, ` + "`Low`" + `)
- **scan_id**: Internal scan identifier (may be ` + "`null`" + `)
- **package**: Nested object with the scanned package details:
  - **identifier**: Package ` + "`slug_perm`" + `
  - **name**: Package name
  - **version**: Package version or digest
  - **url**: Direct API URL for the package`
}

func (g *GetPackageVulnerabilities) Icon() string {
	return "shield-alert"
}

func (g *GetPackageVulnerabilities) Color() string {
	return "gray"
}

func (g *GetPackageVulnerabilities) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetPackageVulnerabilities) Configuration() []configuration.Field {
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
			Description: "The package to retrieve vulnerability results for",
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

func (g *GetPackageVulnerabilities) Setup(ctx core.SetupContext) error {
	spec := GetPackageVulnerabilitiesSpec{}
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

func (g *GetPackageVulnerabilities) Execute(ctx core.ExecutionContext) error {
	spec := GetPackageVulnerabilitiesSpec{}
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

	results, err := client.GetPackageVulnerabilities(owner, repo, spec.Package)
	if err != nil {
		return fmt.Errorf("failed to get package vulnerabilities: %v", err)
	}

	if len(results) == 0 {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"cloudsmith.package.vulnerabilities",
			[]any{VulnerabilityScanResult{}},
		)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.vulnerabilities",
		[]any{results[0]},
	)
}

func (g *GetPackageVulnerabilities) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPackageVulnerabilities) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPackageVulnerabilities) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetPackageVulnerabilities) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetPackageVulnerabilities) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetPackageVulnerabilities) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
