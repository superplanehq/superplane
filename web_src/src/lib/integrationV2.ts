import type { OrganizationsIntegration } from "@/api-client";

const INTEGRATIONS_WITH_V2_SETUP_FLOW = new Set(["github", "semaphore"]);

/**
 * Integrations created via the new setup flow expose capability state on the installation.
 * Used to route to the v2 integration details page vs. the legacy configuration-focused page.
 */
export function integrationUsesNewSetupFlow(integration: OrganizationsIntegration | null | undefined): boolean {
  const capabilities = integration?.status?.capabilities;
  return Array.isArray(capabilities) && capabilities.length > 0;
}

export function isIntegrationV2SetupEnabled(integrationName?: string | null): boolean {
  if (!integrationName) {
    return false;
  }

  return INTEGRATIONS_WITH_V2_SETUP_FLOW.has(integrationName);
}

export function getIntegrationV2SetupPath(organizationId: string, integrationName: string): string {
  return `/${organizationId}/settings/integrations/${integrationName}/setup`;
}
