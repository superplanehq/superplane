import type {
  FactoriesAutomationRef,
  FactoriesFactoryPullRequest,
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
import {
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  confidenceSuitabilityAnalysis,
  confidenceSuitabilitySummary,
} from "../../lib/confidenceScore";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import { isActiveCanvasRun, statusForCanvasRun } from "../../lib/workOrderPullRequest";
import type { BacklogAnalysisRun } from "../../lib/backlogAnalysis";
import type { PRFeedbackLogRun } from "../prFeedbackSettingsModel";
import {
  buildSplitRunFooter,
  doneFooterForStatus,
  SPLIT_RUN_DRAFT_NOTE,
  SPLIT_RUN_FAILED_NOTE_TEXT,
  SPLIT_RUN_WAITING_NOTE,
  type SplitRunFooter,
  type SplitRunFooterKind,
  type SplitRunFooterTone,
} from "./splitRunFooter";
import { intakeTicketAnalysisFixture, type LineIntakeAnalyzingTicket } from "../lineIntakeModel";
import { implementationPlanMarkdown, reviewCandidateForWorkOrderId } from "../onboarding/first-run/reviewCandidates";
import { DESCRIPTION_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import type {
  RunOverlayProvider,
  RunOverlayStep,
  RunOverlayStepStatus,
} from "../work-order-run-overlay/workOrderRunOverlayMocks";
import type { SplitRunCanvasKey, SplitRunCanvasModel } from "./splitRunCanvases";
import { splitRunSourceForOrder, type SplitRunSource } from "./splitRunSource";
import { withNotifyImplementLog } from "./splitRunNotifyFixture";

export type SplitRunPhaseId = string;

export type SplitRunPhaseStatus = "passed" | "running" | "pending" | "waiting" | "failed" | "cancelled";

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
  pullRequest?: FactoriesFactoryPullRequest;
  /** Agent transcript line. No checkmark. */
  note?: boolean;
  /** Nested tool call under a Claude Code step. */
  noteParentId?: string;
  noteDepth?: number;
  kind?: SplitRunStreamKind;
  /** Catalog identity: `Run Claude Code`, `github.addIssueLabel`. */
  componentType?: string;
  /** Compact session log: user talk vs a survey answer. */
  userTalk?: "message" | "survey";
  action?: string;
  iconSlug?: string;
  iconSrc?: string;
  /** Runner catalog id, e.g. runnerClaudeCode. */
  component?: string;
  /** Node execution id for live runner logs. */
  executionId?: string;
  /**
   * Comparable chronological sort key (epoch ms), when known. Lets the
   * planning session merge interleave a user reply with agent notes by true
   * time instead of guessing from wait-slot position. Absent when the
   * source has no timestamp (falls back to positional heuristics).
   */
  orderKey?: number;
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
  /** Line automation canvas. `null` means a person created the task. */
  canvasKey?: SplitRunCanvasKey | null;
  /** Trigger node that ran when the canvas has more than one start. */
  triggerName?: string;
  /** When set, the popup shows this canvas instead of the phase-name map. */
  canvas?: SplitRunCanvasModel;
  /** Line step index used to rerun this automation. */
  stepIndex?: number;
  /** Ledger cost for this phase, in USD cents. Hidden when zero. */
  costCents?: string;
  /** Ledger token count for this phase. Hidden when zero. */
  totalTokens?: string;
}

export type { SplitRunFooter, SplitRunFooterKind, SplitRunFooterTone };

export type SplitRunBoardColumn = "backlog" | "implement" | "verify" | "done";

export type SplitRunIntakeCanvasKey = "intake" | "sentry" | "slack";

export interface SplitRunFixture {
  title: string;
  /** Work-order description field. Artifact markdown is a fallback. */
  descriptionText?: string;
  owner: OrgUserDisplay;
  assigneeIds?: string[];
  elapsed: string;
  startedLabel: string;
  costUsd: string;
  tokensLabel: string;
  lineName: string;
  currentStepIndex: number;
  lineStatus: SplitRunPhaseStatus;
  currentPhaseId: SplitRunPhaseId;
  /** When set, the popup opens this phase even if it already passed. */
  openPhaseId?: SplitRunPhaseId | null;
  phases: SplitRunPhase[];
  waitingNotes: WorkOrderStatusNotePresentation[];
  checks: WorkOrderCheckPresentation[];
  footer: SplitRunFooter;
  footerTone: SplitRunFooterTone;
  source?: SplitRunSource;
}

export { SPLIT_RUN_RUNNING };

const UNKNOWN_OWNER: OrgUserDisplay = {
  id: "unknown",
  name: UNKNOWN_ORG_USER_NAME,
  initials: getUserInitials(UNKNOWN_ORG_USER_NAME) || "U",
};

function splitRunOwnerDisplay(order: FactoriesWorkOrder): OrgUserDisplay {
  const assignee = order.assignees?.[0];
  if (assignee?.id) {
    const name = assignee.name?.trim() || UNKNOWN_OWNER.name;
    return {
      id: assignee.id,
      name,
      initials: getUserInitials(name) || UNKNOWN_OWNER.initials,
    };
  }
  return workOrderOwnerDisplay(order, UNKNOWN_OWNER);
}

function failedFooterNote(current: FactoriesWorkOrderExecution | undefined): WorkOrderStatusNotePresentation {
  const step = current?.step?.trim();
  return {
    key: "step-failed",
    headline: step ? `${step} did not pass` : "The run did not pass",
    text: SPLIT_RUN_FAILED_NOTE_TEXT,
    cta: { label: "Debug", icon: "bug" },
  };
}

function footerRun(current: FactoriesWorkOrderExecution | undefined): { appId: string; runId: string } | undefined {
  const appId = current?.run?.appId;
  const runId = current?.run?.id;
  if (!appId || !runId) {
    return undefined;
  }
  return { appId, runId };
}

const WAITING_FALLBACK_NOTE: WorkOrderStatusNotePresentation = {
  key: "waiting-person",
  headline: SPLIT_RUN_WAITING_NOTE.headline,
  text: SPLIT_RUN_WAITING_NOTE.text ?? "",
};

const FIXES_PAUSED_HEADLINE = "Automatic fixes did not succeed";

const FIXES_PAUSED_FALLBACK_NOTE: WorkOrderStatusNotePresentation = {
  key: "check-fixes-paused",
  headline: FIXES_PAUSED_HEADLINE,
  text: "SuperPlane paused automatic fixes. Review the pull request and fix the remaining checks.",
};

/**
 * The footer note for a draft. A scored review candidate keeps the plan
 * and confidence. Other drafts tell the person to review the details and
 * start. The log holds the source line.
 */
function draftFooterNote(order: FactoriesWorkOrder): WorkOrderStatusNotePresentation {
  const candidate = reviewCandidateForWorkOrderId(order.id);
  if (candidate) {
    return {
      key: "draft-plan-ready",
      headline: "Review the plan, then start",
      text: [
        "Review **plan.md**. Change anything you need. Then click Start to send it to the line.",
        "",
        `From GitHub issue [${candidate.ticketKey}](${candidate.issue.url}).`,
        "",
        `Confidence ${candidate.confidenceScore}/${CONFIDENCE_SCORE_MAX} (${candidate.confidenceBand}):`,
        ...candidate.reasons.map((reason) => `- ${reason}`),
      ].join("\n"),
    };
  }
  return {
    key: "draft-start",
    headline: SPLIT_RUN_DRAFT_NOTE.headline,
    text: SPLIT_RUN_DRAFT_NOTE.text ?? "",
  };
}

function draftSourceSentence(order: FactoriesWorkOrder): string {
  const automation = order.createdBy?.automation;
  if (automation) {
    const { componentName } = lineAutomationPresentation({ id: automation.appId, name: automation.appName });
    return `${componentName} created this task.`;
  }
  const creator = order.createdBy?.user?.name?.trim();
  if (creator) {
    return `${creator} created this task.`;
  }
  return "A person created this task.";
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

const AUTO_EXPAND_STATUSES = new Set<SplitRunPhaseStatus>(["running", "waiting", "failed", "cancelled"]);

/** Open the current step only when it is running, waiting, or failed. */
export function autoExpandedPhaseId(fixture: SplitRunFixture): SplitRunPhaseId | null {
  if (fixture.openPhaseId) {
    return fixture.openPhaseId;
  }
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
  if (status === "cancelled") return "Canceled";
  return "Pending";
}

export type SplitRunFixtureOptions = {
  checks?: FactoriesWorkOrderCheck[];
  lineId?: string | null;
  /** Storybook keeps invented files and pull requests. Live orders do not. */
  demoArtifacts?: boolean;
  /** PR-feedback canvas runs for this task, shown as extra Log phases. */
  prFeedbackRuns?: PRFeedbackLogRun[];
  /** Person who stopped the current automation, when known. */
  stoppedBy?: OrgUserDisplay;
  /** Person or automation that closed the task, when known. */
  closer?: { actor?: OrgUserDisplay; automationName?: string };
  /** Backlog analysis runs for this task, shown as extra Log phases. */
  analysisRuns?: BacklogAnalysisRun[];
};

export function splitRunFixtureForWorkOrder(
  order?: FactoriesWorkOrder,
  options?: SplitRunFixtureOptions,
): SplitRunFixture {
  if (!order) {
    return SPLIT_RUN_RUNNING;
  }
  return mappedWorkOrderFixture(order, options);
}

function mappedWorkOrderFixture(order: FactoriesWorkOrder, options?: SplitRunFixtureOptions): SplitRunFixture {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order, options?.lineId);
  const current = pickCurrentExecution(executions);
  const demoArtifacts = options?.demoArtifacts !== false;
  const phases = phasesForOrder(order, executions, options, demoArtifacts);
  const activeAutomationId = activeAutomationPhaseId(phases);
  const fixture: SplitRunFixture = {
    title: order.title ?? "Task",
    descriptionText: order.description ?? "",
    owner: splitRunOwnerDisplay(order),
    assigneeIds: (order.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id)),
    elapsed: elapsedForDisplay(displayStatus, order),
    startedLabel: startedLabelForOrder(order),
    costUsd: costUsdForDisplay(order),
    tokensLabel: tokensLabelForDisplay(order),
    lineName: visibleDispatchForLine(order, options?.lineId)?.line?.name ?? SPLIT_RUN_RUNNING.lineName,
    currentStepIndex: current?.stepIndex ?? 0,
    lineStatus: lineStatusForDisplay(displayStatus),
    currentPhaseId: activeAutomationId ?? (current ? phaseIdForExecution(current, executions) : (phases[0]?.id ?? "")),
    openPhaseId: activeAutomationId,
    phases,
    source: splitRunSourceForOrder(order),
    ...reviewSurfaces(order, displayStatus, {
      lineId: options?.lineId,
      phases,
      apiChecks: options?.checks,
      demoArtifacts,
      hideWaitingDecision: shouldHideWaitingDecision(options?.prFeedbackRuns),
      fixesPaused: latestPRFeedbackRun(options?.prFeedbackRuns)?.kind === "fixes-paused",
      stoppedBy: options?.stoppedBy ?? options?.closer?.actor,
      closer: options?.closer,
    }),
  };
  if (order.id === "wo-board-implement-notify") {
    return withNotifyImplementLog(fixture, order);
  }
  return fixture;
}

