import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesLineMetrics,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderExecution,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";
import { canvasAppIds } from "@/pages/app/__fixtures__/handlers";

/** Shared with the home fixture so routes stay in sync across HomePage → Factories navigation. */
export const FACTORIES_ORGANIZATION_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";

export const PRIMARY_FACTORY_ID = "factory-refunds";
export const EMPTY_FACTORY_ID = "factory-payments";

/** Workspace key for `PRIMARY_FACTORY_ID` — routes use this, not the raw id. */
export const PRIMARY_FACTORY_KEY = "RF";
/** Workspace key for `EMPTY_FACTORY_ID` — routes use this, not the raw id. */
export const EMPTY_FACTORY_KEY = "PF";

export const STORYBOOK_ME_USER_ID = "storybook-user";
export const STORYBOOK_ME_USER_NAME = "Storybook User";
export const STORYBOOK_ME_USER_EMAIL = "storybook@superplane.dev";

// Relative timestamps so `formatTimeAgo` stays stable across story loads.
const NOW_MS = Date.now();
const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;
const relativeIso = (offsetMs: number) => new Date(NOW_MS - offsetMs).toISOString();

export const HOUR_AGO = relativeIso(HOUR_MS);
export const TWO_HOURS_AGO = relativeIso(2 * HOUR_MS);
export const YESTERDAY = relativeIso(DAY_MS);
export const LAST_WEEK = relativeIso(7 * DAY_MS);

/**
 * Canvas run ids on Line phase cards. Must be UUIDs: AppPage strips any
 * `?run=` value that fails `isValidRunId`, which drops factory run autolayout.
 *
 * `LINE_RUN_IMPLEMENT_FAILED_ID` matches the captured Software Factory
 * published run so Storybook reuses that run's executions and root event.
 */
export const LINE_RUN_IMPLEMENT_ID = "8f3a1c2e-4b5d-46f0-a789-0b1c2d3e4f50";
export const LINE_RUN_IMPLEMENT_FAILED_ID = canvasAppIds.publishedRunId ?? "fef4cee8-fdd7-47af-b5da-e739664cd31d";
export const LINE_RUN_IMPLEMENT_PASSED_ID = "9a4b2d3f-5c6e-47f0-b890-1c2d3e4f5061";
export const LINE_RUN_VERIFY_PASSED_ID = "0b5c3e4a-6d7f-4081-8901-2d3e4f506172";
export const LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID =
  canvasAppIds.rootEventId ?? "755a4430-2481-43f6-94cb-089c331a5d2f";

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
  key: "RF",
  description:
    "Handles reconciliation work: plan a change, implement across affected services, and verify with regression suites.",
  lines: REFUND_FACTORY_LINES,
};

export const EMPTY_FACTORY: FactoriesFactory = {
  id: EMPTY_FACTORY_ID,
  name: "Payments Factory",
  key: "PF",
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
  number: "101",
  key: "RF-101",
  title: "Reconcile duplicate refunds in ledger",
  description:
    "Users are seeing duplicate refund entries after retries. Reconcile the ledger, patch retry logic, and add a regression test.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
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
  number: "102",
  key: "RF-102",
  title: "Add refund reason enum to schema",
  description: "Extend the refund ledger schema with a nullable reason enum so audit can categorize.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: TWO_HOURS_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [],
};

// Storybook user is co-assigned so "mine + running" surfaces this order.
export const RUNNING_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-running-refunds",
  number: "103",
  key: "RF-103",
  title: "Add refund reconciliation test",
  description: "Cover the newly discovered duplicate refund case with a regression test running in CI.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: REVIEWER_USER.id, name: REVIEWER_USER.name } },
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
      run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: HOUR_AGO,
    }),
  ],
};

// Storybook user is co-assigned so "mine + failed" surfaces this order.
export const FAILED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-failed-refunds",
  number: "104",
  key: "RF-104",
  title: "Ship idempotent refund retries",
  description: "Retry logic should be idempotent so replays don't create duplicate refunds.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [
    planLineExecution("plan", { id: "3", state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: TWO_HOURS_AGO }),
    planLineExecution("implement", {
      id: "4",
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      run: { id: LINE_RUN_IMPLEMENT_FAILED_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: HOUR_AGO,
    }),
  ],
};

export const DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-draft-refunds",
  number: "105",
  key: "RF-105",
  title: "Draft: rework refund telemetry",
  description: "Still scoping the metric shape and dashboards before we mark this ready.",
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [],
};

export const CLOSED_FAILED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-closed-failed-refunds",
  number: "92",
  key: "RF-92",
  title: "Failed: reconcile refund ledger for Q1 audit",
  description: "Line completed but validation flagged a mismatch; closed as failed for follow-up.",
  state: "STATE_CLOSED",
  result: "RESULT_FAILED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [],
};

