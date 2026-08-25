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

import { VERIFY_STEP_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { CONFIDENCE_SCORE_MAX } from "../../lib/confidenceScore";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import {
  buildSplitRunFooter,
  doneFooterForStatus,
  type SplitRunFooter,
  type SplitRunFooterKind,
  type SplitRunFooterTone,
} from "./splitRunFooter";
import { intakeTicketAnalysisFixture, type LineIntakeAnalyzingTicket } from "../lineIntakeModel";
import { implementationPlanMarkdown, reviewCandidateForWorkOrderId } from "../onboarding/first-run/reviewCandidates";
import { DESCRIPTION_ARTIFACT, PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import type {
  RunOverlayProvider,
  RunOverlayStep,
  RunOverlayStepStatus,
} from "../work-order-run-overlay/workOrderRunOverlayMocks";
import type { SplitRunCanvasKey, SplitRunCanvasModel } from "./splitRunCanvases";

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
  canvasKey?: SplitRunCanvasKey | null;
  /** Trigger node that ran when the canvas has more than one start. */
  triggerName?: string;
  /** When set, the popup shows this canvas instead of the phase-name map. */
  canvas?: SplitRunCanvasModel;
}

export type { SplitRunFooter, SplitRunFooterKind, SplitRunFooterTone };

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
  footer: SplitRunFooter;
  footerTone: SplitRunFooterTone;
}

export { SPLIT_RUN_RUNNING };

const UNKNOWN_OWNER: OrgUserDisplay = {
  id: "unknown",
  name: UNKNOWN_ORG_USER_NAME,
  initials: getUserInitials(UNKNOWN_ORG_USER_NAME) || "U",
};

const IMPLEMENT_FAILED_NOTE: WorkOrderStatusNotePresentation = {
  key: "implement-failed",
  headline: "Implement did not pass",
  text: "Backend tests failed on the reconciliation worker. The implement step stopped. Open the run to see the diagnosis, then dispatch the line again.",
  cta: { label: "Open failed run", href: "https://superplanehq.semaphoreci.com/" },
  source: { name: "Implementation" },
};

const WAITING_FALLBACK_NOTE: WorkOrderStatusNotePresentation = {
  key: "waiting-person",
  headline: "A person must act",
  text: "The line stopped and waits on a person. Open the last step in the log to see what stopped it.",
};

/**
 * The footer note for a draft: where the order came from, that a plan
 * exists and is editable, and the confidence reasoning. The log holds the
 * step-by-step detail; this is the readable summary.
 */
function draftFooterNote(order: FactoriesWorkOrder): WorkOrderStatusNotePresentation {
  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate) {
    return {
      key: "draft-plan-ready",
      headline: "Review the plan, then start",
      text: [
        `From GitHub issue [${candidate.ticketKey}](${candidate.issue.url}). SuperPlane analyzed the ticket and wrote an implementation plan. Open **plan.md** in the Create plan step to review or edit it.`,
        "",
        `Confidence ${candidate.confidenceScore}/${CONFIDENCE_SCORE_MAX} (${candidate.confidenceBand}):`,
        ...candidate.reasons.map((reason) => `- ${reason}`),
      ].join("\n"),
    };
  }
  return {
    key: "draft-start",
    headline: "Start this work order",
    text: `${draftSourceSentence(order)} The details are in **description.md**. Start it to plan and implement the change.`,
  };
}

function draftSourceSentence(order: FactoriesWorkOrder): string {
  const automation = order.createdBy?.automation;
  if (automation) {
    const { componentName } = lineAutomationPresentation({ id: automation.appId, name: automation.appName });
    return `${componentName} created this work order.`;
  }
  const creator = order.createdBy?.user?.name?.trim();
  if (creator) {
    return `${creator} created this work order manually.`;
  }
  return "A person created this work order manually.";
}

function runningFooterNote(current: FactoriesWorkOrderExecution | undefined): WorkOrderStatusNotePresentation {
  if (!current) {
    return {
      key: "running-step",
      headline: "The line is running",
      text: "SuperPlane works on this order now. The log shows live progress.",
    };
  }
  const { name, componentName } = lineAutomationPresentation(current.run, current.step);
  return {
    key: "running-step",
    headline: `${name} is running`,
    text: `${componentName} works on this step now. The log shows live progress.`,
  };
}

const AUTO_EXPAND_STATUSES = new Set<SplitRunPhaseStatus>(["running", "waiting", "failed"]);

