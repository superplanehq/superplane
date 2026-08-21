import type { FactoriesWorkOrder, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import {
  CLOSED_FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FAILED_WORK_ORDER,
  HOUR_AGO,
  INGEST_DRAFT_WORK_ORDER,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  OPERATOR_USER,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  REVIEWER_USER,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_ID,
  TWO_HOURS_AGO,
  YESTERDAY,
} from "./factoryPageResponses";
import { OPEN_WORK_ORDER_ARTIFACTS } from "./factoryPageFixtureVariants";

const REFUND_LINE = { id: "line-plan-and-implement", name: "plan-and-implement" };

/** Fresh timestamp for the risk re-score, matching the checks fixture's "updated 12 minutes ago". */
const TWELVE_MINUTES_AGO = new Date(Date.now() - 12 * 60_000).toISOString();
/** Fresh timestamp for the CI pass, matching the checks fixture's "updated 3 minutes ago". */
const THREE_MINUTES_AGO = new Date(Date.now() - 3 * 60_000).toISOString();

const RISK_REVIEW_AUTOMATION = { appId: "app-pr-risk-review", appName: "PR Risk Review" };
const CI_LOOP_AUTOMATION = { appId: "app-ci-loop", appName: "CI Loop" };

interface StepExecutionEventFixture {
  order: FactoriesWorkOrder;
  stepName: string;
  at: string;
  runId: string;
  appId: string;
  result?: "passed" | "failed";
}

interface CommentAuthorFixture {
  kind: "user" | "automation";
  userId?: string;
  automation?: {
    nodeId?: string;
    nodeName?: string;
    appId?: string;
    appName?: string;
    lineId?: string;
    lineName?: string;
    stepIndex?: number;
    stepName?: string;
  };
}

interface AutomationRefFixture {
  nodeId?: string;
  nodeName?: string;
  appId?: string;
  appName?: string;
  lineId?: string;
  lineName?: string;
  stepIndex?: number;
  stepName?: string;
}

// The lifecycle emits `order.status.updated` for every transition (creation,
// open, close). This helper wraps the common `draft → open` promotion — the
// closest analogue to the old `order.opened` event — for readable fixtures.
function openedWorkOrderEvent(order: FactoriesWorkOrder, at: string): FactoriesWorkOrderEvent {
  return statusUpdatedEvent(order, at, {
    fromState: "draft",
    toState: "open",
    actor: { id: order.createdBy?.user?.id ?? STORYBOOK_ME_USER_ID },
  });
}

function stepExecutionCreatedEvent(input: StepExecutionEventFixture): FactoriesWorkOrderEvent {
  const { order, stepName, at, runId, appId } = input;
  return {
    type: "step.execution.created",
    timestamp: at,
    event: {
      stepName,
      order: { id: order.id, title: order.title },
      line: REFUND_LINE,
      app: { id: appId, name: stepName },
      run: { id: runId, state: "pending" },
    },
  };
}

function stepExecutionFinishedEvent(
  input: StepExecutionEventFixture & { result: "passed" | "failed" },
): FactoriesWorkOrderEvent {
  const { order, stepName, at, runId, appId, result } = input;
  return {
    type: "step.execution.finished",
    timestamp: at,
    event: {
      stepName,
      order: { id: order.id, title: order.title },
      line: REFUND_LINE,
      app: { id: appId, name: stepName },
      run: { id: runId, state: "finished", result },
    },
  };
}

// `actor` is intentionally NOT defaulted: canvas-driven transitions in
// production carry `automation` (+ run/app) with no `user`, and the
// timeline's attribution logic prefers a user over an automation actor.
// A silent default would spoof user attribution onto canvas events and
// hide the factory-line actor stories were meant to showcase.
function statusUpdatedEvent(
  order: FactoriesWorkOrder,
  at: string,
  transition: {
    fromState: string;
    toState: string;
    toResult?: string;
    actor?: { id: string };
    automation?: AutomationRefFixture;
    run?: { id: string };
    app?: { id: string };
  },
): FactoriesWorkOrderEvent {
  const { fromState, toState, toResult, actor, automation, run, app } = transition;
  return {
    type: "order.status.updated",
    timestamp: at,
    event: {
      ...(actor ? { user: { id: actor.id } } : {}),
      ...(automation ? { automation } : {}),
      ...(run ? { run } : {}),
      ...(app ? { app } : {}),
      order: { id: order.id, title: order.title },
      fromState,
      toState,
      ...(toResult ? { toResult } : {}),
    },
  };
}

function commentAddedEvent(
  order: FactoriesWorkOrder,
  at: string,
  body: string,
  author: CommentAuthorFixture,
  run?: { id: string },
): FactoriesWorkOrderEvent {
  return {
    type: "order.comment.added",
    timestamp: at,
    event: {
      order: { id: order.id, title: order.title },
      body,
      author,
      ...(run ? { run } : {}),
    },
  };
}

function artifactAddedEvent(
  order: FactoriesWorkOrder,
  at: string,
  artifact: {
    id: string;
    type: "pr" | "markdown" | "branch";
    data?: Record<string, unknown>;
  },
  actor: { id: string } | null = { id: STORYBOOK_ME_USER_ID },
  automation?: AutomationRefFixture,
): FactoriesWorkOrderEvent {
  return {
    type: "order.artifact.added",
    timestamp: at,
    event: {
      ...(actor ? { user: { id: actor.id } } : {}),
      ...(automation ? { automation } : {}),
      order: { id: order.id, title: order.title },
      artifact,
    },
  };
}

// A score reported by a dedicated automation (`order.check.reported`).
// `previousScore` marks a re-score, which the timeline renders as a trend
// ("82 → 65") instead of a bare number.
function checkReportedEvent(
  order: FactoriesWorkOrder,
  at: string,
  check: {
    name: string;
    score: number;
    maxScore: number;
    format?: "fraction" | "percent" | "boolean";
    previousScore?: number;
  },
  automation: AutomationRefFixture,
  run?: { id: string },
): FactoriesWorkOrderEvent {
  return {
    type: "order.check.reported",
    timestamp: at,
    event: {
      order: { id: order.id, title: order.title },
      check,
      automation,
      ...(run ? { run } : {}),
      ...(automation.appId ? { app: { id: automation.appId } } : {}),
    },
  };
}

// Emit a canvas-attributed close as an `order.status.updated` (open → closed)
// carrying the automation ref, matching how FactoryContext records it in
// production.
function automationClosedEvent(
  order: FactoriesWorkOrder,
  at: string,
  result: "completed" | "failed" | "rejected",
  automation: AutomationRefFixture,
): FactoriesWorkOrderEvent {
  return statusUpdatedEvent(order, at, {
    fromState: "open",
    toState: "closed",
    toResult: result,
    automation,
  });
}

export const OPEN_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [openedWorkOrderEvent(OPEN_WORK_ORDER, HOUR_AGO)];

export const DRAFT_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  statusUpdatedEvent(DRAFT_WORK_ORDER, TWO_HOURS_AGO, {
    fromState: "",
    toState: "draft",
    actor: { id: STORYBOOK_ME_USER_ID },
  }),
  commentAddedEvent(
    DRAFT_WORK_ORDER,
    HOUR_AGO,
    "Scoping notes: need product sign-off on metric names before moving to ready.",
    { kind: "user", userId: STORYBOOK_ME_USER_ID },
  ),
];

