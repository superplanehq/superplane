import type { FactoriesWorkOrder } from "@/api-client";

import { CUSTOM_AUTOMATION_CATALOG_ID } from "./columnAutomationCatalog";
import type { ColumnAutomation } from "./columnAutomations";
import { customAutomationModeCopy, type CustomAutomationMode } from "./customAutomation";

export function customAutomationColumnAutomation(
  app: {
    id?: string;
    name?: string;
    columnKey?: string;
    templateId?: string;
    customAutomationMode?: CustomAutomationMode;
    customAutomationIcon?: string;
    customAutomationCronExpression?: string;
    customAutomationTriggers?: Array<{ mode: CustomAutomationMode; cronExpression?: string }>;
  },
  workOrders: FactoriesWorkOrder[],
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation | null {
  const id = app.id?.trim();
  if (!id) {
    return null;
  }
  const columnKey = app.columnKey === "backlog" ? "backlog" : "verify";
  const listed = app.customAutomationTriggers ?? [];
  const hasSchedule =
    listed.some((trigger) => trigger.mode === "scheduled") || app.customAutomationMode === "scheduled";
  const primary = listed.some((trigger) => trigger.mode === "automatic")
    ? "automatic"
    : listed.some((trigger) => trigger.mode === "scheduled")
      ? "scheduled"
      : (app.customAutomationMode ?? "manual");
  const copy = customAutomationModeCopy(primary, columnKey);
  const trigger =
    listed.length > 1
      ? listed
          .map((item) =>
            item.mode === "scheduled" && item.cronExpression
              ? `On schedule: ${item.cronExpression}`
              : customAutomationModeCopy(item.mode, columnKey).trigger,
          )
          .join(", ")
      : app.customAutomationMode === "scheduled" && app.customAutomationCronExpression
        ? `On schedule: ${app.customAutomationCronExpression}`
        : copy.trigger;
  return {
    id: `custom-automation-${id}`,
    kind: "custom-verification",
    name: app.name?.trim() || "Custom automation",
    trigger,
    action: copy.action,
    iconSrc: "",
    iconAlt: "",
    iconName: app.customAutomationIcon,
    health: "healthy",
    runningCount: runningCountForApp(id, workOrders),
    catalogId: CUSTOM_AUTOMATION_CATALOG_ID,
    canvasId: id,
    canRunNow: columnKey === "backlog" && hasSchedule ? true : undefined,
  };
}
