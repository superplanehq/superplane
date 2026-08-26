import { Loader2, Plug, Search, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/usePermissions";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { showErrorToast } from "@/lib/toast";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { analytics } from "@/lib/analytics";
import { isCapabilityBasedIntegrationDefinition } from "@/lib/integrations";
import { posthog, isPostHogEnabled } from "@/posthog";
import { integrationDetailPath, integrationSetupPath, useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { factoryCardClassName } from "../factoryPageLayoutStyles";
import { FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

const INTEGRATION_SURVEY_NAME = "Integration Survey";

export function OrganizationSettingsIntegrationsPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  usePageTitle(["Integrations"]);
  const navigate = useNavigate();
  const integrationsBasePath = useIntegrationsBasePath(organizationId);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationDefinition | null>(null);
  const [integrationName, setIntegrationName] = useState("");
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
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

  const selectedInstructions = selectedIntegration?.instructions?.trim() ?? "";

  const handleConnectClick = (integration: IntegrationsIntegrationDefinition) => {
    if (!canCreateIntegrations) return;

    if (isCapabilityBasedIntegrationDefinition(integration)) {
      if (!integration.name) return;
      analytics.integrationConnectStart(integration.name, "integrations_page", organizationId);
      navigate(integrationSetupPath(integrationsBasePath, integration.name));
      return;
    }

    setSelectedIntegration(integration);
    setIntegrationName(getNextIntegrationName(integration.name || "integration", integrationNames));
    setConfiguration({});
    setIsModalOpen(true);
    analytics.integrationConnectStart(integration.name ?? "", "integrations_page", organizationId);
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

      const createdId = result.data?.integration?.metadata?.id;
      if (createdId) {
        navigate(integrationDetailPath(integrationsBasePath, createdId));
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

  const createIntegrationNotice = createIntegrationMutation.isError
    ? getUsageLimitNotice(createIntegrationMutation.error, organizationId)
    : null;

  return (
    <FactorySettingsPageFrame title="Integrations" subtitle="Connect external tools and services to extend SuperPlane.">
      {isLoading ? (
        <p className="text-[13px] text-muted-foreground">Loading integrations...</p>
      ) : (
        <>
          <div className="relative mb-4">
            <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              value={filterQuery}
              onChange={(event) => setFilterQuery(event.target.value)}
              placeholder="Filter integrations..."
              className="pr-9 pl-9"
            />
            {filterQuery.length > 0 ? (
              <button
                type="button"
                onClick={() => setFilterQuery("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                aria-label="Clear filter"
              >
                <X className="size-4" />
              </button>
            ) : null}
          </div>
          {filteredIntegrationCatalog.length === 0 ? (
            <div className="py-12 text-center">
              <Plug className="mx-auto mb-2 size-6 text-muted-foreground" />
              <p className="text-[13px] text-foreground">
                {integrationCatalog.length === 0 ? "No integrations available." : "No integrations match your filter."}
              </p>
              {isIntegrationSurveyActive ? (
                <p className="mt-3 text-[13px] text-muted-foreground">
                  Cannot find your integration?{" "}
                  <button
                    type="button"
                    onClick={handleRequestIntegration}
                    className="font-medium text-foreground underline underline-offset-2"
                  >
                    Request it
                  </button>
                </p>
              ) : null}
            </div>
          ) : (
            <div className="space-y-3">
              {filteredIntegrationCatalog.map((item) => {
                const connectedCount = item.instances.length;

                return (
                  <section key={item.providerName} className={`${factoryCardClassName} p-4`}>
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 flex size-8 items-center justify-center">
                          <IntegrationIcon
                            integrationName={item.providerName}
                            iconSlug={item.integrationDef?.icon}
                            className="size-8 text-muted-foreground"
                          />
                        </div>
                        <div>
                          <h3 className="text-[13px] font-medium text-foreground">{item.providerLabel}</h3>
                          {item.integrationDef?.description ? (
                            <p className="mt-0.5 text-[13px] text-muted-foreground">
                              {item.integrationDef.description}
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
                      <div className="mt-3 pl-[44px]">
                        <p className="mb-2 text-[12px] text-muted-foreground">
                          {connectedCount} connected instance{connectedCount === 1 ? "" : "s"}
                        </p>
                        {item.instances.map((integration) => {
                          const integrationDisplayName = integration.metadata?.name;
                          const statusLabel = integration.status?.state
                            ? integration.status.state.charAt(0).toUpperCase() + integration.status.state.slice(1)
                            : "Unknown";

                          return (
                            <div
                              key={integration.metadata?.id}
                              className="flex items-center gap-2 border-t border-border py-1.5"
                            >
                              <Plug
                                className={`size-4 shrink-0 ${
                                  integration.status?.state === "ready"
                                    ? "text-green-600"
                                    : integration.status?.state === "error"
                                      ? "text-destructive"
                                      : "text-amber-600"
                                }`}
                              />
                              <span className="w-16 text-[12px] font-medium text-muted-foreground">{statusLabel}</span>
                              <p className="truncate text-[13px] font-medium text-foreground">
                                {integrationDisplayName}
                              </p>
                              <div className="ml-auto">
                                <PermissionTooltip
                                  allowed={canUpdateIntegrations || permissionsLoading}
                                  message="You don't have permission to update integrations."
                                >
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => {
                                      if (!canUpdateIntegrations) return;
                                      const providerName = integration.metadata?.integrationName;
                                      const integrationId = integration.metadata?.id;
                                      if (providerName && integration.status?.setupState?.currentStep) {
                                        navigate(integrationSetupPath(integrationsBasePath, providerName), {
                                          state: { integrationId },
                                        });
                                        return;
                                      }
                                      if (integrationId) {
                                        navigate(integrationDetailPath(integrationsBasePath, integrationId), {
                                          state: { tab: "configuration" },
                                        });
                                      }
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
                  </section>
                );
              })}
              {isIntegrationSurveyActive ? (
                <p className="mt-6 text-center text-[13px] text-muted-foreground">
                  Cannot find your integration?{" "}
                  <button
                    type="button"
                    onClick={handleRequestIntegration}
                    className="font-medium text-foreground underline underline-offset-2"
                  >
                    Request it
                  </button>
                </p>
              ) : null}
            </div>
          )}

          <Dialog
            open={isModalOpen && Boolean(selectedIntegration)}
            onOpenChange={(open) => !open && handleCloseModal()}
          >
            <DialogContent className="max-h-[80vh] max-w-2xl overflow-y-auto">
              {selectedIntegration ? (
                <>
                  <DialogHeader>
                    <DialogTitle className="flex items-center gap-3">
                      <IntegrationIcon
                        integrationName={selectedIntegration.name}
                        iconSlug={selectedIntegration.icon}
                        className="size-6 text-muted-foreground"
                      />
                      Connect {selectedIntegration.label || selectedIntegration.name}
                    </DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4">
                    {selectedInstructions ? <IntegrationInstructions description={selectedInstructions} /> : null}
                    <div>
                      <Label className="mb-2">
                        Integration name
                        <span className="ml-1">*</span>
                      </Label>
                      <Input
                        type="text"
                        value={integrationName}
                        onChange={(event) => setIntegrationName(event.target.value)}
                        placeholder="e.g., my-app-integration"
                        required
                        disabled={!canCreateIntegrations}
                      />
                      <p className="mt-2 text-[12px] text-muted-foreground">A unique name for this integration.</p>
                    </div>
                    {selectedIntegration.configuration && selectedIntegration.configuration.length > 0 ? (
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
                    ) : null}
                  </div>
                  <div className="mt-6 flex justify-start gap-3">
                    <Button
                      onClick={() => void handleConnect()}
                      disabled={
                        createIntegrationMutation.isPending || !integrationName?.trim() || !canCreateIntegrations
                      }
                      className="flex items-center gap-2"
                    >
                      {createIntegrationMutation.isPending ? (
                        <>
                          <Loader2 className="size-4 animate-spin" />
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
                </>
              ) : null}
            </DialogContent>
          </Dialog>
        </>
      )}
    </FactorySettingsPageFrame>
  );
}
