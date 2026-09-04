import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";

export function githubIntegrationOwner(integration: OrganizationsIntegration): string | undefined {
  const owner = integration.status?.metadata?.owner;
  return typeof owner === "string" && owner.trim() ? owner.trim() : undefined;
}

export function shouldNameOrganizationFromGitHub(factory: FactoriesFactory | null, selectNewest: boolean): boolean {
  return selectNewest && factory?.onboarding?.initial === true;
}
