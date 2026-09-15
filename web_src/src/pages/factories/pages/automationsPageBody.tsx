import type { FactoriesFactoryLine, FactoryApp } from "@/api-client";
import type { AutomationCardActions } from "./automationCardActions";
import { AutomationsPageList, EmptyAutomationsState } from "./automationsPageParts";

type AutomationsPageBodyProps = {
  organizationId: string;
  factoryKey: string;
  apps: FactoryApp[];
  workOrders: Parameters<typeof AutomationsPageList>[0]["workOrders"];
  appsLoading: boolean;
  actionsForApp: (app: FactoryApp) => AutomationCardActions;
  canCreate: boolean;
  onCreate: () => void;
  lines?: FactoriesFactoryLine[];
};

export function AutomationsPageBody({
  organizationId,
  factoryKey,
  apps,
  workOrders,
  appsLoading,
  actionsForApp,
  canCreate,
  onCreate,
  lines,
}: AutomationsPageBodyProps) {
  if (appsLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading automations…</p>;
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
      lines={lines}
    />
  );
}
