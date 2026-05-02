import type { IntegrationCapabilityState, OrganizationsIntegration } from "@/api-client";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useAvailableIntegrations, useIntegrationMutations } from "@/hooks/useIntegrations";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Alert, AlertDescription } from "@/ui/alert";
import { CircleX } from "lucide-react";
import { DeleteModal } from "./DeleteModal";
import { Header } from "./Header";
import { IntegrationTabs } from "./IntegrationTabs";
import { useIntegrationDetailsState } from "./useIntegrationDetailsState";
import { useState } from "react";
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
  const [activeTab, setActiveTab] = useState<"properties" | "secrets" | "capabilities" | "usage">("properties");
  const canUpdateIntegrations = canAct("integrations", "update");
  const canDeleteIntegrations = canAct("integrations", "delete");
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const integrationDef = integration ? availableIntegrations.find((i) => i.name === providerName) : undefined;
  const detailsState = useIntegrationDetailsState(integration);
  const integrationMutations = useIntegrationMutations(organizationId, integrationId || "");

  const handleDelete = async () => {
    if (!canDeleteIntegrations) return;
    try {
      await integrationMutations.deleteMutation.mutateAsync({ integrationName: providerName });
      navigate(`/${organizationId}/settings/integrations`);
    } catch {
      showErrorToast("Failed to delete integration");
    }
  };

  const handleCapabilitiesSubmit = async (newStates: IntegrationCapabilityState[]) => {
    if (!canUpdateIntegrations || newStates.length === 0) return;
    try {
      const response = await integrationMutations.updateCapabilitiesMutation.mutateAsync(newStates);
      const updated = response.data?.integration ?? null;

      if (updated?.status?.setupState?.currentStep) {
        navigate(`/${organizationId}/settings/integrations/${providerName}/setup`, {
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
    if (!canUpdateIntegrations || integrationMutations.settingsMutationBusy) return;
    try {
      await integrationMutations.updatePropertyMutation.mutateAsync({ propertyName, value });
      showSuccessToast("Property saved");
    } catch (_error) {
      showErrorToast(`Failed to save property: ${getApiErrorMessage(_error)}`);
    }
  };

  const saveSecret = async (secretName: string, value: string, draftFieldKey: string) => {
    if (!canUpdateIntegrations || integrationMutations.settingsMutationBusy || value.trim() === "") return;
    try {
      await integrationMutations.updateSecretMutation.mutateAsync({ secretName, value });
      detailsState.setSecretDrafts((previous) => ({ ...previous, [draftFieldKey]: "" }));
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

        <IntegrationTabs
          activeTab={activeTab}
          onActiveTabChange={setActiveTab}
          integration={integration}
          integrationDef={integrationDef}
          detailsState={detailsState}
          integrationMutations={integrationMutations}
          organizationId={organizationId}
          canUpdateIntegrations={canUpdateIntegrations}
          permissionsLoading={permissionsLoading}
          saveProperty={saveProperty}
          saveSecret={saveSecret}
          isSavingProperty={(propertyName) =>
            Boolean(
              integrationMutations.updatePropertyMutation.isPending &&
                integrationMutations.updatePropertyMutation.variables?.propertyName === propertyName,
            )
          }
          isSavingSecret={(secretName) =>
            Boolean(
              integrationMutations.updateSecretMutation.isPending &&
                integrationMutations.updateSecretMutation.variables?.secretName === secretName,
            )
          }
          onApplyCapabilityChanges={handleCapabilitiesSubmit}
        />
      </div>

      <DeleteModal
        open={showDeleteConfirm}
        integrationName={integrationName}
        canDeleteIntegrations={canDeleteIntegrations}
        isDeleting={integrationMutations.deleteMutation.isPending}
        hasDeleteError={integrationMutations.deleteMutation.isError}
        onDelete={handleDelete}
        onClose={() => setShowDeleteConfirm(false)}
      />
    </div>
  );
}
