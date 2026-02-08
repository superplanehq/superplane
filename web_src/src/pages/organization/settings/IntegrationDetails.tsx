import { ArrowLeft, ExternalLink, Loader2, Trash2 } from "lucide-react";
import { useNavigate, useParams, useLocation } from "react-router-dom";
import { useState, useEffect, useMemo } from "react";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useIntegration,
  useUpdateIntegration,
} from "@/hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/ui/tabs";
import type { ConfigurationField } from "@/api-client";
import { showErrorToast } from "@/utils/toast";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";

interface IntegrationDetailsProps {
  organizationId: string;
}

export function IntegrationDetails({ organizationId }: IntegrationDetailsProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { integrationId } = useParams<{ integrationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");

  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration
    ? availableIntegrations.find((i) => i.name === integration.spec?.integrationName)
    : undefined;

  const updateMutation = useUpdateIntegration(organizationId, integrationId || "");
  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");

  // Initialize config values when installation loads
  useEffect(() => {
    if (integration?.spec?.configuration) {
      setConfigValues(integration.spec.configuration);
    }
  }, [integration]);

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

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canUpdateIntegrations) return;
    try {
      await updateMutation.mutateAsync(configValues);
      navigate(`/${organizationId}/settings/integrations`);
    } catch (_error) {
      showErrorToast("Failed to update configuration");
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
    } catch (_error) {
      showErrorToast("Failed to delete integration");
    }
  };

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate(`/${organizationId}/settings/integrations`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
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
          <button
            onClick={() => navigate(`/${organizationId}/settings/integrations`)}
            className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <h4 className="text-2xl font-semibold">Integration Details</h4>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6">
          <p className="text-gray-500 dark:text-gray-400">Integration not found</p>
        </div>
      </div>
    );
  }

  const defaultTab = location.state?.tab === "configuration" ? "configuration" : "overview";

  return (
    <div className="pt-6">
      <div className="flex items-center gap-4 mb-6">
        <button
          onClick={() => navigate(`/${organizationId}/settings/integrations`)}
          className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <IntegrationIcon
          integrationName={integration?.spec?.integrationName}
          iconSlug={integrationDef?.icon}
          className="w-6 h-6"
        />
        <div className="flex-1">
          <h4 className="text-2xl font-semibold">
            {integration.metadata?.name ||
              getIntegrationTypeDisplayName(undefined, integration.spec?.integrationName) ||
              integration.spec?.integrationName}
          </h4>
          {integration.spec?.integrationName && integration.metadata?.name !== integration.spec?.integrationName && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Integration:{" "}
              {getIntegrationTypeDisplayName(undefined, integration.spec?.integrationName) ||
                integration.spec?.integrationName}
            </p>
          )}
        </div>
      </div>

      <Tabs defaultValue={defaultTab} className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              <h2 className="text-lg font-medium mb-4">Integration Details</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Integration ID</h3>
                  <p className="text-sm text-gray-800 dark:text-gray-100 font-mono">{integration.metadata?.id}</p>
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">State</h3>
                  <span
                    className={`inline-flex px-2 py-0.5 text-xs font-medium rounded ${
                      integration.status?.state === "ready"
                        ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                        : integration.status?.state === "error"
                          ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
                          : "bg-orange-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
                    }`}
                  >
                    {(integration.status?.state || "unknown").charAt(0).toUpperCase() +
                      (integration.status?.state || "unknown").slice(1)}
                  </span>
                  {integration.status?.stateDescription && (
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
                      {integration.status.stateDescription}
                    </p>
                  )}
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
                Once you delete this integration, all its data will be permanently deleted. This action cannot be
                undone.
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
        </TabsContent>

        <TabsContent value="configuration">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              {integration?.status?.browserAction && (
                <IntegrationInstructions
                  description={integration.status.browserAction.description}
                  onContinue={integration.status.browserAction.url ? handleBrowserAction : undefined}
                  className="mb-6"
                />
              )}

              {integrationDef?.configuration && integrationDef.configuration.length > 0 ? (
                <PermissionTooltip
                  allowed={canUpdateIntegrations || permissionsLoading}
                  message="You don't have permission to update integrations."
                  className="w-full"
                >
                  <form onSubmit={handleConfigSubmit} className="space-y-4">
                    {integrationDef.configuration.map((field: ConfigurationField) => (
                      <ConfigurationFieldRenderer
                        key={field.name}
                        field={field}
                        value={configValues[field.name!]}
                        onChange={(value) => setConfigValues({ ...configValues, [field.name!]: value })}
                        allValues={configValues}
                        domainId={organizationId}
                        domainType="DOMAIN_TYPE_ORGANIZATION"
                        organizationId={organizationId}
                        appInstallationId={integration?.metadata?.id}
                      />
                    ))}

                    <div className="flex items-center gap-3 pt-4">
                      <Button type="submit" color="blue" disabled={updateMutation.isPending || !canUpdateIntegrations}>
                        {updateMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            Saving...
                          </>
                        ) : (
                          "Save Configuration"
                        )}
                      </Button>
                      {updateMutation.isSuccess && (
                        <span className="text-sm text-green-600 dark:text-green-400">
                          Configuration updated successfully!
                        </span>
                      )}
                      {updateMutation.isError && (
                        <span className="text-sm text-red-600 dark:text-red-400">Failed to update configuration</span>
                      )}
                    </div>
                  </form>
                </PermissionTooltip>
              ) : (
                <p className="text-sm text-gray-500 dark:text-gray-400">No configuration fields available.</p>
              )}
            </div>
          </div>
        </TabsContent>
      </Tabs>

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
                <Button
                  color="blue"
                  onClick={handleDelete}
                  disabled={deleteMutation.isPending || !canDeleteIntegrations}
                  className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
                >
                  {deleteMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    "Delete"
                  )}
                </Button>
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
