import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { Plus } from "lucide-react";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";
import { AutomationDetail } from "./AutomationDetail";
import { AutomationsPageBody } from "./automationsPageBody";
import { AutomationsLegacyRedirect } from "./automationsPageRedirect";
import { useAutomationsPageModel } from "./useAutomationsPageModel";
import { useActivePRFeedbackWorkOrderIds } from "./useWorkOrderPRFeedbackRunHref";

export function AutomationsPage() {
  const model = useAutomationsPageModel();
  const cardActions = useWorkOrderCardActions(model.organizationId, model.factoryId);
  const addressingFeedbackOrderIds = useActivePRFeedbackWorkOrderIds(model.workOrders);

  // Above the list/detail branching below (hooks can't be conditional): this
  // single call covers both the Automations list and the in-page detail view
  // for a selected app, re-firing when the selection changes without an
  // unmount/mount.
  usePageTitle([model.selectedApp?.name ?? "Automations", model.factory?.name ?? "Workspace"]);

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
  const workOrderCardContext = {
    organizationId: model.organizationId,
    factoryId: model.factoryId,
    factoryKey: model.factoryKey,
    factoryLines: model.factory?.lines ?? [],
    canDispatch: model.canUpdateWorkOrders,
    canAssign: model.canUpdateWorkOrders,
    addressingFeedbackOrderIds,
    ...cardActions,
  };

  if (selectedApp && model.selectedAppActions) {
    return (
      <div className="flex h-full min-h-0 flex-col" data-testid="automations-detail-page">
        <AutomationDetail
          organizationId={model.organizationId}
          factoryKey={model.factoryKey}
          app={selectedApp}
          actions={model.selectedAppActions}
          factory={model.factory}
          workOrders={model.workOrders}
          workOrderCardContext={workOrderCardContext}
        />
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col pb-8" data-testid="automations-list-page">
      <WorkspacePageHeader
        className={factorySectionHeaderClassName}
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

      <div className={factorySectionBodyClassName}>
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

      <CreateFactoryAppDialog
        open={model.createOpen}
        isSaving={model.createCanvas.isPending}
        onClose={() => model.setCreateOpen(false)}
        onCreate={model.handleCreateAutomation}
      />
    </div>
  );
}
