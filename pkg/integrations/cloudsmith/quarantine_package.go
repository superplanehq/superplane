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
	QuarantineActionQuarantine = "Quarantine"
	QuarantineActionRelease    = "Release"
)

type QuarantinePackage struct{}

type QuarantinePackageSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
	Action     string `json:"action" mapstructure:"action"`
}

func (q *QuarantinePackage) Name() string {
	return "cloudsmith.quarantinePackage"
}

func (q *QuarantinePackage) Label() string {
	return "Quarantine Package"
}

func (q *QuarantinePackage) Description() string {
	return "Quarantine or release a Cloudsmith package"
}

func (q *QuarantinePackage) Documentation() string {
	return `The Quarantine Package component quarantines or releases a Cloudsmith package.

## Actions

- **Quarantine**: Makes the package unavailable for download while keeping it in the repository.
  Useful when a package has known vulnerabilities or should not be consumed.
- **Release**: Removes the quarantine on a previously quarantined package, making it available
  for download again once any remediation steps are complete.

## Use Cases

- **Vulnerability response**: Automatically quarantine packages when critical vulnerabilities are detected
- **Policy enforcement**: Quarantine packages that violate license or security policies
- **Incident response**: Immediately isolate a compromised package version
- **Remediation workflow**: Release a package after confirming vulnerabilities have been addressed

## Configuration

- **Repository** (required): The repository containing the package, in the form ` + "`owner/repository`" + `.
- **Package** (required): The unique package identifier (` + "`slug_perm`" + `). Supports expressions — use
  ` + "`{{ $['On Vulnerability Scan Completed'].data.slug_perm }}`" + ` to reference an upstream trigger.
- **Action** (required): Whether to quarantine or release the package.

## Output

Returns the updated Cloudsmith package object reflecting the new quarantine state, including:
- **name** / **version**: Package identifiers
- **status** / **status_str**: Updated status — ` + "`Quarantined`" + ` after quarantine, or the previous
  status (e.g. ` + "`Available`" + `) after release
- **format**: Package format
- **cdn_url** / **self_html_url**: Download and web UI URLs`
}

func (q *QuarantinePackage) Icon() string {
	return "shield-off"
}

func (q *QuarantinePackage) Color() string {
	return "gray"
}

func (q *QuarantinePackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (q *QuarantinePackage) Configuration() []configuration.Field {
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
			Description: "The package to quarantine or release",
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
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     QuarantineActionQuarantine,
			Description: "Whether to quarantine or release the package",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Value:       QuarantineActionQuarantine,
							Label:       "Quarantine",
							Description: "Make the package unavailable for download",
						},
						{
							Value:       QuarantineActionRelease,
							Label:       "Release",
							Description: "Remove quarantine and make the package available again",
						},
					},
				},
			},
		},
	}
}

func (q *QuarantinePackage) Setup(ctx core.SetupContext) error {
	spec := QuarantinePackageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	if spec.Package == "" {
		return errors.New("package is required")
	}

	// An empty action falls back to the field default (Quarantine), since field
	// defaults are not merged into saved configuration server-side. Only reject
	// an explicitly set value that is neither valid option.
	if spec.Action != "" && spec.Action != QuarantineActionQuarantine && spec.Action != QuarantineActionRelease {
		return fmt.Errorf("action must be %q or %q", QuarantineActionQuarantine, QuarantineActionRelease)
	}

	return resolvePackageMetadata(ctx, spec.Repository, spec.Package)
}

func (q *QuarantinePackage) Execute(ctx core.ExecutionContext) error {
	spec := QuarantinePackageSpec{}
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

	release := spec.Action == QuarantineActionRelease
	pkg, err := client.QuarantinePackage(owner, repo, spec.Package, release)
	if err != nil {
		if release {
			return fmt.Errorf("failed to release package: %v", err)
		}
		return fmt.Errorf("failed to quarantine package: %v", err)
	}

	eventType := "cloudsmith.package.quarantined"
	if release {
		eventType = "cloudsmith.package.released"
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		eventType,
		[]any{pkg},
	)
}

func (q *QuarantinePackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (q *QuarantinePackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (q *QuarantinePackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (q *QuarantinePackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (q *QuarantinePackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (q *QuarantinePackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
