import type {
  FactoriesAutomationRef,
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestRevision,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
  FactoriesWorkOrderExecution,
} from "@/api-client";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import {
  UNKNOWN_ORG_USER_NAME,
  getUserInitials,
  type OrgUserDisplay,
  type OrgUserDisplayLookup,
} from "@/lib/orgUserDisplay";
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
  clarityScoreFromChecks,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  confidenceScoreFromChecks,
  confidenceSuitabilityAnalysis,
  confidenceSuitabilitySummary,
  isScoreCheckName,
} from "../../lib/confidenceScore";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import {
  getWorkOrderDisplayStatus,
  getWorkOrderDisplayStatusMeta,
  type WorkOrderDisplayStatus,
} from "../../lib/workOrderProgress";
import { workOrderExecutionCreditFailure } from "../../lib/workOrderFailureReason";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import {
  parseWorkOrderMetric,
  type WorkOrderUsageByMachineType,
  type WorkOrderUsageByModel,
} from "../../lib/workOrderUsage";
import { joinRunnerModels } from "./draftStartModel";
import { isActiveCanvasRun, statusForCanvasRun } from "../../lib/workOrderPullRequest";
import { analysisResultDeliveredForRun, statusForAnalysisRun } from "../../lib/analysisOutcome";
import {
  backlogAnalysisCreditFailure,
  hasActiveBacklogAnalysisRun,
  type BacklogAnalysisRun,
} from "../../lib/backlogAnalysis";
import type { PRFeedbackLogRun } from "../prFeedbackSettingsModel";
import {
  buildSplitRunFooter,
  doneFooterForStatus,
  SPLIT_RUN_DRAFT_NOTE,
  SPLIT_RUN_FAILED_NOTE_TEXT,
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
import { trackedPullRequestReviewNote } from "./splitRunPullRequestReview";

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
  /** Whether the displayed duration should keep ticking. Defaults to a running status. */
  durationRunning?: boolean;
  detail?: string;
  commandScript?: string;
  commandStdout?: string;
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
  /** Markdown details shown below the activity title. */
  description?: string;
  status: SplitRunPhaseStatus;
  duration: string;
  /** Whether the displayed duration should keep ticking. Defaults to a running status. */
  durationRunning?: boolean;
  /** When this automation started. */
  startedAt?: string;
  /** When this automation finished. Used for board-column dwell. */
  endedAt?: string;
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
  /** Run from an earlier dispatch. Console run history; hidden in the classic tabs. */
  historyRun?: boolean;
  /** Ledger cost for this phase, in USD cents. Hidden when zero. */
  costCents?: string;
  /** Ledger token count for this phase. Hidden when zero. */
  totalTokens?: string;
  /** Runner model this automation used. Hidden when empty. */
  model?: string;
  /** Start thinking level from the dispatch that owns this run. Hidden when empty or Auto. */
  thinkingLevel?: string;
  /** Board column this app sits on, when it is a column automation. */
  columnKey?: SplitRunBoardColumn;
  /** Pull request and revision that started this activity. */
  pullRequestActivity?: {
    pullRequest?: FactoriesFactoryPullRequest;
    revision?: SplitRunRevision;
    startedAt?: string;
    waitingForAccess?: boolean;
  };
}

/**
 * Pushed commit. The API sends only `sha` and `createdAt`; `message` is the
 * commit subject and is not yet persisted by the backend.
 */
export interface SplitRunRevision extends FactoriesFactoryPullRequestRevision {
  message?: string;
  /** Automation that pushed this commit. Not yet persisted by the backend. */
  pushedBy?: string;
}

export type { SplitRunFooter, SplitRunFooterKind, SplitRunFooterTone };

export type SplitRunBoardColumn = "backlog" | "implement" | "verify" | "done";

/** Factory app attached to a board column. */
export type SplitRunColumnApp = {
  id: string;
  name?: string;
  columnKey?: string;
};

export function columnAppsFromFactoryApps(
  apps: Array<{ id?: string; name?: string; columnKey?: string }>,
): SplitRunColumnApp[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    return id ? [{ id, name: app.name, columnKey: app.columnKey }] : [];
  });
}

export type SplitRunIntakeCanvasKey = "intake" | "sentry" | "slack";

