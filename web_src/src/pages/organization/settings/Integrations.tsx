import { Loader2, Plug, Search, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useUpdateIntegration,
} from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type {
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  OrganizationsBrowserAction,
} from "../../../api-client/types.gen";
import { getApiErrorMessage } from "@/utils/errors";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { renderIntegrationMetadata } from "./integrationMetadataRenderers";

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationDefinition | null>(null);
  const [integrationName, setIntegrationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
  const [createdIntegration, setCreatedIntegration] = useState<OrganizationsIntegration | null>(null);
  const [wizardBrowserAction, setWizardBrowserAction] = useState<OrganizationsBrowserAction | undefined>(undefined);
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);
  const createIntegrationMutation = useCreateIntegration(organizationId);
  const createdIntegrationId = createdIntegration?.metadata?.id ?? "";
  const updateIntegrationMutation = useUpdateIntegration(organizationId, createdIntegrationId);

  const isLoading = loadingAvailable || loadingInstalled;
  const integrationNames = useMemo(() => {
    return new Set(
      organizationIntegrations.map((integration) => integration.metadata?.name?.trim()).filter(Boolean) as string[],
    );
  }, [organizationIntegrations]);
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
        (instance.metadata?.name || instance.spec?.integrationName || "").toLowerCase().includes(normalizedQuery),
      );
    });
  }, [filterQuery, integrationCatalog]);

  const selectedInstructions = useMemo(() => {
    return selectedIntegration?.instructions?.trim();
  }, [selectedIntegration?.instructions]);

  const getNextIntegrationName = (baseName?: string) => {
    const normalizedBaseName = baseName?.trim() || "integration";
    if (!integrationNames.has(normalizedBaseName)) {
      return normalizedBaseName;
    }

    let suffix = 2;
    let candidate = `${normalizedBaseName}-${suffix}`;
    while (integrationNames.has(candidate)) {
      suffix += 1;
      candidate = `${normalizedBaseName}-${suffix}`;
    }

    return candidate;
  };

  const handleConnectClick = (integration: IntegrationsIntegrationDefinition) => {
    if (!canCreateIntegrations) return;
    setSelectedIntegration(integration);
    setIntegrationName(getNextIntegrationName(integration.name));
    setConfiguration({});
    setIsModalOpen(true);
  };
  const handleConnect = async () => {
    if (!canCreateIntegrations) return;
    if (!selectedIntegration?.name) return;

    try {
      const result = await createIntegrationMutation.mutateAsync({
        integrationName: selectedIntegration.name,
        name: integrationName,
        configuration,
      });
      const integration = result.data?.integration;

      const hasMetadataContent =
        integration && selectedIntegration
          ? renderIntegrationMetadata(selectedIntegration.name, integration) !== null
          : false;

      if (integration?.status?.state === "pending" || hasMetadataContent) {
        setCreatedIntegration(integration!);
        setConfiguration(integration?.spec?.configuration ?? {});
        if (integration?.status?.browserAction) {
          setWizardBrowserAction(integration.status.browserAction);
        }
        return;
      }

      setIsModalOpen(false);
      setSelectedIntegration(null);
      setIntegrationName("");
      setConfiguration({});
      if (integration?.metadata?.id) {
        navigate(`/${organizationId}/settings/integrations/${integration.metadata.id}`);
      }
    } catch (_error) {
      showErrorToast("Failed to create integration");
    }
  };

  const handleWizardSave = async () => {
    if (!createdIntegration?.metadata?.id) return;

    try {
      const result = await updateIntegrationMutation.mutateAsync({
        name: integrationName,
        configuration,
      });
      const integration = result.data?.integration;

      if (integration?.status?.browserAction) {
        setCreatedIntegration(integration);
        setWizardBrowserAction(integration.status.browserAction);
        return;
      }

      handleCloseModal();
      if (integration?.metadata?.id) {
        navigate(`/${organizationId}/settings/integrations/${integration.metadata.id}`);
      }
    } catch (_error) {
      showErrorToast("Failed to update integration");
    }
  };

  const handleWizardBrowserAction = () => {
    if (!wizardBrowserAction) return;
    const { url, method, formFields } = wizardBrowserAction;

    if (method?.toUpperCase() === "POST" && formFields) {
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";
      Object.entries(formFields).forEach(([key, value]) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = String(value);
        form.appendChild(input);
      });
      document.body.appendChild(form);
      form.submit();
      document.body.removeChild(form);
    } else if (url) {
      window.open(url, "_blank");
    }
  };

  const handleCloseModal = () => {
    const integrationId = createdIntegration?.metadata?.id;
    setIsModalOpen(false);
    setSelectedIntegration(null);
    setIntegrationName("");
    setConfiguration({});
    setCreatedIntegration(null);
    setWizardBrowserAction(undefined);
    createIntegrationMutation.reset();
    updateIntegrationMutation.reset();

    if (integrationId) {
      navigate(`/${organizationId}/settings/integrations/${integrationId}`);
    }
  };

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
                        if (!item.integrationDef) return;
                        handleConnectClick(item.integrationDef);
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

      {/* Connect Modal */}
      {isModalOpen &&
        selectedIntegration &&
        (() => {
          const integrationTypeName = selectedIntegration.name;
          const isSetupPhase = createdIntegration != null;
          const isReadyWithMetadata = isSetupPhase && createdIntegration?.status?.state !== "pending";
          const creationFields =
            selectedIntegration.configuration?.filter(
              (f) => f.required || (f.requiredConditions && f.requiredConditions.length > 0),
            ) ?? [];
          const allFields = selectedIntegration.configuration ?? [];
          const fieldsToShow = isSetupPhase ? allFields : creationFields;
          const isBusy = createIntegrationMutation.isPending || updateIntegrationMutation.isPending;
          const metadataContent = isSetupPhase
            ? renderIntegrationMetadata(selectedIntegration.name, createdIntegration!)
            : null;
          const activeBrowserAction = isSetupPhase
            ? (wizardBrowserAction ?? createdIntegration!.status?.browserAction)
            : undefined;

          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
                <div className="p-6">
                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                      <IntegrationIcon
                        integrationName={integrationTypeName}
                        iconSlug={selectedIntegration.icon}
                        className="w-6 h-6 text-gray-500 dark:text-gray-400"
                      />
                      <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                        {isSetupPhase ? "Set up" : "Connect"} {selectedIntegration.label || selectedIntegration.name}
                      </h3>
                    </div>
                    <button
                      onClick={handleCloseModal}
                      className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                      disabled={isBusy}
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>

                  <div className="space-y-4">
                    {!isSetupPhase && selectedInstructions && (
                      <IntegrationInstructions description={selectedInstructions} />
                    )}

                    {metadataContent}

                    {activeBrowserAction && (
                      <IntegrationInstructions
                        description={activeBrowserAction.description}
                        onContinue={activeBrowserAction.url ? handleWizardBrowserAction : undefined}
                      />
                    )}

                    {!isSetupPhase && (
                      <div>
                        <Label className="text-gray-800 dark:text-gray-100 mb-2">
                          Integration Name
                          <span className="text-gray-800 ml-1">*</span>
                        </Label>
                        <Input
                          type="text"
                          value={integrationName}
                          onChange={(e) => setIntegrationName(e.target.value)}
                          placeholder="e.g., my-app-integration"
                          required
                          disabled={!canCreateIntegrations}
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                          A unique name for this integration
                        </p>
                      </div>
                    )}

                    {fieldsToShow.length > 0 && (
                      <div className="space-y-4">
                        {fieldsToShow.map((field) => {
                          if (!field.name) return null;
                          return (
                            <ConfigurationFieldRenderer
                              key={field.name}
                              field={field}
                              value={configuration[field.name]}
                              onChange={(value) => setConfiguration({ ...configuration, [field.name || ""]: value })}
                              allValues={configuration}
                              domainId={organizationId}
                              domainType="DOMAIN_TYPE_ORGANIZATION"
                              organizationId={organizationId}
                              appInstallationId={createdIntegration?.metadata?.id}
                            />
                          );
                        })}
                      </div>
                    )}
                  </div>

                  <div className="flex justify-start gap-3 mt-6">
                    {isSetupPhase ? (
                      <>
                        {isReadyWithMetadata ? (
                          <Button color="blue" onClick={handleCloseModal}>
                            Done
                          </Button>
                        ) : (
                          <>
                            <Button
                              color="blue"
                              onClick={() => void handleWizardSave()}
                              disabled={isBusy}
                              className="flex items-center gap-2"
                            >
                              {updateIntegrationMutation.isPending ? (
                                <>
                                  <Loader2 className="w-4 h-4 animate-spin" />
                                  Saving...
                                </>
                              ) : (
                                "Save"
                              )}
                            </Button>
                            <Button variant="outline" onClick={handleCloseModal} disabled={isBusy}>
                              Close
                            </Button>
                          </>
                        )}
                      </>
                    ) : (
                      <>
                        <Button
                          color="blue"
                          onClick={() => void handleConnect()}
                          disabled={isBusy || !integrationName?.trim() || !canCreateIntegrations}
                          className="flex items-center gap-2"
                        >
                          {createIntegrationMutation.isPending ? (
                            <>
                              <Loader2 className="w-4 h-4 animate-spin" />
                              Connecting...
                            </>
                          ) : (
                            "Connect"
                          )}
                        </Button>
                        <Button variant="outline" onClick={handleCloseModal} disabled={isBusy}>
                          Cancel
                        </Button>
                      </>
                    )}
                  </div>

                  {(createIntegrationMutation.isError || updateIntegrationMutation.isError) && (
                    <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                      <p className="text-sm text-red-800 dark:text-red-200">
                        {createIntegrationMutation.isError
                          ? `Failed to create integration: ${getApiErrorMessage(createIntegrationMutation.error)}`
                          : `Failed to update integration: ${getApiErrorMessage(updateIntegrationMutation.error)}`}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        })()}
    </div>
  );
}