function reviewSurfaces(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  input: {
    lineId?: string | null;
    phases: SplitRunPhase[];
    apiChecks?: FactoriesWorkOrderCheck[];
    demoArtifacts?: boolean;
    hideWaitingDecision?: boolean;
    fixesPaused?: boolean;
    stoppedBy?: OrgUserDisplay;
    closer?: { actor?: OrgUserDisplay; automationName?: string };
  },
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  const demoArtifacts = input.demoArtifacts !== false;
  const executions = latestDispatchExecutions(order, input.lineId);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const checks = overviewChecks(input.phases, input.apiChecks, demoArtifacts);

  if (displayStatus === "draft") {
    return surfaces(
      buildSplitRunFooter({ kind: "draft", note: draftFooterNote(order), status: displayStatus }),
      [],
      checks,
    );
  }
  if (displayStatus === "completed" || displayStatus === "rejected" || displayStatus === "cancelled") {
    return surfaces(doneFooterForStatus(displayStatus, input.closer), [], checks);
  }
  if (current?.result === "RESULT_FAILED") {
    return failedReviewSurface(current, displayStatus, checks);
  }
  if (current?.result === "RESULT_CANCELLED") {
    return stoppedReviewSurface(current, displayStatus, checks, input.stoppedBy);
  }
  if (displayStatus === "waiting" || (column === "implement" && current?.state === "STATE_PENDING")) {
    return waitingReviewSurface(order, displayStatus, checks, input.hideWaitingDecision, input.fixesPaused);
  }
  if (displayStatus === "running") {
    return surfaces(
      buildSplitRunFooter({
        kind: "running",
        note: runningFooterNote(current),
        run: footerRun(current),
        status: displayStatus,
      }),
      [],
      checks,
    );
  }
  return surfaces(doneFooterForStatus(displayStatus, input.closer), [], checks);
}

