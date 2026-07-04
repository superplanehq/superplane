import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
import { CapabilitySection } from "./CapabilitySection";
import { DisableCapabilitiesInUseDialog } from "./DisableCapabilitiesInUseDialog";
import type { Dispatch, SetStateAction } from "react";
import { useCapabilitiesTabModel } from "./useCapabilitiesTabModel";

export interface CapabilitiesTabProps {
  organizationId: string;
  integration: OrganizationsIntegration;
  integrationDef: IntegrationsIntegrationDefinition | undefined;
  capabilityStates: Record<string, IntegrationCapabilityStateState>;
  setCapabilityStates: Dispatch<SetStateAction<Record<string, IntegrationCapabilityStateState>>>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  capabilitiesMutationPending: boolean;
  onApplyCapabilityChanges: (updates: IntegrationCapabilityState[]) => void | Promise<void>;
}

export function CapabilitiesTab({
  organizationId,
  integration,
  integrationDef,
  capabilityStates,
  setCapabilityStates,
  canUpdateIntegrations,
  permissionsLoading,
  capabilitiesMutationPending,
  onApplyCapabilityChanges,
}: CapabilitiesTabProps) {
  const {
    capabilityByName,
    capabilityGroupSections,
    stagedCapabilityUpdates,
    pendingCapabilityUpdates,
    capabilityDisableCanvasRows,
    requestCapabilityUpdates,
    confirmPendingCapabilityUpdates,
    clearPendingCapabilityUpdates,
    queueCapabilityStateChange,
  } = useCapabilitiesTabModel({
    integration,
    integrationDef,
    capabilityStates,
    setCapabilityStates,
    canUpdateIntegrations,
    capabilitiesMutationPending,
    onApplyCapabilityChanges,
  });

  if (capabilityByName.size === 0) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No capabilities available.</p>;
  }

  return (
    <>
      <DisableCapabilitiesInUseDialog
        organizationId={organizationId}
        open={pendingCapabilityUpdates !== null}
        onOpenChange={(open) => {
          if (!open && !capabilitiesMutationPending) {
            clearPendingCapabilityUpdates();
          }
        }}
        capabilityDisableCanvasRows={capabilityDisableCanvasRows}
        canUpdateIntegrations={canUpdateIntegrations}
        capabilitiesMutationPending={capabilitiesMutationPending}
        onConfirm={confirmPendingCapabilityUpdates}
        onCancel={clearPendingCapabilityUpdates}
      />

      <div className="space-y-4">
        {capabilityGroupSections.map((section) => (
          <CapabilitySection
            key={section.key}
            section={section}
            capabilityByName={capabilityByName}
            capabilityStates={capabilityStates}
            canUpdateIntegrations={canUpdateIntegrations}
            permissionsLoading={permissionsLoading}
            capabilitiesMutationPending={capabilitiesMutationPending}
            onQueueCapabilityStateChange={queueCapabilityStateChange}
          />
        ))}
      </div>
      {stagedCapabilityUpdates.length > 0 ? (
        <div className="mt-4 flex justify-end">
          <LoadingButton
            type="button"
            color="blue"
            onClick={() => requestCapabilityUpdates(stagedCapabilityUpdates)}
            disabled={!canUpdateIntegrations}
            loading={capabilitiesMutationPending}
            loadingText="Updating…"
          >
            Update capabilities
          </LoadingButton>
        </div>
      ) : null}
    </>
  );
}
