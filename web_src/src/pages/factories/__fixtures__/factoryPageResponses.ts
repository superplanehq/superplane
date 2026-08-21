import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  MeNotificationSettings,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";

import type { FactoriesWorkOrderCheck } from "@/api-client";
import { DEFAULT_FACTORY_USAGE, EMPTY_USAGE_REPORT, type StorybookUsageReport } from "./usageReportFixtures";
import { planLineDispatch, planLineExecution } from "./factoryPagePlanLine";
import {
  EMPTY_FACTORY_ID,
  FACTORIES_ORGANIZATION_ID,
  HOUR_AGO,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  OPERATOR_USER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
  YESTERDAY,
  minutesAgo,
} from "./factoryPageIds";

export * from "./factoryPageIds";

const RUN_APP_TYPE = "runApp";

function runAppStep(appId: string, entrypoint: string): FactoryLineStep {
  return {
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

export const REFUND_FACTORY_LINES: FactoriesFactoryLine[] = [
  {
    id: REFUND_LINE_PLAN_ID,
    name: "plan-and-implement",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [
      runAppStep("app-refund-planner", "start-plan"),
      runAppStep("app-refund-implementer", "start-implementation"),
      runAppStep("app-refund-verifier", "start-verification"),
    ],
  },
  {
    id: REFUND_LINE_HOTFIX_ID,
    name: "hotfix",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [runAppStep("app-refund-verifier", "start-verification")],
  },
];

export const REFUND_FACTORY: FactoriesFactory = {
  id: PRIMARY_FACTORY_ID,
  name: "Semaphore",
  key: "RF",
  description:
    "Handles reconciliation work: plan a change, implement across affected services, and verify with regression suites.",
  lines: REFUND_FACTORY_LINES,
};

export const EMPTY_FACTORY: FactoriesFactory = {
  id: EMPTY_FACTORY_ID,
  name: "SuperPlane",
  key: "PF",
  description: "New factory. No lines or work orders configured yet.",
  lines: [],
};

export const OPEN_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-open-refunds",
  number: "101",
  key: "RF-101",
  title: "Reconcile duplicate refunds in ledger",
  description: [
    "Users see duplicate refund entries in the ledger after payment retries. Support reported 14 affected accounts this week, and finance cannot close the monthly report until the totals match.",
    "",
    "### Scope",
    "",
    "- Reconcile the ledger and remove duplicate entries created by retries.",
    "- Patch the retry logic in `ledger-writer` so replays reuse the original idempotency key.",
    "- Add a regression test that replays a retry burst against a seeded ledger.",
    "",
    "### Out of scope",
    "",
    "- Migrating the ledger to the new event schema (tracked separately).",
    "- Changes to the refund approval flow.",
    "",
    "First repro: retry a refund three times with the writer under load — the second and third attempts land outside the idempotency window and insert new rows.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
  // A watcher automation announcing why the order needs attention — the detail
  // page renders it as the "next step" panel above the checks.
  statusNotes: [
    {
      key: "pr-closure",
      kind: "info",
      headline: "Review the pull request",
      body: "The Refund Processing line opened [PR #6812](https://github.com/superplanehq/superplane/pull/6812). When it merges, this work order completes automatically. If it closes without a merge, the work order is rejected.",
      ctaLabel: "Review PR #6812",
      ctaUrl: "https://github.com/superplanehq/superplane/pull/6812",
      automation: { appId: "app-refund-verifier", appName: "PR Closure" },
      updatedAt: minutesAgo(25),
    },
  ],
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
  description: [
    "Audit cannot categorize refunds because the ledger stores the reason as free text. Extend the schema with a nullable `reason` enum so reports can group refunds by cause.",
    "",
    "Proposed values: `duplicate_charge`, `customer_request`, `fraud`, `service_outage`, `other`. Keep the column nullable so the backfill can run separately.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: TWO_HOURS_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
};

// Storybook user owns this order so "mine + running" surfaces it.
export const RUNNING_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-running-refunds",
  number: "103",
  key: "RF-103",
  title: "Add refund reconciliation test",
  description: [
    "The duplicate refund case found in RF-101 has no automated coverage. Add a regression test that runs in CI so the bug cannot return unnoticed.",
    "",
    "- Seed a ledger with one refund and replay the retry burst from the incident.",
    "- Assert the ledger contains exactly one entry per refund after reconciliation.",
    "- Run the test in the `verify` step of the plan-and-implement line.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: REVIEWER_USER.id, name: REVIEWER_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("plan", {
        id: "1",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: TWO_HOURS_AGO,
        totalTokens: "1800",
        costCents: "45",
      }),
      planLineExecution("implement", {
        id: "2",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
        updatedAt: HOUR_AGO,
        totalTokens: "900",
        costCents: "28",
      }),
    ]),
  ],
  totalTokens: "2700",
  totalCostCents: "73",
};

