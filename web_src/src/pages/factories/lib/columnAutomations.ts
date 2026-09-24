import type { FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler, FactoriesWorkOrder } from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";

import { lineIntakeSourceForApiSource } from "../pages/lineIntakeModel";
import { PR_FEEDBACK_SOURCES, prFeedbackSourceId } from "../pages/prFeedbackSettingsModel";
import {
  buildColumnAutomationActivity,
  emptyColumnAutomationActivity,
  type ColumnAutomationActivity,
} from "./columnAutomationActivity";
import {
  AGENT_STEP_CATALOG_ID,
  ANALYSIS_CATALOG_ID,
  ANALYSIS_ENTRY,
  CUSTOM_CATALOG_ID,
  CUSTOM_ENTRY,
  EVENT_CUSTOM_ENTRY,
  PR_CLOSURE_CATALOG_ID,
  PR_CLOSURE_ENTRY,
  prFeedbackSentence,
} from "./columnAutomationCatalog";
import {
  factoryColumnAutomationViewPath,
  factoryIntakePath,
  factoryPlanningPath,
  factoryPlanningSetupPath,
  factoryPRFeedbackPath,
} from "./factoryPagePaths";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";
import { findBacklogAutomationApp, findClosureAutomationApp, type LinePhaseColumn } from "./linePhaseRuns";

export { catalogForColumn, onlyCustomCatalogRemains, takenCatalogIds } from "./columnAutomationCatalog";
export { COLUMN_AUTOMATIONS_COPY } from "./columnAutomationsCopy";

export type ColumnKey = "backlog" | `phase-${number}` | "verify" | "done";

export type ColumnAutomationKind =
  | "intake"
  | "analysis"
  | "agent-step"
  | "custom"
  | "pr-discussion"
  | "pr-checks"
  | "pr-closure";

export type ColumnAutomationHealth = "healthy" | "needs-repair" | "disabled";

export type ColumnAutomation = {
  id: string;
  kind: ColumnAutomationKind;
  name: string;
  trigger: string;
  action: string;
  iconSrc: string;
  iconAlt: string;
  health: ColumnAutomationHealth;
  runningCount: number;
  catalogId: string;
  canvasId?: string;
  activity?: ColumnAutomationActivity;
};

export type ColumnAutomationCatalogEntry = {
  id: string;
  kind: ColumnAutomationKind;
  name: string;
  description: string;
  trigger: string;
  action: string;
  iconSrc: string;
  iconAlt: string;
  unique: boolean;
};

export type ColumnAutomationsInput = {
  columnTitle: string;
  columns?: LinePhaseColumn[];
  intakes?: FactoriesFactoryIntake[];
  prFeedbackHandlers?: FactoriesFactoryPrFeedbackHandler[];
  apps?: Array<{ id?: string; name?: string; columnKey?: string }>;
  workOrders?: FactoriesWorkOrder[];
  planningEnabled?: boolean;
};

const PHASE_KEY_PATTERN = /^phase-(\d+)$/;

export function isColumnKey(value: string | null | undefined): value is ColumnKey {
  if (!value) {
    return false;
  }
  return value === "backlog" || value === "verify" || value === "done" || PHASE_KEY_PATTERN.test(value);
}

export function phaseIndexFromColumnKey(key: ColumnKey): number | undefined {
  const match = PHASE_KEY_PATTERN.exec(key);
  if (!match) {
    return undefined;
  }
  return Number.parseInt(match[1] ?? "", 10);
}

export function columnTitleForKey(key: ColumnKey, columns: LinePhaseColumn[] = []): string {
  if (key === "backlog") {
    return "Backlog";
  }
  if (key === "verify") {
    return "Verify";
  }
  if (key === "done") {
    return "Done";
  }
  const stepIndex = phaseIndexFromColumnKey(key);
  return columns.find((column) => column.stepIndex === stepIndex)?.stepName ?? `Phase ${(stepIndex ?? 0) + 1}`;
}

export function columnAutomationsEmptyCopy(columnTitle: string, key: ColumnKey): string {
  if (key === "backlog") {
    return "No automations. Add an intake to create tasks from a source.";
  }
  if (key === "verify") {
    return "No automations. Add a pull request listener.";
  }
  if (key === "done") {
    return "No automations. Tasks stay here when they finish.";
  }
  return `No automations. Tasks pass through ${columnTitle} without action.`;
}

export function columnAutomationsNeedRepair(automations: ColumnAutomation[]): boolean {
  return automations.some((automation) => automation.health === "needs-repair");
}

