import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client/types.gen";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

export type IntegrationCatalogItem = {
  providerName: string;
  providerLabel: string;
  integrationDef: IntegrationsIntegrationDefinition | null;
  instances: OrganizationsIntegration[];
};

export function integrationNameSet(organizationIntegrations: OrganizationsIntegration[]) {
  return new Set(
    organizationIntegrations.map((integration) => integration.metadata?.name?.trim()).filter(Boolean) as string[],
  );
}

export function buildIntegrationCatalog(
  availableIntegrations: IntegrationsIntegrationDefinition[],
  organizationIntegrations: OrganizationsIntegration[],
): IntegrationCatalogItem[] {
  const connectedByProvider = groupConnectedInstances(organizationIntegrations);
  const catalogByProvider = new Map<string, IntegrationCatalogItem>();

  for (const integrationDef of availableIntegrations) {
    const providerName = integrationDef.name || "";
    const providerLabel =
      integrationDef.label ||
      getIntegrationTypeDisplayName(undefined, integrationDef.name) ||
      integrationDef.name ||
      "Integration";
    catalogByProvider.set(providerName, {
      providerName,
      providerLabel,
      integrationDef,
      instances: sortedInstances(connectedByProvider.get(providerName) ?? [], providerLabel),
    });
  }

  for (const [providerName, instances] of connectedByProvider) {
    if (catalogByProvider.has(providerName)) {
      continue;
    }
    const providerLabel = getIntegrationTypeDisplayName(undefined, providerName) || providerName || "Integration";
    catalogByProvider.set(providerName, {
      providerName,
      providerLabel,
      integrationDef: null,
      instances: sortedInstances(instances, providerLabel),
    });
  }

  return [...catalogByProvider.values()].sort((left, right) => {
    const leftHasInstances = left.instances.length > 0;
    const rightHasInstances = right.instances.length > 0;
    if (leftHasInstances !== rightHasInstances) {
      return leftHasInstances ? -1 : 1;
    }
    return left.providerLabel.localeCompare(right.providerLabel);
  });
}

export function filterIntegrationCatalog(catalog: IntegrationCatalogItem[], filterQuery: string) {
  const normalizedQuery = filterQuery.trim().toLowerCase();
  if (!normalizedQuery) {
    return catalog;
  }
  return catalog.filter((item) => catalogItemMatchesQuery(item, normalizedQuery));
}

export function integrationStatusLabel(state: string | undefined) {
  if (!state) {
    return "Unknown";
  }
  return state.charAt(0).toUpperCase() + state.slice(1);
}

function groupConnectedInstances(organizationIntegrations: OrganizationsIntegration[]) {
  const groups = new Map<string, OrganizationsIntegration[]>();
  for (const integration of organizationIntegrations) {
    const provider = integration.metadata?.integrationName;
    if (!provider) {
      continue;
    }
    const current = groups.get(provider) ?? [];
    current.push(integration);
    groups.set(provider, current);
  }
  return groups;
}

function sortedInstances(instances: OrganizationsIntegration[], providerLabel: string) {
  return [...instances].sort((left, right) =>
    (left.metadata?.name || providerLabel).localeCompare(right.metadata?.name || providerLabel),
  );
}

function catalogItemMatchesQuery(item: IntegrationCatalogItem, normalizedQuery: string) {
  const providerText = [item.providerLabel, item.providerName, item.integrationDef?.description]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  if (providerText.includes(normalizedQuery)) {
    return true;
  }
  return item.instances.some((instance) =>
    (instance.metadata?.name || instance.metadata?.integrationName || "").toLowerCase().includes(normalizedQuery),
  );
}
