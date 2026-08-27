import { useMemo } from "react";
import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

export function useIntegrationCatalog(
  availableIntegrations: IntegrationsIntegrationDefinition[],
  organizationIntegrations: OrganizationsIntegration[],
  filterQuery: string,
) {
  const integrationNames = useMemo(() => {
    return new Set(
      organizationIntegrations.map((integration) => integration.metadata?.name?.trim()).filter(Boolean) as string[],
    );
  }, [organizationIntegrations]);

  const connectedInstancesByProvider = useMemo(() => {
    const groups = new Map<string, typeof organizationIntegrations>();

    organizationIntegrations.forEach((integration) => {
      const provider = integration.metadata?.integrationName;
      if (!provider) return;
      const current = groups.get(provider) || [];
      current.push(integration);
      groups.set(provider, current);
    });

    return groups;
  }, [organizationIntegrations]);

  const integrationCatalog = useMemo(() => {
    const catalogByProvider = new Map<
      string,
      {
        providerName: string;
        providerLabel: string;
        integrationDef: IntegrationsIntegrationDefinition | null;
        instances: typeof organizationIntegrations;
      }
    >();

    availableIntegrations.forEach((integrationDef) => {
      const providerName = integrationDef.name || "";
      const providerLabel =
        integrationDef.label ||
        getIntegrationTypeDisplayName(undefined, integrationDef.name) ||
        integrationDef.name ||
        "Integration";
      const instances = [...(connectedInstancesByProvider.get(providerName) || [])].sort((a, b) =>
        (a.metadata?.name || providerLabel).localeCompare(b.metadata?.name || providerLabel),
      );

      catalogByProvider.set(providerName, {
        providerName,
        providerLabel,
        integrationDef,
        instances,
      });
    });

    connectedInstancesByProvider.forEach((instances, providerName) => {
      if (catalogByProvider.has(providerName)) {
        return;
      }

      const providerLabel = getIntegrationTypeDisplayName(undefined, providerName) || providerName || "Integration";
      const sortedInstances = [...instances].sort((a, b) =>
        (a.metadata?.name || providerLabel).localeCompare(b.metadata?.name || providerLabel),
      );

      catalogByProvider.set(providerName, {
        providerName,
        providerLabel,
        integrationDef: null,
        instances: sortedInstances,
      });
    });

    return [...catalogByProvider.values()].sort((a, b) => {
      const aHasInstances = a.instances.length > 0;
      const bHasInstances = b.instances.length > 0;
      if (aHasInstances !== bHasInstances) {
        return aHasInstances ? -1 : 1;
      }
      return a.providerLabel.localeCompare(b.providerLabel);
    });
  }, [availableIntegrations, connectedInstancesByProvider]);

  const filteredIntegrationCatalog = useMemo(() => {
    const normalizedQuery = filterQuery.trim().toLowerCase();
    if (!normalizedQuery) {
      return integrationCatalog;
    }

    return integrationCatalog.filter((item) => {
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
    });
  }, [filterQuery, integrationCatalog]);

  return { integrationNames, integrationCatalog, filteredIntegrationCatalog };
}