export function buildColumnAutomations(key: ColumnKey, input: ColumnAutomationsInput): ColumnAutomation[] {
  const workOrders = input.workOrders ?? [];
  const automations = automationsForColumn(key, input, workOrders);
  return automations.map((automation) => attachAutomationActivity(automation, workOrders));
}

function automationsForColumn(
  key: ColumnKey,
  input: ColumnAutomationsInput,
  workOrders: FactoriesWorkOrder[],
): ColumnAutomation[] {
  if (key === "backlog") {
    return [
      ...intakeAutomations(input.intakes ?? [], workOrders),
      ...analysisAutomation(input.apps ?? [], workOrders, input.planningEnabled !== false),
    ];
  }
  if (key === "verify") {
    return [
      ...prFeedbackAutomations(input.prFeedbackHandlers ?? [], workOrders),
      ...customColumnAutomations(input.apps ?? [], "verify", workOrders, new Set()),
    ];
  }
  if (key === "done") {
    const closure = closureAutomation(input.apps ?? [], workOrders);
    const skip = new Set(closure.flatMap((automation) => (automation.canvasId ? [automation.canvasId] : [])));
    return [...closure, ...customColumnAutomations(input.apps ?? [], "done", workOrders, skip)];
  }
  return agentStepAutomation(key, input.columnTitle, input.columns ?? [], workOrders);
}

function attachAutomationActivity(automation: ColumnAutomation, workOrders: FactoriesWorkOrder[]): ColumnAutomation {
  return {
    ...automation,
    activity: buildColumnAutomationActivity({
      canvasId: automation.canvasId,
      workOrders,
      runningCount: automation.runningCount,
    }),
  };
}

function intakeAutomations(intakes: FactoriesFactoryIntake[], workOrders: FactoriesWorkOrder[]): ColumnAutomation[] {
  return intakes.flatMap((intake) => {
    const intakeId = intake.id?.trim();
    const source = lineIntakeSourceForApiSource(intake.source);
    if (!intakeId || !source) {
      return [];
    }
    return [
      {
        id: intakeId,
        kind: "intake" as const,
        name: intake.name?.trim() || source.name,
        trigger: source.listen.label,
        action: source.accept.label,
        iconSrc: source.iconSrc,
        iconAlt: source.iconAlt,
        health: "healthy",
        runningCount: runningCountForApp(intake.canvasId, workOrders),
        catalogId: source.id,
        canvasId: intake.canvasId?.trim() || undefined,
      },
    ];
  });
}

function analysisAutomation(
  apps: Array<{ id?: string; name?: string }>,
  workOrders: FactoriesWorkOrder[],
  planningEnabled: boolean,
): ColumnAutomation[] {
  const app = findBacklogAutomationApp(apps);
  if (!app) {
    return [];
  }
  return [
    {
      id: `analysis-${app.id}`,
      kind: "analysis",
      name: app.name === "Ingest" || app.name === "Backlog" ? "Task analysis" : app.name,
      trigger: ANALYSIS_ENTRY.trigger,
      action: ANALYSIS_ENTRY.action,
      iconSrc: ANALYSIS_ENTRY.iconSrc,
      iconAlt: ANALYSIS_ENTRY.iconAlt,
      health: planningEnabled ? "healthy" : "disabled",
      runningCount: runningCountForApp(app.id, workOrders),
      catalogId: ANALYSIS_CATALOG_ID,
      canvasId: app.id,
    },
  ];
}

function prFeedbackAutomation(
  handler: FactoriesFactoryPrFeedbackHandler,
  workOrders: FactoriesWorkOrder[],
): ColumnAutomation | null {
  const handlerId = handler.id?.trim();
  if (!handlerId) {
    return null;
  }
  const sourceId = prFeedbackSourceId(handler.source);
  const source = PR_FEEDBACK_SOURCES.find((entry) => entry.id === sourceId);
  const sentence = prFeedbackSentence(sourceId);
  return {
    id: handlerId,
    kind: sentence.kind,
    name: handler.name?.trim() || source?.name || "PR feedback",
    trigger: sentence.trigger,
    action: sentence.action,
    iconSrc: source?.iconSrc ?? githubIcon,
    iconAlt: source?.iconAlt ?? "GitHub",
    health: handler.healthy === false ? "needs-repair" : "healthy",
    runningCount: runningCountForApp(handler.canvasId, workOrders),
    catalogId: sourceId,
    canvasId: handler.canvasId?.trim() || undefined,
  };
}

function prFeedbackAutomations(
  handlers: FactoriesFactoryPrFeedbackHandler[],
  workOrders: FactoriesWorkOrder[],
): ColumnAutomation[] {
  return handlers.flatMap((handler) => {
    const automation = prFeedbackAutomation(handler, workOrders);
    return automation ? [automation] : [];
  });
}

