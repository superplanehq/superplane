import type {
  IntegrationCapabilityState,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { CapabilitiesTab } from "./CapabilitiesTab";
import { PropertiesTab } from "./PropertiesTab";
import { SecretsTab } from "./SecretsTab";
import { UsageTab } from "./UsageTab";
import { getActiveTabClass } from "./lib";
import type { useIntegrationMutations } from "@/hooks/useIntegrations";
import type { IntegrationDetailsState } from "./useIntegrationDetailsState";

type IntegrationTab = "properties" | "secrets" | "capabilities" | "usage";
type IntegrationMutations = ReturnType<typeof useIntegrationMutations>;

const TABS: Array<{ value: IntegrationTab; label: string }> = [
  { value: "properties", label: "Properties" },
  { value: "secrets", label: "Secrets" },
  { value: "capabilities", label: "Capabilities" },
  { value: "usage", label: "Usage" },
];

interface IntegrationTabsProps {
  activeTab: IntegrationTab;
  onActiveTabChange: (tab: IntegrationTab) => void;
  integration: OrganizationsIntegration;
  integrationDef?: IntegrationsIntegrationDefinition;
  detailsState: IntegrationDetailsState;
  integrationMutations: IntegrationMutations;
  organizationId: string;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  saveProperty: (propertyName: string, value: string) => Promise<void>;
  saveSecret: (secretName: string, value: string, draftFieldKey: string) => Promise<void>;
  isSavingProperty: (propertyName: string | undefined) => boolean;
  isSavingSecret: (secretName: string | undefined) => boolean;
  onApplyCapabilityChanges: (updates: IntegrationCapabilityState[]) => void | Promise<void>;
}

function TabButton({
  activeTab,
  value,
  label,
  onActiveTabChange,
}: {
  activeTab: IntegrationTab;
  value: IntegrationTab;
  label: string;
  onActiveTabChange: (tab: IntegrationTab) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onActiveTabChange(value)}
      className={cn(
        "py-2 mr-4 text-sm mb-[-1px] font-medium border-b transition-colors",
        getActiveTabClass(activeTab === value),
      )}
    >
      {label}
    </button>
  );
}

export function IntegrationTabs({
  activeTab,
  onActiveTabChange,
  integration,
  integrationDef,
  detailsState,
  integrationMutations,
  organizationId,
  canUpdateIntegrations,
  permissionsLoading,
  saveProperty,
  saveSecret,
  isSavingProperty,
  isSavingSecret,
  onApplyCapabilityChanges,
}: IntegrationTabsProps) {
  return (
    <Tabs value={activeTab} onValueChange={(value) => onActiveTabChange(value as IntegrationTab)} className="w-full">
      <div className="border-border border-b-1">
        <div className="flex flex-wrap px-4">
          {TABS.map((tab) => (
            <TabButton
              key={tab.value}
              activeTab={activeTab}
              value={tab.value}
              label={tab.label}
              onActiveTabChange={onActiveTabChange}
            />
          ))}
        </div>
      </div>

      <TabsContent value="properties" className="mt-4">
        <PropertiesTab
          integrationProperties={detailsState.integrationProperties}
          propertyDrafts={detailsState.propertyDrafts}
          setPropertyDrafts={detailsState.setPropertyDrafts}
          canUpdateIntegrations={canUpdateIntegrations}
          permissionsLoading={permissionsLoading}
          settingsMutationBusy={integrationMutations.settingsMutationBusy}
          saveProperty={saveProperty}
          isSavingProperty={isSavingProperty}
        />
      </TabsContent>

      <TabsContent value="secrets" className="mt-4">
        <SecretsTab
          integrationSecrets={detailsState.integrationSecrets}
          secretDrafts={detailsState.secretDrafts}
          setSecretDrafts={detailsState.setSecretDrafts}
          canUpdateIntegrations={canUpdateIntegrations}
          permissionsLoading={permissionsLoading}
          settingsMutationBusy={integrationMutations.settingsMutationBusy}
          saveSecret={saveSecret}
          isSavingSecret={isSavingSecret}
        />
      </TabsContent>

      <TabsContent value="capabilities" className="mt-4">
        <CapabilitiesTab
          integration={integration}
          integrationDef={integrationDef}
          capabilityStates={detailsState.capabilityStates}
          setCapabilityStates={detailsState.setCapabilityStates}
          canUpdateIntegrations={canUpdateIntegrations}
          permissionsLoading={permissionsLoading}
          capabilitiesMutationPending={integrationMutations.updateCapabilitiesMutation.isPending}
          onApplyCapabilityChanges={onApplyCapabilityChanges}
        />
      </TabsContent>

      <TabsContent value="usage" className="mt-4">
        <UsageTab organizationId={organizationId} workflowGroups={detailsState.workflowGroups} />
      </TabsContent>
    </Tabs>
  );
}
