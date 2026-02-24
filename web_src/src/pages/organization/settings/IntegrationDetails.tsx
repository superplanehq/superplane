import {
  ArrowLeft,
  Check,
  CircleX,
  Copy,
  ExternalLink,
  Loader2,
  Pencil,
  Plug,
  Settings2,
  Trash2,
  TriangleAlert,
  Workflow,
  X,
} from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { useState, useEffect, useMemo } from "react";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useIntegration,
  useInvokeIntegrationAction,
  useUpdateIntegration,
} from "@/hooks/useIntegrations";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import type { ConfigurationField } from "@/api-client";
import { showErrorToast } from "@/utils/toast";
import { getApiErrorMessage } from "@/utils/errors";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { executeSetupRedirectAction } from "@/utils/setupInstructions";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";
import { Alert, AlertDescription } from "@/ui/alert";

interface IntegrationDetailsProps {
  organizationId: string;
}

export function IntegrationDetails({ organizationId }: IntegrationDetailsProps) {
  const navigate = useNavigate();
  const { integrationId } = useParams<{ integrationId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});
  const [isEditingName, setIsEditingName] = useState(false);
  const [nameDraft, setNameDraft] = useState("");
  const [integrationIdCopied, setIntegrationIdCopied] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [activeInstructionAction, setActiveInstructionAction] = useState<number | null>(null);
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");

  const { data: integration, isLoading, error } = useIntegration(organizationId, integrationId || "");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration
    ? availableIntegrations.find((i) => i.name === integration.spec?.integrationName)
    : undefined;
  const displayName =
    integration?.metadata?.name ||
    getIntegrationTypeDisplayName(undefined, integration?.spec?.integrationName) ||
    integration?.spec?.integrationName ||
    "";

  const updateMutation = useUpdateIntegration(organizationId, integrationId || "");
  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");
  const invokeActionMutation = useInvokeIntegrationAction(organizationId, integrationId || "");

  // Initialize config values when installation loads
  useEffect(() => {
    if (integration?.spec?.configuration) {
      setConfigValues(integration.spec.configuration);
    }
  }, [integration]);

  useEffect(() => {
    if (!isEditingName) {
      setNameDraft(displayName);
    }
  }, [displayName, isEditingName]);

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
  const hasActionCallSetupAction = Boolean(
    integration?.status?.instruction?.actions?.some((action) => Boolean(action.actionCall)),
  );

  const handleConfigSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canUpdateIntegrations) return;

    const nextName = (integration?.metadata?.name || integration?.spec?.integrationName || "").trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      await updateMutation.mutateAsync({
        name: nextName,
        configuration: configValues,
      });
    } catch (_error) {
      showErrorToast("Failed to update integration");
    }
  };

  const handleStartNameEdit = () => {
    if (!canUpdateIntegrations) return;
    setNameDraft(displayName);
    setIsEditingName(true);
  };

  const handleCancelNameEdit = () => {
    setNameDraft(displayName);
    setIsEditingName(false);
  };

  const handleSaveName = async () => {
    if (!canUpdateIntegrations) return;

    const nextName = nameDraft.trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }
    if (nextName === displayName.trim()) {
      setIsEditingName(false);
      return;
    }

    try {
      await updateMutation.mutateAsync({
        name: nextName,
        configuration: configValues,
      });
      setIsEditingName(false);
    } catch (_error) {
      showErrorToast("Failed to update integration");
    }
  };

  const handleInstructionAction = async (index: number) => {
    const action = integration?.status?.instruction?.actions?.[index];
    if (!action) return;

    if (action.redirect) {
      executeSetupRedirectAction(action.redirect);
      return;
    }

    const actionName = action.actionCall?.actionName?.trim();
    if (!actionName) {
      showErrorToast("Missing action name in setup instruction");
      return;
    }

    try {
      setActiveInstructionAction(index);
      await invokeActionMutation.mutateAsync({
        actionName,
        parameters: action.actionCall?.parameters || {},
      });
    } catch (error) {
      showErrorToast(`Failed to run integration action: ${getApiErrorMessage(error)}`);
    } finally {
      setActiveInstructionAction(null);
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

  const handleCopyIntegrationID = async () => {
    const id = integration?.metadata?.id;
    if (!id) return;
    await navigator.clipboard.writeText(id);
    setIntegrationIdCopied(true);
    setTimeout(() => setIntegrationIdCopied(false), 1200);
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

  return (
    <div className="pt-6">
      <div className="flex flex-wrap items-center gap-4 mb-6">
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
        <div className="flex-1 min-w-[280px]">
          {isEditingName ? (
            <div className="flex items-center gap-2 max-w-xl">
              <Input
                type="text"
                value={nameDraft}
                onChange={(e) => setNameDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    void handleSaveName();
                  }
                  if (e.key === "Escape") {
                    e.preventDefault();
                    handleCancelNameEdit();
                  }
                }}
                className="h-9 text-base"
                autoFocus
                disabled={updateMutation.isPending}
              />
              <Button
                type="button"
                size="icon"
                color="blue"
                onClick={() => void handleSaveName()}
                disabled={updateMutation.isPending || !nameDraft.trim()}
              >
                {updateMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Check className="h-4 w-4" />
                )}
              </Button>
              <Button
                type="button"
                size="icon"
                variant="outline"
                onClick={handleCancelNameEdit}
                disabled={updateMutation.isPending}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-1">
              <h4 className="text-2xl font-medium break-all">{displayName}</h4>
              {canUpdateIntegrations ? (
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="h-8 w-8"
                  onClick={handleStartNameEdit}
                  disabled={updateMutation.isPending}
                >
                  <Pencil className="h-4 w-4" />
                </Button>
              ) : null}
            </div>
          )}
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
      <div className="mb-6 flex flex-wrap items-center gap-2">
        <span className="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">ID</span>
        <code className="rounded-md border border-gray-300 bg-gray-50 px-2 py-1 text-xs text-gray-800 dark:border-gray-700 dark:bg-gray-800/60 dark:text-gray-100">
          {integration.metadata?.id}
        </code>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-8 gap-1"
          onClick={() => void handleCopyIntegrationID()}
        >
          {integrationIdCopied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {integrationIdCopied ? "Copied" : "Copy"}
        </Button>
      </div>

      <div className="space-y-6">
        {integration.status?.state === "error" && integration.status?.stateDescription && (
          <Alert variant="destructive" className="[&>svg+div]:translate-y-0 [&>svg]:top-[14px]">
            <CircleX className="h-4 w-4" />
            <AlertDescription>{integration.status.stateDescription}</AlertDescription>
          </Alert>
        )}

        {integration?.status?.instruction ? (
          <div className="rounded-lg border border-orange-950/15 bg-orange-100 dark:border-orange-900/40 dark:bg-orange-950/30">
            <div className="p-4">
              <IntegrationInstructions
                description={integration.status.instruction.text}
                className="rounded-none border-0 bg-transparent p-0 text-gray-800 dark:text-gray-200"
                actions={(integration.status.instruction.actions || []).map((action, index) => ({
                  label: action.redirect?.label || action.actionCall?.label || "Continue",
                  onClick: () => {
                    void handleInstructionAction(index);
                  },
                  external: Boolean(action.redirect),
                  disabled: invokeActionMutation.isPending,
                  isPending: invokeActionMutation.isPending && activeInstructionAction === index,
                }))}
              />
            </div>
          </div>
        ) : null}

        <div>
          <h2 className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            <Settings2 className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
            Configuration
          </h2>
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
              {integrationDef?.configuration && integrationDef.configuration.length > 0 ? (
                <PermissionTooltip
                  allowed={canUpdateIntegrations || permissionsLoading}
                  message="You don't have permission to update integrations."
                  className="w-full"
                >
                  <form onSubmit={handleConfigSubmit} className="space-y-4">
                    {!hasActionCallSetupAction
                      ? integrationDef.configuration.map((field: ConfigurationField) => (
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
                        ))
                      : null}

                    <div className="flex items-center gap-3 pt-4">
                      <Button type="submit" color="blue" disabled={updateMutation.isPending || !canUpdateIntegrations}>
                        {updateMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            Saving...
                          </>
                        ) : (
                          "Save"
                        )}
                      </Button>
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
        </div>

        {/* Used By */}
        <div>
          <h2 className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            <Workflow className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
            Used By
          </h2>
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800">
            <div className="p-6">
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
        </div>

        {/* Danger Zone */}
        <div>
          <h2 className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            <TriangleAlert className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
            Danger Zone
          </h2>
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-red-200 dark:border-red-800">
            <div className="p-6">
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
