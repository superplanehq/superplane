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
	return `The Get Package component retrieves complete metadata for a specific Cloudsmith package.

## Use Cases

- **Audit trails**: Record full package metadata (checksums, format, upload time) for compliance
- **Downstream enrichment**: Pass package details such as format or CDN URL to later workflow steps
- **Checksum verification**: Retrieve SHA-256 or MD5 checksums to validate package integrity
- **Package promotion**: Read source package attributes before replicating to another repository

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use ` + "`{{ $['On Package Uploaded'].package.slug_perm }}`" + ` to reference an upstream trigger.

## Output

Returns the complete package object including:
- **name** / **version**: Package name and version string
- **format**: Package format (e.g., ` + "`python`" + `, ` + "`debian`" + `, ` + "`docker`" + `, ` + "`maven`" + `)
- **status** / **status_str**: Processing status
- **uploaded_at**: ISO 8601 upload timestamp
- **checksum_md5** / **checksum_sha1** / **checksum_sha256** / **checksum_sha512**: Package checksums
- **size** / **size_str**: Package size in bytes and human-readable form
- **cdn_url**: Direct CDN download URL
- **self_html_url**: Link to the package page in the Cloudsmith web UI`
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
