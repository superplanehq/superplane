import type { FactoriesWorkOrder } from "@/api-client";

import { HUMAN_REVIEW_CATALOG_ID, HUMAN_REVIEW_ENTRY } from "./columnAutomationCatalog";
import type { ColumnAutomation } from "./columnAutomations";
import type { ColumnAutomationApp } from "./customColumnAutomations";

export function humanReviewColumnAutomations(
  apps: ColumnAutomationApp[],
  workOrders: FactoriesWorkOrder[],
  runningCount: (appId: string, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    if (!id || app.columnKey !== "verify" || app.templateId !== HUMAN_REVIEW_CATALOG_ID) return [];
    return [
      {
        id: `human-review-${id}`,
        kind: "human-review" as const,
        name: app.name?.trim() || HUMAN_REVIEW_ENTRY.name,
        trigger: HUMAN_REVIEW_ENTRY.trigger,
        action: HUMAN_REVIEW_ENTRY.action,
        iconSrc: HUMAN_REVIEW_ENTRY.iconSrc,
        iconAlt: HUMAN_REVIEW_ENTRY.iconAlt,
        health: "healthy" as const,
        runningCount: runningCount(id, workOrders),
        catalogId: HUMAN_REVIEW_CATALOG_ID,
        canvasId: id,
      },
    ];
  });
}