/** Open the current step only when it is running, waiting, or failed. */
export function autoExpandedPhaseId(fixture: SplitRunFixture): SplitRunPhaseId | null {
  const current = fixture.phases.find((phase) => phase.id === fixture.currentPhaseId);
  if (!current || !AUTO_EXPAND_STATUSES.has(current.status)) {
    return null;
  }
  return current.id;
}

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
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null },
): SplitRunFixture {
  if (!order) {
    return SPLIT_RUN_RUNNING;
  }
  return mappedWorkOrderFixture(order, options);
}

function mappedWorkOrderFixture(
  order: FactoriesWorkOrder,
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null },
): SplitRunFixture {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order, options?.lineId);
  const current = pickCurrentExecution(executions);
  const phases = phasesForOrder(order, executions, options?.checks);
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
    ...reviewSurfaces(order, displayStatus, options?.lineId, phases, options?.checks),
  };
}

function reviewSurfaces(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  lineId: string | null | undefined,
  phases: SplitRunPhase[],
  apiChecks?: FactoriesWorkOrderCheck[],
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  const executions = latestDispatchExecutions(order, lineId);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const implementFailed = column === "implement" && current?.result === "RESULT_FAILED";
  const checks = overviewChecks(phases, apiChecks);

  if (displayStatus === "draft") {
    return surfaces(buildSplitRunFooter({ kind: "draft", note: draftFooterNote(order) }), [], checks);
  }
  if (implementFailed) {
    return surfaces(
      buildSplitRunFooter({ kind: "failed", note: IMPLEMENT_FAILED_NOTE }),
      [IMPLEMENT_FAILED_NOTE],
      checks,
    );
  }
  if (displayStatus === "waiting") {
    const notes = presentWorkOrderStatusNotes(order.statusNotes, displayStatus);
    return surfaces(buildSplitRunFooter({ kind: "waiting", note: notes[0] ?? WAITING_FALLBACK_NOTE }), notes, checks);
  }
  if (column === "implement" && current?.state === "STATE_PENDING") {
    const notes = presentWorkOrderStatusNotes(order.statusNotes, displayStatus);
    return surfaces(buildSplitRunFooter({ kind: "waiting", note: notes[0] ?? WAITING_FALLBACK_NOTE }), notes, checks);
  }
  if (displayStatus === "running") {
    return surfaces(buildSplitRunFooter({ kind: "running", note: runningFooterNote(current) }), [], checks);
  }
  return surfaces(doneFooterForStatus(displayStatus), [], checks);
}

function overviewChecks(phases: SplitRunPhase[], apiChecks?: FactoriesWorkOrderCheck[]): WorkOrderCheckPresentation[] {
  const presented = presentWorkOrderChecks(apiChecks ?? []);
  if (presented.length > 0) {
    return presented;
  }
  return phases.flatMap((phase) => phase.checks ?? []);
}

function surfaces(
  footer: SplitRunFooter,
  waitingNotes: WorkOrderStatusNotePresentation[] = [],
  checks: WorkOrderCheckPresentation[] = [],
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  return { waitingNotes, checks, footer, footerTone: footer.kind };
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

const VERIFY_STEP_KEYS = new Set(VERIFY_STEP_CHECKS.map((check) => check.key).filter(Boolean));

function phasesForOrder(
  order: FactoriesWorkOrder,
  executions: FactoriesWorkOrderExecution[],
  apiChecks?: FactoriesWorkOrderCheck[],
): SplitRunPhase[] {
  return [
    ...sourcePhasesForOrder(order, executions.length > 0),
    ...executions.map((execution) => executionToPhase(order, execution, apiChecks)),
  ];
}

function sourcePhasesForOrder(order: FactoriesWorkOrder, isDownstream: boolean): SplitRunPhase[] {
  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate) {
    return intakeTicketAnalysisFixture(analysisTicketForOrder(order), { complete: true }).phases;
  }
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

function executionToPhase(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
  apiChecks?: FactoriesWorkOrderCheck[],
): SplitRunPhase {
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
    checks: checksForLineExecution(execution, apiChecks),
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: execution.run?.appId,
    runId: execution.run?.id,
  };
}

function checksForLineExecution(
  execution: FactoriesWorkOrderExecution,
  apiChecks?: FactoriesWorkOrderCheck[],
): WorkOrderCheckPresentation[] | undefined {
  if (!(execution.step ?? "").toLowerCase().includes("verify")) {
    return undefined;
  }
  const source = apiChecks ?? VERIFY_STEP_CHECKS;
  return presentWorkOrderChecks(source.filter((check) => VERIFY_STEP_KEYS.has(check.key ?? "")));
}

function artifactsForLineExecution(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
): FactoriesWorkOrderArtifact[] {
  const step = (execution.step ?? "").toLowerCase();
  if (step.includes("implement")) {
    return [branchArtifactForOrder(order), pullRequestArtifactForOrder(order, "open")];
  }
  if (step.includes("verify")) {
    return [];
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
