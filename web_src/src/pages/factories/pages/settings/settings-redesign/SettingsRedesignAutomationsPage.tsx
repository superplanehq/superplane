import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Plus } from "lucide-react";

import { CreateFactoryAppDialog } from "../../../CreateFactoryAppDialog";
import { FactoriesLayoutContext } from "../../../layout/factoriesLayoutContext";
import { AutomationsPageBody } from "../../automationsPageBody";
import { useAutomationsPageModel } from "../../useAutomationsPageModel";
import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";

export function SettingsRedesignAutomationsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();

  return (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId,
        factoryId,
        factoryKey: factory.key ?? "",
        factory,
        factories: [factory],
        openCreateWorkOrder: () => undefined,
      }}
    >
      <AutomationsSettingsContent />
    </FactoriesLayoutContext.Provider>
  );
}

function AutomationsSettingsContent() {
  const model = useAutomationsPageModel();
  usePageTitle(["Automations", "Settings", model.factory?.name ?? "Workspace"]);

  return (
    <>
      <FactorySettingsPageFrame
        title="Automations"
        subtitle="One-step lines that listen for a trigger and run a canvas."
        actions={
          <PermissionTooltip
            allowed={model.canCreateApp || model.permissionsLoading}
            message="You do not have permission to create automations."
          >
            <Button
              type="button"
              size="sm"
              disabled={!model.canCreateApp}
              onClick={() => model.setCreateOpen(true)}
              data-testid="automations-create-button"
            >
              <Plus className="size-3.5" aria-hidden />
              New automation
            </Button>
          </PermissionTooltip>
        }
      >
        <div data-testid="settings-redesign-automations">
          <AutomationsPageBody
            organizationId={model.organizationId}
            factoryKey={model.factoryKey}
            apps={model.apps}
            workOrders={model.workOrders}
            appsLoading={model.appsLoading}
            actionsForApp={model.actionsForApp}
            canCreate={model.canCreateApp || model.permissionsLoading}
            onCreate={() => model.setCreateOpen(true)}
          />
        </div>
      </FactorySettingsPageFrame>

      <CreateFactoryAppDialog
        open={model.createOpen}
        isSaving={model.createCanvas.isPending}
        onClose={() => model.setCreateOpen(false)}
        onCreate={model.handleCreateAutomation}
      />
    </>
  );
}
