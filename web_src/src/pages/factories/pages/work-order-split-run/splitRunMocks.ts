import type {
  FactoriesAutomationRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
  FactoriesWorkOrderExecution,
} from "@/api-client";
import { UNKNOWN_ORG_USER_NAME, getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";
import { workOrderOwnerDisplay } from "../../lib/workOrderCreator";
import { latestDispatchForLine } from "../../lib/workOrderNumberResolution";
import { clockLabel, providerForName } from "./splitRunFormat";
import { canvasKeyForAutomation, lineAutomationPresentation } from "./splitRunCanvases";
import { SPLIT_RUN_RUNNING } from "./splitRunRunningFixture";
import {
  costUsdForDisplay,
  durationForExecution,
  elapsedForDisplay,
  lineStatusForDisplay,
  startedLabelForOrder,
  tokensLabelForDisplay,
} from "./splitRunWorkOrderDisplay";

import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import { intakeTicketAnalysisFixture, type LineIntakeAnalyzingTicket } from "../lineIntakeModel";
import { implementationPlanMarkdown, reviewCandidateForWorkOrderId } from "../onboarding/first-run/reviewCandidates";
import { DESCRIPTION_ARTIFACT, PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import type {
  RunOverlayProvider,
  RunOverlayStep,
  RunOverlayStepStatus,
} from "../work-order-run-overlay/workOrderRunOverlayMocks";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

export type SplitRunPhaseId = string;

export type SplitRunPhaseStatus = "passed" | "running" | "pending" | "waiting" | "failed";

export type SplitRunStreamEvent = "started" | "expanded" | "completed";

export type SplitRunStreamKind = "trigger" | "filter" | "if" | "action" | "agent" | "check";

export interface SplitRunStreamLine {
  id: string;
  nodeId?: string;
  event?: SplitRunStreamEvent;
  at: string;
  componentName: string;
  status: SplitRunPhaseStatus;
  duration?: string;
  detail?: string;
  artifact?: FactoriesWorkOrderArtifact;
  /** Agent transcript line. No checkmark. */
  note?: boolean;
  /** Nested tool call under a Claude Code step. */
  noteParentId?: string;
  noteDepth?: number;
  kind?: SplitRunStreamKind;
  /** Catalog identity: `Run Claude Code`, `github.addIssueLabel`. */
  componentType?: string;
  action?: string;
  iconSlug?: string;
  iconSrc?: string;
}

export interface SplitRunPhase {
  id: SplitRunPhaseId;
  name: string;
  status: SplitRunPhaseStatus;
  duration: string;
  /** Component that ran or is running in this phase. */
  componentName: string;
  artifacts: FactoriesWorkOrderArtifact[];
  /** Checks this automation reported, shown on the log row. */
  checks?: WorkOrderCheckPresentation[];
  stream: SplitRunStreamLine[];
  canvasSteps: RunOverlayStep[];
  appId?: string;
  runId?: string;
  /** Line automation canvas. `null` means a person created the work order. */
  canvasKey?: SplitRunIntakeCanvasKey | null;
  /** Trigger node that ran when the canvas has more than one start. */
  triggerName?: string;
  /** When set, the popup shows this canvas instead of the phase-name map. */
  canvas?: SplitRunCanvasModel;
}

export type SplitRunFooterTone = "waiting" | "draft" | "failed";

export type SplitRunBoardColumn = "backlog" | "implement" | "verify" | "done";

export type SplitRunIntakeCanvasKey = "intake" | "sentry" | "slack";

export interface SplitRunFixture {
  title: string;
  owner: OrgUserDisplay;
  elapsed: string;
  startedLabel: string;
  costUsd: string;
  tokensLabel: string;
  lineName: string;
  lineStatus: SplitRunPhaseStatus;
  currentPhaseId: SplitRunPhaseId;
  phases: SplitRunPhase[];
  waitingNotes: WorkOrderStatusNotePresentation[];
  checks: WorkOrderCheckPresentation[];
  footerTone?: SplitRunFooterTone;
}

export { SPLIT_RUN_RUNNING };

const UNKNOWN_OWNER: OrgUserDisplay = {
  id: "unknown",
  name: UNKNOWN_ORG_USER_NAME,
  initials: getUserInitials(UNKNOWN_ORG_USER_NAME) || "U",
};

const DRAFT_NEXT_STEP: WorkOrderStatusNotePresentation = {
  key: "start-plan",
  headline: "Start the next stage",
  text: "This work order is a draft. Dispatch it to the Plan and Implement line to start Implement.",
  cta: { label: "Dispatch" },
};

const IMPLEMENT_FAILED_NOTE: WorkOrderStatusNotePresentation = {
  key: "implement-failed",
  headline: "Implement did not pass",
  text: "Backend tests failed on the reconciliation worker. The implement step stopped. Open the run to see the diagnosis, then dispatch the line again.",
  cta: { label: "Open failed run", href: "https://superplanehq.semaphoreci.com/" },
  source: { name: "Implementation" },
};

export function phaseById(fixture: SplitRunFixture, id: SplitRunPhaseId): SplitRunPhase {
  const phase = fixture.phases.find((entry) => entry.id === id);
  if (!phase) {
    throw new Error(`Unknown split-run phase: ${id}`);
  }
  return phase;
}

export function splitRunStatusLabel(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "Completed";
  if (status === "running") return "Running";
  if (status === "waiting") return "Needs attention";
  if (status === "failed") return "Failed";
  return "Pending";
}

const PR_CLOSURE_APP_NAME = "PR Closure";

export function splitRunFixtureForWorkOrder(
  order?: FactoriesWorkOrder,
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null; detailHref?: string },
): SplitRunFixture {
  if (!order) {
    return SPLIT_RUN_RUNNING;
  }
  return mappedWorkOrderFixture(order, options);
}

function mappedWorkOrderFixture(
  order: FactoriesWorkOrder,
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null; detailHref?: string },
): SplitRunFixture {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order, options?.lineId);
  const current = pickCurrentExecution(executions);
  const phases = phasesForOrder(order, executions);
  return {
    title: order.title ?? "Work order",
    owner: workOrderOwnerDisplay(order, UNKNOWN_OWNER),
    elapsed: elapsedForDisplay(displayStatus, order),
    startedLabel: startedLabelForOrder(order),
    costUsd: costUsdForDisplay(order),
    tokensLabel: tokensLabelForDisplay(order),
    lineName: latestDispatchForLine(order, options?.lineId)?.line?.name ?? SPLIT_RUN_RUNNING.lineName,
    lineStatus: lineStatusForDisplay(displayStatus),
    currentPhaseId: current ? phaseIdForExecution(current) : (phases[0]?.id ?? ""),
    phases,
    ...reviewSurfaces(order, displayStatus, options?.checks, options?.lineId, options?.detailHref),
  };
}