export const CLOSED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-closed-refunds",
  number: "87",
  key: "RF-87",
  title: "Backfill refund audit trail",
  description: "Add historical audit entries so the reconciliation report has a full retroactive picture.",
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  executions: [
    planLineExecution("plan", { id: "5", state: "STATE_FINISHED", result: "RESULT_PASSED", updatedAt: LAST_WEEK }),
    planLineExecution("implement", {
      id: "6",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      run: { id: LINE_RUN_IMPLEMENT_PASSED_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
      updatedAt: LAST_WEEK,
    }),
    planLineExecution("verify", {
      id: "7",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      run: { id: LINE_RUN_VERIFY_PASSED_ID, appId: "app-refund-verifier", appName: "Refund Verifier" },
      updatedAt: YESTERDAY,
    }),
  ],
};

export const OPEN_WORK_ORDER_ARTIFACTS: FactoriesWorkOrderArtifact[] = [
  {
    id: "art-pr-1",
    type: "TYPE_PR",
    data: {
      url: "https://github.com/example/ledger/pull/482",
      title: "Fix duplicate refund on retry",
      number: 482,
    },
    createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
    createdAt: HOUR_AGO,
  },
  {
    id: "art-md-1",
    type: "TYPE_MARKDOWN",
    data: {
      title: "Investigation notes",
      body: "Retry policy exceeded idempotency window when the ledger writer was under load; details captured in the design doc.",
    },
    createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
    createdAt: HOUR_AGO,
  },
  {
    id: "art-branch-1",
    type: "TYPE_BRANCH",
    data: {
      name: "feature/refund-retry",
      url: "https://github.com/example/ledger/tree/feature/refund-retry",
    },
    createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
    createdAt: HOUR_AGO,
  },
];

export const DEFAULT_WORK_ORDERS: FactoriesWorkOrder[] = [
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  RUNNING_WORK_ORDER,
  FAILED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  CLOSED_WORK_ORDER,
  CLOSED_FAILED_WORK_ORDER,
];

/** Plan-and-implement has real throughput; hotfix is left out to exercise the "no data yet" dashes state. */
export const REFUND_LINE_PLAN_METRICS: FactoriesLineMetrics = {
  lineId: REFUND_LINE_PLAN_ID,
  successRatePct: 87,
  mergedCount: "20",
  totalClosedCount: "23",
  reworkPerWorkOrder: 1.3,
  costPerSuccessCents: "842",
  successTrendPct: [72, 75, 78, 80, 82, 84, 85, 86, 87, 87],
  successDeltaPts: 5,
  reworkDelta: -0.4,
  costDeltaCents: "-63",
  throughputPerDay: 0.67,
  throughputTrend: [1, 1, 2, 2, 2, 3, 2, 3, 2, 2],
};

export interface FactoriesFixture {
  organizationId: string;
  factories: FactoriesFactory[];
  workOrdersByFactoryId: Record<string, FactoriesWorkOrder[]>;
  appsByFactoryId: Record<string, FactoryApp[]>;
  lineMetricsByFactoryId: Record<string, FactoriesLineMetrics[]>;
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
  lineMetricsByFactoryId: {
    [PRIMARY_FACTORY_ID]: [REFUND_LINE_PLAN_METRICS],
    [EMPTY_FACTORY_ID]: [],
  },
};

export const emptyFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [],
  workOrdersByFactoryId: {},
  appsByFactoryId: {},
  lineMetricsByFactoryId: {},
};

/** Same shape as {@link defaultFactoriesFixture} but with only closed orders in the primary factory. */
export const emptyWorkOrdersFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  workOrdersByFactoryId: {
    [PRIMARY_FACTORY_ID]: [CLOSED_WORK_ORDER],
    [EMPTY_FACTORY_ID]: [],
  },
};

/** Story-only clone: Plan and Implement has five phases so the board can scroll on x. */
export const fiveStepLineFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  factories: defaultFactoriesFixture.factories.map((factory) => {
    if (factory.id !== PRIMARY_FACTORY_ID) {
      return factory;
    }
    return {
      ...factory,
      lines: (factory.lines ?? []).map((line) => {
        if (line.id !== REFUND_LINE_PLAN_ID) {
          return line;
        }
        return {
          ...line,
          steps: [
            ...(line.steps ?? []),
            runAppStep("release", "app-refund-verifier", "start-verification"),
            runAppStep("observe", "app-refund-planner", "start-plan"),
          ],
        };
      }),
    };
  }),
};
