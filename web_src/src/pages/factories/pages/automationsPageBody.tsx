import type { FactoryApp } from "@/api-client";
import type { AutomationCardActions } from "./automationCardActions";
import { AutomationDetail, AutomationsPageList, EmptyAutomationsState } from "./automationsPageParts";

type AutomationsPageBodyProps = {
  organizationId: string;
  factoryId: string;
  apps: FactoryApp[];
  workOrders: Parameters<typeof AutomationsPageList>[0]["workOrders"];
  appsLoading: boolean;
  selectedApp: FactoryApp | null;
  selectedAppActions: AutomationCardActions | null;
  actionsForApp: (app: FactoryApp) => AutomationCardActions;
  canCreate: boolean;
  onCreate: () => void;
};

export function AutomationsPageBody({
  organizationId,
  factoryId,
  apps,
  workOrders,
  appsLoading,
  selectedApp,
  selectedAppActions,
  actionsForApp,
  canCreate,
  onCreate,
}: AutomationsPageBodyProps) {
  if (appsLoading && !selectedApp) {
    return <p className="text-[13px] text-muted-foreground">Loading automations…</p>;
  }
  if (selectedApp && selectedAppActions) {
    return (
      <AutomationDetail
        organizationId={organizationId}
        factoryId={factoryId}
        app={selectedApp}
        actions={selectedAppActions}
      />
    );
  }
  if (apps.length === 0) {
    return <EmptyAutomationsState canCreate={canCreate} onCreate={onCreate} />;
  }
  return (
    <AutomationsPageList
      organizationId={organizationId}
      factoryId={factoryId}
      apps={apps}
      workOrders={workOrders}
      actionsForApp={actionsForApp}
    />
  );
}
