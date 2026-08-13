import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Plus } from "lucide-react";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";
import { AutomationsPageBody } from "./automationsPageBody";
import { AutomationsLegacyRedirect } from "./automationsPageRedirect";
import { useAutomationsPageModel } from "./useAutomationsPageModel";

export function AutomationsPage() {
  const model = useAutomationsPageModel();

  if (model.showLegacyRedirect) {
    return (
      <AutomationsLegacyRedirect
        organizationId={model.organizationId}
        factoryId={model.factoryId}
        factoryLoaded={Boolean(model.factory)}
        legacyLineId={model.legacyLineId}
      />
    );
  }

  const selectedApp = model.selectedApp;

  return (
    <div
      className={cn(selectedApp && "flex h-full min-h-0 flex-col")}
      data-testid={selectedApp ? "automations-detail-page" : "automations-list-page"}
    >
      <header className={cn(factoryContentHeaderClassName, selectedApp && "shrink-0")}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            Automations
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Automations are one-step lines. Each one listens for a trigger and runs a canvas when it fires.
          </p>
        </div>
        {selectedApp == null ? (
          <PermissionTooltip
            allowed={model.canCreateApp || model.permissionsLoading}
            message="You don't have permission to create automations."
          >
            <Button
              type="button"
              variant="outline"
              asChild={false}
              disabled={!model.canCreateApp}
              onClick={() => model.setCreateOpen(true)}
              data-testid="automations-create-button"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden />
              Add automation
            </Button>
          </PermissionTooltip>
        ) : null}
      </header>

      <div
        className={cn(factoryContentBodyClassName, selectedApp && "flex min-h-0 flex-1 flex-col overflow-hidden py-6")}
      >
        <AutomationsPageBody
          organizationId={model.organizationId}
          factoryId={model.factoryId}
          apps={model.apps}
          workOrders={model.workOrders}
          appsLoading={model.appsLoading}
          selectedApp={selectedApp}
          selectedAppActions={model.selectedAppActions}
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
