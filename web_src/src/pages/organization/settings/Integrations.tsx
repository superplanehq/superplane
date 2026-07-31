import { Loader2, Plug, Search, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import { useNavigate } from "react-router-dom";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useListKeyboardNavigation } from "@/hooks/useListKeyboardNavigation";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
} from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePermissions } from "@/contexts/usePermissions";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "../../../api-client/types.gen";
import { IntegrationCatalogRow } from "./IntegrationCatalogRow";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { Icon } from "@/components/Icon";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { showErrorToast } from "@/lib/toast";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { analytics } from "@/lib/analytics";
import { isCapabilityBasedIntegrationDefinition } from "@/lib/integrations";
import { posthog, isPostHogEnabled } from "@/posthog";
import { cn } from "@/lib/utils";
import {
  settingsEmptyStateIconClassName,
  settingsEmptyStateTitleClassName,
  settingsModalClassName,
} from "./settingsPageStyles";

const INTEGRATION_SURVEY_NAME = "Integration Survey";

const NAVIGATION_KEYS = ["ArrowDown", "ArrowUp", "Home", "End"];

function filterCountLabel(matching: number, total: number, isFiltering: boolean): string {
  return isFiltering ? `${matching} of ${total} integrations` : `${total} integrations`;
}

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  usePageTitle(["Integrations"]);
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationDefinition | null>(null);
  const [integrationName, setIntegrationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
  // The highlight is a keyboard affordance only: it appears while filtering or
  // once the arrow keys are used, and never follows the pointer.
  const [highlightVisible, setHighlightVisible] = useState(false);
  const [isIntegrationSurveyActive, setIsIntegrationSurveyActive] = useState(false);
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  useEffect(() => {
    if (!isPostHogEnabled) return;

    posthog.getSurveys((surveys) => {
      const isActive = surveys.some(
        (survey) => survey.name === INTEGRATION_SURVEY_NAME && survey.start_date != null && survey.end_date == null,
      );
      setIsIntegrationSurveyActive(isActive);
    }, true);
  }, []);

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);
  const createIntegrationMutation = useCreateIntegration(organizationId, "integrations_page");

  const isLoading = loadingAvailable || loadingInstalled;

  useReportPageReady(!isLoading && !permissionsLoading);

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

  // Connected providers are shown in their own section above the rest. The
  // flattened order is what the arrow keys walk, so the highlight moves through
  // both sections as one list.
  const { connectedItems, availableItems, orderedItems } = useMemo(() => {
    const connected = filteredIntegrationCatalog.filter((item) => item.instances.length > 0);
    const available = filteredIntegrationCatalog.filter((item) => item.instances.length === 0);
    return { connectedItems: connected, availableItems: available, orderedItems: [...connected, ...available] };
  }, [filteredIntegrationCatalog]);

  const selectedInstructions = useMemo(() => {
    return selectedIntegration?.instructions?.trim() ?? "";
  }, [selectedIntegration?.instructions]);

  const getNextIntegrationName = useCallback(
    (baseName?: string) => {
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
    },
    [integrationNames],
  );

  const handleConnectClick = useCallback(
    (integration: IntegrationsIntegrationDefinition) => {
      if (!canCreateIntegrations) return;

      if (isCapabilityBasedIntegrationDefinition(integration)) {
        if (!integration.name) return;
        analytics.integrationConnectStart(integration.name, "integrations_page", organizationId);
        navigate(`/${organizationId}/settings/integrations/${integration.name}/setup`);
        return;
      }

      setSelectedIntegration(integration);
      setIntegrationName(getNextIntegrationName(integration.name));
      setConfiguration({});
      setIsModalOpen(true);
      analytics.integrationConnectStart(integration.name ?? "", "integrations_page", organizationId);
    },
    [canCreateIntegrations, getNextIntegrationName, navigate, organizationId],
  );

  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const rowRefs = useRef<Array<HTMLLIElement | null>>([]);

  const handleActivateRow = useCallback(
    (index: number) => {
      // Never act on a highlight the user cannot see.
      if (!highlightVisible) return;
      const integrationDef = orderedItems[index]?.integrationDef;
      if (!integrationDef) return;
      handleConnectClick(integrationDef);
    },
    [handleConnectClick, highlightVisible, orderedItems],
  );

  const { activeIndex, setActiveIndex, handleKeyDown } = useListKeyboardNavigation({
    itemCount: orderedItems.length,
    onActivate: handleActivateRow,
  });

  // Keep the highlighted row on screen while arrowing through a long catalog.
  useEffect(() => {
    if (!highlightVisible || activeIndex < 0) return;
    rowRefs.current[activeIndex]?.scrollIntoView?.({ block: "nearest" });
  }, [activeIndex, highlightVisible]);

  // `/` jumps to the filter, matching the convention used across the app. The
  // global command palette owns the modified variants, so ignore those here.
  useEffect(() => {
    const focusFilter = (event: globalThis.KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;

      const target = event.target as HTMLElement | null;
      const tagName = target?.tagName;
      if (tagName === "INPUT" || tagName === "TEXTAREA" || tagName === "SELECT" || target?.isContentEditable) {
        return;
      }

      event.preventDefault();
      searchInputRef.current?.focus();
    };

    document.addEventListener("keydown", focusFilter);
    return () => document.removeEventListener("keydown", focusFilter);
  }, []);

  const handleFilterKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      if (filterQuery.length === 0) {
        searchInputRef.current?.blur();
        return;
      }
      event.preventDefault();
      setFilterQuery("");
      setActiveIndex(0);
      setHighlightVisible(false);
      return;
    }

    // The first arrow press reveals the highlight where it already sits rather
    // than skipping the top result.
    if (!highlightVisible && NAVIGATION_KEYS.includes(event.key)) {
      event.preventDefault();
      setHighlightVisible(true);
      return;
    }

    handleKeyDown(event);
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
      setIsModalOpen(false);
      setSelectedIntegration(null);
      setIntegrationName("");
      setConfiguration({});

      // Redirect to the integration details page
      if (result.data?.integration?.metadata?.id) {
        navigate(`/${organizationId}/settings/integrations/${result.data.integration.metadata.id}`);
      }
    } catch (_error) {
      showErrorToast(getUsageLimitToastMessage(_error, "Failed to create integration"));
    }
  };

  const handleRequestIntegration = () => {
    analytics.integrationRequested(organizationId);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedIntegration(null);
    setIntegrationName("");
    setConfiguration({});
    createIntegrationMutation.reset();
  };

  const handleConfigureClick = useCallback(
    (integration: OrganizationsIntegration) => {
      const providerName = integration.metadata?.integrationName;
      if (providerName && integration.status?.setupState?.currentStep) {
        navigate(`/${organizationId}/settings/integrations/${providerName}/setup`, {
          state: { integrationId: integration.metadata?.id },
        });
        return;
      }

      navigate(`/${organizationId}/settings/integrations/${integration.metadata?.id}`, {
        state: { tab: "configuration" },
      });
    },
    [navigate, organizationId],
  );

  const createIntegrationNotice = createIntegrationMutation.isError
    ? getUsageLimitNotice(createIntegrationMutation.error, organizationId)
    : null;

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
      <div className="relative mb-2">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 dark:text-gray-400" />
        <Input
          ref={searchInputRef}
          type="text"
          value={filterQuery}
          onChange={(e) => {
            setFilterQuery(e.target.value);
            setActiveIndex(0);
            setHighlightVisible(e.target.value.trim().length > 0);
          }}
          onKeyDown={handleFilterKeyDown}
          placeholder="Filter integrations..."
          className="pl-9 pr-9"
          autoFocus
          aria-describedby="integration-filter-count"
        />
        {filterQuery.length > 0 ? (
          <button
            type="button"
            onClick={() => {
              setFilterQuery("");
              setActiveIndex(0);
              setHighlightVisible(false);
              searchInputRef.current?.focus();
            }}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            aria-label="Clear filter"
          >
            <X className="w-4 h-4" />
          </button>
        ) : null}
      </div>
      <p
        id="integration-filter-count"
        role="status"
        aria-live="polite"
        className="mb-4 text-xs text-gray-500 dark:text-gray-400"
      >
        {filterCountLabel(orderedItems.length, integrationCatalog.length, filterQuery.trim().length > 0)}
        <span className="sr-only">
          {" "}
          Use the arrow keys to move between integrations and press Enter to connect the highlighted one.
        </span>
      </p>
      {filteredIntegrationCatalog.length === 0 ? (
        <div className="py-12 text-center">
          <Plug className={cn("mx-auto mb-2 h-6 w-6", settingsEmptyStateIconClassName)} />
          <p className={settingsEmptyStateTitleClassName}>
            {integrationCatalog.length === 0 ? "No integrations available." : "No integrations match your filter."}
          </p>
          {isIntegrationSurveyActive ? (
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">
              Can't find your integration?{" "}
              <button
                type="button"
                onClick={handleRequestIntegration}
                className="text-blue-600 hover:underline dark:text-blue-400 font-medium"
              >
                Request it
              </button>
            </p>
          ) : null}
        </div>
      ) : (
        <div className="space-y-8">
          {[
            { title: "Connected", items: connectedItems, offset: 0 },
            { title: "Available", items: availableItems, offset: connectedItems.length },
          ]
            .filter((section) => section.items.length > 0)
            .map((section) => (
              <section key={section.title} aria-label={`${section.title} (${section.items.length})`}>
                <div className="mb-3 flex items-center gap-3">
                  <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                    {section.title}
                  </h2>
                  <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700/50 dark:text-gray-300">
                    {section.items.length}
                  </span>
                  <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700/70" />
                </div>
                <ul className="space-y-4">
                  {section.items.map((item, indexInSection) => {
                    const flatIndex = section.offset + indexInSection;

                    return (
                      <IntegrationCatalogRow
                        key={item.providerName}
                        entry={item}
                        isActive={highlightVisible && flatIndex === activeIndex}
                        canCreateIntegrations={canCreateIntegrations}
                        canUpdateIntegrations={canUpdateIntegrations}
                        permissionsLoading={permissionsLoading}
                        onConnect={handleConnectClick}
                        onConfigure={handleConfigureClick}
                        rowRef={(node) => {
                          rowRefs.current[flatIndex] = node;
                        }}
                      />
                    );
                  })}
                </ul>
              </section>
            ))}
          {isIntegrationSurveyActive ? (
            <p className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
              Can't find your integration?{" "}
              <button
                type="button"
                onClick={handleRequestIntegration}
                className="text-blue-600 hover:underline dark:text-blue-400 font-medium"
              >
                Request it
              </button>
            </p>
          ) : null}
        </div>
      )}
      {/* Connect Modal */}
      {isModalOpen &&
        selectedIntegration &&
        (() => {
          const integrationTypeName = selectedIntegration.name;
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <div className={cn(settingsModalClassName, "max-h-[80vh] max-w-2xl overflow-y-auto")}>
                <div className="p-6">
                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                      <IntegrationIcon
                        integrationName={integrationTypeName}
                        iconSlug={selectedIntegration.icon}
                        className="w-6 h-6 text-gray-500 dark:text-gray-400"
                      />
                      <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                        Connect {selectedIntegration.label || selectedIntegration.name}
                      </h3>
                    </div>
                    <button
                      onClick={handleCloseModal}
                      className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
                      disabled={createIntegrationMutation.isPending}
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>

                  <div className="space-y-4">
                    {selectedInstructions && <IntegrationInstructions description={selectedInstructions} />}
                    {/* Integration Name Field */}
                    <div>
                      <Label className="text-gray-800 dark:text-gray-100 mb-2">
                        Integration Name
                        <span className="ml-1 text-gray-800 dark:text-gray-100">*</span>
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

                    {/* Configuration Fields */}
                    {selectedIntegration.configuration && selectedIntegration.configuration.length > 0 && (
                      <div className="space-y-4">
                        {selectedIntegration.configuration
                          .filter((field) => Boolean(field.name))
                          .map((field) => (
                            <ConfigurationFieldRenderer
                              key={field.name!}
                              field={field}
                              value={configuration[field.name!]}
                              onChange={(value) => setConfiguration({ ...configuration, [field.name!]: value })}
                              allValues={configuration}
                              organizationId={organizationId}
                            />
                          ))}
                      </div>
                    )}
                  </div>

                  <div className="flex justify-start gap-3 mt-6">
                    <Button
                      color="blue"
                      onClick={handleConnect}
                      disabled={
                        createIntegrationMutation.isPending || !integrationName?.trim() || !canCreateIntegrations
                      }
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
                    <Button variant="outline" onClick={handleCloseModal} disabled={createIntegrationMutation.isPending}>
                      Cancel
                    </Button>
                  </div>

                  {createIntegrationMutation.isError && createIntegrationNotice ? (
                    <UsageLimitAlert notice={createIntegrationNotice} className="mt-4" />
                  ) : null}
                  {createIntegrationMutation.isError && !createIntegrationNotice ? (
                    <Alert variant="destructive" className="mt-4">
                      <AlertTitle>Unable to create integration</AlertTitle>
                      <AlertDescription>
                        Failed to create integration: {getApiErrorMessage(createIntegrationMutation.error)}
                      </AlertDescription>
                    </Alert>
                  ) : null}
                </div>
              </div>
            </div>
          );
        })()}
    </div>
  );
}
