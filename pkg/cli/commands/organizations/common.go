package organizations

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

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

func renderOrganization(stdout io.Writer, org openapi_client.OrganizationsOrganization) error {
	metadata := org.GetMetadata()

	_, _ = fmt.Fprintf(stdout, "ID: %s\n", metadata.GetId())
	_, _ = fmt.Fprintf(stdout, "Name: %s\n", metadata.GetName())
	_, _ = fmt.Fprintf(stdout, "Description: %s\n", metadata.GetDescription())
	_, _ = fmt.Fprintf(stdout, "Versioning Enabled: %t\n", metadata.GetVersioningEnabled())
	if metadata.HasCreatedAt() {
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", metadata.GetCreatedAt().Format(time.RFC3339))
	}
	if metadata.HasUpdatedAt() {
		_, _ = fmt.Fprintf(stdout, "Updated At: %s\n", metadata.GetUpdatedAt().Format(time.RFC3339))
	}

	return nil
}
