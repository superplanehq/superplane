package groups

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	GroupKind = "Group"
)

type groupResource struct {
	APIVersion string                              `json:"apiVersion"`
	Kind       string                              `json:"kind"`
	Metadata   *openapi_client.GroupsGroupMetadata `json:"metadata,omitempty"`
	Spec       *openapi_client.GroupsGroupSpec     `json:"spec,omitempty"`
}

func parseGroupFile(path string) (*groupResource, error) {
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

	if kind != GroupKind {
		return nil, fmt.Errorf("unsupported resource kind %q", kind)
	}

	resource := groupResource{}
	if err := yaml.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse group resource: %w", err)
	}

	return &resource, nil
}

func resourceToGroup(resource groupResource) openapi_client.GroupsGroup {
	group := openapi_client.GroupsGroup{}
	if resource.Metadata != nil {
		group.SetMetadata(*resource.Metadata)
	}
	if resource.Spec != nil {
		group.SetSpec(*resource.Spec)
	}
	return group
}

func renderGroupListText(stdout io.Writer, items []openapi_client.GroupsGroup) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(stdout, "No groups found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "NAME\tDISPLAY_NAME\tROLE\tMEMBERS\tCREATED_AT")

	for _, item := range items {
		metadata := item.GetMetadata()
		spec := item.GetSpec()
		status := item.GetStatus()

		createdAt := ""
		if metadata.HasCreatedAt() {
			createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%s\n",
			metadata.GetName(),
			spec.GetDisplayName(),
			spec.GetRole(),
			status.GetMembersCount(),
			createdAt,
		)
	}

	return writer.Flush()
}

func renderGroupText(stdout io.Writer, group openapi_client.GroupsGroup) error {
	metadata := group.GetMetadata()
	spec := group.GetSpec()
	status := group.GetStatus()

	_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
	_, _ = fmt.Fprintf(stdout, "Display Name: %s\n", spec.GetDisplayName())
	_, _ = fmt.Fprintf(stdout, "Role: %s\n", spec.GetRole())
	if description := strings.TrimSpace(spec.GetDescription()); description != "" {
		_, _ = fmt.Fprintf(stdout, "Description: %s\n", description)
	}
	_, _ = fmt.Fprintf(stdout, "Members: %d\n", status.GetMembersCount())
	if metadata.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", metadata.GetCreatedAt().Format(time.RFC3339))
	}
	if metadata.HasUpdatedAt() {
		_, _ = fmt.Fprintf(stdout, "Updated At: %s\n", metadata.GetUpdatedAt().Format(time.RFC3339))
	}
	return nil
}

func renderGroupUsersText(stdout io.Writer, users []openapi_client.SuperplaneUsersUser) error {
	if len(users) == 0 {
		_, err := fmt.Fprintln(stdout, "No members found.")
		return err
	}

	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tEMAIL\tNAME\tCREATED_AT")

	for _, user := range users {
		metadata := user.GetMetadata()
		spec := user.GetSpec()

		createdAt := ""
		if metadata.HasCreatedAt() {
			createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			metadata.GetId(),
			metadata.GetEmail(),
			spec.GetDisplayName(),
			createdAt,
		)
	}

	return writer.Flush()
}
