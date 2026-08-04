import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";

/** Shared with the home fixture so routes stay in sync across HomePage → Factories navigation. */
export const FACTORIES_ORGANIZATION_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";

export const PRIMARY_FACTORY_ID = "factory-refunds";
export const EMPTY_FACTORY_ID = "factory-payments";

export const STORYBOOK_ME_USER_ID = "storybook-user";
export const STORYBOOK_ME_USER_NAME = "Storybook User";
export const STORYBOOK_ME_USER_EMAIL = "storybook@superplane.dev";

// Anchored to Date.now() at module load so `formatTimeAgo` renders the intended
// "1 hour ago" / "yesterday" / "last week" labels regardless of when a story is
// opened. (Absolute dates cause labels to drift over time.)
const NOW_MS = Date.now();
const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;
const relativeIso = (offsetMs: number) => new Date(NOW_MS - offsetMs).toISOString();

const HOUR_AGO = relativeIso(HOUR_MS);
const TWO_HOURS_AGO = relativeIso(2 * HOUR_MS);
const YESTERDAY = relativeIso(DAY_MS);
const LAST_WEEK = relativeIso(7 * DAY_MS);

export const REVIEWER_USER = {
  id: "user-reviewer-alex",
  name: "Alex Reviewer",
  email: "alex@superplane.dev",
} as const;

export const OPERATOR_USER = {
  id: "user-operator-jamie",
  name: "Jamie Operator",
  email: "jamie@superplane.dev",
} as const;

export const ORGANIZATION_USERS = [
  { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME, email: STORYBOOK_ME_USER_EMAIL },
  REVIEWER_USER,
  OPERATOR_USER,
];

const RUN_APP_TYPE = "runApp";

function runAppStep(name: string, appId: string, entrypoint: string): FactoryLineStep {
  return {
    name,
    type: RUN_APP_TYPE,
    app: { app: appId, entrypoint },
  };
}

