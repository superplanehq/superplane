import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";

/**
 * Guided setup is available when the integration registers a setup provider (API: legacySetupOnly is false).
 */
export function integrationSupportsGuidedSetup(definition?: IntegrationsIntegrationDefinition | null): boolean {
  return definition?.legacySetupOnly === false;
}

/**
 * Route to the v2 integration details page when the installation was not created via the legacy setup flow.
 * Prefer `status.legacySetup` from the API; fall back to capability presence for older responses.
 */
export function integrationUsesNewSetupFlow(integration: OrganizationsIntegration | null | undefined): boolean {
  if (integration?.status?.legacySetup !== undefined) {
    return integration.status.legacySetup === false;
  }
  const capabilities = integration?.status?.capabilities;
  return Array.isArray(capabilities) && capabilities.length > 0;
}

export function getIntegrationV2SetupPath(organizationId: string, integrationName: string): string {
  return `/${organizationId}/settings/integrations/${integrationName}/setup`;
}
