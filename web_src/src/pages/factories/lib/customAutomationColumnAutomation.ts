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
  const primary = primaryCustomAutomationMode(app.customAutomationMode, listed);
  const copy = customAutomationModeCopy(primary, columnKey);
  const trigger = customAutomationTriggerSummary(app, listed, columnKey, copy.trigger);
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

function primaryCustomAutomationMode(
  saved: CustomAutomationMode | undefined,
  listed: Array<{ mode: CustomAutomationMode }>,
): CustomAutomationMode {
  if (listed.some((trigger) => trigger.mode === "automatic")) {
    return "automatic";
  }
  if (listed.some((trigger) => trigger.mode === "scheduled")) {
    return "scheduled";
  }
  return saved ?? "manual";
}

function customAutomationTriggerSummary(
  app: { customAutomationMode?: CustomAutomationMode; customAutomationCronExpression?: string },
  listed: Array<{ mode: CustomAutomationMode; cronExpression?: string }>,
  columnKey: "backlog" | "verify",
  fallback: string,
): string {
  if (listed.length > 1) {
    return listed.map((item) => customAutomationTriggerLabel(item, columnKey)).join(", ");
  }
  if (app.customAutomationMode === "scheduled" && app.customAutomationCronExpression) {
    return `On schedule: ${app.customAutomationCronExpression}`;
  }
  return fallback;
}

function customAutomationTriggerLabel(
  item: { mode: CustomAutomationMode; cronExpression?: string },
  columnKey: "backlog" | "verify",
): string {
  if (item.mode === "scheduled" && item.cronExpression) {
    return `On schedule: ${item.cronExpression}`;
  }
  return customAutomationModeCopy(item.mode, columnKey).trigger;
}
