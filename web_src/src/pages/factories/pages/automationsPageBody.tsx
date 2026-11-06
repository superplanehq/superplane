import type { FactoryAutomation } from "@/api-client";
import type { AutomationCardActions } from "./automationCardActions";
import { AutomationsPageList, EmptyAutomationsState } from "./automationsPageParts";

type AutomationsPageBodyProps = {
  organizationId: string;
  factoryKey: string;
  apps: FactoryAutomation[];
  failedAutomationIds?: ReadonlySet<string>;
  workOrders: Parameters<typeof AutomationsPageList>[0]["workOrders"];
  appsLoading: boolean;
  actionsForApp: (app: FactoryAutomation) => AutomationCardActions;
  canCreate: boolean;
  onCreate: () => void;
};

export function AutomationsPageBody({
  organizationId,
  factoryKey,
  apps,
  failedAutomationIds,
  workOrders,
  appsLoading,
  actionsForApp,
  canCreate,
  onCreate,
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
      failedAutomationIds={failedAutomationIds}
      workOrders={workOrders}
      actionsForApp={actionsForApp}
    />
  );
}