function customColumnAutomations(
  apps: Array<{ id?: string; name?: string; columnKey?: string }>,
  columnKey: "verify" | "done",
  workOrders: FactoriesWorkOrder[],
  skipIds: Set<string>,
): ColumnAutomation[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    if (!id || skipIds.has(id) || app.columnKey !== columnKey) {
      return [];
    }
    return [
      {
        id: `custom-${id}`,
        kind: "custom" as const,
        name: app.name?.trim() || EVENT_CUSTOM_ENTRY.name,
        trigger: EVENT_CUSTOM_ENTRY.trigger,
        action: EVENT_CUSTOM_ENTRY.action,
        iconSrc: EVENT_CUSTOM_ENTRY.iconSrc,
        iconAlt: EVENT_CUSTOM_ENTRY.iconAlt,
        health: "healthy" as const,
        runningCount: runningCountForApp(id, workOrders),
        catalogId: CUSTOM_CATALOG_ID,
        canvasId: id,
      },
    ];
  });
}

function closureAutomation(
  apps: Array<{ id?: string; name?: string }>,
  workOrders: FactoriesWorkOrder[],
): ColumnAutomation[] {
  const app = findClosureAutomationApp(apps);
  if (!app) {
    return [];
  }
  return [
    {
      id: `closure-${app.id}`,
      kind: "pr-closure",
      name: app.name,
      trigger: PR_CLOSURE_ENTRY.trigger,
      action: PR_CLOSURE_ENTRY.action,
      iconSrc: githubIcon,
      iconAlt: "GitHub",
      health: "healthy",
      runningCount: runningCountForApp(app.id, workOrders),
      catalogId: PR_CLOSURE_CATALOG_ID,
      canvasId: app.id,
    },
  ];
}

function agentStepAutomation(
  key: ColumnKey,
  columnTitle: string,
  columns: LinePhaseColumn[],
  workOrders: FactoriesWorkOrder[],
): ColumnAutomation[] {
  const stepIndex = phaseIndexFromColumnKey(key);
  const column = columns.find((entry) => entry.stepIndex === stepIndex);
  const appId = column?.appId?.trim();
  if (!appId) {
    return [];
  }
  const title = column?.stepName || columnTitle;
  return [
    {
      id: `step-${stepIndex}-${appId}`,
      kind: "agent-step",
      name: title,
      trigger: `On task in ${title}`,
      action: `Run the ${title} agent`,
      iconSrc: "",
      iconAlt: "",
      health: "healthy",
      runningCount: runningCountForApp(appId, workOrders),
      catalogId: AGENT_STEP_CATALOG_ID,
      canvasId: appId,
    },
  ];
}

export function runningCountForApp(appId: string | undefined, workOrders: FactoriesWorkOrder[]): number {
  const id = appId?.trim();
  if (!id) {
    return 0;
  }
  let count = 0;
  for (const order of workOrders) {
    for (const dispatch of order.lineDispatches ?? []) {
      for (const execution of dispatch.stepExecutions ?? []) {
        if (execution.run?.appId === id && isActiveWorkOrderExecution(execution)) {
          count += 1;
        }
      }
    }
  }
  return count;
}

/**
 * Path for an existing column automation. Opens the popup on the first tab.
 * Task analysis opens the Planning setup wizard until the factory confirms it.
 */
export function columnAutomationOpenPath(
  automation: ColumnAutomation,
  args: { organizationId: string; factoryKey: string; lineId?: string; planningSetupCompleted?: boolean },
): string | undefined {
  if (automation.kind === "intake") {
    return factoryIntakePath(args.organizationId, args.factoryKey, args.lineId, automation.id);
  }
  if (automation.kind === "pr-discussion" || automation.kind === "pr-checks") {
    return factoryPRFeedbackPath(args.organizationId, args.factoryKey, args.lineId, undefined, automation.id);
  }
  if (automation.kind === "analysis") {
    if (args.planningSetupCompleted === false && args.lineId) {
      return factoryPlanningSetupPath(args.organizationId, args.factoryKey, args.lineId);
    }
    return factoryPlanningPath(args.organizationId, args.factoryKey, args.lineId);
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
  const isPhaseCustom = entry.kind === "custom" && entry.trigger === CUSTOM_ENTRY.trigger;
  const trigger = entry.kind === "agent-step" || isPhaseCustom ? `On task in ${columnTitle}` : entry.trigger;
  const action =
    entry.kind === "agent-step"
      ? `Run the ${columnTitle} agent`
      : isPhaseCustom
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
