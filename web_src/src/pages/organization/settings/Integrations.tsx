import { Plug, Search, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAvailableIntegrations, useConnectedIntegrations } from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import type { IntegrationsIntegrationDefinition } from "../../../api-client/types.gen";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [filterQuery, setFilterQuery] = useState("");
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);

  const isLoading = loadingAvailable || loadingInstalled;
  const connectedInstancesByProvider = useMemo(() => {
    const groups = new Map<string, typeof organizationIntegrations>();

    organizationIntegrations.forEach((integration) => {
      const provider = integration.spec?.integrationName;
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

    return [...catalogByProvider.values()].sort((a, b) => a.providerLabel.localeCompare(b.providerLabel));
  }, [availableIntegrations, connectedInstancesByProvider, organizationIntegrations]);

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
        (instance.metadata?.name || instance.spec?.integrationName || "").toLowerCase().includes(normalizedQuery),
      );
    });
  }, [filterQuery, integrationCatalog]);

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex justify-center items-center h-32">
          <p className="text-gray-500 dark:text-gray-400">Loading integrations...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="relative mb-4">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 dark:text-gray-400" />
        <Input
          type="text"
          value={filterQuery}
          onChange={(e) => setFilterQuery(e.target.value)}
          placeholder="Filter integrations..."
          className="pl-9 pr-9"
        />
        {filterQuery.length > 0 ? (
          <button
            type="button"
            onClick={() => setFilterQuery("")}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            aria-label="Clear filter"
          >
            <X className="w-4 h-4" />
          </button>
        ) : null}
      </div>
      {filteredIntegrationCatalog.length === 0 ? (
        <div className="text-center py-12">
          <Plug className="w-6 h-6 text-gray-800 mx-auto mb-2" />
          <p className="text-sm text-gray-800">
            {integrationCatalog.length === 0 ? "No integrations available." : "No integrations match your filter."}
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {filteredIntegrationCatalog.map((item) => {
            const connectedCount = item.instances.length;

            return (
              <div key={item.providerName} className="bg-white border border-gray-300 dark:border-gray-700 rounded-md">
                <div className="p-4 flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3">
                    <div className="mt-0.5 flex h-8 w-8 items-center justify-center">
                      <IntegrationIcon
                        integrationName={item.providerName}
                        iconSlug={item.integrationDef?.icon}
                        className="w-8 h-8 text-gray-500 dark:text-gray-400"
                      />
                    </div>
                    <div>
                      <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">{item.providerLabel}</h3>
                      {item.integrationDef?.description ? (
                        <p className="mt-0.5 text-sm text-gray-800 dark:text-gray-400">
                          {item.integrationDef?.description}
                        </p>
                      ) : null}
                    </div>
                  </div>
                  <PermissionTooltip
                    allowed={Boolean(item.integrationDef) && (canCreateIntegrations || permissionsLoading)}
                    message={
                      item.integrationDef
                        ? "You don't have permission to connect integrations."
                        : "This integration provider is no longer available for new connections."
                    }
                  >
                    <Button
                      variant="default"
                      size="sm"
                      onClick={() => {
                        if (!item.integrationDef?.name) return;
                        navigate(`/${organizationId}/settings/integrations/new/${item.integrationDef.name}`);
                      }}
                      className="self-start"
                      disabled={!item.integrationDef || !canCreateIntegrations}
                    >
                      {item.integrationDef ? "Connect" : "Unavailable"}
                    </Button>
                  </PermissionTooltip>
                </div>
                {item.instances.length > 0 ? (
                  <div className="pr-4 pb-4 pl-[60px]">
                    <p className="mb-2 text-xs text-gray-500 dark:text-gray-400">
                      {connectedCount} connected instance{connectedCount === 1 ? "" : "s"}
                    </p>
                    {item.instances.map((integration, index) => {
                      const integrationDisplayName =
                        integration.metadata?.name ||
                        getIntegrationTypeDisplayName(undefined, integration.spec?.integrationName) ||
                        integration.spec?.integrationName;
                      const statusLabel = integration.status?.state
                        ? integration.status.state.charAt(0).toUpperCase() + integration.status.state.slice(1)
                        : "Unknown";

                      return (
                        <div
                          key={integration.metadata?.id}
                          className={`flex items-center gap-2 py-1.5 border-t border-gray-200 dark:border-gray-700 ${index === 0 ? "mt-1" : ""}`}
                        >
                          <Plug
                            className={`w-4 h-4 shrink-0 ${
                              integration.status?.state === "ready"
                                ? "text-green-500"
                                : integration.status?.state === "error"
                                  ? "text-red-500"
                                  : "text-amber-600"
                            }`}
                          />
                          <span
                            className={`inline-flex w-16 items-center justify-start rounded text-xs font-medium ${
                              integration.status?.state === "ready"
                                ? "bg-white text-green-500"
                                : integration.status?.state === "error"
                                  ? "bg-white text-red-500"
                                  : "bg-white text-amber-600"
                            }`}
                          >
                            {statusLabel}
                          </span>
                          <p className="text-sm font-medium text-gray-800 dark:text-gray-100 truncate">
                            {integrationDisplayName}
                          </p>
                          <div className="ml-auto flex items-center gap-4">
                            <PermissionTooltip
                              allowed={canUpdateIntegrations || permissionsLoading}
                              message="You don't have permission to update integrations."
                            >
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                  if (!canUpdateIntegrations) return;
                                  navigate(`/${organizationId}/settings/integrations/${integration.metadata?.id}`, {
                                    state: { tab: "configuration" },
                                  });
                                }}
                                disabled={!canUpdateIntegrations}
                              >
                                Configure
                              </Button>
                            </PermissionTooltip>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