export const INGEST_DRAFT_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  statusUpdatedEvent(INGEST_DRAFT_WORK_ORDER, TWO_HOURS_AGO, {
    fromState: "",
    toState: "draft",
    automation: {
      appId: "app-refund-backlog",
      appName: "Ingest",
      nodeName: "On Issue Label",
    },
    app: { id: "app-refund-backlog" },
  }),
];

export const CLOSED_FAILED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(CLOSED_FAILED_WORK_ORDER, LAST_WEEK),
  artifactAddedEvent(
    CLOSED_FAILED_WORK_ORDER,
    YESTERDAY,
    {
      id: "art-audit-report",
      type: "markdown",
      data: {
        title: "Reconciliation report",
        body: "Ledger totals mismatched by $412.66 — see attached JIRA for follow-up.",
      },
    },
    { id: OPERATOR_USER.id },
  ),
  automationClosedEvent(CLOSED_FAILED_WORK_ORDER, YESTERDAY, "failed", {
    nodeName: "node-close",
    appName: "Refund Diagnostics",
    lineName: "Plan",
    stepName: "Refund Diagnostics",
  }),
];

export const RUNNING_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(RUNNING_WORK_ORDER, YESTERDAY),
  stepExecutionCreatedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "Refund Planner",
    at: TWO_HOURS_AGO,
    runId: "run-plan",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "Refund Planner",
    at: TWO_HOURS_AGO,
    runId: "run-plan",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "Refund Implementer",
    at: HOUR_AGO,
    runId: LINE_RUN_IMPLEMENT_ID,
    appId: "app-refund-implementer",
  }),
  // Automation-authored comment attached to the in-flight "implement" step.
  // It is matched to the step by line id ("plan-and-implement"), not the
  // canvas node name, and renders as plain text — the step's own title
  // already names the line and links to its run.
  commentAddedEvent(
    RUNNING_WORK_ORDER,
    HOUR_AGO,
    "Applying the ledger fix now, will report back once tests pass.",
    {
      kind: "automation",
      automation: {
        nodeId: "node-implement",
        nodeName: "node-implement",
        appId: "app-refund-implementer",
        appName: "Refund Implementer",
        lineId: REFUND_LINE.id,
        lineName: REFUND_LINE.name,
        stepName: "Refund Implementer",
      },
    },
    { id: LINE_RUN_IMPLEMENT_ID },
  ),
];

