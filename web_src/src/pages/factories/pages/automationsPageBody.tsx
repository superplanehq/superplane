import type { FactoryApp } from "@/api-client";
import type { AutomationCardActions } from "./automationCardActions";
import { AutomationDetail } from "./AutomationDetail";
import { AutomationsPageList, EmptyAutomationsState } from "./automationsPageParts";

type AutomationsPageBodyProps = {
  organizationId: string;
  factoryKey: string;
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
  factoryKey,
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
        factoryKey={factoryKey}
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
      factoryKey={factoryKey}
      apps={apps}
      workOrders={workOrders}
      actionsForApp={actionsForApp}
    />
  );
}