function attentionNote(order: FactoriesWorkOrder, detailHref?: string): WorkOrderStatusNotePresentation {
  const name = order.assignees?.[0]?.name?.trim();
  return {
    key: "needs-attention",
    headline: "Needs attention",
    text: name ? `This work order needs attention from ${name}.` : "This work order needs attention.",
    cta: detailHref ? { label: "Open work order", href: detailHref } : undefined,
  };
}

function reviewSurfaces(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  apiChecks?: FactoriesWorkOrderCheck[],
  lineId?: string | null,
  detailHref?: string,
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footerTone"> {
  const executions = latestDispatchExecutions(order, lineId);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const implementFailed = column === "implement" && current?.result === "RESULT_FAILED";
  const showChecks = column === "verify" || column === "done";
  const checks = showChecks ? presentWorkOrderChecks(apiChecks ?? []) : [];

  if (displayStatus === "draft") {
    return { waitingNotes: [DRAFT_NEXT_STEP], checks: [], footerTone: "draft" };
  }
  if (implementFailed) {
    return { waitingNotes: [IMPLEMENT_FAILED_NOTE], checks: [], footerTone: "failed" };
  }
  if (displayStatus === "waiting") {
    const notes = presentWorkOrderStatusNotes(order.statusNotes, displayStatus);
    return {
      waitingNotes: notes.length > 0 ? notes : [attentionNote(order, detailHref)],
      checks: [],
      footerTone: "waiting",
    };
  }
  if (column === "implement" && current?.state === "STATE_PENDING") {
    return {
      waitingNotes: presentWorkOrderStatusNotes(order.statusNotes, displayStatus),
      checks: [],
      footerTone: "waiting",
    };
  }
  return { waitingNotes: [], checks };
}

