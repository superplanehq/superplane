package roles

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	RoleKind = "Role"
)

type roleResource struct {
	APIVersion string                            `json:"apiVersion"`
	Kind       string                            `json:"kind"`
	Metadata   *openapi_client.RolesRoleMetadata `json:"metadata,omitempty"`
	Spec       *openapi_client.RolesRoleSpec     `json:"spec,omitempty"`
}

func parseRoleFile(path string) (*roleResource, error) {
	// #nosec
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource file: %w", err)
	}

	apiVersion, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return nil, err
	}

	if apiVersion != core.APIVersion {
		return nil, fmt.Errorf("unsupported apiVersion %q", apiVersion)
	}

	if kind != RoleKind {
		return nil, fmt.Errorf("unsupported resource kind %q", kind)
	}

	resource := roleResource{}
	if err := core.NewDecoder(data).DecodeYAML(&resource); err != nil {
		return nil, fmt.Errorf("failed to parse role resource: %w", err)
	}

	return &resource, nil
}

func resourceToRole(resource roleResource) openapi_client.RolesRole {
	role := openapi_client.RolesRole{}
	if resource.Metadata != nil {
		role.SetMetadata(*resource.Metadata)
	}
	if resource.Spec != nil {
		role.SetSpec(*resource.Spec)
	}
	return role
}

func renderRoleListText(stdout io.Writer, items []openapi_client.RolesRole) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(stdout, "No roles found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "NAME\tDISPLAY_NAME\tPERMISSIONS\tINHERITS\tCREATED_AT")

	for _, item := range items {
		metadata := item.GetMetadata()
		spec := item.GetSpec()

		createdAt := ""
		if metadata.HasCreatedAt() {
			createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
		}

		inherits := "-"
		if spec.HasInheritedRole() {
			inherited := spec.GetInheritedRole()
			inheritedMetadata := inherited.GetMetadata()
			inherits = inheritedMetadata.GetName()
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%d\t%s\t%s\n",
			metadata.GetName(),
			spec.GetDisplayName(),
			len(spec.GetPermissions()),
			inherits,
			createdAt,
		)
	}

	return writer.Flush()
}

func renderRoleText(stdout io.Writer, role openapi_client.RolesRole) error {
	metadata := role.GetMetadata()
	spec := role.GetSpec()

	_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
	_, _ = fmt.Fprintf(stdout, "Display Name: %s\n", spec.GetDisplayName())
	if description := strings.TrimSpace(spec.GetDescription()); description != "" {
		_, _ = fmt.Fprintf(stdout, "Description: %s\n", description)
	}
	if metadata.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", metadata.GetCreatedAt().Format(time.RFC3339))
	}
	if spec.HasInheritedRole() {
		inherited := spec.GetInheritedRole()
		inheritedMetadata := inherited.GetMetadata()
		_, _ = fmt.Fprintf(stdout, "Inherits From: %s\n", inheritedMetadata.GetName())
	}

	permissions := spec.GetPermissions()
	_, _ = fmt.Fprintln(stdout, "Permissions:")
	if len(permissions) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none)")
	}
	for _, p := range permissions {
		_, _ = fmt.Fprintf(stdout, "- %s:%s\n", p.GetResource(), p.GetAction())
	}

	return nil
}
