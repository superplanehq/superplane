import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { usePermissions } from "@/contexts/PermissionsContext";
import {
  useAvailableIntegrations,
  useDeleteIntegration,
  useUpdateIntegrationCapabilities,
  useUpdateIntegrationProperty,
  useUpdateIntegrationSecret,
} from "@/hooks/useIntegrations";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Alert, AlertDescription } from "@/ui/alert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { CopyButton } from "@/ui/CopyButton";
import { ArrowLeft, CircleX, Plug, Trash2 } from "lucide-react";
import { CapabilitiesTab } from "./CapabilitiesTab";
import { PropertiesTab } from "./PropertiesTab";
import { SecretsTab } from "./SecretsTab";
import { UsageTab } from "./UsageTab";
import { DEFAULT_CAPABILITY_STATE, getActiveTabClass } from "./lib";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

interface CapabilityBasedIntegrationDetailsProps {
  organizationId: string;
  integration: OrganizationsIntegration;
}

export function CapabilityBasedIntegrationDetails({
  organizationId,
  integration,
}: CapabilityBasedIntegrationDetailsProps) {
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  usePageTitle(["Integrations", integration?.metadata?.name]);

  const integrationId = integration.metadata?.id;
  const providerName = integration.metadata?.integrationName ?? "";
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [activeTab, setActiveTab] = useState("properties");
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration ? availableIntegrations.find((i) => i.name === providerName) : undefined;

  const deleteMutation = useDeleteIntegration(organizationId, integrationId || "");
  const updateCapabilitiesMutation = useUpdateIntegrationCapabilities(organizationId, integrationId || "");
  const updatePropertyMutation = useUpdateIntegrationProperty(organizationId, integrationId || "");
  const updateSecretMutation = useUpdateIntegrationSecret(organizationId, integrationId || "");
  const integrationProperties = useMemo(() => integration?.status?.properties ?? [], [integration?.status?.properties]);
  const integrationSecrets = useMemo(() => integration?.status?.secrets ?? [], [integration?.status?.secrets]);

  const [propertyDrafts, setPropertyDrafts] = useState<Record<string, string>>({});
  const [secretDrafts, setSecretDrafts] = useState<Record<string, string>>({});
  const [capabilityStates, setCapabilityStates] = useState<Record<string, IntegrationCapabilityStateState>>({});

  useEffect(() => {
    const nextStates: Record<string, IntegrationCapabilityStateState> = {};
    (integration?.status?.capabilities || []).forEach((capability) => {
      if (!capability.name) return;
      nextStates[capability.name] = capability.state || DEFAULT_CAPABILITY_STATE;
    });
    setCapabilityStates(nextStates);
  }, [integration?.status?.capabilities]);

  useEffect(() => {
    const next: Record<string, string> = {};
    integrationProperties.forEach((property, index) => {
      const key = property.name?.trim() || `__property_${index}`;
      next[key] = property.value ?? "";
    });
    setPropertyDrafts(next);
  }, [integration?.metadata?.id, integration?.metadata?.updatedAt, integrationProperties]);

  useEffect(() => {
    setSecretDrafts({});
  }, [integration?.metadata?.id, integration?.metadata?.updatedAt, integrationSecrets]);

  const settingsMutationBusy = updatePropertyMutation.isPending || updateSecretMutation.isPending;

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

  const handleDelete = async () => {
    if (!canDeleteIntegrations) return;
    try {
      await deleteMutation.mutateAsync({ integrationName: providerName });
      navigate(`/${organizationId}/settings/integrations`);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  const handleCapabilitiesSubmit = async (newStates: IntegrationCapabilityState[]) => {
    if (!canUpdateIntegrations || newStates.length === 0) return;
    try {
      const response = await updateCapabilitiesMutation.mutateAsync(newStates);
      const updated = response.data?.integration ?? null;

      if (updated?.status?.setupState?.currentStep) {
        navigate(`/${organizationId}/settings/integrations/${integrationId}/setup`, {
          state: { integrationId },
        });
        return;
      }

      showSuccessToast("Integration capabilities updated");
    } catch (_error) {
      showErrorToast(`Failed to update capabilities: ${getApiErrorMessage(_error)}`);
    }
  };

  const saveProperty = async (propertyName: string, value: string) => {
    if (!canUpdateIntegrations || settingsMutationBusy) return;
    try {
      await updatePropertyMutation.mutateAsync({ propertyName, value });
      showSuccessToast("Property saved");
    } catch (_error) {
      showErrorToast(`Failed to save property: ${getApiErrorMessage(_error)}`);
    }
  };

  const saveSecret = async (secretName: string, value: string, draftFieldKey: string) => {
    if (!canUpdateIntegrations || settingsMutationBusy || value.trim() === "") return;
    try {
      await updateSecretMutation.mutateAsync({ secretName, value });
      setSecretDrafts((previous) => ({ ...previous, [draftFieldKey]: "" }));
      showSuccessToast("Secret saved");
    } catch (_error) {
      showErrorToast(`Failed to save secret: ${getApiErrorMessage(_error)}`);
    }
  };

  return (
    <div className="pt-6">
      <Header
        organizationId={organizationId}
        integration={integration}
        integrationDef={integrationDef}
        canDeleteIntegrations={canDeleteIntegrations}
        permissionsLoading={permissionsLoading}
        setShowDeleteConfirm={setShowDeleteConfirm}
      />

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
                onClick={() => setActiveTab("properties")}
                className={cn(
                  "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors",
                  getActiveTabClass(activeTab === "properties"),
                )}
              >
                Properties
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("secrets")}
                className={cn(
                  "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors",
                  getActiveTabClass(activeTab === "secrets"),
                )}
              >
                Secrets
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("capabilities")}
                className={cn(
                  "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors",
                  getActiveTabClass(activeTab === "capabilities"),
                )}
              >
                Capabilities
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("usage")}
                className={cn(
                  "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors",
                  getActiveTabClass(activeTab === "usage"),
                )}
              >
                Usage
              </button>
            </div>
          </div>

          <TabsContent value="properties" className="mt-4">
            <PropertiesTab
              integrationProperties={integrationProperties}
              propertyDrafts={propertyDrafts}
              setPropertyDrafts={setPropertyDrafts}
              canUpdateIntegrations={canUpdateIntegrations}
              permissionsLoading={permissionsLoading}
              settingsMutationBusy={settingsMutationBusy}
              saveProperty={saveProperty}
              isSavingProperty={(propertyName) =>
                Boolean(
                  updatePropertyMutation.isPending && updatePropertyMutation.variables?.propertyName === propertyName,
                )
              }
            />
          </TabsContent>

          <TabsContent value="secrets" className="mt-4">
            <SecretsTab
              integrationSecrets={integrationSecrets}
              secretDrafts={secretDrafts}
              setSecretDrafts={setSecretDrafts}
              canUpdateIntegrations={canUpdateIntegrations}
              permissionsLoading={permissionsLoading}
              settingsMutationBusy={settingsMutationBusy}
              saveSecret={saveSecret}
              isSavingSecret={(secretName) =>
                Boolean(updateSecretMutation.isPending && updateSecretMutation.variables?.secretName === secretName)
              }
            />
          </TabsContent>

          <TabsContent value="capabilities" className="mt-4">
            <CapabilitiesTab
              integration={integration}
              integrationDef={integrationDef}
              capabilityStates={capabilityStates}
              setCapabilityStates={setCapabilityStates}
              canUpdateIntegrations={canUpdateIntegrations}
              permissionsLoading={permissionsLoading}
              capabilitiesMutationPending={updateCapabilitiesMutation.isPending}
              onApplyCapabilityChanges={handleCapabilitiesSubmit}
            />
          </TabsContent>

          <TabsContent value="usage" className="mt-4">
            <UsageTab organizationId={organizationId} workflowGroups={workflowGroups} />
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

interface HeaderProps {
  organizationId: string;
  integration: OrganizationsIntegration;
  integrationDef?: IntegrationsIntegrationDefinition;
  canDeleteIntegrations: boolean;
  permissionsLoading: boolean;
  setShowDeleteConfirm: (show: boolean) => void;
}

function Header({
  organizationId,
  integration,
  integrationDef,
  canDeleteIntegrations,
  permissionsLoading,
  setShowDeleteConfirm,
}: HeaderProps) {
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const integrationId = integration.metadata?.id;
  const integrationName = integration.metadata?.name;
  const integrationStatus = integration.status?.state || "unknown";

  return (
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
        <h4 className="flex items-center text-2xl font-medium">
          <span
            className="inline-flex shrink-0"
            title={integrationStatus.charAt(0).toUpperCase() + integrationStatus.slice(1)}
          ></span>
          <span>{integrationName}</span>
        </h4>
        {integrationId ? (
          <div className="mt-1.5 flex max-w-full items-center gap-1.5">
            <span className="min-w-0 truncate font-mono text-xs text-gray-700 dark:text-gray-300">{integrationId}</span>
            <CopyButton text={integrationId} />
          </div>
        ) : null}
      </div>
      <div className="ml-auto flex items-center gap-2">
        <Plug
          className={`h-5 w-5 ${
            integrationStatus === "ready"
              ? "text-green-500"
              : integrationStatus === "error"
                ? "text-red-600"
                : "text-amber-600"
          }`}
          aria-label={`Integration status: ${integrationStatus}`}
        />
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
  );
}
