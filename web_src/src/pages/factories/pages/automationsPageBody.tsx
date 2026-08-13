import type { FactoryApp } from "@/api-client";
import { AutomationDetail, AutomationsPageList, EmptyAutomationsState } from "./automationsPageParts";

type AutomationsPageBodyProps = {
  organizationId: string;
  factoryKey: string;
  apps: FactoryApp[];
  workOrders: Parameters<typeof AutomationsPageList>[0]["workOrders"];
  appsLoading: boolean;
  selectedApp: FactoryApp | null;
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
  canCreate,
  onCreate,
}: AutomationsPageBodyProps) {
  if (appsLoading && !selectedApp) {
    return <p className="text-[13px] text-muted-foreground">Loading automations…</p>;
  }
  if (selectedApp) {
    return <AutomationDetail organizationId={organizationId} factoryKey={factoryKey} app={selectedApp} />;
  }
  if (apps.length === 0) {
    return <EmptyAutomationsState canCreate={canCreate} onCreate={onCreate} />;
  }
  return (
    <AutomationsPageList organizationId={organizationId} factoryKey={factoryKey} apps={apps} workOrders={workOrders} />
  );
}