function stoppedReviewSurface(
  current: FactoriesWorkOrderExecution | undefined,
  displayStatus: WorkOrderDisplayStatus,
  checks: WorkOrderCheckPresentation[],
  stoppedBy?: OrgUserDisplay,
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  return surfaces(
    buildSplitRunFooter({
      kind: "stopped",
      actor: stoppedBy,
      run: footerRun(current),
      status: displayStatus,
    }),
    [],
    checks,
  );
}

function failedReviewSurface(
  current: FactoriesWorkOrderExecution | undefined,
  displayStatus: WorkOrderDisplayStatus,
  checks: WorkOrderCheckPresentation[],
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  const note = failedFooterNote(current);
  return surfaces(
    buildSplitRunFooter({
      kind: "failed",
      note,
      run: footerRun(current),
      status: displayStatus,
    }),
    [note],
    checks,
  );
}

function waitingReviewSurface(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  checks: WorkOrderCheckPresentation[],
  hideWaitingDecision?: boolean,
  fixesPaused?: boolean,
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  if (hideWaitingDecision) {
    return surfaces(buildSplitRunFooter({ kind: "waiting", status: displayStatus, decision: false }), [], checks);
  }
  const notes = presentWorkOrderStatusNotes(order.statusNotes, displayStatus);
  if (fixesPaused) {
    const note = pauseFooterNote(notes);
    return surfaces(
      buildSplitRunFooter({
        kind: "waiting",
        note,
        status: displayStatus,
      }),
      [note],
      checks,
    );
  }
  return surfaces(
    buildSplitRunFooter({
      kind: "waiting",
      note: notes[0] ?? WAITING_FALLBACK_NOTE,
      status: displayStatus,
    }),
    notes,
    checks,
  );
}