function boardColumnFor(current: FactoriesWorkOrderExecution | undefined, executionCount: number): SplitRunBoardColumn {
  if (!current || executionCount === 0) {
    return "backlog";
  }
  const step = (current.step ?? "").toLowerCase();
  const index = current.stepIndex ?? -1;
  if (step.includes("done")) {
    return "done";
  }
  if (step.includes("verify")) {
    return "verify";
  }
  if (step.includes("implement")) {
    return "implement";
  }
  if (index >= 2) {
    return "done";
  }
  if (index === 1) {
    return "verify";
  }
  if (index === 0) {
    return "implement";
  }
  return "backlog";
}

function phasesForOrder(order: FactoriesWorkOrder, executions: FactoriesWorkOrderExecution[]): SplitRunPhase[] {
  return [
    ...sourcePhasesForOrder(order, executions.length > 0),
    ...executions.map((execution) => executionToPhase(order, execution)),
  ];
}

function sourcePhasesForOrder(order: FactoriesWorkOrder, isDownstream: boolean): SplitRunPhase[] {
  const automation = order.createdBy?.automation;
  const fromGitHubIngest =
    automation && intakeCanvasKeyFor({ id: automation.appId, name: automation.appName }) === "intake";
  if (fromGitHubIngest && isDownstream) {
    return intakeTicketAnalysisFixture(analysisTicketForOrder(order), { complete: true }).phases;
  }
  return [backlogSourcePhase(order)];
}

function analysisTicketForOrder(order: FactoriesWorkOrder): LineIntakeAnalyzingTicket {
  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate) {
    return {
      id: candidate.workOrderId,
      title: candidate.title,
      detailsMarkdown: candidate.issue.bodyMarkdown,
      issueKey: candidate.ticketKey,
      issueUrl: candidate.issue.url,
      planMarkdown: candidate.planMarkdown,
      confidenceScore: candidate.confidenceScore,
      confidenceSummary: candidate.summary,
      confidenceAnalysis: candidate.reasons.map((reason) => `- ${reason}`).join("\n"),
    };
  }
  return {
    id: order.id ?? "work-order",
    title: order.title ?? "Work order",
    detailsMarkdown: order.description,
    issueKey: order.key,
    planMarkdown: fallbackPlanMarkdown(order),
    confidenceScore: exampleConfidenceScore(order),
    confidenceSummary: order.title,
  };
}

function fallbackPlanMarkdown(order: FactoriesWorkOrder): string {
  const goal = order.title ?? "Implement the change.";
  return implementationPlanMarkdown({
    goal,
    files: ["See the work-order description for the files to change."],
    steps: [
      "Read the work order and the current implementation.",
      "Apply the change described in the work order.",
      "Add or update tests for the new behavior.",
    ],
    verify: ["The existing suite passes.", "The notes in the work order hold."],
  });
}

const EXAMPLE_CONFIDENCE_BY_ORDER_ID: Record<string, number> = {
  "wo-running-refunds": 4,
  "wo-board-implement-failed": 3,
  "wo-failed-refunds": 4,
  "wo-pr-closure-receipts": 5,
  "wo-board-done-canceled": 3,
};

function exampleConfidenceScore(order: FactoriesWorkOrder): number {
  if (order.id && EXAMPLE_CONFIDENCE_BY_ORDER_ID[order.id] != null) {
    return EXAMPLE_CONFIDENCE_BY_ORDER_ID[order.id];
  }
  return 4;
}