// Storybook user is co-assigned so "mine + failed" surfaces this order.
export const FAILED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-failed-refunds",
  number: "104",
  key: "RF-104",
  title: "Ship idempotent refund retries",
  description: [
    "Retry logic must be idempotent so replays do not create duplicate refunds. Today each retry attempt generates a new request id, which defeats the dedupe check downstream.",
    "",
    "Derive the idempotency key from the refund id plus the original attempt, and extend the dedupe window in `ledger-writer` to cover delayed replays from the queue.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("plan", {
        id: "3",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: TWO_HOURS_AGO,
        totalTokens: "2200",
        costCents: "55",
      }),
      planLineExecution("implement", {
        id: "4",
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        run: { id: LINE_RUN_IMPLEMENT_FAILED_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
        updatedAt: HOUR_AGO,
        totalTokens: "6400",
        costCents: "210",
      }),
    ]),
  ],
  totalTokens: "8600",
  totalCostCents: "265",
};

export const DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-draft-refunds",
  number: "105",
  key: "RF-105",
  title: "Draft: rework refund telemetry",
  description: [
    "Still scoping. The current refund metrics count events but hide latency, so we cannot tell whether reconciliation slows down under load.",
    "",
    "Open questions before this is ready:",
    "",
    "- Which percentiles do the dashboards need (p50/p95/p99)?",
    "- Do we tag metrics by provider, by line, or both?",
    "- Can we reuse the payment-poller histogram buckets?",
  ].join("\n"),
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
};

export const INGEST_DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-ingest-api-key",
  number: "72",
  key: "RF-72",
  title: "Duplicate API key name returns HTTP 500 instead of a validation/conflict error",
  description: [
    "Creating an API key with a name that already exists in the organization returns `HTTP 500 Internal Server Error` with a generic message, instead of a client-actionable validation or conflict response.",
    "",
    "### What happens",
    "",
    "`POST /api/v1/api-keys` with a name that collides with an existing key (the name is trimmed before creation, so surrounding whitespace still collides) returns:",
    "",
    "```",
    "HTTP/1.1 500 Internal Server Error",
    '{"code": ...,"message":"failed to create API key","details":[]}',
    "```",
    "",
    "The database correctly rejects the duplicate:",
    "",
    "```",
    'ERROR: duplicate key value violates unique constraint "unique_api_key_in_organization" (SQLSTATE 23505)',
    "```",
    "",
    "No second key is created, so the data stays consistent, but a foreseeable user input (reusing a name) surfaces as a server error with no indication of the real cause.",
    "",
    "### Expected",
    "",
    "A duplicate name should return a client-visible conflict or validation error explaining that the name is already in use, rather than a generic `500`.",
  ].join("\n"),
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: TWO_HOURS_AGO,
  createdBy: {
    automation: {
      appId: "app-refund-backlog",
      appName: "Ingest",
      nodeName: "On Issue Label",
    },
  },
  assignees: [],
  lineDispatches: [],
};

export const CLOSED_FAILED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-closed-failed-refunds",
  number: "92",
  key: "RF-92",
  title: "Failed: reconcile refund ledger for Q1 audit",
  description: [
    "Reconcile the refund ledger for the Q1 audit. The line completed, but validation flagged a $412.66 mismatch between the ledger and the payment provider export.",
    "",
    "Closed as failed. Follow-up: trace the mismatch to its source transactions before the audit deadline — see the reconciliation report artifact for the affected date range.",
  ].join("\n"),
  state: "STATE_CLOSED",
  result: "RESULT_FAILED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
};