export const REFUND_FACTORY_APPS: FactoryApp[] = [
  {
    id: "app-refund-planner",
    name: "Refund Planner",
    description: "Plans reconciliation work across ledger + payment services.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-refund-implementer",
    name: "Refund Implementer",
    description: "Applies the plan across affected repos and opens PRs.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-refund-verifier",
    name: "Refund Verifier",
    description: "Runs verification suites and gates merge.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
];

const REFUND_LINE_PLAN_ID = "line-plan-and-implement";
const REFUND_LINE_HOTFIX_ID = "line-hotfix";

export const REFUND_FACTORY_LINES: FactoriesFactoryLine[] = [
  {
    id: REFUND_LINE_PLAN_ID,
    name: "plan-and-implement",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [
      runAppStep("plan", "app-refund-planner", "start-plan"),
      runAppStep("implement", "app-refund-implementer", "start-implementation"),
      runAppStep("verify", "app-refund-verifier", "start-verification"),
    ],
  },
  {
    id: REFUND_LINE_HOTFIX_ID,
    name: "hotfix",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [runAppStep("verify", "app-refund-verifier", "start-verification")],
  },
];

export const REFUND_FACTORY: FactoriesFactory = {
  id: PRIMARY_FACTORY_ID,
  name: "Refunds Factory",
  description:
    "Handles reconciliation work: plan a change, implement across affected services, and verify with regression suites.",
  lines: REFUND_FACTORY_LINES,
};

export const EMPTY_FACTORY: FactoriesFactory = {
  id: EMPTY_FACTORY_ID,
  name: "Payments Factory",
  description: "New factory. No lines or work orders configured yet.",
  lines: [],
};

function planLineExecution(
  step: string,
  overrides: Partial<FactoriesWorkOrderExecution> = {},
): FactoriesWorkOrderExecution {
  return {
    id: `exec-${step}-${overrides.id ?? Math.random().toString(36).slice(2, 8)}`,
    line: { id: REFUND_LINE_PLAN_ID, name: "plan-and-implement" },
    step,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: TWO_HOURS_AGO,
    updatedAt: HOUR_AGO,
    run: {
      id: `run-${step}`,
      appId: "app-refund-planner",
      appName: "Refund Planner",
    },
    ...overrides,
  };
}

export const OPEN_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-open-refunds",
  title: "Reconcile duplicate refunds in ledger",
  description:
    "Users are seeing duplicate refund entries after retries. Reconcile the ledger, patch retry logic, and add a regression test.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
  assignees: [
    { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
    { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  ],
  executions: [],
};

/**
 * Second open work order assigned to the storybook user so the FactoryDetailPage
 * "Populated" story shows a real list under the default `mine + open` filters.
 */
export const OPEN_WORK_ORDER_SECONDARY: FactoriesWorkOrder = {
  id: "wo-open-refunds-schema",
  title: "Add refund reason enum to schema",
  description: "Extend the refund ledger schema with a nullable reason enum so audit can categorize.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: TWO_HOURS_AGO,
  createdBy: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [],
};

// Assignees include the storybook user so the running/failed variants surface
// under the "mine + running" and "mine + failed" filters. Reviewer/operator
// remain listed to demonstrate multi-assignee cards.
export const RUNNING_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-running-refunds",
  title: "Add refund reconciliation test",
  description: "Cover the newly discovered duplicate refund case with a regression test running in CI.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  assignees: [
    { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
    { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  ],
  executions: [
    planLineExecution("plan", { id: "1", state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: TWO_HOURS_AGO }),
    planLineExecution("implement", {
      id: "2",
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      run: { id: "run-implement", appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: HOUR_AGO,
    }),
  ],
};

// Failed order stays authored by operator; storybook user co-assigned so the
// "mine + failed" filter is meaningfully populated.
export const FAILED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-failed-refunds",
  title: "Ship idempotent refund retries",
  description: "Retry logic should be idempotent so replays don't create duplicate refunds.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { id: OPERATOR_USER.id, name: OPERATOR_USER.name },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [
    planLineExecution("plan", { id: "3", state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: TWO_HOURS_AGO }),
    planLineExecution("implement", {
      id: "4",
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      run: { id: "run-implement-2", appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: HOUR_AGO,
    }),
  ],
};

export const CLOSED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-closed-refunds",
  title: "Backfill refund audit trail",
  description: "Add historical audit entries so the reconciliation report has a full retroactive picture.",
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [
    planLineExecution("plan", { id: "5", state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: LAST_WEEK }),
    planLineExecution("implement", {
      id: "6",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      run: { id: "run-implement-3", appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: LAST_WEEK,
    }),
    planLineExecution("verify", {
      id: "7",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      run: { id: "run-verify-3", appId: "app-refund-verifier", appName: "Refund Verifier" },
      updatedAt: YESTERDAY,
    }),
  ],
};

const REFUND_LINE = { id: "line-plan-and-implement", name: "plan-and-implement" };

interface StepExecutionEventFixture {
  order: FactoriesWorkOrder;
  stepName: string;
  at: string;
  runId: string;
  appId: string;
  result?: "passed" | "failed";
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

function stepExecutionFinishedEvent(input: StepExecutionEventFixture & { result: "passed" | "failed" }): FactoriesWorkOrderEvent {
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

export const OPEN_WORK_ORDER_EVENTS: FactoriesWorkOrderEvent[] = [openedWorkOrderEvent(OPEN_WORK_ORDER, HOUR_AGO)];

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

export const DEFAULT_WORK_ORDERS: FactoriesWorkOrder[] = [
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  RUNNING_WORK_ORDER,
  FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
];

export interface FactoriesFixture {
  organizationId: string;
  factories: FactoriesFactory[];
  workOrdersByFactoryId: Record<string, FactoriesWorkOrder[]>;
  appsByFactoryId: Record<string, FactoryApp[]>;
}

export const defaultFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [REFUND_FACTORY, EMPTY_FACTORY],
  workOrdersByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_WORK_ORDERS,
    [EMPTY_FACTORY_ID]: [],
  },
  appsByFactoryId: {
    [PRIMARY_FACTORY_ID]: REFUND_FACTORY_APPS,
    [EMPTY_FACTORY_ID]: [],
  },
};

export const emptyFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [],
  workOrdersByFactoryId: {},
  appsByFactoryId: {},
};

/** Same shape as {@link defaultFactoriesFixture} but with only closed orders in the primary factory. */
export const emptyWorkOrdersFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  workOrdersByFactoryId: {
    [PRIMARY_FACTORY_ID]: [CLOSED_WORK_ORDER],
    [EMPTY_FACTORY_ID]: [],
  },
};