function backlogSourcePhase(order: FactoriesWorkOrder): SplitRunPhase {
  const description = descriptionArtifactForOrder(order);
  const automation = order.createdBy?.automation;
  if (automation) {
    return automationBacklogPhase(order, automation, description);
  }
  return manualBacklogPhase(order, description);
}

function automationBacklogPhase(
  order: FactoriesWorkOrder,
  automation: FactoriesAutomationRef,
  description: FactoriesWorkOrderArtifact,
): SplitRunPhase {
  const app = { id: automation.appId, name: automation.appName };
  const { name, componentName } = lineAutomationPresentation(app);
  const at = clockLabel(order.createdAt);
  return {
    id: "backlog",
    name,
    status: "passed",
    duration: "2s",
    componentName,
    artifacts: [description],
    stream: [
      {
        id: "backlog-create",
        at,
        componentName: automation.nodeName?.trim() || "Create Work Order",
        status: "passed",
        duration: "2s",
        artifact: description,
        kind: "action",
        componentType: "Create Work Order",
        action: "passed",
        iconSlug: "factory",
      },
    ],
    canvasSteps: [],
    appId: automation.appId,
    canvasKey: intakeCanvasKeyFor(app),
    triggerName: automation.nodeName,
  };
}

function manualBacklogPhase(order: FactoriesWorkOrder, description: FactoriesWorkOrderArtifact): SplitRunPhase {
  const creator = order.createdBy?.user?.name?.trim();
  const at = clockLabel(order.createdAt);
  const line = creator ? `${creator} created this work order manually.` : "Created this work order manually.";
  return {
    id: "backlog",
    name: "Backlog",
    status: "passed",
    duration: "2s",
    componentName: "Created manually",
    artifacts: [description],
    stream: [
      {
        id: "backlog-created",
        at,
        componentName: line,
        status: "passed",
        duration: "2s",
        artifact: description,
        kind: "action",
        componentType: "Create Work Order",
        action: "passed",
        iconSlug: "user",
      },
    ],
    canvasSteps: [],
    canvasKey: null,
  };
}

function intakeCanvasKeyFor(app: { id?: string; name?: string }): SplitRunIntakeCanvasKey {
  const key = canvasKeyForAutomation(app);
  if (key === "sentry" || key === "slack" || key === "intake") {
    return key;
  }
  return "intake";
}

function descriptionArtifactForOrder(order: FactoriesWorkOrder): FactoriesWorkOrderArtifact {
  return {
    ...DESCRIPTION_ARTIFACT,
    id: `art-description-${order.id ?? "draft"}`,
    data: {
      name: "description.md",
      title: "description.md",
      body: order.description ?? "",
    },
  };
}

function executionToPhase(order: FactoriesWorkOrder, execution: FactoriesWorkOrderExecution): SplitRunPhase {
  const status = statusForExecution(execution);
  const { name, componentName } = lineAutomationPresentation(execution.run, execution.step);
  const duration = durationForExecution(execution, status);
  const artifacts = artifactsForLineExecution(order, execution);
  const line: SplitRunStreamLine = {
    id: execution.id ?? name,
    at: clockLabel(execution.updatedAt ?? execution.createdAt),
    componentName,
    status,
    duration,
    detail: execution.step,
    artifact: artifacts[0],
    kind: "action",
    componentType: componentName,
    action: status === "passed" ? "passed" : status === "failed" ? "failed" : status === "running" ? "running" : "—",
    iconSlug: "box",
  };
  return {
    id: phaseIdForExecution(execution),
    name,
    status,
    duration,
    componentName,
    artifacts,
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: execution.run?.appId,
    runId: execution.run?.id,
  };
}

function artifactsForLineExecution(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
): FactoriesWorkOrderArtifact[] {
  const step = (execution.step ?? "").toLowerCase();
  if (step.includes("implement")) {
    return [branchArtifactForOrder(order)];
  }
  if (step.includes("verify")) {
    return [pullRequestArtifactForOrder(order, "open")];
  }
  if (step.includes("done")) {
    if (execution.result === "RESULT_CANCELLED") {
      return [canceledNotesArtifact(order)];
    }
    if (isPrClosureRun(execution)) {
      return [PR_CLOSURE_PR_ARTIFACT];
    }
    return [pullRequestArtifactForOrder(order, order.result === "RESULT_REJECTED" ? "closed" : "merged")];
  }
  return [];
}

