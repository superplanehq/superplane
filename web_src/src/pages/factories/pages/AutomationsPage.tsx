import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";
import { AutomationDetail } from "./automationsPageParts";
import { AutomationsPageBody } from "./automationsPageBody";
import { AutomationsLegacyRedirect } from "./automationsPageRedirect";
import { useAutomationsPageModel } from "./useAutomationsPageModel";

export function AutomationsPage() {
  const model = useAutomationsPageModel();

  if (model.showLegacyRedirect) {
    return (
      <AutomationsLegacyRedirect
        organizationId={model.organizationId}
        factoryKey={model.factoryKey}
        factoryLoaded={Boolean(model.factory)}
        legacyLineId={model.legacyLineId}
      />
    );
  }

  const selectedApp = model.selectedApp;

  if (selectedApp && model.selectedAppActions) {
    return (
      <div className="flex h-full min-h-0 flex-col" data-testid="automations-detail-page">
        <AutomationDetail
          organizationId={model.organizationId}
          factoryKey={model.factoryKey}
          app={selectedApp}
          actions={model.selectedAppActions}
        />
      </div>
    );
  }

  return (
    <div data-testid="automations-list-page">
      <WorkspacePageHeader
        title="Automations"
        subtitle="Automations are one-step lines. Each one listens for a trigger and runs a canvas when it fires."
        actions={
          <PermissionTooltip
            allowed={model.canCreateApp || model.permissionsLoading}
            message="You don't have permission to create automations."
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
      />

      <div className={factoryContentBodyClassName}>
        <AutomationsPageBody
          organizationId={model.organizationId}
          factoryKey={model.factoryKey}
          apps={model.apps}
          workOrders={model.workOrders}
          appsLoading={model.appsLoading}
          selectedApp={null}
          selectedAppActions={null}
          actionsForApp={model.actionsForApp}
          canCreate={model.canCreateApp || model.permissionsLoading}
          onCreate={() => model.setCreateOpen(true)}
        />
      </div>

      <CreateFactoryAppDialog
        open={model.createOpen}
        isSaving={model.createCanvas.isPending}
        onClose={() => model.setCreateOpen(false)}
        onCreate={model.handleCreateAutomation}
      />
    </div>
  );
}