function pauseFooterNote(notes: WorkOrderStatusNotePresentation[]): WorkOrderStatusNotePresentation {
  const written = notes.find((note) => note.headline === FIXES_PAUSED_HEADLINE);
  if (written) {
    return written;
  }
  const review = notes[0];
  if (!review?.cta) {
    return FIXES_PAUSED_FALLBACK_NOTE;
  }
  return { ...FIXES_PAUSED_FALLBACK_NOTE, cta: review.cta };
}

function overviewChecks(
  phases: SplitRunPhase[],
  apiChecks?: FactoriesWorkOrderCheck[],
  demoArtifacts = true,
): WorkOrderCheckPresentation[] {
  const presented = presentWorkOrderChecks(apiChecks ?? []);
  if (!demoArtifacts) {
    return presented;
  }
  const intake = phases.find((phase) => phase.id === "score")?.checks ?? [];
  if (presented.length === 0) {
    return phases.flatMap((phase) => phase.checks ?? []);
  }
  const later = intake.length > 0 ? presented.filter((check) => check.name !== CONFIDENCE_CHECK_NAME) : presented;
  return [...intake, ...later];
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

/**
 * The log reads in the order the work happened: the source that created the
 * order, the Backlog analysis that scored it, the line steps that acted on it,
 * and last the PR feedback runs.
 */
function phasesForOrder(
  order: FactoriesWorkOrder,
  executions: FactoriesWorkOrderExecution[],
  options: SplitRunFixtureOptions | undefined,
  demoArtifacts: boolean,
): SplitRunPhase[] {
  const apiChecks = options?.checks;
  return [
    ...sourcePhasesForOrder(order, executions.length > 0, demoArtifacts),
    ...phasesForAnalysisRuns(options?.analysisRuns ?? [], apiChecks),
    ...executions.map((execution) => executionToPhase(order, execution, apiChecks, demoArtifacts, executions)),
    ...phasesForPRFeedbackRuns(options?.prFeedbackRuns ?? []),
  ];
}

const ANALYSIS_PHASE_ID_PREFIX = "backlog-analysis-";

/**
 * Backlog analysis runs of this task, oldest first. Each phase keeps
 * its run, so the log panel streams the analysis while the automation
 * still works. The newest phase carries the reported score.
 */
function phasesForAnalysisRuns(runs: BacklogAnalysisRun[], apiChecks?: FactoriesWorkOrderCheck[]): SplitRunPhase[] {
  const ordered = [...runs]
    .filter((entry) => Boolean(entry.canvasId && entry.run.id))
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""));

  return ordered.map((entry, index) =>
    analysisRunToPhase(entry, index === ordered.length - 1 ? confidenceChecks(apiChecks) : undefined),
  );
}

