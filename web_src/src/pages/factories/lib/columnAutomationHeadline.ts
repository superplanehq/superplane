import { lineIntakeSourceById } from "../pages/lineIntakeModel";
import type { ColumnAutomation, ColumnAutomationKind } from "./columnAutomations";

/**
 * One short sentence for the column header. It states what the automation
 * does for tasks in that column, in the simple present.
 */
export function columnAutomationHeadline(automation: ColumnAutomation): string {
  if (automation.kind === "intake") {
    const source = lineIntakeSourceById(automation.catalogId);
    return `Listens to ${source?.name ?? automation.name}`;
  }
  if (automation.kind === "agent-step") {
    return `Runs the ${automation.name} agent`;
  }
  if (automation.kind === "custom") {
    return `Runs the ${automation.name} automation`;
  }
  return FIXED_HEADLINES[automation.kind];
}

const FIXED_HEADLINES: Record<Exclude<ColumnAutomationKind, "intake" | "agent-step" | "custom">, string> = {
  analysis: "Scores new tasks",
  "pr-discussion": "Addresses pull request comments",
  "pr-checks": "Fixes failing status checks",
  "pr-closure": "Closes tasks when pull requests merge",
  "issue-closure": "Closes backlog tasks when GitHub issues close",
};

/**
 * Rows every column header reserves. The largest column sets the count so
 * card lists start at the same height across the board. Never below one.
 */
export function columnAutomationHeaderRowCount(columns: ReadonlyArray<ReadonlyArray<ColumnAutomation>>): number {
  return Math.max(1, ...columns.map((automations) => automations.length));
}