function branchArtifactForOrder(order: FactoriesWorkOrder): FactoriesWorkOrderArtifact {
  const name = `feature/${(order.key ?? order.id ?? "change").toLowerCase()}`;
  return {
    id: `art-branch-${order.id ?? "order"}`,
    type: "TYPE_BRANCH",
    data: {
      name,
      url: `https://github.com/example/ledger/tree/${name}`,
    },
  };
}

function pullRequestNumberForOrder(order: FactoriesWorkOrder): number {
  if (order.id === "wo-failed-refunds") {
    return 6812;
  }
  if (order.id === "wo-pr-closure-receipts") {
    return 510;
  }
  const number = Number(order.number);
  return Number.isFinite(number) && number > 0 ? number + 400 : 400;
}

function pullRequestArtifactForOrder(order: FactoriesWorkOrder, state: string): FactoriesWorkOrderArtifact {
  const number = pullRequestNumberForOrder(order);
  return {
    id: `art-pr-${order.id ?? "order"}`,
    type: "TYPE_PR",
    data: {
      url: `https://github.com/example/ledger/pull/${number}`,
      title: order.title ?? "Pull request",
      number,
      state,
    },
  };
}

function canceledNotesArtifact(order: FactoriesWorkOrder): FactoriesWorkOrderArtifact {
  return {
    id: `art-notes-${order.id ?? "order"}`,
    type: "TYPE_MARKDOWN",
    data: {
      name: "notes.md",
      title: "notes.md",
      body: "The work order was canceled.",
    },
  };
}

function streamLineToCanvasStep(line: SplitRunStreamLine, provider: RunOverlayProvider): RunOverlayStep {
  return {
    id: line.id,
    title: line.componentName,
    componentName: line.componentName,
    provider,
    status: canvasStatus(line.status),
    detail: line.detail,
    duration: line.duration === "—" ? undefined : line.duration,
  };
}

function canvasStatus(status: SplitRunPhaseStatus): RunOverlayStepStatus {
  if (status === "waiting") return "pending";
  return status;
}

function statusForExecution(execution: FactoriesWorkOrderExecution): SplitRunPhaseStatus {
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING") {
    return "running";
  }
  if (execution.state === "STATE_PENDING") {
    return "pending";
  }
  if (execution.result === "RESULT_FAILED") {
    return "failed";
  }
  if (execution.result === "RESULT_PASSED") {
    return "passed";
  }
  return "waiting";
}

function phaseIdForExecution(execution: FactoriesWorkOrderExecution): string {
  const name = (execution.step ?? "step").toLowerCase().replace(/[^a-z0-9]+/g, "-");
  return `${name}-${execution.stepIndex ?? 0}`;
}

function latestDispatchExecutions(order: FactoriesWorkOrder, lineId?: string | null): FactoriesWorkOrderExecution[] {
  return latestDispatchForLine(order, lineId)?.stepExecutions ?? [];
}

function pickCurrentExecution(executions: FactoriesWorkOrderExecution[]): FactoriesWorkOrderExecution | undefined {
  const active = executions.filter(
    (execution) =>
      execution.state === "STATE_STARTED" ||
      execution.state === "STATE_PENDING" ||
      execution.state === "STATE_CANCELLING" ||
      execution.result === "RESULT_FAILED",
  );
  const pool = active.length > 0 ? active : executions;
  return pool.reduce<FactoriesWorkOrderExecution | undefined>((best, candidate) => {
    if (!best) {
      return candidate;
    }
    return (candidate.stepIndex ?? -1) >= (best.stepIndex ?? -1) ? candidate : best;
  }, undefined);
}

function isPrClosureRun(execution: FactoriesWorkOrderExecution): boolean {
  return execution.run?.appName === PR_CLOSURE_APP_NAME;
}
