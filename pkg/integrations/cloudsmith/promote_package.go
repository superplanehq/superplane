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

const (
	PromoteModeMove = "move"
	PromoteModeCopy = "copy"
)

type PromotePackage struct{}

type PromotePackageSpec struct {
	SourceRepository      string `json:"sourceRepository" mapstructure:"sourceRepository"`
	Package               string `json:"package" mapstructure:"package"`
	DestinationRepository string `json:"destinationRepository" mapstructure:"destinationRepository"`
	Mode                  string `json:"mode" mapstructure:"mode"`
}

func (p *PromotePackage) Name() string {
	return "cloudsmith.promotePackage"
}

func (p *PromotePackage) Label() string {
	return "Promote Package"
}

func (p *PromotePackage) Description() string {
	return "Copy or move a package from one Cloudsmith repository to another"
}

func (p *PromotePackage) Documentation() string {
	return `The Promote Package component copies or moves a package from a source repository to a destination repository within the same Cloudsmith namespace.

## Use Cases

- **Promotion pipelines**: Move a package from staging to production after all checks pass
- **Multi-environment distribution**: Copy a package to multiple target repositories simultaneously
- **Artifact archiving**: Move packages from active repositories to archive repositories
- **Release management**: Promote a vetted version from a dev channel to a release channel

## Configuration

- **Source Repository** (required): The repository that currently holds the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use ` + "`{{ $['On Package Created'].data.slug_perm }}`" + ` to reference an upstream trigger.
- **Destination Repository** (required): The target repository to promote the package into, in the form ` + "`owner/repository`" + `.
- **Mode** (required): Whether to ` + "`copy`" + ` (keep the original) or ` + "`move`" + ` (remove from source).

## Output

Returns the promoted package as it appears in the destination repository, including:
- **name** / **version**: Package name and version
- **format**: Package format (e.g., ` + "`docker`" + `, ` + "`python`" + `, ` + "`debian`" + `)
- **repository** / **namespace**: Where the package now lives
- **self_webapp_url**: URL to the promoted package in the Cloudsmith web app
- **slug_perm**: Permanent identifier of the package in the destination
- **uploaded_at**: Original upload timestamp`
}

func (p *PromotePackage) Icon() string {
	return "copy"
}

func (p *PromotePackage) Color() string {
	return "blue"
}

func (p *PromotePackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (p *PromotePackage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sourceRepository",
			Label:       "Source Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository currently holding the package",
			Placeholder: "Select source repository",
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
			Description: "The package to promote",
			Placeholder: "Select package",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "package",
					UseNameAsValue: false,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "repository",
							ValueFrom: &configuration.ParameterValueFrom{Field: "sourceRepository"},
						},
					},
				},
			},
		},
		{
			Name:        "destinationRepository",
			Label:       "Destination Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to promote the package into",
			Placeholder: "Select destination repository",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Copy keeps the original; Move removes it from the source",
			Default:     PromoteModeCopy,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Copy", Value: PromoteModeCopy, Description: "Copy the package; the original remains in the source repository"},
						{Label: "Move", Value: PromoteModeMove, Description: "Move the package; it is removed from the source repository"},
					},
				},
			},
		},
	}
}

func (p *PromotePackage) Setup(ctx core.SetupContext) error {
	spec := PromotePackageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.SourceRepository == "" {
		return errors.New("sourceRepository is required")
	}

	if spec.Package == "" {
		return errors.New("package is required")
	}

	if spec.DestinationRepository == "" {
		return errors.New("destinationRepository is required")
	}

	if spec.Mode != PromoteModeCopy && spec.Mode != PromoteModeMove {
		return fmt.Errorf("mode must be %q or %q, got %q", PromoteModeCopy, PromoteModeMove, spec.Mode)
	}

	return resolvePackageMetadata(ctx, spec.SourceRepository, spec.Package)
}

func (p *PromotePackage) Execute(ctx core.ExecutionContext) error {
	spec := PromotePackageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	owner, sourceRepo, err := parseRepositoryID(spec.SourceRepository)
	if err != nil {
		return fmt.Errorf("invalid sourceRepository %q: %w", spec.SourceRepository, err)
	}

	destOwner, destRepo, err := parseRepositoryID(spec.DestinationRepository)
	if err != nil {
		return fmt.Errorf("invalid destinationRepository %q: %w", spec.DestinationRepository, err)
	}

	if destOwner != owner {
		return fmt.Errorf("cross-namespace promotion is not supported: source owner %q and destination owner %q must match", owner, destOwner)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var pkg *Package
	switch spec.Mode {
	case PromoteModeMove:
		pkg, err = client.MovePackage(owner, sourceRepo, spec.Package, destRepo)
		if err != nil {
			return fmt.Errorf("failed to move package: %v", err)
		}
	default:
		pkg, err = client.CopyPackage(owner, sourceRepo, spec.Package, destRepo)
		if err != nil {
			return fmt.Errorf("failed to copy package: %v", err)
		}
	}

	if pkg == nil {
		return errors.New("promote returned empty response")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.package.promoted",
		[]any{pkg},
	)
}

func (p *PromotePackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (p *PromotePackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (p *PromotePackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (p *PromotePackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (p *PromotePackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *PromotePackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
