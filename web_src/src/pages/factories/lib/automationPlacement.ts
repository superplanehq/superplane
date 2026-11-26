import type { ColumnAutomation } from "./columnAutomations";
import type { LineBoardStage } from "./lineBoardStages";

export type AutomationMove = {
  type: string;
  automationId: string;
  executionKey: string;
  sourceKey: string;
  destinationKey: string;
};

export function cardMove(
  card: ColumnAutomation,
  sourceKey: string,
  destinationKey: string,
): AutomationMove | undefined {
  const type =
    card.kind === "agent-step" || card.kind === "intake"
      ? card.kind
      : card.kind === "pr-discussion" || card.kind === "pr-checks"
        ? "feedback"
        : "canvas";
  const automationId = type === "feedback" || type === "intake" ? card.id : card.canvasId;
  if (!automationId || !card.executionKey) return undefined;
  return { type, automationId, executionKey: card.executionKey, sourceKey, destinationKey };
}

export function supportsCard(stage: LineBoardStage, card: ColumnAutomation) {
  if (card.kind !== "intake") return stage.kind !== "STAGE_KIND_BACKLOG";
  if (card.executionKey?.startsWith("custom-")) return stage.key.startsWith("custom-") || stage.key.startsWith("new-");
  return !card.taskStageTargets || card.taskStageTargets.includes(stage.key);
}

export function savedCardStage(card: ColumnAutomation, originKey: string, saved: LineBoardStage[]) {
  const key =
    card.kind === "agent-step"
      ? saved.find((stage) => stage.key === originKey)?.agentDisplayKey
      : card.displayColumnKey;
  return key && saved.some((stage) => stage.key === key) ? key : originKey;
}
