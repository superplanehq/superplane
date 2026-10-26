import type { FactoriesWorkOrder } from "@/api-client";

import { CUSTOM_CATALOG_ID, EVENT_CUSTOM_ENTRY } from "./columnAutomationCatalog";
import type { ColumnAutomation } from "./columnAutomations";
import { CUSTOM_VERIFICATION_TEMPLATE_ID, type CustomVerificationMode } from "./customVerification";
import { customVerificationColumnAutomation } from "./customVerificationColumnAutomation";

export type ColumnAutomationApp = {
  id?: string;
  name?: string;
  columnKey?: string;
  templateId?: string;
  customVerificationMode?: CustomVerificationMode;
  customVerificationIcon?: string;
};

export function customColumnAutomations(
  apps: ColumnAutomationApp[],
  columnKey: "backlog" | "verify" | "done",
  workOrders: FactoriesWorkOrder[],
  skipIds: Set<string>,
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    if (!id || skipIds.has(id) || app.columnKey !== columnKey) {
      return [];
    }
    if (columnKey === "verify" && app.templateId === CUSTOM_VERIFICATION_TEMPLATE_ID) {
      const automation = customVerificationColumnAutomation(app, workOrders, runningCountForApp);
      return automation ? [automation] : [];
    }
    return [
      {
        id: `custom-${id}`,
        kind: "custom",
        name: app.name?.trim() || EVENT_CUSTOM_ENTRY.name,
        iconName: columnKey === "backlog" ? "alarm-clock" : undefined,
        trigger: columnKey === "backlog" ? "On schedule" : EVENT_CUSTOM_ENTRY.trigger,
        action: columnKey === "backlog" ? "Proposes a task" : EVENT_CUSTOM_ENTRY.action,
        iconSrc: EVENT_CUSTOM_ENTRY.iconSrc,
        iconAlt: EVENT_CUSTOM_ENTRY.iconAlt,
        health: "healthy",
        runningCount: runningCountForApp(id, workOrders),
        catalogId: columnKey === "backlog" ? "scheduled-intake" : CUSTOM_CATALOG_ID,
        canvasId: id,
      },
    ];
  });
}
