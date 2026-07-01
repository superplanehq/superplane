package core

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	APIVersion = "v1"
)

// OrganizationDomainType returns the authorization domain type used for all
// organization-scoped CLI requests. Packages should use this instead of
// referencing the openapi_client enum directly so the scoping rule lives in
// one place.
func OrganizationDomainType() openapi_client.AuthorizationDomainType {
	return openapi_client.AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_ORGANIZATION
}

func ParseYamlResourceHeaders(raw []byte) (string, string, error) {
	m := make(map[string]interface{})

	err := yaml.Unmarshal(raw, &m)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse resource; %s", err)
	}

	apiVersion, ok := m["apiVersion"].(string)
	if !ok {
		return "", "", fmt.Errorf("failed to parse resource's api version")
	}

	kind, ok := m["kind"].(string)
	if !ok {
		return "", "", fmt.Errorf("failed to parse resource's kind")
	}

	return apiVersion, kind, nil
}

func ParseIntegrationScopedName(name string) (string, string, bool) {
	integrationName, resourceName, hasDot := strings.Cut(name, ".")
	if !hasDot || integrationName == "" || resourceName == "" {
		return "", "", false
	}

	return integrationName, resourceName, true
}

func FindIntegrationDefinition(ctx CommandContext, name string) (openapi_client.IntegrationsIntegrationDefinition, error) {
	response, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(ctx.Context).Execute()
	if err != nil {
		return openapi_client.IntegrationsIntegrationDefinition{}, err
	}

	for _, integration := range response.GetIntegrations() {
		if integration.GetName() == name {
			return integration, nil
		}
	}

	return openapi_client.IntegrationsIntegrationDefinition{}, fmt.Errorf("integration %q not found", name)
}

func ResolveOrganizationID(ctx CommandContext) (string, error) {
	me, _, err := ctx.API.MeAPI.MeMe(ctx.Context).Execute()
	if err != nil {
		return "", err
	}

	if !me.User.HasOrganizationId() || strings.TrimSpace(me.User.GetOrganizationId()) == "" {
		return "", fmt.Errorf("organization id not found for authenticated user")
	}

	return me.User.GetOrganizationId(), nil
}

func ResolveAppID(ctx CommandContext, appID string) (string, error) {
	appID = strings.TrimSpace(appID)
	if appID != "" {
		return appID, nil
	}

	activeApp := strings.TrimSpace(ctx.Config.GetActiveApp())
	if activeApp == "" {
		return "", fmt.Errorf("app id is required; pass --app-id or set one with \"superplane apps active\"")
	}

	return activeApp, nil
}

// BindAppIDFlag registers --app-id and a deprecated --canvas-id alias.
func BindAppIDFlag(cmd *cobra.Command, dest *string, usage string) {
	cmd.Flags().StringVar(dest, "app-id", "", usage)
	cmd.Flags().StringVar(dest, "canvas-id", "", usage)
	_ = cmd.Flags().MarkDeprecated("canvas-id", "use --app-id")
}
