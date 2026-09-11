import { emptyColumnAutomationActivity } from "./columnAutomationActivity";
import type { ColumnAutomation, ColumnAutomationCatalogEntry } from "./columnAutomations";
import { factoryColumnAutomationViewPath, factoryIntakePath, factoryPRFeedbackPath } from "./factoryPagePaths";

/** Path for an existing column automation. Opens the popup on the first tab. */
export function columnAutomationOpenPath(
  automation: ColumnAutomation,
  args: { organizationId: string; factoryKey: string; lineId?: string },
): string | undefined {
  if (automation.kind === "intake") {
    return factoryIntakePath(args.organizationId, args.factoryKey, args.lineId, automation.id);
  }
  if (automation.kind === "pr-discussion" || automation.kind === "pr-checks") {
    return factoryPRFeedbackPath(args.organizationId, args.factoryKey, args.lineId, undefined, automation.id);
  }
  if (!automation.canvasId) {
    return undefined;
  }
  return factoryColumnAutomationViewPath(args.organizationId, args.factoryKey, args.lineId, automation.canvasId);
}

export type ColumnAutomationViewTab = "general" | "agent" | "automation";

/** Tab order: form first, then agent, then the canvas. */
export function columnAutomationViewTabs(input: { hasGeneral: boolean; hasAgent: boolean }): ColumnAutomationViewTab[] {
  const tabs: ColumnAutomationViewTab[] = [];
  if (input.hasGeneral) {
    tabs.push("general");
  }
  if (input.hasAgent) {
    tabs.push("agent");
  }
  tabs.push("automation");
  return tabs;
}

export function applyColumnAutomationsOverlay(
  automations: ColumnAutomation[],
  extra: ColumnAutomation[] | undefined,
  disabledIds: readonly string[],
  removedIds: readonly string[],
): ColumnAutomation[] {
  const removed = new Set(removedIds);
  const disabled = new Set(disabledIds);
  return [...automations, ...(extra ?? [])]
    .filter((automation) => !removed.has(automation.id))
    .map((automation) => (disabled.has(automation.id) ? { ...automation, health: "disabled" } : automation));
}

export function catalogEntryToAutomation(
  entry: ColumnAutomationCatalogEntry,
  columnTitle: string,
  id: string,
): ColumnAutomation {
  const trigger = entry.kind === "agent-step" || entry.kind === "custom" ? `On task in ${columnTitle}` : entry.trigger;
  const action =
    entry.kind === "agent-step"
      ? `Run the ${columnTitle} agent`
      : entry.kind === "custom"
        ? `Run the ${columnTitle} automation`
        : entry.action;
  return {
    id,
    kind: entry.kind,
    name: entry.name,
    trigger,
    action,
    iconSrc: entry.iconSrc,
    iconAlt: entry.iconAlt,
    health: "healthy",
    runningCount: 0,
    catalogId: entry.id,
    activity: emptyColumnAutomationActivity(0),
  };
}

export const COLUMN_AUTOMATIONS_COPY = {
  menuLabel: "Automations",
  addLabel: "Add automation",
  addHint: "Choose a trigger and an action.",
  pickerTitle: "Add automation",
  pickerDescription: "Choose an automation for this column.",
  sourceTaken: "This automation is already configured.",
  needsRepairLabel: "Needs repair",
  disabledLabel: "Disabled",
  editLabel: "Edit automation",
  tabsLabel: "Automation sections",
  generalTab: "General",
  agentTab: "Agent",
  automationTab: "Automation",
  viewLoading: "The automation is loading.",
  viewEmpty: "This automation has no canvas yet.",
  viewError: "SuperPlane could not load the automation.",
  viewRetry: "Try again",
  rowsEmpty: "No automations",
  rowMenu: "Open automation",
  viewMenuLabel: "Board view",
  viewOptions: "View options",
  viewNames: "Automation names",
  viewIcons: "Automation icons",
  lastRunPassed: "Passed",
  lastRunFailed: "Failed",
  activityRunning: "running",
} as const;
