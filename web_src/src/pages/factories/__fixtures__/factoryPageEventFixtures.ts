import type { FactoriesWorkOrder, FactoriesWorkOrderEvent } from "@/api-client";

import {
  CLOSED_FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FAILED_WORK_ORDER,
  HOUR_AGO,
  LAST_WEEK,
  OPEN_WORK_ORDER,
  OPERATOR_USER,
  REVIEWER_USER,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_ID,
  TWO_HOURS_AGO,
  YESTERDAY,
} from "./factoryPageResponses";

const REFUND_LINE = { id: "line-plan-and-implement", name: "plan-and-implement" };

interface StepExecutionEventFixture {
  order: FactoriesWorkOrder;
  stepName: string;
  at: string;
  runId: string;
  appId: string;
  result?: "passed" | "failed";
}

interface CommentAuthorFixture {
  kind: "user" | "automation" | "system";
  userId?: string;
  automation?: {
    nodeId?: string;
    nodeName?: string;
    appId?: string;
    appName?: string;
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

function openedWorkOrderEvent(order: FactoriesWorkOrder, at: string): FactoriesWorkOrderEvent {
  return {
    type: "order.opened",
    timestamp: at,
    event: {
      user: { id: order.createdBy?.id },
      order: { id: order.id, title: order.title },
    },
  };
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
      app: { id: appId },
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
      app: { id: appId },
      run: { id: runId, state: "finished", result },
    },
  };
}

function statusUpdatedEvent(
  order: FactoriesWorkOrder,
  at: string,
  transition: { fromState: string; toState: string; toResult?: string; actor?: { id: string } },
): FactoriesWorkOrderEvent {
  const { fromState, toState, toResult, actor = { id: STORYBOOK_ME_USER_ID } } = transition;
  return {
    type: "order.status.updated",
    timestamp: at,
    event: {
      user: { id: actor.id },
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
): FactoriesWorkOrderEvent {
  return {
    type: "order.comment.added",
    timestamp: at,
    event: {
      order: { id: order.id, title: order.title },
      body,
      author,
    },
  };
}

function artifactAddedEvent(
  order: FactoriesWorkOrder,
  at: string,
  artifact: {
    id: string;
    type: "pr" | "markdown";
    url?: string;
    title?: string;
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

function automationClosedEvent(
  order: FactoriesWorkOrder,
  at: string,
  result: "completed" | "failed" | "rejected",
  automation: AutomationRefFixture,
): FactoriesWorkOrderEvent {
  return {
    type: "order.closed",
    timestamp: at,
    event: {
      automation,
      order: { id: order.id, title: order.title },
      result,
    },
  };
}

export const OPEN_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [openedWorkOrderEvent(OPEN_WORK_ORDER, HOUR_AGO)];

export const DRAFT_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  statusUpdatedEvent(DRAFT_WORK_ORDER, TWO_HOURS_AGO, { fromState: "", toState: "draft" }),
  commentAddedEvent(
    DRAFT_WORK_ORDER,
    HOUR_AGO,
    "Scoping notes: need product sign-off on metric names before moving to ready.",
    { kind: "user", userId: STORYBOOK_ME_USER_ID },
  ),
];

export const CLOSED_FAILED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(CLOSED_FAILED_WORK_ORDER, LAST_WEEK),
  artifactAddedEvent(
    CLOSED_FAILED_WORK_ORDER,
    YESTERDAY,
    {
      id: "art-audit-report",
      type: "markdown",
      title: "Reconciliation report",
      data: { body: "Ledger totals mismatched by $412.66 — see attached JIRA for follow-up." },
    },
    { id: OPERATOR_USER.id },
  ),
  automationClosedEvent(CLOSED_FAILED_WORK_ORDER, YESTERDAY, "failed", {
    nodeName: "node-close",
    appName: "Refund Diagnostics",
    lineName: "Plan",
    stepName: "step-01",
  }),
];

export const RUNNING_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(RUNNING_WORK_ORDER, YESTERDAY),
  stepExecutionCreatedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "plan",
    at: TWO_HOURS_AGO,
    runId: "run-plan",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "plan",
    at: TWO_HOURS_AGO,
    runId: "run-plan",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: RUNNING_WORK_ORDER,
    stepName: "implement",
    at: HOUR_AGO,
    runId: "run-implement",
    appId: "app-refund-implementer",
  }),
];

export const FAILED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(FAILED_WORK_ORDER, YESTERDAY),
  stepExecutionCreatedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "plan",
    at: TWO_HOURS_AGO,
    runId: "run-plan-2",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "plan",
    at: TWO_HOURS_AGO,
    runId: "run-plan-2",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "implement",
    at: HOUR_AGO,
    runId: "run-implement-2",
    appId: "app-refund-implementer",
  }),
  stepExecutionFinishedEvent({
    order: FAILED_WORK_ORDER,
    stepName: "implement",
    at: HOUR_AGO,
    runId: "run-implement-2",
    appId: "app-refund-implementer",
    result: "failed",
  }),
];

export const CLOSED_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [
  openedWorkOrderEvent(CLOSED_WORK_ORDER, LAST_WEEK),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "plan",
    at: LAST_WEEK,
    runId: "run-plan-3",
    appId: "app-refund-planner",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "plan",
    at: LAST_WEEK,
    runId: "run-plan-3",
    appId: "app-refund-planner",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "implement",
    at: LAST_WEEK,
    runId: "run-implement-3",
    appId: "app-refund-implementer",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "implement",
    at: LAST_WEEK,
    runId: "run-implement-3",
    appId: "app-refund-implementer",
    result: "passed",
  }),
  stepExecutionCreatedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "verify",
    at: YESTERDAY,
    runId: "run-verify-3",
    appId: "app-refund-verifier",
  }),
  stepExecutionFinishedEvent({
    order: CLOSED_WORK_ORDER,
    stepName: "verify",
    at: YESTERDAY,
    runId: "run-verify-3",
    appId: "app-refund-verifier",
    result: "passed",
  }),
  {
    type: "order.closed",
    timestamp: YESTERDAY,
    event: {
      user: { id: STORYBOOK_ME_USER_ID },
      order: { id: CLOSED_WORK_ORDER.id, title: CLOSED_WORK_ORDER.title },
      result: "completed",
    },
  },
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
    "I re-ran the failing test locally and confirmed the duplicate entry appears only on retry #3.",
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
      url: "https://github.com/example/ledger/pull/482",
      title: "Fix duplicate refund on retry",
      data: { number: 482 },
    },
    { id: REVIEWER_USER.id },
  ),
  artifactAddedEvent(
    OPEN_WORK_ORDER,
    HOUR_AGO,
    {
      id: "art-md-1",
      type: "markdown",
      title: "Investigation notes",
      data: {
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
      url: "https://github.com/example/ledger/pull/483",
      title: "Automated retry fix",
      data: { number: 483 },
    },
    null,
    { nodeName: "attach-artifact", appName: "Refund Diagnostics", lineName: "Plan", stepName: "step-01" },
  ),
];