function analysisRunToPhase(entry: BacklogAnalysisRun, checks?: WorkOrderCheckPresentation[]): SplitRunPhase {
  const status = statusForCanvasRun(entry.run);
  const componentName = CONFIDENCE_CHECK_NAME;
  const duration = durationForExecution(
    {
      createdAt: entry.run.createdAt,
      updatedAt: entry.run.finishedAt ?? entry.run.updatedAt ?? entry.run.createdAt,
    },
    status,
  );
  const line: SplitRunStreamLine = {
    id: entry.run.id ?? componentName,
    at: clockLabel(entry.run.createdAt),
    componentName,
    status,
    duration,
    kind: "action",
    componentType: componentName,
    action: status === "passed" ? "passed" : status === "failed" ? "failed" : status === "running" ? "running" : "—",
    iconSlug: "box",
  };
  return {
    id: `${ANALYSIS_PHASE_ID_PREFIX}${entry.run.id}`,
    name: "Analysis",
    status,
    duration,
    componentName,
    artifacts: [],
    checks,
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: entry.canvasId,
    runId: entry.run.id,
  };
}

function confidenceChecks(apiChecks?: FactoriesWorkOrderCheck[]): WorkOrderCheckPresentation[] | undefined {
  const reported = (apiChecks ?? []).filter((check) => (check.name ?? "") === CONFIDENCE_CHECK_NAME);
  if (reported.length === 0) {
    return undefined;
  }
  return presentWorkOrderChecks(reported);
}

function phasesForPRFeedbackRuns(runs: PRFeedbackLogRun[]): SplitRunPhase[] {
  return [...runs]
    .filter((entry) => Boolean(entry.canvasId && entry.run.id))
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""))
    .map(prFeedbackRunToPhase);
}

function activePRFeedbackPhaseId(phases: SplitRunPhase[]): SplitRunPhaseId | undefined {
  return activePhaseIdWithPrefix(phases, "pr-feedback-");
}

function hasActivePRFeedbackRun(runs?: PRFeedbackLogRun[]): boolean {
  return (runs ?? []).some((entry) => isActiveCanvasRun(entry.run));
}

