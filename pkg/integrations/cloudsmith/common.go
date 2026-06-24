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

const (
	PackageResyncedPayloadType = "cloudsmith.package.resynced"
	PackageTaggedPayloadType   = "cloudsmith.package.tagged"
	PackageDeletedPayloadType  = "cloudsmith.package.deleted"
)

type PackageSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Package    string `json:"package" mapstructure:"package"`
}

type PackageResult struct {
	Repository string   `json:"repository"`
	Package    string   `json:"package"`
	Data       *Package `json:"data,omitempty"`
}

func packageConfigurationFields(extra ...configuration.Field) []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository that contains the package",
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
			Description: "The package to operate on",
			Placeholder: "Select package",
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

	return append(fields, extra...)
}

func decodePackageSpec(configuration any) (PackageSpec, error) {
	spec := PackageSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return PackageSpec{}, fmt.Errorf("error decoding configuration: %v", err)
	}

	spec.Repository = strings.TrimSpace(spec.Repository)
	if spec.Repository == "" {
		return PackageSpec{}, errors.New("repository is required")
	}

	spec.Package = strings.TrimSpace(spec.Package)
	if spec.Package == "" {
		return PackageSpec{}, errors.New("package is required")
	}

	return spec, nil
}

func setupPackageComponent(ctx core.SetupContext) error {
	spec, err := decodePackageSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return resolvePackageMetadata(ctx, spec.Repository, spec.Package)
}

func packageRequestParts(spec PackageSpec) (owner string, repository string, identifier string, err error) {
	owner, repository, err = parseRepositoryID(spec.Repository)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid repository %q: %w", spec.Repository, err)
	}

	return owner, repository, spec.Package, nil
}

func packageResult(spec PackageSpec, pkg *Package) PackageResult {
	return PackageResult{
		Repository: spec.Repository,
		Package:    spec.Package,
		Data:       pkg,
	}
}

func defaultProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func defaultHandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
