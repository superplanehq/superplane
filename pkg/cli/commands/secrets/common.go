package secrets

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	SecretKind = "Secret"
)

type secretResource struct {
	APIVersion string                                `json:"apiVersion"`
	Kind       string                                `json:"kind"`
	Metadata   *openapi_client.SecretsSecretMetadata `json:"metadata,omitempty"`
	Spec       *openapi_client.SecretsSecretSpec     `json:"spec,omitempty"`
}

func resolveOrganizationID(ctx core.CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	if !me.HasOrganizationId() || strings.TrimSpace(me.GetOrganizationId()) == "" {
		return "", fmt.Errorf("organization id not found for authenticated user")
	}

	return me.GetOrganizationId(), nil
}

func organizationDomainType() openapi_client.AuthorizationDomainType {
	return openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION
}

func parseSecretFile(path string) (*secretResource, error) {
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

	if kind != SecretKind {
		return nil, fmt.Errorf("unsupported resource kind %q", kind)
	}

	resource := secretResource{}
	if err := yaml.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse secret resource: %w", err)
	}

	return &resource, nil
}

func resourceToSecret(resource secretResource) openapi_client.SecretsSecret {
	secret := openapi_client.SecretsSecret{}
	if resource.Metadata != nil {
		secret.SetMetadata(*resource.Metadata)
	}
	if resource.Spec != nil {
		secret.SetSpec(*resource.Spec)
	}
	return secret
}

func renderSecretListText(stdout io.Writer, items []openapi_client.SecretsSecret) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tNAME\tPROVIDER\tKEYS\tCREATED_AT")

	for _, item := range items {
		metadata := item.GetMetadata()
		spec := item.GetSpec()

		keyCount := 0
		if local, ok := spec.GetLocalOk(); ok && local.HasData() {
			keyCount = len(local.GetData())
		}

		createdAt := ""
		if metadata.HasCreatedAt() {
			createdAt = metadata.GetCreatedAt().Format(time.RFC3339)
		}

		_, _ = fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%s\n",
			metadata.GetId(),
			metadata.GetName(),
			spec.GetProvider(),
			keyCount,
			createdAt,
		)
	}

	return writer.Flush()
}

func renderSecretText(stdout io.Writer, item openapi_client.SecretsSecret) error {
	metadata := item.GetMetadata()
	spec := item.GetSpec()

	_, _ = fmt.Fprintf(stdout, "ID: %s\n", metadata.GetId())
	_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
	_, _ = fmt.Fprintf(stdout, "Provider: %s\n", spec.GetProvider())
	_, _ = fmt.Fprintf(stdout, "DomainType: %s\n", metadata.GetDomainType())
	_, _ = fmt.Fprintf(stdout, "DomainID: %s\n", metadata.GetDomainId())
	if metadata.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "CreatedAt: %s\n", metadata.GetCreatedAt().Format(time.RFC3339))
	}

	_, _ = fmt.Fprintln(stdout, "Keys:")
	keys := make([]string, 0)
	if local, ok := spec.GetLocalOk(); ok && local.HasData() {
		for key := range local.GetData() {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(stdout, "- %s\n", key)
	}

	return nil
}
