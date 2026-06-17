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

type GetPackageStatus struct{}

type GetPackageStatusSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

func (g *GetPackageStatus) Name() string {
	return "cloudsmith.getPackageStatus"
}

func (g *GetPackageStatus) Label() string {
	return "Get Package Status"
}

func (g *GetPackageStatus) Description() string {
	return "Retrieve the current status of a Cloudsmith package"
}

func (g *GetPackageStatus) Documentation() string {
	return `The Get Package Status component retrieves the current processing status of a Cloudsmith package.

## Use Cases

- **Release gating**: Check that a package is Available before triggering downstream deployment steps
- **Polling workflows**: Repeatedly check status until a package leaves the Processing state
- **Quarantine detection**: Alert teams when a package enters a Quarantined or Failed state
- **Compliance checks**: Verify that a package has passed all scans before promotion

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use ` + "`{{ $['On Package Uploaded'].package.slug_perm }}`" + ` to reference an upstream trigger.

## Output

Returns a status snapshot containing:
- **slug**: The package identifier
- **name** / **version**: Human-readable package name and version
- **status** / **status_str**: Numeric status code and its string representation (Available, Processing, Failed, Quarantined, Awaiting Scanning)
- **repository** / **namespace**: Repository and namespace the package belongs to
- **uploaded_at**: When the package was uploaded`
}

func (g *GetPackageStatus) Icon() string {
	return "activity"
}

func (g *GetPackageStatus) Color() string {
	return "gray"
}

func (g *GetPackageStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetPackageStatus) Configuration() []configuration.Field {
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
			Description: "The package to check status for",
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

func (g *GetPackageStatus) Setup(ctx core.SetupContext) error {
	spec := GetPackageStatusSpec{}
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

func (g *GetPackageStatus) Execute(ctx core.ExecutionContext) error {
	spec := GetPackageStatusSpec{}
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

	info, err := client.GetPackageStatusInfo(owner, repo, spec.Package)
	if err != nil {
		return fmt.Errorf("failed to get package status: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.status",
		[]any{info},
	)
}

func (g *GetPackageStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPackageStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPackageStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetPackageStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetPackageStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetPackageStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