export interface SplitRunFixture {
  title: string;
  /** Stored task key, for example `RF-101`. Empty when the order has no key. */
  identifier?: string;
  /** Work-order description field. Artifact markdown is a fallback. */
  descriptionText?: string;
  owner: OrgUserDisplay;
  assigneeIds?: string[];
  elapsed: string;
  startedLabel: string;
  costUsd: string;
  tokensLabel: string;
  savedTokens?: number;
  savedCostCents?: number;
  usageByModel?: WorkOrderUsageByModel[];
  usageByMachineType?: WorkOrderUsageByMachineType[];
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

function splitRunOwnerDisplay(order: FactoriesWorkOrder, resolveUser?: OrgUserDisplayLookup): OrgUserDisplay {
  const assignee = order.assignees?.[0];
  if (assignee?.id) {
    // `resolveUser` looks the owner up against the org members list, which
    // carries the avatar image. Without it we can only show initials.
    const resolved = resolveUser?.(assignee.id, assignee.name);
    if (resolved) {
      return resolved;
    }
    const name = assignee.name?.trim() || UNKNOWN_OWNER.name;
    return {
      id: assignee.id,
      name,
      initials: getUserInitials(name) || UNKNOWN_OWNER.initials,
    };
  }
  return workOrderOwnerDisplay(order, UNKNOWN_OWNER, resolveUser);
}

function failedFooterNote(current: FactoriesWorkOrderExecution | undefined): WorkOrderStatusNotePresentation {
  const step = current?.step?.trim();
  const credit = workOrderExecutionCreditFailure(current);
  return {
    key: "step-failed",
    headline: step ? `${step} did not pass` : "The run did not pass",
    text: credit?.message ?? SPLIT_RUN_FAILED_NOTE_TEXT,
    cta: credit ? { label: credit.actionLabel, destination: "billing" } : { label: "Debug", icon: "bug" },
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
function draftCreditFooterNote(
  credit: NonNullable<ReturnType<typeof backlogAnalysisCreditFailure>>,
): WorkOrderStatusNotePresentation {
  return {
    key: "draft-credit",
    headline: credit.label,
    text: credit.message,
    cta: { label: credit.actionLabel, destination: "billing" },
  };
}

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
  if (status === "waiting") return "Waiting";
  if (status === "failed") return "Failed";
  if (status === "cancelled") return "Canceled";
  return "Pending";
}

export type SplitRunFixtureOptions = {
  checks?: FactoriesWorkOrderCheck[];
  lineId?: string | null;
  lineName?: string;
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
  /**
   * Factory apps on board columns. A related run becomes a console card
   * in that column, under the app name.
   */
  columnApps?: SplitRunColumnApp[];
  /** Task files used to decide if a cancelled analysis already delivered a plan. */
  artifacts?: FactoriesWorkOrderArtifact[];
  /**
   * Whether the Backlog automation is still scoring this draft. Covers the
   * optimistic window where a fresh draft is known to be analyzing before its
   * run appears in `analysisRuns`, so the popup matches the board card.
   */
  isAnalyzing?: boolean;
  /** Looks up an org member's display (name, initials, avatar) by id. */
  resolveUser?: OrgUserDisplayLookup;
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
  const phases = [
    ...phasesForOrder(order, executions, options, demoArtifacts),
    ...closurePhaseForOrder(order, displayStatus, options?.closer),
  ];
  const activeAutomationId = activeAutomationPhaseId(phases);
  const fixture: SplitRunFixture = {
    title: order.title ?? "Task",
    identifier: order.key?.trim() ?? "",
    descriptionText: order.description ?? "",
    owner: splitRunOwnerDisplay(order, options?.resolveUser),
    assigneeIds: (order.assignees ?? []).map((assignee) => assignee.id).filter((id): id is string => Boolean(id)),
    elapsed: elapsedForDisplay(displayStatus, order),
    startedLabel: startedLabelForOrder(order),
    costUsd: costUsdForDisplay(order),
    tokensLabel: tokensLabelForDisplay(order),
    savedTokens: parseWorkOrderMetric(order.totalTokens),
    savedCostCents: parseWorkOrderMetric(order.totalCostCents),
    usageByModel: order.usageByModel,
    usageByMachineType: order.usageByMachineType,
    lineName:
      options?.lineName?.trim() ||
      visibleDispatchForLine(order, options?.lineId)?.line?.name ||
      SPLIT_RUN_RUNNING.lineName,
    currentStepIndex: current?.stepIndex ?? 0,
    lineStatus: lineStatusForDisplay(displayStatus),
    currentPhaseId: activeAutomationId ?? (current ? phaseIdForExecution(current, executions) : (phases[0]?.id ?? "")),
    openPhaseId: activeAutomationId,
    phases,
    source: splitRunSourceForOrder(order, options?.resolveUser),
    ...reviewSurfaces(order, displayStatus, {
      lineId: options?.lineId,
      phases,
      apiChecks: options?.checks,
      demoArtifacts,
      hideWaitingDecision: shouldHideWaitingDecision(options?.prFeedbackRuns),
      fixesPaused: latestPRFeedbackRun(options?.prFeedbackRuns)?.kind === "fixes-paused",
      stoppedBy: options?.stoppedBy ?? options?.closer?.actor,
      closer: options?.closer,
      analysisRuns: options?.analysisRuns,
      isAnalyzing: options?.isAnalyzing,
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
    analysisRuns?: BacklogAnalysisRun[];
    isAnalyzing?: boolean;
  },
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  const demoArtifacts = input.demoArtifacts !== false;
  const executions = latestDispatchExecutions(order, input.lineId);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const checks = overviewChecks(input.phases, input.apiChecks, demoArtifacts);

  if (displayStatus === "draft") {
    return draftReviewSurface(order, checks, input);
  }
  if (displayStatus === "completed" || displayStatus === "rejected") {
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

/**
 * Whether a draft is still being scored. The optimistic `isAnalyzing` flag
 * covers the window before a fresh draft's run appears in `analysisRuns`, so
 * the popup matches the board card even during run discovery.
 */
function draftIsAnalyzing(input: { isAnalyzing?: boolean; analysisRuns?: BacklogAnalysisRun[] }): boolean {
  return Boolean(input.isAnalyzing) || hasActiveBacklogAnalysisRun(input.analysisRuns ?? []);
}

function draftReviewSurface(
  order: FactoriesWorkOrder,
  checks: WorkOrderCheckPresentation[],
  input: { isAnalyzing?: boolean; analysisRuns?: BacklogAnalysisRun[] },
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footer" | "footerTone"> {
  const credit = backlogAnalysisCreditFailure(input.analysisRuns ?? []);
  return surfaces(
    buildSplitRunFooter({
      kind: "draft",
      note: credit ? draftCreditFooterNote(credit) : draftFooterNote(order),
      status: "draft",
      isAnalyzing: credit ? false : draftIsAnalyzing(input),
      clarityScore: clarityScoreFromChecks(checks),
      confidenceScore: confidenceScoreFromChecks(checks),
    }),
    [],
    checks,
  );
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
  const note = notes[0] ?? trackedPullRequestReviewNote(order.pullRequests, order.id);
  return surfaces(
    buildSplitRunFooter({
      kind: "waiting",
      note,
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
  const later = intake.length > 0 ? presented.filter((check) => !isScoreCheckName(check.name)) : presented;
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
  const prior = priorLineExecutions(order, options?.lineId, executions);
  const peers = [...prior, ...executions];
  const columnApps = options?.columnApps ?? [];
  const knownPhases = [
    ...sourcePhasesForOrder(order, executions.length > 0, demoArtifacts),
    ...phasesForAnalysisRuns(options?.analysisRuns ?? [], apiChecks, options?.artifacts),
    ...prior.map((execution) => ({
      ...executionToPhase(order, execution, apiChecks, demoArtifacts, peers),
      historyRun: true,
    })),
    ...executions.map((execution) => executionToPhase(order, execution, apiChecks, demoArtifacts, executions)),
    ...phasesForPRFeedbackRuns(options?.prFeedbackRuns ?? [], columnApps),
  ];
  return [...knownPhases, ...phasesForColumnAppChecks(columnApps, apiChecks, options?.artifacts, knownPhases)];
}

export const SPLIT_RUN_CLOSURE_PHASE_ID = "done-closure";

const CLOSED_DISPLAY_STATUSES = new Set<WorkOrderDisplayStatus>(["completed", "rejected", "cancelled", "failed"]);

/**
 * The decision that closed the task, as a Done stage. Mirrors the
 * creation stage in Backlog. The console shows it in Done when no
 * closer automation run sits there.
 */
function closurePhaseForOrder(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  closer?: { actor?: OrgUserDisplay; automationName?: string },
): SplitRunPhase[] {
  if (!CLOSED_DISPLAY_STATUSES.has(displayStatus)) {
    return [];
  }
  const note = doneFooterForStatus(displayStatus, closer).note;
  const sentence = note ? `${note.actor?.name ? `${note.actor.name} ` : ""}${note.headline}.` : undefined;
  return [
    {
      id: SPLIT_RUN_CLOSURE_PHASE_ID,
      name: "Done",
      description: sentence,
      status: closureStatus(displayStatus),
      duration: "",
      startedAt: order.updatedAt,
      componentName: getWorkOrderDisplayStatusMeta(displayStatus).label,
      artifacts: [],
      stream: [],
      canvasSteps: [],
      canvasKey: null,
    },
  ];
}

function closureStatus(displayStatus: WorkOrderDisplayStatus): SplitRunPhaseStatus {
  if (displayStatus === "completed") {
    return "passed";
  }
  if (displayStatus === "cancelled") {
    return "cancelled";
  }
  return "failed";
}

/**
 * Executions from earlier dispatches of this line. A stop-and-rerun makes a
 * new dispatch, so these are the earlier runs of the same steps. The console
 * groups them with the current run as run history.
 */
function priorLineExecutions(
  order: FactoriesWorkOrder,
  lineId: string | null | undefined,
  current: FactoriesWorkOrderExecution[],
): FactoriesWorkOrderExecution[] {
  const visible = visibleDispatchForLine(order, lineId);
  if (!visible) {
    return [];
  }
  const shown = new Set(current.map((execution) => execution.id));
  return (order.lineDispatches ?? [])
    .filter((dispatch) => dispatch.id !== visible.id && (!lineId || dispatch.line?.id === lineId))
    .flatMap((dispatch) => dispatch.stepExecutions ?? [])
    .filter((execution) => !shown.has(execution.id))
    .sort((left, right) => (Date.parse(left.createdAt ?? "") || 0) - (Date.parse(right.createdAt ?? "") || 0));
}

const ANALYSIS_PHASE_ID_PREFIX = "backlog-analysis-";

/** One task analysis can span multiple canvas runs. Present those attempts as one phase. */
function phasesForAnalysisRuns(
  runs: BacklogAnalysisRun[],
  apiChecks?: FactoriesWorkOrderCheck[],
  artifacts?: FactoriesWorkOrderArtifact[],
): SplitRunPhase[] {
  const ordered = [...runs]
    .filter((entry) => Boolean(entry.canvasId && entry.workOrderId && entry.run.id))
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""));
  const latest = ordered.at(-1);
  if (!latest) {
    return [];
  }

  const result = { checks: apiChecks, artifacts };
  return [
    analysisAttemptsToPhase(
      ordered,
      confidenceChecks(apiChecks),
      analysisResultDeliveredForRun(latest.run, { ...result, isLast: true }),
    ),
  ];
}

function analysisAttemptsToPhase(
  attempts: BacklogAnalysisRun[],
  checks?: WorkOrderCheckPresentation[],
  delivered = false,
): SplitRunPhase {
  const first = attempts[0];
  const latest = attempts.at(-1);
  if (!first || !latest) {
    throw new Error("analysis phase requires at least one attempt");
  }

  const status = statusForAnalysisRun(latest.run, statusForCanvasRun(latest.run), delivered);
  const credit = backlogAnalysisCreditFailure([latest]);
  const durationRunning = latest.run.state === "STATE_STARTED";
  // The card title is the Backlog automation. The score stays on the check.
  const componentName = "Backlog";
  const latestDuration = analysisAttemptsDuration([latest], durationRunning);
  const line: SplitRunStreamLine = {
    id: latest.run.id ?? componentName,
    at: clockLabel(latest.run.createdAt),
    componentName,
    status,
    duration: latestDuration,
    durationRunning,
    kind: "action",
    componentType: componentName,
    action: status === "passed" ? "passed" : status === "failed" ? "failed" : status === "running" ? "running" : "—",
    ...(credit ? { detail: credit.message } : {}),
    iconSlug: "box",
  };
  return {
    id: `${ANALYSIS_PHASE_ID_PREFIX}${latest.workOrderId}`,
    name: "Analysis",
    status,
    duration: analysisAttemptsDuration(attempts, durationRunning),
    durationRunning,
    startedAt: first.run.createdAt,
    componentName,
    artifacts: [],
    checks,
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: latest.canvasId,
    runId: latest.run.id,
    costCents: sumAnalysisMetric(attempts, (attempt) => attempt.run.costCents),
    totalTokens: sumAnalysisMetric(attempts, (attempt) => attempt.run.totalTokens),
    model: joinRunnerModels(attempts.flatMap((attempt) => attempt.run.models ?? [])),
  };
}

function analysisAttemptsDuration(attempts: BacklogAnalysisRun[], durationRunning: boolean, now = Date.now()): string {
  const lastIndex = attempts.length - 1;
  const durationMs = attempts.reduce((total, attempt, index) => {
    const startedAt = Date.parse(attempt.run.createdAt ?? "");
    if (!Number.isFinite(startedAt)) {
      return total;
    }
    const recordedEnd = Date.parse(attempt.run.finishedAt ?? attempt.run.updatedAt ?? "");
    const isLatestRunningAttempt = index === lastIndex && durationRunning;
    const endedAt = isLatestRunningAttempt ? now : Number.isFinite(recordedEnd) ? recordedEnd : startedAt;
    return total + Math.max(0, endedAt - startedAt);
  }, 0);
  return formatMinutesSecondsDuration(durationMs) || "<1s";
}

function sumAnalysisMetric(
  attempts: BacklogAnalysisRun[],
  select: (attempt: BacklogAnalysisRun) => string | undefined,
): string | undefined {
  const values = attempts.map(select).filter((value): value is string => value != null && value !== "");
  if (values.length === 0) {
    return undefined;
  }
  return String(values.reduce((total, value) => total + parseWorkOrderMetric(value), 0));
}

function confidenceChecks(apiChecks?: FactoriesWorkOrderCheck[]): WorkOrderCheckPresentation[] | undefined {
  const reported = (apiChecks ?? []).filter((check) => isScoreCheckName(check.name));
  if (reported.length === 0) {
    return undefined;
  }
  return presentWorkOrderChecks(reported);
}

function phasesForPRFeedbackRuns(runs: PRFeedbackLogRun[], columnApps: SplitRunColumnApp[] = []): SplitRunPhase[] {
  return [...runs]
    .filter((entry) => Boolean(entry.canvasId && entry.run.id))
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""))
    .map((entry) => prFeedbackRunToPhase(entry, columnApps));
}

function phasesForColumnAppChecks(
  columnApps: SplitRunColumnApp[],
  checks: FactoriesWorkOrderCheck[] | undefined,
  artifacts: FactoriesWorkOrderArtifact[] | undefined,
  knownPhases: SplitRunPhase[],
): SplitRunPhase[] {
  const checksByRun = columnAppChecksByRun(columnApps, checks);
  const extras: SplitRunPhase[] = [];
  for (const [runId, runChecks] of checksByRun) {
    const presented = presentWorkOrderChecks(runChecks);
    const existing = knownPhases.find((phase) => phase.runId === runId);
    if (existing) {
      existing.checks = uniquePresentedChecks([...(existing.checks ?? []), ...presented]);
      continue;
    }
    const phase = phaseForColumnAppCheck(columnApps, runChecks, artifacts);
    if (phase) {
      extras.push(phase);
    }
  }
  return extras;
}

function columnAppChecksByRun(
  columnApps: SplitRunColumnApp[],
  checks: FactoriesWorkOrderCheck[] | undefined,
): Map<string, FactoriesWorkOrderCheck[]> {
  const checksByRun = new Map<string, FactoriesWorkOrderCheck[]>();
  for (const check of checks ?? []) {
    const ref = columnAppCheckRef(check);
    if (!ref) {
      continue;
    }
    const app = columnApps.find((entry) => entry.id === ref.appId);
    if (!app || !consoleColumnForAppKey(app.columnKey)) {
      continue;
    }
    const runChecks = checksByRun.get(ref.runId) ?? [];
    runChecks.push(check);
    checksByRun.set(ref.runId, runChecks);
  }
  return checksByRun;
}

function uniquePresentedChecks(checks: WorkOrderCheckPresentation[]): WorkOrderCheckPresentation[] {
  const seen = new Set<string>();
  return checks.filter((check) => {
    const key = check.id || check.name;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}

function phaseForColumnAppCheck(
  columnApps: SplitRunColumnApp[],
  checks: FactoriesWorkOrderCheck[],
  artifacts: FactoriesWorkOrderArtifact[] | undefined,
): SplitRunPhase | undefined {
  const first = checks[0];
  const ref = first ? columnAppCheckRef(first) : undefined;
  if (!first || !ref) {
    return undefined;
  }
  const app = columnApps.find((entry) => entry.id === ref.appId);
  const columnKey = consoleColumnForAppKey(app?.columnKey);
  if (!app || !columnKey) {
    return undefined;
  }
  const name = phaseNameForColumn(columnKey);
  const componentName = columnAppCheckName(app, first, name);
  const latest = checks[checks.length - 1] ?? first;
  const status: SplitRunPhaseStatus = "passed";
  const line: SplitRunStreamLine = {
    id: ref.runId,
    at: clockLabel(latest.updatedAt),
    componentName,
    status,
    duration: "",
    kind: "action",
    componentType: componentName,
    action: "passed",
    iconSlug: "box",
  };
  return {
    id: `column-app-${ref.runId}`,
    name,
    status,
    duration: "",
    startedAt: first.updatedAt,
    componentName,
    artifacts: artifactsForCanvasRun(artifacts, ref.runId),
    checks: presentWorkOrderChecks(checks),
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: ref.appId,
    runId: ref.runId,
    columnKey,
  };
}

function columnAppCheckRef(check: FactoriesWorkOrderCheck): { appId: string; runId: string } | undefined {
  const appId = check.automation?.appId?.trim();
  const runId = check.runId?.trim();
  if (!appId || !runId) {
    return undefined;
  }
  return { appId, runId };
}

function columnAppCheckName(app: SplitRunColumnApp, check: FactoriesWorkOrderCheck, fallback: string): string {
  return app.name?.trim() || check.automation?.appName?.trim() || check.name || fallback;
}

function consoleColumnForAppKey(columnKey?: string): SplitRunBoardColumn | undefined {
  const key = columnKey?.trim();
  if (key === "verify" || key === "done" || key === "backlog") {
    return key;
  }
  if (key === "implement" || key?.startsWith("phase-")) {
    return "implement";
  }
  return undefined;
}

function phaseNameForColumn(column: SplitRunBoardColumn): string {
  if (column === "verify") {
    return "Verify";
  }
  if (column === "done") {
    return "Done";
  }
  if (column === "backlog") {
    return "Backlog";
  }
  return "Implement";
}

function artifactsForCanvasRun(
  artifacts: FactoriesWorkOrderArtifact[] | undefined,
  runId: string,
): FactoriesWorkOrderArtifact[] {
  return (artifacts ?? []).filter((artifact) => artifactCanvasRunId(artifact) === runId);
}

function artifactCanvasRunId(artifact: FactoriesWorkOrderArtifact): string | undefined {
  const data = artifact.data;
  if (!data || typeof data !== "object") {
    return undefined;
  }
  const value = data.canvasRunId;
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function columnAppForCanvas(columnApps: SplitRunColumnApp[], canvasId?: string): SplitRunColumnApp | undefined {
  const id = canvasId?.trim();
  if (!id) {
    return undefined;
  }
  return columnApps.find((app) => app.id === id);
}

function prFeedbackComponentName(entry: PRFeedbackLogRun, columnApps: SplitRunColumnApp[]): string {
  const handlerName = entry.handlerName?.trim();
  if (handlerName) {
    return handlerName;
  }
  const appName = columnAppForCanvas(columnApps, entry.canvasId)?.name?.trim();
  if (appName) {
    return appName;
  }
  return "Address PR feedback";
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

function prFeedbackActivityBaseName(entry: PRFeedbackLogRun): string {
  const title = entry.title?.trim();
  if (title) {
    return title;
  }
  const description = entry.description?.trim();
  if (description) {
    return description;
  }
  if (entry.pullRequestNumber) {
    return `Activity on PR #${String(entry.pullRequestNumber).replace(/^#/, "")}`;
  }
  return "Activity on PR";
}

function prFeedbackActivityName(entry: PRFeedbackLogRun): string {
  const attempt = entry.attemptLabel?.trim();
  const baseName = prFeedbackActivityBaseName(entry);
  return attempt ? `${baseName} ${attempt}` : baseName;
}

function prFeedbackPhaseStatus(entry: PRFeedbackLogRun): SplitRunPhaseStatus {
  if (entry.waitingForAccess && isActiveCanvasRun(entry.run)) {
    return "waiting";
  }
  return statusForCanvasRun(entry.run);
}

function prFeedbackStreamAction(status: SplitRunPhaseStatus): string {
  if (status === "passed") {
    return "passed";
  }
  if (status === "failed") {
    return "failed";
  }
  if (status === "running") {
    return "running";
  }
  return "—";
}

function prFeedbackUpdatedAt(entry: PRFeedbackLogRun) {
  return entry.run.finishedAt ?? entry.run.updatedAt ?? entry.run.createdAt;
}

function prFeedbackAttachedPullRequest(entry: PRFeedbackLogRun) {
  if (entry.pullRequest) {
    return entry.pullRequest;
  }
  if (entry.pullRequestNumber) {
    return { number: entry.pullRequestNumber };
  }
  return undefined;
}

function prFeedbackPhaseDescription(entry: PRFeedbackLogRun) {
  const title = entry.title?.trim();
  if (!title) {
    return undefined;
  }
  return entry.description?.trim();
}

function prFeedbackRunToPhase(entry: PRFeedbackLogRun, columnApps: SplitRunColumnApp[] = []): SplitRunPhase {
  const status = prFeedbackPhaseStatus(entry);
  const name = prFeedbackActivityName(entry);
  const componentName = prFeedbackComponentName(entry, columnApps);
  const duration = durationForExecution(
    { createdAt: entry.run.createdAt, updatedAt: prFeedbackUpdatedAt(entry) },
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
    action: prFeedbackStreamAction(status),
    iconSlug: "box",
  };
  return {
    id: `pr-feedback-${entry.run.id}`,
    name,
    description: prFeedbackPhaseDescription(entry),
    status,
    duration,
    startedAt: entry.run.createdAt,
    componentName,
    artifacts: [],
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: entry.canvasId,
    runId: entry.run.id,
    costCents: entry.costCents,
    totalTokens: entry.totalTokens,
    columnKey: consoleColumnForAppKey(columnAppForCanvas(columnApps, entry.canvasId)?.columnKey),
    pullRequestActivity: {
      pullRequest: prFeedbackAttachedPullRequest(entry),
      revision: entry.revision,
      startedAt: entry.run.createdAt,
      waitingForAccess: entry.waitingForAccess,
    },
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

/** Card title: the factory automation name, or the template label when the run has none. */
function automationCardName(appName: string | undefined, fallback: string): string {
  const name = appName?.trim();
  return name || fallback;
}

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
  if (order.origin?.url?.trim()) {
    return importedBacklogPhase(order, description);
  }
  return manualBacklogPhase(order, description);
}

function automationBacklogPhase(
  order: FactoriesWorkOrder,
  automation: FactoriesAutomationRef,
  description: FactoriesWorkOrderArtifact,
): SplitRunPhase {
  const app = { id: automation.appId, name: automation.appName };
  const presentation = lineAutomationPresentation(app);
  const componentName = automation.appName?.trim() || presentation.componentName;
  const at = clockLabel(order.createdAt);
  return {
    id: "backlog",
    name: "Backlog",
    status: "passed",
    duration: "2s",
    startedAt: order.createdAt,
    componentName,
    description: intakeCreationDescription(order),
    artifacts: [description],
    stream: [
      {
        id: "backlog-create",
        at,
        componentName: automation.nodeName?.trim() || "Create Task",
        status: "passed",
        duration: "2s",
        artifact: description,
        kind: "action",
        componentType: "Create Task",
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

function importedBacklogPhase(order: FactoriesWorkOrder, description: FactoriesWorkOrderArtifact): SplitRunPhase {
  const source = splitRunSourceForOrder(order);
  if (source.kind !== "intake") {
    return manualBacklogPhase(order, description);
  }
  const at = clockLabel(order.createdAt);
  const sentence = importedSourceSentence(order, source.iconAlt, source.ticket);
  return {
    id: "backlog",
    name: "Backlog",
    description: sentence.markdown,
    status: "passed",
    duration: "2s",
    startedAt: order.createdAt,
    componentName: `Imported from ${source.iconAlt}`,
    artifacts: [description],
    stream: [
      {
        id: "backlog-imported",
        at,
        componentName: sentence.plain,
        status: "passed",
        duration: "2s",
        artifact: description,
        kind: "action",
        componentType: "Create Task",
        action: "passed",
        iconSlug: "user",
      },
    ],
    canvasSteps: [],
    canvasKey: null,
  };
}

function intakeCreationDescription(order: FactoriesWorkOrder): string {
  const source = splitRunSourceForOrder(order);
  const ticket = source.kind === "intake" ? source.ticket : undefined;
  if (!ticket) {
    return "Created this task.";
  }
  return `Created this task from [${ticket.label}](${ticket.href}).`;
}

function importedSourceSentence(
  order: FactoriesWorkOrder,
  product: string,
  ticket?: { label: string; href: string },
): { plain: string; markdown: string } {
  const person = order.createdBy?.user?.name?.trim() || "A person";
  if (!ticket) {
    const text = `${person} imported this task from ${product}.`;
    return { plain: text, markdown: text };
  }
  return {
    plain: `${person} imported this task from ${ticket.label}.`,
    markdown: `${person} imported this task from [${ticket.label}](${ticket.href}).`,
  };
}

function manualBacklogPhase(order: FactoriesWorkOrder, description: FactoriesWorkOrderArtifact): SplitRunPhase {
  const at = clockLabel(order.createdAt);
  const line = draftSourceSentence(order);
  return {
    id: "backlog",
    name: "Backlog",
    description: line,
    status: "passed",
    duration: "2s",
    startedAt: order.createdAt,
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
        componentType: "Create Task",
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
    // The description is written when the task is created, so it carries the
    // task's createdAt and sorts first in age-ordered artifact lists.
    createdAt: order.createdAt,
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
  const presentation = lineAutomationPresentation(execution.run, execution.step);
  const name = presentation.name;
  const componentName = automationCardName(execution.run?.appName, presentation.componentName);
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
    startedAt: execution.createdAt || execution.updatedAt,
    endedAt: endedAtForExecution(execution, status),
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
    model: modelsForExecution(order, execution),
    thinkingLevel: thinkingLevelForExecution(order, execution),
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
    : source.filter((check) => !isScoreCheckName(check.name));
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

function modelsForExecution(order: FactoriesWorkOrder, execution: FactoriesWorkOrderExecution): string | undefined {
  const reported = joinRunnerModels(execution.models ?? []);
  if (reported) {
    return reported;
  }
  return dispatchModelForExecution(order, execution);
}

function thinkingLevelForExecution(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
): string | undefined {
  const value = dispatchForExecution(order, execution)?.thinkingLevel?.trim();
  return value || undefined;
}

function dispatchModelForExecution(
  order: FactoriesWorkOrder,
  execution: FactoriesWorkOrderExecution,
): string | undefined {
  const value = dispatchForExecution(order, execution)?.model?.trim();
  return value || undefined;
}

function dispatchForExecution(order: FactoriesWorkOrder, execution: FactoriesWorkOrderExecution) {
  return (order.lineDispatches ?? []).find((dispatch) =>
    (dispatch.stepExecutions ?? []).some((step) => step.id && step.id === execution.id),
  );
}

function endedAtForExecution(execution: FactoriesWorkOrderExecution, status: SplitRunPhaseStatus): string | undefined {
  if (status === "running" || status === "pending") {
    return undefined;
  }
  const value = execution.finishedAt || execution.updatedAt;
  return Date.parse(value ?? "") ? value : undefined;
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
