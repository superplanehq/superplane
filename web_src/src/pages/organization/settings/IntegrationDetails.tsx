import type {
  ConfigurationField,
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsCapabilityDefinition,
} from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePermissions } from "@/contexts/PermissionsContext";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useIntegration,
  useUpdateIntegration,
  useUpdateIntegrationCapabilities,
} from "@/hooks/useIntegrations";
import { Alert, AlertDescription } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { Switch } from "@/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { usePageTitle } from "@/hooks/usePageTitle";
import { ArrowLeft, CircleX, ExternalLink, Loader2, Plug, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { renderIntegrationMetadata } from "./integrationMetadataRenderers";

interface IntegrationDetailsProps {
  organizationId: string;
}

const defaultCapabilityState: IntegrationCapabilityStateState = "STATE_UNAVAILABLE";

type DisplayCapability = {
  name: string;
  definition?: IntegrationsCapabilityDefinition;
  state: IntegrationCapabilityStateState;
};

const getCapabilityDisplayName = (capability: DisplayCapability) => {
  return capability.definition?.label || capability.definition?.name || capability.name || "Unnamed capability";
};

const getCapabilityDescription = (capability: DisplayCapability) => {
  return capability.definition?.description;
};

const getCapabilityTypeLabel = (capability: DisplayCapability) => {
  if (capability.definition?.type === "TYPE_ACTION") return "Action";
  if (capability.definition?.type === "TYPE_TRIGGER") return "Trigger";
  return "Unknown";
};

const getCapabilityStateLabel = (state: IntegrationCapabilityStateState | undefined) => {
  if (state === "STATE_ENABLED") return "Enabled";
  if (state === "STATE_DISABLED") return "Disabled";
  if (state === "STATE_REQUESTED") return "Requested";
  return "Unavailable";
};

const getCapabilityStateStyles = (state: IntegrationCapabilityStateState | undefined) => {
  if (state === "STATE_ENABLED") {
    return "bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-300";
  }

  if (state === "STATE_DISABLED") {
    return "bg-amber-100 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300";
  }

  if (state === "STATE_REQUESTED") {
    return "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300";
  }

  return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300";
};

export function IntegrationDetails({ organizationId }: IntegrationDetailsProps) {
  const navigate = useNavigate();
  const { integrationId } = useParams<{ integrationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");
  usePageTitle(["Integrations", integration?.metadata?.name]);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [integrationName, setIntegrationName] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration
    ? availableIntegrations.find((i) => i.name === integration.metadata?.integrationName)
    : undefined;

  const updateMutation = useUpdateIntegration(organizationId, integrationId || "");
  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");
  const updateCapabilitiesMutation = useUpdateIntegrationCapabilities(organizationId, integrationId || "");
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const [capabilityStates, setCapabilityStates] = useState<Record<string, IntegrationCapabilityStateState>>({});

  // Initialize config values when installation loads
  const [configLoaded, setConfigLoaded] = useState(false);
  useEffect(() => {
    if (integration?.spec?.configuration) {
      setConfigValues(integration.spec.configuration);
      setConfigLoaded(true);
    }
  }, [integration]);

  useEffect(() => {
    setIntegrationName(integration?.metadata?.name || integration?.metadata?.integrationName || "");
  }, [integration?.metadata?.name, integration?.metadata?.integrationName]);

  useEffect(() => {
    const nextStates: Record<string, IntegrationCapabilityStateState> = {};
    (integration?.status?.capabilities || []).forEach((capability) => {
      if (!capability.name) return;
      nextStates[capability.name] = capability.state || defaultCapabilityState;
    });
    setCapabilityStates(nextStates);
  }, [integration?.status?.capabilities]);

  // Full instructions (same for all integrations)
  const instructionsContent = useMemo(() => {
    const raw = integrationDef?.instructions?.trim();
    if (!raw) return null;
    return <IntegrationInstructions description={raw} />;
  }, [integrationDef?.instructions]);

  // Webhook block: show when integration exposes a webhook URL in metadata (generic, no integration name check)
  const webhookSection = useMemo(() => {
    const webhookUrl =
      integration?.status?.metadata && typeof integration.status.metadata.webhookUrl === "string"
        ? integration.status.metadata.webhookUrl
        : null;
    if (!webhookUrl) return null;
    const webhookConfigured =
      integration?.status?.metadata &&
      typeof integration.status.metadata.webhookSigningSecretConfigured === "boolean" &&
      integration.status.metadata.webhookSigningSecretConfigured === true;
    return (
      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
        <div className="p-6">
          <h2 className="text-lg font-medium mb-4">Webhook</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
            {webhookConfigured
              ? "Webhook is configured. You can copy the URL below if needed."
              : "Add this URL in your external service to receive webhooks. Complete any setup steps described in the instructions above."}
          </p>
          <div className="flex items-center gap-2">
            <Input type="text" value={webhookUrl} readOnly className="font-mono text-sm flex-1" />
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={async () => {
                await navigator.clipboard.writeText(webhookUrl);
                showSuccessToast("Webhook URL copied");
              }}
            >
              Copy
            </Button>
          </div>
        </div>
      </div>
    );
  }, [integration?.status?.metadata?.webhookUrl, integration?.status?.metadata?.webhookSigningSecretConfigured]);

  // Group usedIn nodes by workflow
  const workflowGroups = useMemo(() => {
    if (!integration?.status?.usedIn) return [];

    const groups = new Map<string, { canvasName: string; nodes: Array<{ nodeId: string; nodeName: string }> }>();
    integration.status.usedIn.forEach((nodeRef) => {
      const canvasId = nodeRef.canvasId || "";
      const canvasName = nodeRef.canvasName || canvasId;
      const nodeId = nodeRef.nodeId || "";
      const nodeName = nodeRef.nodeName || nodeId;

      if (!groups.has(canvasId)) {
        groups.set(canvasId, { canvasName, nodes: [] });
      }
      groups.get(canvasId)?.nodes.push({ nodeId, nodeName });
    });

    return Array.from(groups.entries()).map(([canvasId, data]) => ({
      canvasId,
      canvasName: data.canvasName,
      nodes: data.nodes,
    }));
  }, [integration?.status?.usedIn]);

  const metadataContent = useMemo(
    () => renderIntegrationMetadata(integration?.metadata?.integrationName, integration!),
    [integration],
  );

  const capabilities = useMemo(() => {
    const byName = new Map<string, DisplayCapability>();

    (integrationDef?.capabilities || []).forEach((definition) => {
      if (!definition.name) return;
      byName.set(definition.name, {
        name: definition.name,
        definition,
        state: defaultCapabilityState,
      });
    });

    (integration?.status?.capabilities || []).forEach((capability) => {
      if (!capability.name) return;
      const existing = byName.get(capability.name);
      byName.set(capability.name, {
        name: capability.name,
        definition: existing?.definition,
        state: capability.state || defaultCapabilityState,
      });
    });

    return Array.from(byName.values()).sort((left, right) =>
      getCapabilityDisplayName(left).localeCompare(getCapabilityDisplayName(right)),
    );
  }, [integration?.status?.capabilities, integrationDef?.capabilities]);

  const stagedCapabilityUpdates = useMemo(() => {
    return capabilities.reduce<IntegrationCapabilityState[]>((updates, capability) => {
      const currentState = capabilityStates[capability.name] || capability.state || defaultCapabilityState;
      if (currentState === capability.state) return updates;
      if (currentState !== "STATE_ENABLED" && currentState !== "STATE_DISABLED") return updates;

      updates.push({
        name: capability.name,
        state: currentState,
      });
      return updates;
    }, []);
  }, [capabilities, capabilityStates]);

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canUpdateIntegrations) return;

    const nextName = integrationName.trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      await updateMutation.mutateAsync({
        name: nextName,
        configuration: configValues,
      });
      showSuccessToast("Integration saved");
    } catch {
      showErrorToast("Failed to update integration");
    }
  };

  const handleBrowserAction = () => {
    if (!integration?.status?.browserAction) return;

    const { url, method, formFields } = integration.status.browserAction;

    if (method?.toUpperCase() === "POST" && formFields) {
      // Create a hidden form and submit it
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";

      // Add form fields
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
    } else {
      // For GET requests or no form fields, just open the URL
      if (url) {
        window.open(url, "_blank");
      }
    }
  };

  const handleDelete = async () => {
    if (!canDeleteIntegrations) return;
    try {
      await deleteMutation.mutateAsync();
      navigate(`/${organizationId}/settings/integrations`);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  const handleCapabilityToggle = (capability: DisplayCapability, enabled: boolean) => {
    if (!canUpdateIntegrations) return;

    const currentState = capabilityStates[capability.name] || capability.state || defaultCapabilityState;
    if (currentState === "STATE_UNAVAILABLE" || currentState === "STATE_REQUESTED") return;

    if ((enabled && currentState === "STATE_ENABLED") || (!enabled && currentState === "STATE_DISABLED")) {
      return;
    }

    const nextState: IntegrationCapabilityStateState = enabled ? "STATE_ENABLED" : "STATE_DISABLED";

    setCapabilityStates((previous) => ({
      ...previous,
      [capability.name]: nextState,
    }));
  };

  const handleCapabilitiesSubmit = async () => {
    if (!canUpdateIntegrations || stagedCapabilityUpdates.length === 0) return;

    try {
      await updateCapabilitiesMutation.mutateAsync(stagedCapabilityUpdates);
      showSuccessToast("Integration capabilities updated");
    } catch (_error) {
      showErrorToast(`Failed to update capabilities: ${getApiErrorMessage(_error)}`);
    }
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <Link
            to={integrationsHref}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="flex justify-center items-center h-32">
          <Loader2 className="w-8 h-8 animate-spin text-gray-500 dark:text-gray-400" />
        </div>
      </div>
    );
  }

  if (error || !integration) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <Link
            to={integrationsHref}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
            aria-label="Back to integrations"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <p className="text-gray-500 dark:text-gray-400">Integration not found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <Link
          to={integrationsHref}
          className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          aria-label="Back to integrations"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <IntegrationIcon
          integrationName={integration?.metadata?.integrationName}
          iconSlug={integrationDef?.icon}
          className="w-6 h-6"
        />
        <div className="flex-1 min-w-[200px]">
          <h4 className="text-2xl font-medium">
            {integration.metadata?.name ||
              getIntegrationTypeDisplayName(undefined, integration.metadata?.integrationName) ||
              integration.metadata?.integrationName}
          </h4>
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <Plug
            className={`w-4 h-4 ${
              integration.status?.state === "ready"
                ? "text-green-500"
                : integration.status?.state === "error"
                  ? "text-red-600"
                  : "text-amber-600"
            }`}
          />
          <span
            className={`text-sm font-medium ${
              integration.status?.state === "ready"
                ? "text-green-500"
                : integration.status?.state === "error"
                  ? "text-red-600"
                  : "text-amber-600"
            }`}
          >
            {(integration.status?.state || "unknown").charAt(0).toUpperCase() +
              (integration.status?.state || "unknown").slice(1)}
          </span>
        </div>
      </div>

      <div className="space-y-6">
        {integration.status?.state === "error" && integration.status?.stateDescription && (
          <Alert variant="destructive" className="[&>svg+div]:translate-y-0 [&>svg]:top-[14px]">
            <CircleX className="h-4 w-4" />
            <AlertDescription>{integration.status.stateDescription}</AlertDescription>
          </Alert>
        )}

        {integration?.status?.browserAction && (
          <IntegrationInstructions
            description={integration.status.browserAction.description}
            onContinue={integration.status.browserAction.url ? handleBrowserAction : undefined}
          />
        )}

        {instructionsContent}
        {metadataContent}

        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Configuration</h2>
            {integrationDef?.configuration && integrationDef.configuration.length > 0 ? (
              <PermissionTooltip
                allowed={canUpdateIntegrations || permissionsLoading}
                message="You don't have permission to update integrations."
                className="w-full"
              >
                <form onSubmit={handleConfigSubmit} className="space-y-4">
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
                      disabled={!canUpdateIntegrations}
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">A unique name for this integration</p>
                  </div>

                  {configLoaded &&
                    integrationDef.configuration.map((field: ConfigurationField) => (
                      <ConfigurationFieldRenderer
                        key={field.name}
                        field={field}
                        value={configValues[field.name!]}
                        onChange={(value) => setConfigValues({ ...configValues, [field.name!]: value })}
                        allValues={configValues}
                        domainId={organizationId}
                        domainType="DOMAIN_TYPE_ORGANIZATION"
                        organizationId={organizationId}
                        integrationId={integration?.metadata?.id}
                      />
                    ))}

                  <div className="flex items-center gap-3 pt-4">
                    <LoadingButton
                      type="submit"
                      color="blue"
                      disabled={!integrationName.trim() || !canUpdateIntegrations}
                      loading={updateMutation.isPending}
                      loadingText="Saving..."
                    >
                      Save
                    </LoadingButton>
                    {updateMutation.isError && (
                      <span className="text-sm text-red-600 dark:text-red-400">
                        Failed to update integration: {getApiErrorMessage(updateMutation.error)}
                      </span>
                    )}
                  </div>
                </form>
              </PermissionTooltip>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">No configuration fields available.</p>
            )}
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Capabilities</h2>
            {capabilities.length > 0 ? (
              <>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                  Enable or disable exposed capabilities for this integration.
                </p>
                <div className="overflow-x-auto rounded-md border border-gray-300 dark:border-gray-700">
                  <table className="w-full min-w-[720px] divide-y divide-gray-200 dark:divide-gray-800">
                    <thead className="bg-gray-50 dark:bg-gray-800/60">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                          Capability
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                          Type
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                          State
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                          Enabled
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                      {capabilities.map((capability, index) => {
                        const capabilityState =
                          (capability.name ? capabilityStates[capability.name] : undefined) ||
                          capability.state ||
                          defaultCapabilityState;
                        const isUnavailable = capabilityState === "STATE_UNAVAILABLE";
                        const isRequested = capabilityState === "STATE_REQUESTED";
                        const isEnabled = capabilityState === "STATE_ENABLED";
                        const disabled =
                          isUnavailable ||
                          isRequested ||
                          updateCapabilitiesMutation.isPending ||
                          !canUpdateIntegrations;
                        const isChanged = capabilityState !== capability.state;
                        const description = getCapabilityDescription(capability);

                        return (
                          <tr key={capability.name || `capability-${index}`}>
                            <td className="px-4 py-4 align-top">
                              <div className="flex flex-col gap-1">
                                <div className="flex flex-wrap items-center gap-2">
                                  <span className="text-sm font-medium text-gray-800 dark:text-gray-100">
                                    {getCapabilityDisplayName(capability)}
                                  </span>
                                  {isChanged && (
                                    <span className="inline-flex items-center rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300">
                                      Unsaved
                                    </span>
                                  )}
                                </div>
                                {description && (
                                  <span className="text-sm text-gray-500 dark:text-gray-400">{description}</span>
                                )}
                                {isUnavailable && (
                                  <span className="text-xs text-gray-500 dark:text-gray-400">
                                    This capability is currently unavailable for this integration setup.
                                  </span>
                                )}
                                {isRequested && (
                                  <span className="text-xs text-gray-500 dark:text-gray-400">
                                    This capability was requested but is not available yet.
                                  </span>
                                )}
                              </div>
                            </td>
                            <td className="px-4 py-4 align-top">
                              <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                                {getCapabilityTypeLabel(capability)}
                              </span>
                            </td>
                            <td className="px-4 py-4 align-top">
                              <span
                                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${getCapabilityStateStyles(
                                  capabilityState,
                                )}`}
                              >
                                {getCapabilityStateLabel(capabilityState)}
                              </span>
                            </td>
                            <td className="px-4 py-4 align-top">
                              <div className="flex justify-end">
                                <PermissionTooltip
                                  allowed={canUpdateIntegrations || permissionsLoading}
                                  message="You don't have permission to update integrations."
                                >
                                  <Switch
                                    checked={isEnabled}
                                    onCheckedChange={(checked) => handleCapabilityToggle(capability, checked)}
                                    disabled={disabled}
                                    aria-label={`Toggle ${getCapabilityDisplayName(capability)}`}
                                  />
                                </PermissionTooltip>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
                {stagedCapabilityUpdates.length > 0 && (
                  <div className="mt-4 flex justify-end">
                    <LoadingButton
                      type="button"
                      color="blue"
                      onClick={() => void handleCapabilitiesSubmit()}
                      disabled={!canUpdateIntegrations}
                      loading={updateCapabilitiesMutation.isPending}
                      loadingText="Updating..."
                    >
                      Update Capabilities
                    </LoadingButton>
                  </div>
                )}
              </>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">No capabilities available.</p>
            )}
          </div>
        </div>

        {webhookSection}

        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Integration Details</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Integration ID</h3>
                <p className="text-sm text-gray-800 dark:text-gray-100 font-mono">{integration.metadata?.id}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Used By */}
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
          <div className="p-6">
            <h2 className="text-lg font-medium mb-4">Used By</h2>
            {workflowGroups.length > 0 ? (
              <>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-3">
                  This integration is currently used in the following canvases:
                </p>
                <div className="space-y-2">
                  {workflowGroups.map((group) => (
                    <button
                      key={group.canvasId}
                      onClick={() => window.open(`/${organizationId}/canvases/${group.canvasId}`, "_blank")}
                      className="w-full flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-md border border-gray-300 dark:border-gray-700 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors text-left"
                    >
                      <div className="flex-1">
                        <p className="text-sm font-medium text-gray-800 dark:text-gray-100">
                          Canvas: {group.canvasName}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          Used in {group.nodes.length} node{group.nodes.length !== 1 ? "s" : ""}:{" "}
                          {group.nodes.map((node) => node.nodeName).join(", ")}
                        </p>
                      </div>
                      <ExternalLink className="w-4 h-4 text-gray-400 dark:text-gray-500 shrink-0" />
                    </button>
                  ))}
                </div>
              </>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                This integration is not used in any workflow yet.
              </p>
            )}
          </div>
        </div>

        {/* Danger Zone */}
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-red-200 dark:border-red-800">
          <div className="p-6">
            <h2 className="text-lg font-medium text-red-600 dark:text-red-400 mb-2">Danger Zone</h2>
            <p className="text-sm text-gray-800 dark:text-gray-100 mb-4">
              Once you delete this integration, all its data will be permanently deleted. This action cannot be undone.
            </p>
            <PermissionTooltip
              allowed={canDeleteIntegrations || permissionsLoading}
              message="You don't have permission to delete integrations."
            >
              <Button
                variant="outline"
                onClick={() => {
                  if (!canDeleteIntegrations) return;
                  setShowDeleteConfirm(true);
                }}
                className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 hover:text-red-600 dark:hover:text-red-400 gap-1"
                disabled={!canDeleteIntegrations}
              >
                <Trash2 className="w-4 h-4" />
                Delete Integration
              </Button>
            </PermissionTooltip>
          </div>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-2">
                Delete {integration?.metadata?.name || "integration"}?
              </h3>
              <p className="text-sm text-gray-800 dark:text-gray-100 mb-6">
                This cannot be undone. All data will be permanently deleted.
              </p>
              <div className="flex justify-start gap-3">
                <LoadingButton
                  color="blue"
                  onClick={handleDelete}
                  disabled={!canDeleteIntegrations}
                  loading={deleteMutation.isPending}
                  loadingText="Deleting..."
                  className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
                >
                  Delete
                </LoadingButton>
                <Button
                  variant="outline"
                  onClick={() => setShowDeleteConfirm(false)}
                  disabled={deleteMutation.isPending}
                >
                  Cancel
                </Button>
              </div>
              {deleteMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                  <p className="text-sm text-red-800 dark:text-red-200">
                    Failed to delete integration. Please try again.
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