function shouldHideWaitingDecision(runs?: PRFeedbackLogRun[]): boolean {
  return hasActivePRFeedbackRun(runs);
}

function latestPRFeedbackRun(runs?: PRFeedbackLogRun[]): PRFeedbackLogRun | undefined {
  if (!runs || runs.length === 0) {
    return undefined;
  }
  return [...runs].sort(
    (left, right) => Date.parse(right.run.createdAt ?? "") - Date.parse(left.run.createdAt ?? ""),
  )[0];
}

/**
 * Factory-level automation that works on this order right now: a PR-feedback
 * run or a Backlog analysis run. The popup opens its log so the progress is
 * visible without another click.
 */
function activeAutomationPhaseId(phases: SplitRunPhase[]): SplitRunPhaseId | undefined {
  return activePRFeedbackPhaseId(phases) ?? activePhaseIdWithPrefix(phases, ANALYSIS_PHASE_ID_PREFIX);
}

function activePhaseIdWithPrefix(phases: SplitRunPhase[], prefix: string): SplitRunPhaseId | undefined {
  return phases.find(
    (phase) => phase.id.startsWith(prefix) && (phase.status === "running" || phase.status === "pending"),
  )?.id;
}

function prFeedbackRunToPhase(entry: PRFeedbackLogRun): SplitRunPhase {
  const status = statusForCanvasRun(entry.run);
  const description = entry.description?.trim();
  const baseName = description
    ? description
    : entry.pullRequestNumber
      ? `Activity on PR #${String(entry.pullRequestNumber).replace(/^#/, "")}`
      : "Activity on PR";
  const name = entry.attemptLabel ? `${baseName} · ${entry.attemptLabel}` : baseName;
  const componentName = entry.handlerName?.trim() || "Address PR feedback";
  const duration = durationForExecution(
    {
      createdAt: entry.run.createdAt,
      updatedAt: entry.run.finishedAt ?? entry.run.updatedAt ?? entry.run.createdAt,
    },
    status,
  );
  const line: SplitRunStreamLine = {
    id: entry.run.id ?? name,
    at: clockLabel(entry.run.createdAt),
    componentName,
    status,
    duration,
    kind: "action",
    componentType: componentName,
    action: status === "passed" ? "passed" : status === "failed" ? "failed" : status === "running" ? "running" : "—",
    iconSlug: "box",
  };
  return {
    id: `pr-feedback-${entry.run.id}`,
    name,
    status,
    duration,
    componentName,
    artifacts: [],
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: entry.canvasId,
    runId: entry.run.id,
    costCents: entry.costCents,
    totalTokens: entry.totalTokens,
  };
}

function sourcePhasesForOrder(
  order: FactoriesWorkOrder,
  isDownstream: boolean,
  demoArtifacts: boolean,
): SplitRunPhase[] {
  if (!demoArtifacts) {
    return [backlogSourcePhase(order)];
  }
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
      confidenceAnalysis: confidenceSuitabilityAnalysis({
        source: "GitHub",
        reasons: candidate.reasons,
      }),
    };
  }
  const confidenceScore = exampleConfidenceScore(order);
  return {
    id: order.id ?? "work-order",
    title: order.title ?? "Task",
    detailsMarkdown: order.description,
    issueKey: order.key,
    planMarkdown: fallbackPlanMarkdown(order),
    confidenceScore,
    confidenceSummary: confidenceSuitabilitySummary(confidenceBandForScore(confidenceScore)),
    confidenceAnalysis: confidenceSuitabilityAnalysis({ source: issueSourceForOrder(order) }),
  };
}

function issueSourceForOrder(order: FactoriesWorkOrder): string | undefined {
  const automation = order.createdBy?.automation;
  if (!automation) {
    return undefined;
  }
  const key = canvasKeyForAutomation({ id: automation.appId, name: automation.appName });
  if (key === "intake") {
    return "GitHub";
  }
  if (key === "sentry") {
    return "Sentry";
  }
  if (key === "slack") {
    return "Slack";
  }
  const name = automation.appName?.trim();
  if (name && /pagerduty/i.test(name)) {
    return "PagerDuty";
  }
  return name || undefined;
}

