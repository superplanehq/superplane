import type { ColumnAutomation } from "./columnAutomations";
import { isCustomStageColumn } from "./columnKeys";
import { customAutomationColumn } from "./customAutomation";
import { factoryCustomAutomationSetupPath } from "./customAutomationPaths";
import {
  factoryColumnAutomationViewPath,
  factoryIntakePath,
  factoryPlanningPath,
  factoryPRFeedbackPath,
} from "./factoryPagePaths";

export function columnAutomationOpenPath(
  automation: ColumnAutomation,
  args: { organizationId: string; factoryKey: string; lineId?: string; customEditingEnabled?: boolean },
): string | undefined {
  if (automation.kind === "intake") {
    return factoryIntakePath(args.organizationId, args.factoryKey, args.lineId, automation.id);
  }
  if (automation.kind === "pr-discussion" || automation.kind === "pr-checks") {
    return factoryPRFeedbackPath(args.organizationId, args.factoryKey, args.lineId, undefined, automation.id);
  }
  if (automation.kind === "analysis") {
    return factoryPlanningPath(args.organizationId, args.factoryKey, args.lineId);
  }
  if (usesCustomSetupEditor(automation, args.customEditingEnabled) && automation.canvasId && args.lineId) {
    return factoryCustomAutomationSetupPath(
      args.organizationId,
      args.factoryKey,
      args.lineId,
      automation.canvasId,
      automation.columnKey ? customAutomationColumn(automation.columnKey) : undefined,
    );
  }
  if (!automation.canvasId) return undefined;
  return factoryColumnAutomationViewPath(args.organizationId, args.factoryKey, args.lineId, automation.canvasId);
}

function usesCustomSetupEditor(automation: ColumnAutomation, customEditingEnabled = false): boolean {
  if (isCustomStageColumn(automation.columnKey)) {
    return customEditingEnabled && (automation.kind === "custom" || automation.kind === "custom-verification");
  }
  return automation.kind === "custom-verification";
}
