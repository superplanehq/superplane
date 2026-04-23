const INTEGRATIONS_WITH_V2_SETUP_FLOW = new Set(["github", "semaphore"]);

export function isIntegrationV2SetupEnabled(integrationName?: string | null): boolean {
  if (!integrationName) {
    return false;
  }

  return INTEGRATIONS_WITH_V2_SETUP_FLOW.has(integrationName);
}

export function getIntegrationV2SetupPath(organizationId: string, integrationName: string): string {
  return `/${organizationId}/settings/integrations/${integrationName}/setup`;
}
