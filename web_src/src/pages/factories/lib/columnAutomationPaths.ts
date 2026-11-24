import type { ColumnAutomation } from "./columnAutomations";
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
  const customStageEditor =
    automation.kind === "custom" && automation.columnKey?.startsWith("custom-") && args.customEditingEnabled;
  const customSetupEditor =
    automation.kind === "custom-verification" &&
    (!automation.columnKey?.startsWith("custom-") || args.customEditingEnabled);
  if ((customSetupEditor || customStageEditor) && automation.canvasId && args.lineId) {
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