function fallbackPlanMarkdown(order: FactoriesWorkOrder): string {
  const goal = order.title ?? "Implement the change.";
  return implementationPlanMarkdown({
    goal,
    files: ["See the work-order description for the files to change."],
    steps: [
      "Read the task and the current implementation.",
      "Apply the change described in the task.",
      "Add or update tests for the new behavior.",
    ],
    verify: ["The existing suite passes.", "The notes in the task hold."],
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
  const at = clockLabel(order.createdAt);
  const line = draftSourceSentence(order);
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
  demoArtifacts = true,
  peers: FactoriesWorkOrderExecution[] = [],
): SplitRunPhase {
  const status = statusForExecution(execution);
  const { name, componentName } = lineAutomationPresentation(execution.run, execution.step);
  const duration = durationForExecution(execution, status);
  const artifacts = demoArtifacts ? artifactsForLineExecution(order, execution) : [];
  const pullRequest = demoArtifacts ? pullRequestForLineExecution(order, execution) : undefined;
  const line: SplitRunStreamLine = {
    id: execution.id ?? name,
    at: clockLabel(execution.updatedAt ?? execution.createdAt),
    componentName,
    status,
    duration,
    detail: execution.step,
    artifact: artifacts[0],
    pullRequest,
    kind: "action",
    componentType: componentName,
    action: status === "passed" ? "passed" : status === "failed" ? "failed" : status === "running" ? "running" : "—",
    iconSlug: "box",
  };
  return {
    id: phaseIdForExecution(execution, peers),
    name,
    status,
    duration,
    componentName,
    artifacts,
    checks: checksForLineExecution(execution, apiChecks, demoArtifacts),
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: execution.run?.appId,
    runId: execution.run?.id,
    stepIndex: execution.stepIndex,
    costCents: execution.costCents,
    totalTokens: execution.totalTokens,
  };
}

function checksForLineExecution(
  execution: FactoriesWorkOrderExecution,
  apiChecks?: FactoriesWorkOrderCheck[],
  demoArtifacts = true,
): WorkOrderCheckPresentation[] | undefined {
  if (!(execution.step ?? "").toLowerCase().includes("verify")) {
    return undefined;
  }
  const source = demoArtifacts ? (apiChecks ?? VERIFY_STEP_CHECKS) : (apiChecks ?? []);
  const verifyChecks = demoArtifacts
    ? source.filter((check) => VERIFY_STEP_KEYS.has(check.key ?? ""))
    : source.filter((check) => (check.name ?? "") !== CONFIDENCE_CHECK_NAME);
  return presentWorkOrderChecks(verifyChecks);
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
    return [];
  }
  if (step.includes("done")) {
    if (execution.result === "RESULT_CANCELLED") {
      return [canceledNotesArtifact(order)];
    }
    return [];
  }
  return [];
}

function pullRequestForLineExecution(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
): FactoriesFactoryPullRequest | undefined {
  const step = (execution.step ?? "").toLowerCase();
  if (step.includes("implement")) {
    return pullRequestForOrder(order, "STATE_OPEN");
  }
  if (!step.includes("done") || execution.result === "RESULT_CANCELLED") {
    return undefined;
  }
  if (order.result === "RESULT_REJECTED") {
    return pullRequestForOrder(order, "STATE_CLOSED");
  }
  return pullRequestForOrder(order, "STATE_MERGED");
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

function pullRequestForOrder(
  order: FactoriesWorkOrder,
  state: FactoriesFactoryPullRequest["state"],
): FactoriesFactoryPullRequest {
  const number = pullRequestNumberForOrder(order);
  return {
    id: `pr-${order.id ?? "order"}`,
    workOrderId: order.id,
    number: String(number),
    url: `https://github.com/example/ledger/pull/${number}`,
    title: order.title ?? "Pull request",
    state,
  };
}

function canceledNotesArtifact(order: FactoriesWorkOrder): FactoriesWorkOrderArtifact {
  return {
    id: `art-notes-${order.id ?? "order"}`,
    type: "TYPE_MARKDOWN",
    data: {
      name: "notes.md",
      title: "notes.md",
      body: "The task was canceled.",
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
  if (status === "waiting" || status === "cancelled") return "pending";
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
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }
  if (execution.result === "RESULT_PASSED") {
    return "passed";
  }
  return "waiting";
}

function phaseIdForExecution(
  execution: FactoriesWorkOrderExecution,
  peers: FactoriesWorkOrderExecution[] = [],
): string {
  const name = (execution.step ?? "step").toLowerCase().replace(/[^a-z0-9]+/g, "-");
  const stepIndex = execution.stepIndex ?? 0;
  const base = `${name}-${stepIndex}`;
  const sameStep = peers.filter((peer) => (peer.stepIndex ?? 0) === stepIndex);
  if (sameStep.length <= 1) {
    return base;
  }
  return `${base}-${execution.id ?? String(sameStep.indexOf(execution))}`;
}

function visibleDispatchForLine(order: FactoriesWorkOrder, lineId?: string | null) {
  const onLine = (order.lineDispatches ?? []).filter((dispatch) => !lineId || dispatch.line?.id === lineId);
  const active = [...onLine].reverse().find((dispatch) => dispatch.state === "STATE_ACTIVE");
  return active ?? latestDispatchForLine(order, lineId);
}

function latestDispatchExecutions(order: FactoriesWorkOrder, lineId?: string | null): FactoriesWorkOrderExecution[] {
  const latest = visibleDispatchForLine(order, lineId);
  const current = latest?.stepExecutions ?? [];
  if (!latest || current.length === 0) {
    return current;
  }

  const present = new Set(current.map((execution) => execution.stepIndex ?? 0));
  const missing = missingEarlierStepIndexes(present);
  if (missing.length === 0) {
    return current;
  }

  const older = (order.lineDispatches ?? [])
    .filter((dispatch) => dispatch.id !== latest.id && (!lineId || dispatch.line?.id === lineId))
    .sort((left, right) => (Date.parse(right.createdAt ?? "") || 0) - (Date.parse(left.createdAt ?? "") || 0));

  const prior: FactoriesWorkOrderExecution[] = [];
  for (const stepIndex of missing) {
    const match = older
      .flatMap((dispatch) => dispatch.stepExecutions ?? [])
      .find((execution) => (execution.stepIndex ?? 0) === stepIndex);
    if (match) {
      prior.push(match);
    }
  }

  return [...prior, ...current].sort((left, right) => (left.stepIndex ?? 0) - (right.stepIndex ?? 0));
}

function missingEarlierStepIndexes(present: Set<number>): number[] {
  const currentMin = Math.min(...present);
  if (!Number.isFinite(currentMin) || currentMin <= 0) {
    return [];
  }

  const missing: number[] = [];
  for (let index = 0; index < currentMin; index++) {
    if (!present.has(index)) {
      missing.push(index);
    }
  }
  return missing;
}

function pickCurrentExecution(executions: FactoriesWorkOrderExecution[]): FactoriesWorkOrderExecution | undefined {
  const inFlight = executions.filter(
    (execution) =>
      execution.state === "STATE_STARTED" ||
      execution.state === "STATE_PENDING" ||
      execution.state === "STATE_CANCELLING",
  );
  const pool = inFlight.length > 0 ? inFlight : executions;
  return pool.reduce<FactoriesWorkOrderExecution | undefined>((best, candidate) => {
    if (!best) {
      return candidate;
    }
    const bestAt = Date.parse(best.updatedAt ?? best.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.updatedAt ?? candidate.createdAt ?? "") || 0;
    if (candidateAt !== bestAt) {
      return candidateAt > bestAt ? candidate : best;
    }
    return (candidate.stepIndex ?? -1) >= (best.stepIndex ?? -1) ? candidate : best;
  }, undefined);
}
