import { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";

export function isCapabilityBasedIntegration(integration: OrganizationsIntegration) {
  if (!integration) return false;
  return integration.status?.legacySetup === false
}

export function isCapabilityBasedIntegrationDefinition(integration: IntegrationsIntegrationDefinition) {
  return integration.legacySetupOnly === false
}
