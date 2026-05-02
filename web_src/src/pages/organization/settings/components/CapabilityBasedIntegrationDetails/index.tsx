import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  OrganizationsIntegration,
} from "@/api-client";
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
import { CircleX } from "lucide-react";
import { CapabilitiesTab } from "./CapabilitiesTab";
import { DeleteModal } from "./DeleteModal";
import { Header } from "./Header";
import { PropertiesTab } from "./PropertiesTab";
import { SecretsTab } from "./SecretsTab";
import { UsageTab } from "./UsageTab";
import { DEFAULT_CAPABILITY_STATE, getActiveTabClass, groupNodeRefsByCanvas } from "./lib";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

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
  const integrationId = integration.metadata?.id;
  const integrationName = integration.metadata?.name;
  const providerName = integration.metadata?.integrationName ?? "";

  usePageTitle(["Integrations", integrationName]);

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

  const workflowGroups = useMemo(() => {
    return groupNodeRefsByCanvas(integration?.status?.usedIn ?? []);
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
        onRequestDelete={() => setShowDeleteConfirm(true)}
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

      <DeleteModal
        open={showDeleteConfirm}
        integrationName={integrationName}
        canDeleteIntegrations={canDeleteIntegrations}
        isDeleting={deleteMutation.isPending}
        hasDeleteError={deleteMutation.isError}
        onDelete={handleDelete}
        onClose={() => setShowDeleteConfirm(false)}
      />
    </div>
  );
}
