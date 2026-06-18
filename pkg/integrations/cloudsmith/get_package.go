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

type GetPackage struct{}

type GetPackageSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

func (g *GetPackage) Name() string {
	return "cloudsmith.getPackage"
}

func (g *GetPackage) Label() string {
	return "Get Package"
}

func (g *GetPackage) Description() string {
	return "Retrieve metadata for a specific Cloudsmith package"
}

func (g *GetPackage) Documentation() string {
	return `The Get Package component retrieves complete metadata for a specific Cloudsmith package,
including sync status, quarantine state, and security scan results.

## Use Cases

- **Release gating**: Check that a package is Available and sync-completed before triggering downstream deployment steps
- **Quarantine detection**: Detect when a package has been quarantined or has policy violations
- **Audit trails**: Record full package metadata (checksums, format, upload time) for compliance
- **Downstream enrichment**: Pass package details such as format or CDN URL to later workflow steps
- **Checksum verification**: Retrieve SHA-256 or MD5 checksums to validate package integrity
- **Security insights**: Check the security scan status and link to full vulnerability results

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use ` + "`{{ $['On Package Uploaded'].package.slug_perm }}`" + ` to reference an upstream trigger.

## Output

Returns the complete package object including:
- **name** / **version**: Package name and version string
- **format**: Package format (e.g., ` + "`python`" + `, ` + "`debian`" + `, ` + "`docker`" + `, ` + "`maven`" + `)
- **status** / **status_str**: Overall status code and label (e.g. ` + "`Available`" + `, ` + "`Quarantined`" + `, ` + "`Failed`" + `)
- **stage** / **stage_str**: Processing stage (e.g. ` + "`Fully Synchronised`" + `)
- **sync_progress**: Sync completion percentage (0–100)
- **is_sync_completed** / **is_sync_failed**: Final sync outcome flags
- **is_quarantined**: Whether the package has been quarantined
- **security_scan_status**: Result of the most recent security scan
- **vulnerability_scan_results_url**: URL to full vulnerability scan results
- **checksum_md5** / **checksum_sha1** / **checksum_sha256** / **checksum_sha512**: Package checksums
- **size** / **size_str**: Package size in bytes and human-readable form
- **cdn_url** / **self_html_url**: Download and web UI URLs
- **uploaded_at**: ISO 8601 upload timestamp`
}

func (g *GetPackage) Icon() string {
	return "info"
}

func (g *GetPackage) Color() string {
	return "gray"
}

func (g *GetPackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetPackage) Configuration() []configuration.Field {
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
			Description: "The package to fetch",
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

func (g *GetPackage) Setup(ctx core.SetupContext) error {
	spec := GetPackageSpec{}
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

func (g *GetPackage) Execute(ctx core.ExecutionContext) error {
	spec := GetPackageSpec{}
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

	pkg, err := client.GetPackage(owner, repo, spec.Package)
	if err != nil {
		return fmt.Errorf("failed to get package: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.fetched",
		[]any{pkg},
	)
}

func (g *GetPackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetPackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetPackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetPackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