export const FAILED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(FAILED_WORK_ORDER, YESTERDAY),
  stepExecutionCreatedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "Refund Planner",
    at: TWO_HOURS_AGO,
    runId: "run-plan-2",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "Refund Planner",
    at: TWO_HOURS_AGO,
    runId: "run-plan-2",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "Refund Implementer",
    at: HOUR_AGO,
    runId: LINE_RUN_IMPLEMENT_FAILED_ID,
    appId: "app-refund-implementer",
  }),
  stepExecutionFinishedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "Refund Implementer",
    at: HOUR_AGO,
    runId: LINE_RUN_IMPLEMENT_FAILED_ID,
    appId: "app-refund-implementer",
    result: "failed",
  }),
];

export const CLOSED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(CLOSED_WORK_ORDER, LAST_WEEK),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Planner",
    at: LAST_WEEK,
    runId: "run-plan-3",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Planner",
    at: LAST_WEEK,
    runId: "run-plan-3",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Implementer",
    at: LAST_WEEK,
    runId: LINE_RUN_IMPLEMENT_PASSED_ID,
    appId: "app-refund-implementer",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Implementer",
    at: LAST_WEEK,
    runId: LINE_RUN_IMPLEMENT_PASSED_ID,
    appId: "app-refund-implementer",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Verifier",
    at: YESTERDAY,
    runId: LINE_RUN_VERIFY_PASSED_ID,
    appId: "app-refund-verifier",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "Refund Verifier",
    at: YESTERDAY,
    runId: LINE_RUN_VERIFY_PASSED_ID,
    appId: "app-refund-verifier",
    result: "passed",
  }),
  statusUpdatedEvent(CLOSED_WORK_ORDER, YESTERDAY, {
    fromState: "open",
    toState: "closed",
    toResult: "completed",
    actor: { id: STORYBOOK_ME_USER_ID },
  }),
];

export const PR_CLOSURE_COMPLETED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(PR_CLOSURE_COMPLETED_WORK_ORDER, LAST_WEEK),
  automationClosedEvent(PR_CLOSURE_COMPLETED_WORK_ORDER, YESTERDAY, "completed", {
    appId: "app-refund-done",
    appName: "PR Closure",
    nodeName: "Complete Work Order",
  }),
];

