import type { FactoriesWorkOrder } from "@/api-client";

import { CUSTOM_CATALOG_ID, EVENT_CUSTOM_ENTRY } from "./columnAutomationCatalog";
import type { ColumnAutomation, ColumnKey } from "./columnAutomations";
import { CUSTOM_AUTOMATION_TEMPLATE_ID, type CustomAutomationMode } from "./customAutomation";
import { customAutomationColumnAutomation } from "./customAutomationColumnAutomation";
import type { LinePhaseColumn } from "./linePhaseRuns";

export type ColumnAutomationApp = {
  id?: string;
  name?: string;
  columnKey?: string;
  templateId?: string;
  customAutomationMode?: CustomAutomationMode;
  customAutomationIcon?: string;
  customAutomationCronExpression?: string;
  customAutomationTriggers?: Array<{ mode: CustomAutomationMode; cronExpression?: string }>;
};

export function customColumnAutomations(
  apps: ColumnAutomationApp[],
  columnKey: "backlog" | "implement" | "verify" | "done",
  workOrders: FactoriesWorkOrder[],
  skipIds: Set<string>,
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    if (!id || skipIds.has(id) || app.columnKey !== columnKey) {
      return [];
    }
    if (app.templateId === CUSTOM_AUTOMATION_TEMPLATE_ID) {
      const automation = customAutomationColumnAutomation(app, workOrders, runningCountForApp);
      return automation ? [automation] : [];
    }
    return [
      {
        id: `custom-${id}`,
        kind: "custom",
        name: app.name?.trim() || "Custom automation",
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

export function firstPhaseAutomations(
  key: ColumnKey,
  input: { columnTitle: string; columns?: LinePhaseColumn[]; apps?: ColumnAutomationApp[] },
  workOrders: FactoriesWorkOrder[],
  agentSteps: (
    key: ColumnKey,
    columnTitle: string,
    columns: LinePhaseColumn[],
    workOrders: FactoriesWorkOrder[],
  ) => ColumnAutomation[],
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation[] {
  const steps = agentSteps(key, input.columnTitle, input.columns ?? [], workOrders);
  if (key !== "phase-0") {
    return steps;
  }
  return [
    ...steps,
    ...customColumnAutomations(input.apps ?? [], "implement", workOrders, new Set(), runningCountForApp),
  ];
}
