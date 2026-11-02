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
  },
  workOrders: FactoriesWorkOrder[],
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation | null {
  const id = app.id?.trim();
  if (!id) {
    return null;
  }
  const copy = customAutomationModeCopy(
    app.customAutomationMode ?? "manual",
    app.columnKey === "backlog" ? "backlog" : "verify",
  );
  return {
    id: `custom-automation-${id}`,
    kind: "custom-verification",
    name: app.name?.trim() || "Custom automation",
    trigger: copy.trigger,
    action: copy.action,
    iconSrc: "",
    iconAlt: "",
    iconName: app.customAutomationIcon,
    health: "healthy",
    runningCount: runningCountForApp(id, workOrders),
    catalogId: CUSTOM_AUTOMATION_CATALOG_ID,
    canvasId: id,
  };
}
