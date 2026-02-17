import { AppWindow, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
} from "../../../hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import { ConfigurationFieldRenderer } from "../../../ui/configurationFieldRenderer";
import type { IntegrationsIntegrationDefinition } from "../../../api-client/types.gen";
import { getApiErrorMessage } from "@/utils/errors";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { Icon } from "@/components/Icon";
import { showErrorToast } from "@/utils/toast";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";

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
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");

  const { data: availableIntegrations = [], isLoading: loadingAvailable } = useAvailableIntegrations();
  const { data: organizationIntegrations = [], isLoading: loadingInstalled } = useConnectedIntegrations(organizationId);
  const createIntegrationMutation = useCreateIntegration(organizationId);

  const isLoading = loadingAvailable || loadingInstalled;
  const integrationNames = useMemo(() => {
    return new Set(
      organizationIntegrations.map((integration) => integration.metadata?.name?.trim()).filter(Boolean) as string[],
    );
  }, [organizationIntegrations]);

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
      setIsModalOpen(false);
      setSelectedIntegration(null);
      setIntegrationName("");
      setConfiguration({});

      // Redirect to the integration details page
      if (result.data?.integration?.metadata?.id) {
        navigate(`/${organizationId}/settings/integrations/${result.data.integration.metadata.id}`);
      }
    } catch (_error) {
      showErrorToast("Failed to create integration");
    }
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedIntegration(null);
    setIntegrationName("");
    setConfiguration({});
    createIntegrationMutation.reset();
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
      {/* Integrations */}
      {organizationIntegrations.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-medium mb-4">Connected</h2>
          <div className="space-y-4">
            {[...organizationIntegrations]
              .sort((a, b) =>
                (a.metadata?.name || a.spec?.integrationName || "").localeCompare(
                  b.metadata?.name || b.spec?.integrationName || "",
                ),
              )
              .map((integration) => {
                const integrationDefinition = availableIntegrations.find(
                  (a) => a.name === integration.spec?.integrationName,
                );
                const integrationLabel =
                  integrationDefinition?.label ||
                  getIntegrationTypeDisplayName(undefined, integration.spec?.integrationName) ||
                  integration.spec?.integrationName;
                const integrationDisplayName =
                  integration.metadata?.name ||
                  getIntegrationTypeDisplayName(undefined, integration.spec?.integrationName) ||
                  integration.spec?.integrationName;
                const integrationName = integrationDefinition?.name || integration.spec?.integrationName;
                const statusLabel = integration.status?.state
                  ? integration.status.state.charAt(0).toUpperCase() + integration.status.state.slice(1)
                  : "Unknown";

                return (
                  <div
                    key={integration.metadata?.id}
                    className="bg-white border border-gray-300 dark:border-gray-700 rounded-md p-4 flex items-start justify-between gap-4"
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 flex h-4 w-4 items-center justify-center">
                        <IntegrationIcon
                          integrationName={integrationName}
                          iconSlug={integrationDefinition?.icon}
                          className="w-4 h-4 text-gray-500 dark:text-gray-400"
                        />
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                          {integrationDisplayName}
                        </h3>
                        {integrationLabel && integrationDisplayName !== integrationLabel ? (
                          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">Type: {integrationLabel}</p>
                        ) : null}
                        {integrationDefinition?.description ? (
                          <p className="mt-1 text-sm text-gray-800 dark:text-gray-400">
                            {integrationDefinition.description}
                          </p>
                        ) : null}
                      </div>
                    </div>
                    <div className="flex items-start gap-6">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                          integration.status?.state === "ready"
                            ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                            : integration.status?.state === "error"
                              ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                              : "bg-orange-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                        }`}
                      >
                        {statusLabel}
                      </span>
                      <PermissionTooltip
                        allowed={canUpdateIntegrations || permissionsLoading}
                        message="You don't have permission to update integrations."
                      >
                        <Button
                          variant="outline"
                          onClick={() => {
                            if (!canUpdateIntegrations) return;
                            navigate(`/${organizationId}/settings/integrations/${integration.metadata?.id}`, {
                              state: { tab: "configuration" },
                            });
                          }}
                          className="text-sm py-1.5 self-start"
                          disabled={!canUpdateIntegrations}
                        >
                          Configure...
                        </Button>
                      </PermissionTooltip>
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {/* Available Integrations */}
      <div>
        <h2 className="text-lg font-medium mb-4">Available</h2>
        <div>
          {availableIntegrations.length === 0 ? (
            <div className="text-center py-12">
              <AppWindow className="w-6 h-6 text-gray-800 mx-auto mb-2" />
              <p className="text-sm text-gray-800">No integrations available.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {[...availableIntegrations]
                .sort((a, b) => (a.label || a.name || "").localeCompare(b.label || b.name || ""))
                .map((app) => {
                  const appName = app.name;
                  return (
                    <div
                      key={app.name}
                      className="bg-white border border-gray-300 dark:border-gray-700 rounded-md p-4 flex items-start justify-between gap-4"
                    >
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 flex h-4 w-4 items-center justify-center">
                          <IntegrationIcon
                            integrationName={appName}
                            iconSlug={app.icon}
                            className="w-4 h-4 text-gray-500 dark:text-gray-400"
                          />
                        </div>
                        <div>
                          <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
                            {app.label || app.name}
                          </h3>
                          {app.description ? (
                            <p className="mt-1 text-sm text-gray-800 dark:text-gray-400">{app.description}</p>
                          ) : null}
                        </div>
                      </div>

                      <PermissionTooltip
                        allowed={canCreateIntegrations || permissionsLoading}
                        message="You don't have permission to connect integrations."
                      >
                        <Button
                          color="blue"
                          onClick={() => handleConnectClick(app)}
                          className="text-sm py-1.5 self-start"
                          disabled={!canCreateIntegrations}
                        >
                          Connect
                        </Button>
                      </PermissionTooltip>
                    </div>
                  );
                })}
            </div>
          )}
        </div>
      </div>

      {/* Connect Modal */}
      {isModalOpen &&
        selectedIntegration &&
        (() => {
          const integrationName = selectedIntegration.name;
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto">
                <div className="p-6">
                  <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                      <IntegrationIcon
                        integrationName={integrationName}
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
                        <span className="text-gray-800 ml-1">*</span>
                      </Label>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                        A unique name for this integration
                      </p>
                      <Input
                        type="text"
                        value={integrationName}
                        onChange={(e) => setIntegrationName(e.target.value)}
                        placeholder="e.g., my-app-integration"
                        required
                        disabled={!canCreateIntegrations}
                      />
                    </div>

                    {/* Configuration Fields */}
                    {selectedIntegration.configuration && selectedIntegration.configuration.length > 0 && (
                      <div className="border-t border-gray-200 dark:border-gray-700 pt-6 space-y-4">
                        {selectedIntegration.configuration.map((field) => {
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
                            />
                          );
                        })}
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

                  {createIntegrationMutation.isError && (
                    <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                      <p className="text-sm text-red-800 dark:text-red-200">
                        Failed to create integration: {getApiErrorMessage(createIntegrationMutation.error)}
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
