import type { FactoriesWorkOrder } from "@/api-client";

import { CUSTOM_VERIFICATION_CATALOG_ID } from "./columnAutomationCatalog";
import type { ColumnAutomation } from "./columnAutomations";
import { customVerificationModeCopy, type CustomVerificationMode } from "./customVerification";

export function customVerificationColumnAutomation(
  app: {
    id?: string;
    name?: string;
    templateId?: string;
    customVerificationMode?: CustomVerificationMode;
    customVerificationIcon?: string;
  },
  workOrders: FactoriesWorkOrder[],
  runningCountForApp: (appId: string | undefined, orders: FactoriesWorkOrder[]) => number,
): ColumnAutomation | null {
  const id = app.id?.trim();
  if (!id) {
    return null;
  }
  const copy = customVerificationModeCopy(app.customVerificationMode ?? "manual");
  return {
    id: `custom-verification-${id}`,
    kind: "custom-verification",
    name: app.name?.trim() || "Custom automation",
    trigger: copy.trigger,
    action: copy.action,
    iconSrc: "",
    iconAlt: "",
    iconName: app.customVerificationIcon,
    health: "healthy",
    runningCount: runningCountForApp(id, workOrders),
    catalogId: CUSTOM_VERIFICATION_CATALOG_ID,
    canvasId: id,
  };
}
