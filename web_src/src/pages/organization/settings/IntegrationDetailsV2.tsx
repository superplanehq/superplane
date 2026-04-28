import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsCapabilityDefinition,
} from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { usePermissions } from "@/contexts/PermissionsContext";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useIntegration,
  useUpdateIntegrationCapabilities,
} from "@/hooks/useIntegrations";
import { Alert, AlertDescription } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { CopyButton } from "@/ui/CopyButton";
import { getApiErrorMessage } from "@/lib/errors";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { usePageTitle } from "@/hooks/usePageTitle";
import { ArrowLeft, CircleX, ExternalLink, Loader2, Plug, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

interface IntegrationDetailsV2Props {
  organizationId: string;
}

const defaultCapabilityState: IntegrationCapabilityStateState = "STATE_UNAVAILABLE";

type DisplayCapability = {
  name: string;
  definition?: IntegrationsCapabilityDefinition;
  state: IntegrationCapabilityStateState;
};

const getCapabilityLabel = (capability: DisplayCapability) => {
  return capability.definition?.label || capability.definition?.name || capability.name || "Unnamed capability";
};

const getCapabilityDescription = (capability: DisplayCapability) => {
  return capability.definition?.description;
};

const getCapabilityStatusDotClass = (state: IntegrationCapabilityStateState) => {
  if (state === "STATE_ENABLED") return "bg-green-500";
  if (state === "STATE_DISABLED") return "bg-red-500";
  if (state === "STATE_REQUESTED") return "bg-amber-500";
  return "bg-gray-400 dark:bg-gray-500";
};

/** Matches component sidebar tab row (`ui/componentSidebar/index.tsx`). */
const sidebarTabButtonClass = "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors";

const sidebarTabActiveClass = "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600";

const sidebarTabInactiveClass =
  "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300";

export function IntegrationDetailsV2({ organizationId }: IntegrationDetailsV2Props) {
  const navigate = useNavigate();
  const { integrationId } = useParams<{ integrationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");
  usePageTitle(["Integrations", integration?.metadata?.name]);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [activeTab, setActiveTab] = useState("capabilities");
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration
    ? availableIntegrations.find((i) => i.name === integration.metadata?.integrationName)
    : undefined;

  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");
  const updateCapabilitiesMutation = useUpdateIntegrationCapabilities(organizationId, integrationId || "");
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const [capabilityStates, setCapabilityStates] = useState<Record<string, IntegrationCapabilityStateState>>({});

  useEffect(() => {
    const nextStates: Record<string, IntegrationCapabilityStateState> = {};
    (integration?.status?.capabilities || []).forEach((capability) => {
      if (!capability.name) return;
      nextStates[capability.name] = capability.state || defaultCapabilityState;
    });
    setCapabilityStates(nextStates);
  }, [integration?.status?.capabilities]);

  const webhookUrl = useMemo(() => {
    if (!integration?.status?.metadata || typeof integration.status.metadata.webhookUrl !== "string") {
      return null;
    }
    return integration.status.metadata.webhookUrl;
  }, [integration?.status?.metadata]);

  const webhookConfigured = useMemo(() => {
    return (
      integration?.status?.metadata &&
      typeof integration.status.metadata.webhookSigningSecretConfigured === "boolean" &&
      integration.status.metadata.webhookSigningSecretConfigured === true
    );
  }, [integration?.status?.metadata?.webhookSigningSecretConfigured]);

  useEffect(() => {
    if (!webhookUrl && activeTab === "webhook") {
      setActiveTab("capabilities");
    }
  }, [webhookUrl, activeTab]);

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
      getCapabilityLabel(left).localeCompare(getCapabilityLabel(right)),
    );
  }, [integration?.status?.capabilities, integrationDef?.capabilities]);

  const stagedCapabilityUpdates = useMemo(() => {
    return capabilities.reduce<IntegrationCapabilityState[]>((updates, capability) => {
      if (!capability.name) return updates;
      const serverState = capability.state || defaultCapabilityState;
      const effectiveState = capabilityStates[capability.name] ?? serverState;
      if (effectiveState === serverState) return updates;
      if (
        effectiveState !== "STATE_ENABLED" &&
        effectiveState !== "STATE_DISABLED" &&
        effectiveState !== "STATE_REQUESTED"
      ) {
        return updates;
      }
      updates.push({ name: capability.name, state: effectiveState });
      return updates;
    }, []);
  }, [capabilities, capabilityStates]);

  const handleDelete = async () => {
    if (!canDeleteIntegrations) return;
    try {
      await deleteMutation.mutateAsync();
      navigate(`/${organizationId}/settings/integrations`);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  const queueCapabilityStateChange = (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => {
    if (!canUpdateIntegrations || !capability.name || updateCapabilitiesMutation.isPending) return;
    setCapabilityStates((previous) => ({
      ...previous,
      [capability.name!]: nextState,
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
          <h4 className="flex items-center gap-2 text-2xl font-medium">
            <span
              className="inline-flex shrink-0"
              title={
                (integration.status?.state || "unknown").charAt(0).toUpperCase() +
                (integration.status?.state || "unknown").slice(1)
              }
            >
              <Plug
                className={`h-5 w-5 ${
                  integration.status?.state === "ready"
                    ? "text-green-500"
                    : integration.status?.state === "error"
                      ? "text-red-600"
                      : "text-amber-600"
                }`}
                aria-label={`Integration status: ${integration.status?.state || "unknown"}`}
              />
            </span>
            <span>
              {integration.metadata?.name ||
                getIntegrationTypeDisplayName(undefined, integration.metadata?.integrationName) ||
                integration.metadata?.integrationName}
            </span>
          </h4>
          {integration.metadata?.id ? (
            <div className="mt-1.5 flex max-w-full items-center gap-1.5">
              <span className="min-w-0 truncate font-mono text-xs text-gray-700 dark:text-gray-300">
                {integration.metadata.id}
              </span>
              <CopyButton text={integration.metadata.id} />
            </div>
          ) : null}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <PermissionTooltip
            allowed={canDeleteIntegrations || permissionsLoading}
            message="You don't have permission to delete integrations."
          >
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="shrink-0 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
              aria-label="Delete integration"
              disabled={!canDeleteIntegrations}
              onClick={() => {
                if (!canDeleteIntegrations) return;
                setShowDeleteConfirm(true);
              }}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </PermissionTooltip>
        </div>
      </div>

      <div className="space-y-6">
        {integration.status?.state === "error" && integration.status?.stateDescription && (
          <Alert variant="destructive" className="[&>svg+div]:translate-y-0 [&>svg]:top-[14px]">
            <CircleX className="h-4 w-4" />
            <AlertDescription>{integration.status.stateDescription}</AlertDescription>
          </Alert>
        )}

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="border-border border-b-1">
            <div className="flex flex-wrap px-4">
              <button
                type="button"
                onClick={() => setActiveTab("capabilities")}
                className={cn(
                  sidebarTabButtonClass,
                  activeTab === "capabilities" ? sidebarTabActiveClass : sidebarTabInactiveClass,
                )}
              >
                Capabilities
              </button>
              {webhookUrl ? (
                <button
                  type="button"
                  onClick={() => setActiveTab("webhook")}
                  className={cn(
                    sidebarTabButtonClass,
                    activeTab === "webhook" ? sidebarTabActiveClass : sidebarTabInactiveClass,
                  )}
                >
                  Webhook
                </button>
              ) : null}
              <button
                type="button"
                onClick={() => setActiveTab("usage")}
                className={cn(
                  sidebarTabButtonClass,
                  activeTab === "usage" ? sidebarTabActiveClass : sidebarTabInactiveClass,
                )}
              >
                Usage
              </button>
            </div>
          </div>

          <TabsContent value="capabilities" className="mt-4">
            {capabilities.length > 0 ? (
              <>
                <div className="overflow-x-auto rounded-md border border-gray-300 dark:border-gray-700">
                  <table className="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
                    <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                      {capabilities.map((capability, index) => {
                        const serverState = capability.state || defaultCapabilityState;
                        const effectiveState = capabilityStates[capability.name] ?? serverState;
                        const statusDotClass = getCapabilityStatusDotClass(effectiveState);
                        const actionDisabled = !canUpdateIntegrations || updateCapabilitiesMutation.isPending;
                        const isDirty = effectiveState !== serverState;
                        const description = getCapabilityDescription(capability);

                        return (
                          <tr
                            key={capability.name || `capability-${index}`}
                            className={cn(isDirty && "bg-amber-50 dark:bg-amber-950/25")}
                          >
                            <td className="px-4 py-3 align-middle">
                              <div className="flex flex-wrap items-center gap-2">
                                <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)} aria-hidden />
                                <span className="font-mono text-sm text-gray-800 dark:text-gray-100">
                                  {capability.name}
                                </span>
                                <CopyButton text={capability.name} />
                              </div>
                            </td>
                            <td className="px-4 py-3 align-middle">
                              {description ? (
                                <div className="text-sm text-gray-600 dark:text-gray-400">{description}</div>
                              ) : null}
                            </td>
                            <td className="px-4 py-3 align-middle text-right">
                              <PermissionTooltip
                                allowed={canUpdateIntegrations || permissionsLoading}
                                message="You don't have permission to update integrations."
                              >
                                <span className="flex justify-end">
                                  {effectiveState === "STATE_ENABLED" ? (
                                    <Button
                                      type="button"
                                      variant="outline"
                                      size="sm"
                                      disabled={actionDisabled}
                                      onClick={() => queueCapabilityStateChange(capability, "STATE_DISABLED")}
                                    >
                                      Disable
                                    </Button>
                                  ) : null}
                                  {effectiveState === "STATE_DISABLED" ? (
                                    <Button
                                      type="button"
                                      variant="outline"
                                      size="sm"
                                      disabled={actionDisabled}
                                      onClick={() => queueCapabilityStateChange(capability, "STATE_ENABLED")}
                                    >
                                      Enable
                                    </Button>
                                  ) : null}
                                  {effectiveState === "STATE_UNAVAILABLE" ? (
                                    <Button
                                      type="button"
                                      variant="outline"
                                      size="sm"
                                      disabled={actionDisabled}
                                      onClick={() => queueCapabilityStateChange(capability, "STATE_REQUESTED")}
                                    >
                                      Request
                                    </Button>
                                  ) : null}
                                  {effectiveState === "STATE_REQUESTED" ? (
                                    <Button type="button" variant="outline" size="sm" disabled>
                                      Requested
                                    </Button>
                                  ) : null}
                                </span>
                              </PermissionTooltip>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
                {stagedCapabilityUpdates.length > 0 ? (
                  <div className="mt-4 flex justify-end">
                    <LoadingButton
                      type="button"
                      color="blue"
                      onClick={() => void handleCapabilitiesSubmit()}
                      disabled={!canUpdateIntegrations}
                      loading={updateCapabilitiesMutation.isPending}
                      loadingText="Updating…"
                    >
                      Apply changes
                    </LoadingButton>
                  </div>
                ) : null}
              </>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">No capabilities available.</p>
            )}
          </TabsContent>

          {webhookUrl ? (
            <TabsContent value="webhook" className="mt-4">
              <div className="rounded-lg border border-gray-300 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
                <p className="mb-3 text-sm text-gray-600 dark:text-gray-400">
                  {webhookConfigured
                    ? "Webhook is configured. You can copy the URL below if needed."
                    : "Add this URL in your external service to receive webhooks."}
                </p>
                <div className="flex items-center gap-2">
                  <Input type="text" value={webhookUrl} readOnly className="flex-1 font-mono text-sm" />
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
            </TabsContent>
          ) : null}

          <TabsContent value="usage" className="mt-4">
            <div className="rounded-lg border border-gray-300 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
              {workflowGroups.length > 0 ? (
                <>
                  <p className="mb-3 text-sm text-gray-500 dark:text-gray-400">
                    This integration is currently used in the following canvases:
                  </p>
                  <div className="space-y-2">
                    {workflowGroups.map((group) => (
                      <button
                        key={group.canvasId}
                        type="button"
                        onClick={() => window.open(`/${organizationId}/canvases/${group.canvasId}`, "_blank")}
                        className="flex w-full items-center gap-2 rounded-md border border-gray-300 bg-gray-50 p-3 text-left transition-colors hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:bg-gray-800"
                      >
                        <div className="flex-1">
                          <p className="text-sm font-medium text-gray-800 dark:text-gray-100">
                            Canvas: {group.canvasName}
                          </p>
                          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                            Used in {group.nodes.length} node{group.nodes.length !== 1 ? "s" : ""}:{" "}
                            {group.nodes.map((node) => node.nodeName).join(", ")}
                          </p>
                        </div>
                        <ExternalLink className="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
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
          </TabsContent>
        </Tabs>
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
