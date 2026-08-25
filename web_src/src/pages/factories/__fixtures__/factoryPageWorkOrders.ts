import type { FactoriesWorkOrder } from "@/api-client";

import { planLineDispatch, planLineExecution } from "./factoryPagePlanLine";
import {
  ARNOLD_USER,
  HOUR_AGO,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_PASSED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
  YESTERDAY,
  minutesAgo,
} from "./factoryPageIds";

export const INGEST_CREATED_BY = {
  automation: { appId: "app-refund-backlog", appName: "Ingest", nodeName: "On Issue Label" },
} as const;

export const SENTRY_CREATED_BY = {
  automation: { appId: "app-refund-sentry", appName: "Sentry", nodeName: "On Issue" },
} as const;

export const SLACK_CREATED_BY = {
  automation: { appId: "app-refund-slack", appName: "Slack", nodeName: "On Mention" },
} as const;

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
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
  // A watcher automation announcing why the order needs attention — the detail
  // page renders it as the "next step" panel above the checks.
  statusNotes: [
    {
      key: "pr-closure",
      kind: "info",
      headline: "Listening for user review",
      body: "This automation finished and opened [PR #6812](https://github.com/superplanehq/superplane/pull/6812). Tag `@superplaneagent` in comment to request changes. Task will automatically close when the pull request is closed or merged.",
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
  createdBy: SENTRY_CREATED_BY,
  assignees: [{ id: ARNOLD_USER.id, name: ARNOLD_USER.name }],
  lineDispatches: [],
};

export const QUESTION_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-agent-question-refunds",
  number: "108",
  key: "RF-108",
  title: "Clarify retry policy for provider timeouts",
  description: [
    "The payment poller stops on the first provider timeout. Confirm whether it should fail closed or retry with backoff before the next dispatch.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: HOUR_AGO,
  createdBy: SLACK_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [],
  statusNotes: [
    {
      key: "agent-question",
      kind: "info",
      headline: "The agent has a question",
      body: "Should the poller fail closed on the first timeout, or retry with backoff?",
      automation: { appId: "app-refund-backlog", appName: "Ingest" },
      updatedAt: HOUR_AGO,
    },
  ],
};

export const APPROVAL_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-approval-refunds",
  number: "109",
  key: "RF-109",
  title: "Review the refund webhook schema change",
  description: [
    "The refund webhook payload dropped the `event_version` field. Restore it on the schema and keep older clients working.",
    "",
    "- Add `event_version` back to the webhook schema.",
    "- Accept a missing version as `1` so current senders keep working.",
    "- Add a contract test for the v1 and v2 payloads.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: HOUR_AGO,
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: ARNOLD_USER.id, name: ARNOLD_USER.name }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("implement", {
        id: "approval-impl",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Implementation" },
        updatedAt: HOUR_AGO,
      }),
    ]),
  ],
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
    "- Run the test in the `verify` step of the line.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("implement", {
        id: "2",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Implementation" },
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
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: ARNOLD_USER.id, name: ARNOLD_USER.name }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("implement", {
        id: "4",
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        run: { id: LINE_RUN_IMPLEMENT_FAILED_ID, appId: "app-refund-implementer", appName: "Implementation" },
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
    "**Describe the request:**",
    "Let a user add emoji reactions on a work order itself (not only on comments).",
    "",
    "___",
    "",
    "**Describe your use-case:**",
    "There is no reaction UI on a work order. People need a quick signal on the order (acknowledge, +1) without leaving a comment.",
    "",
    "___",
    "",
    "**Describe functionality:**",
    "- React to an existing work order with an emoji.",
    "- Show reactions on the work order details page.",
    "- A user can add or remove their own reaction.",
  ].join("\n"),
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [],
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
  createdBy: INGEST_CREATED_BY,
  assignees: [],
  lineDispatches: [],
};

export const SENTRY_DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-sentry-refund-amount",
  number: "68",
  key: "RF-68",
  title: "TypeError: Cannot read properties of undefined (reading 'amount')",
  description: [
    "Sentry opened this issue after checkout failed to read `amount` on a refund payload.",
    "",
    "### What happens",
    "",
    "The refund worker throws when the provider omits `amount` on a retry callback. The request then returns `HTTP 500`.",
    "",
    "### Expected",
    "",
    "Treat a missing amount as a validation error. Do not fail the worker.",
  ].join("\n"),
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: YESTERDAY,
  createdBy: SENTRY_CREATED_BY,
  assignees: [],
  lineDispatches: [],
};

export const SLACK_DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-slack-missing-email",
  number: "64",
  key: "RF-64",
  title: "Customer reported a missing refund email",
  description: [
    "Support mentioned the SuperPlane agent in Slack. A customer did not receive a refund confirmation email after a successful refund.",
    "",
    "### Context",
    "",
    "Channel: #refunds-support",
    "Message: The customer completed a refund yesterday. The ledger shows the refund. The confirmation email is missing.",
    "",
    "### Expected",
    "",
    "Send the confirmation email after the provider confirms the refund.",
  ].join("\n"),
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: TWO_HOURS_AGO,
  updatedAt: TWO_HOURS_AGO,
  createdBy: SLACK_CREATED_BY,
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
  createdBy: SENTRY_CREATED_BY,
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
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("implement", {
        id: "6",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: LINE_RUN_IMPLEMENT_PASSED_ID, appId: "app-refund-implementer", appName: "Implementation" },
        updatedAt: LAST_WEEK,
        totalTokens: "12000",
        costCents: "480",
      }),
      planLineExecution("verify", {
        id: "7",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: LINE_RUN_VERIFY_PASSED_ID, appId: "app-refund-verifier", appName: "Risk Assessment" },
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
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    planLineDispatch([
      planLineExecution("implement", {
        id: "pr-impl",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: "run-pr-closure-implement", appId: "app-refund-implementer", appName: "Implementation" },
        updatedAt: LAST_WEEK,
        totalTokens: "5400",
        costCents: "180",
      }),
      planLineExecution("verify", {
        id: "pr-verify",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        run: { id: "run-pr-closure-verify", appId: "app-refund-verifier", appName: "Risk Assessment" },
        updatedAt: YESTERDAY,
        totalTokens: "700",
        costCents: "16",
      }),
    ]),
  ],
  totalTokens: "7000",
  totalCostCents: "218",
};