export const CLOSED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-closed-refunds",
  number: "87",
  key: "RF-87",
  title: "Backfill refund audit trail",
  description: [
    "The reconciliation report needs a full retroactive picture, but refunds processed before March have no audit entries.",
    "",
    "Backfill historical audit entries from the provider exports, then verify row counts against the ledger for each month. The backfill ran in batches of 10k rows to keep replication lag flat.",
  ].join("\n"),
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("plan", {
        id: "5",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: LAST_WEEK,
        totalTokens: "1500",
        costCents: "40",
      }),
      planLineExecution("implement", {
        id: "6",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: LINE_RUN_IMPLEMENT_PASSED_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
        updatedAt: LAST_WEEK,
        totalTokens: "12000",
        costCents: "480",
      }),
      planLineExecution("verify", {
        id: "7",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: LINE_RUN_VERIFY_PASSED_ID, appId: "app-refund-verifier", appName: "Refund Verifier" },
        updatedAt: YESTERDAY,
        totalTokens: "800",
        costCents: "18",
      }),
    ]),
  ],
  totalTokens: "14300",
  totalCostCents: "538",
};

export const PR_CLOSURE_COMPLETED_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-pr-closure-receipts",
  number: "88",
  key: "RF-88",
  title: "Send refund receipts after provider confirm",
  description: [
    "Customers do not receive a receipt after a refund confirms at the provider.",
    "",
    "Send the receipt when the provider webhook reports success. PR Closure completes the work order after the pull request merges.",
  ].join("\n"),
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("plan", {
        id: "pr-plan",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: LAST_WEEK,
        totalTokens: "900",
        costCents: "22",
      }),
      planLineExecution("implement", {
        id: "pr-impl",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: "run-pr-closure-implement", appId: "app-refund-implementer", appName: "Refund Implementer" },
        updatedAt: LAST_WEEK,
        totalTokens: "5400",
        costCents: "180",
      }),
      planLineExecution("verify", {
        id: "pr-verify",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: "run-pr-closure-verify", appId: "app-refund-verifier", appName: "Refund Verifier" },
        updatedAt: YESTERDAY,
        totalTokens: "700",
        costCents: "16",
      }),
    ]),
  ],
  totalTokens: "7000",
  totalCostCents: "218",
};

export const DEFAULT_WORK_ORDERS: FactoriesWorkOrder[] = [
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  RUNNING_WORK_ORDER,
  FAILED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  CLOSED_WORK_ORDER,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  CLOSED_FAILED_WORK_ORDER,
];

export interface FactoriesFixture {
  organizationId: string;
  factories: FactoriesFactory[];
  workOrdersByFactoryId: Record<string, FactoriesWorkOrder[]>;
  appsByFactoryId: Record<string, FactoryApp[]>;
  usageByFactoryId?: Record<string, StorybookUsageReport>;
  organizationLlmSpend?: StorybookUsageReport;
  /** Per-user notification settings backing `/api/v1/me/notification-settings`. */
  notificationSettings?: MeNotificationSettings;
  /**
   * Per-order activity timelines. When an order id is absent, the handlers
   * fall back to `DEFAULT_EVENTS_BY_ORDER_ID` from `factoryPageEventFixtures`.
   */
  eventsByOrderId?: Record<string, FactoriesWorkOrderEvent[]>;
  /** Per-order artifacts; same fallback pattern as `eventsByOrderId`. */
  artifactsByOrderId?: Record<string, FactoriesWorkOrderArtifact[]>;
  /** Per-order checks (automation-reported scores); same fallback pattern as `eventsByOrderId`. */
  checksByOrderId?: Record<string, FactoriesWorkOrderCheck[]>;
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
  usageByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_USAGE,
    [EMPTY_FACTORY_ID]: EMPTY_USAGE_REPORT,
  },
  organizationLlmSpend: DEFAULT_FACTORY_USAGE,
};