export const RICH_OPEN_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(OPEN_WORK_ORDER, YESTERDAY),
  commentAddedEvent(
    OPEN_WORK_ORDER,
    TWO_HOURS_AGO,
    "Kicked off — starting with the ledger diff, will attach a repro PR shortly.",
    { kind: "user", userId: REVIEWER_USER.id },
  ),
  commentAddedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    [
      "**Blocked** on the ledger writer — reproduced on retry `#3`.",
      "",
      "Next steps:",
      "- Confirm the idempotency window in staging",
      "- Follow up on [PR #482](https://github.com/example/ledger/pull/482)",
    ].join("\n"),
    {
      kind: "automation",
      automation: { nodeName: "reproduce-failure", appName: "Refund Diagnostics" },
    },
  ),
  artifactAddedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    {
      id: "art-pr-1",
      type: "pr",
      data: {
        url: "https://github.com/example/ledger/pull/482",
        title: "Fix duplicate refund on retry",
        number: 482,
      },
    },
    { id: REVIEWER_USER.id },
  ),
  artifactAddedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    {
      id: "art-md-1",
      type: "markdown",
      data: {
        title: "Investigation notes",
        body: "Retry policy exceeded idempotency window when the ledger writer was under load; details captured in the design doc.",
      },
    },
    { id: REVIEWER_USER.id },
  ),
  artifactAddedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    {
      id: "art-auto-1",
      type: "pr",
      data: {
        url: "https://github.com/example/ledger/pull/483",
        title: "Automated retry fix",
        number: 483,
      },
    },
    null,
    { nodeName: "attach-artifact", appName: "Refund Diagnostics", lineName: "Plan", stepName: "Refund Diagnostics" },
  ),
  // Check history: the first risk score landed before the PR, follow-up
  // commits addressed the concerns, and the re-score dropped 82 → 65 — the
  // timeline tells the trend while the Checks section shows only the latest.
  checkReportedEvent(
    OPEN_WORK_ORDER,
    TWO_HOURS_AGO,
    { name: "Risk review", score: 82, maxScore: 100 },
    RISK_REVIEW_AUTOMATION,
    { id: "run-risk-review-100" },
  ),
  checkReportedEvent(
    OPEN_WORK_ORDER,
    TWELVE_MINUTES_AGO,
    { name: "Risk review", score: 65, maxScore: 100, previousScore: 82 },
    RISK_REVIEW_AUTOMATION,
    { id: "run-risk-review-101" },
  ),
  // Boolean check history: CI failed first, the loop's automated fix landed,
  // and the re-report flipped Fail → Pass.
  checkReportedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    { name: "CI", score: 0, maxScore: 1, format: "boolean" },
    CI_LOOP_AUTOMATION,
    { id: "run-ci-100" },
  ),
  checkReportedEvent(
    OPEN_WORK_ORDER,
    THREE_MINUTES_AGO,
    { name: "CI", score: 1, maxScore: 1, format: "boolean", previousScore: 0 },
    CI_LOOP_AUTOMATION,
    { id: "run-ci-101" },
  ),
];

/**
 * Activity served by the fixture HTTP handlers for harnessed page stories
 * (`GET …/orders/{orderId}/events`). The open order uses the rich set so
 * the page-level detail story shows comments, artifacts, and runs without
 * per-story wiring. Direct-props stories keep choosing their own arrays.
 */
export const DEFAULT_EVENTS_BY_ORDER_ID: Record<string, FactoriesWorkOrderEvent[]> = {
  [OPEN_WORK_ORDER.id!]: RICH_OPEN_WORK_ORDER_EVENTS,
  [OPEN_WORK_ORDER_SECONDARY.id!]: [openedWorkOrderEvent(OPEN_WORK_ORDER_SECONDARY, TWO_HOURS_AGO)],
  [RUNNING_WORK_ORDER.id!]: RUNNING_WORK_ORDER_EVENTS,
  [FAILED_WORK_ORDER.id!]: FAILED_WORK_ORDER_EVENTS,
  [DRAFT_WORK_ORDER.id!]: DRAFT_WORK_ORDER_EVENTS,
  [INGEST_DRAFT_WORK_ORDER.id!]: INGEST_DRAFT_WORK_ORDER_EVENTS,
  [CLOSED_WORK_ORDER.id!]: CLOSED_WORK_ORDER_EVENTS,
  [PR_CLOSURE_COMPLETED_WORK_ORDER.id!]: PR_CLOSURE_COMPLETED_WORK_ORDER_EVENTS,
  [CLOSED_FAILED_WORK_ORDER.id!]: CLOSED_FAILED_WORK_ORDER_EVENTS,
};

/** Artifacts served by the fixture HTTP handlers (`GET …/orders/{orderId}/artifacts`). */
export const DEFAULT_ARTIFACTS_BY_ORDER_ID: Record<string, FactoriesWorkOrderArtifact[]> = {
  [OPEN_WORK_ORDER.id!]: OPEN_WORK_ORDER_ARTIFACTS,
};
