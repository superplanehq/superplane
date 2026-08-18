import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderResult,
} from "@/api-client";

import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  STORYBOOK_ME_USER_ID,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";

/**
 * Backing work orders for every row the Overview redesign mock shows, so
 * clicking a row opens a real work order detail page instead of bouncing
 * back to the Work Orders list (the detail route redirects when it cannot
 * resolve the order number). Each order carries a description and an
 * activity timeline that match its overview state, so click-throughs land
 * on a believable detail page.
 *
 * Titles and numbers mirror `overviewRedesignMocks.ts`; keep the two in sync.
 */

// Reuse the real fixture line id so timeline links resolve; display names
// come from the overview rows ("Backend", "Frontend", "Maintenance").
const LINE_ID = "line-plan-and-implement";

function hoursAgo(hours: number): string {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function overviewOrder(
  number: string,
  title: string,
  description: string[],
  options: {
    state: "STATE_OPEN" | "STATE_CLOSED" | "STATE_DRAFT";
    result?: FactoriesWorkOrderResult;
    ageHours: number;
  },
): FactoriesWorkOrder {
  return {
    id: `wo-overview-${number}`,
    number,
    key: `${PRIMARY_FACTORY_KEY}-${number}`,
    title,
    description: description.join("\n"),
    state: options.state,
    result: options.result ?? "RESULT_UNSPECIFIED",
    createdAt: hoursAgo(options.ageHours + 24),
    updatedAt: hoursAgo(options.ageHours),
    assignees: [],
    executions: [],
  };
}

/* --------------------------- Event builders --------------------------- */

function opened(order: FactoriesWorkOrder): FactoriesWorkOrderEvent {
  return {
    type: "order.status.updated",
    timestamp: order.createdAt,
    event: {
      user: { id: STORYBOOK_ME_USER_ID },
      order: { id: order.id, title: order.title },
      fromState: "draft",
      toState: "open",
    },
  };
}

interface StepEventSpec {
  line: string;
  stepName: string;
  atHours: number;
  runId: string;
  result?: "passed" | "failed";
}

function stepEvent(order: FactoriesWorkOrder, spec: StepEventSpec): FactoriesWorkOrderEvent {
  const finished = spec.result !== undefined;
  return {
    type: finished ? "step.execution.finished" : "step.execution.created",
    timestamp: hoursAgo(spec.atHours),
    event: {
      stepName: spec.stepName,
      order: { id: order.id, title: order.title },
      line: { id: LINE_ID, name: spec.line },
      app: { id: "app-refund-implementer" },
      run: finished ? { id: spec.runId, state: "finished", result: spec.result } : { id: spec.runId, state: "pending" },
    },
  };
}

function agentComment(
  order: FactoriesWorkOrder,
  line: string,
  stepName: string,
  atHours: number,
  body: string,
): FactoriesWorkOrderEvent {
  return {
    type: "order.comment.added",
    timestamp: hoursAgo(atHours),
    event: {
      order: { id: order.id, title: order.title },
      body,
      author: {
        kind: "automation",
        automation: { nodeName: stepName, appName: "Refund Implementer", lineId: LINE_ID, lineName: line, stepName },
      },
    },
  };
}

function prArtifact(order: FactoriesWorkOrder, atHours: number, pr: { number: number; title: string; repo: string }) {
  const artifact: FactoriesWorkOrderArtifact = {
    id: `art-overview-${order.number}`,
    type: "TYPE_PR",
    data: { url: `https://github.com/${pr.repo}/pull/${pr.number}`, title: pr.title, number: pr.number },
    createdBy: { id: STORYBOOK_ME_USER_ID, name: "Storybook User" },
    createdAt: hoursAgo(atHours),
  };
  const event: FactoriesWorkOrderEvent = {
    type: "order.artifact.added",
    timestamp: hoursAgo(atHours),
    event: {
      user: { id: STORYBOOK_ME_USER_ID },
      order: { id: order.id, title: order.title },
      artifact: {
        id: artifact.id,
        type: "pr",
        data: artifact.data as Record<string, unknown>,
      },
    },
  };
  return { artifact, event };
}

function closed(
  order: FactoriesWorkOrder,
  atHours: number,
  result: "completed" | "failed",
  byAutomation?: { line: string; stepName: string },
): FactoriesWorkOrderEvent {
  return {
    type: "order.status.updated",
    timestamp: hoursAgo(atHours),
    event: {
      ...(byAutomation
        ? {
            automation: {
              nodeName: byAutomation.stepName,
              appName: "Refund Implementer",
              lineId: LINE_ID,
              lineName: byAutomation.line,
              stepName: byAutomation.stepName,
            },
          }
        : { user: { id: STORYBOOK_ME_USER_ID } }),
      order: { id: order.id, title: order.title },
      fromState: "open",
      toState: "closed",
      toResult: result,
    },
  };
}

/* ------------------------------- Orders ------------------------------- */

interface OverviewOrderFixture {
  order: FactoriesWorkOrder;
  events: FactoriesWorkOrderEvent[];
  artifacts?: FactoriesWorkOrderArtifact[];
}

function attentionOrderFixtures(): OverviewOrderFixture[] {
  const migrateWebhooks = overviewOrder(
    "61",
    "Migrate refund webhooks to the new event schema",
    [
      "Provider webhooks still send the v1 payload, so retries lose the idempotency key. Migrate refund webhooks to the new event schema end to end.",
      "",
      "The plan covers the schema mapping, a dual-write window, and a cutover checklist for each provider.",
    ],
    { state: "STATE_OPEN", ageHours: 4 },
  );
  const retryLimits = overviewOrder(
    "58",
    "Add retry limits to the payment poller",
    [
      "The payment poller retries forever when the provider is down, which floods the queue and hides real failures. Add a retry limit with exponential backoff and a dead-letter path.",
    ],
    { state: "STATE_OPEN", ageHours: 1 },
  );
  const flakyE2e = overviewOrder(
    "54",
    "Fix flaky checkout E2E test on slow networks",
    [
      "The checkout E2E test fails on slow networks because the confirmation poll times out after 5 seconds. Raise the timeout and stub the rate-limited status endpoint so CI stays green.",
    ],
    { state: "STATE_OPEN", ageHours: 1 },
  );

  return [
    {
      order: migrateWebhooks,
      events: [
        opened(migrateWebhooks),
        stepEvent(migrateWebhooks, { line: "Backend", stepName: "Plan", atHours: 6, runId: "run-ov-61-plan" }),
        stepEvent(migrateWebhooks, {
          line: "Backend",
          stepName: "Plan",
          atHours: 5,
          runId: "run-ov-61-plan",
          result: "passed",
        }),
        agentComment(
          migrateWebhooks,
          "Backend",
          "Plan review",
          4,
          "The migration plan is ready for review. Approve it to start the build step.",
        ),
      ],
    },
    {
      order: retryLimits,
      events: [
        opened(retryLimits),
        stepEvent(retryLimits, { line: "Backend", stepName: "Build", atHours: 2, runId: "run-ov-58-build" }),
        agentComment(
          retryLimits,
          "Backend",
          "Build",
          1,
          "Should exhausted retries land in the dead-letter queue, or fail the run? The requirements do not say.",
        ),
      ],
    },
    {
      order: flakyE2e,
      events: [
        opened(flakyE2e),
        stepEvent(flakyE2e, { line: "Frontend", stepName: "CI check", atHours: 2, runId: "run-ov-54-ci" }),
        stepEvent(flakyE2e, {
          line: "Frontend",
          stepName: "CI check",
          atHours: 1,
          runId: "run-ov-54-ci",
          result: "failed",
        }),
      ],
    },
  ];
}

function inFlightOrderFixtures(): OverviewOrderFixture[] {
  const auditLog = overviewOrder(
    "63",
    "Add audit log entries for refund overrides",
    [
      "Manual refund overrides leave no trace today. Write an audit log entry with the actor, amount, and reason for every override so compliance can review them.",
    ],
    { state: "STATE_OPEN", ageHours: 1 },
  );
  const goToolchain = overviewOrder(
    "62",
    "Bump Go toolchain and fix deprecations",
    [
      "Bump the Go toolchain to the version pinned in `go.mod` and fix the deprecation warnings it surfaces, mostly in the HTTP client wrappers.",
    ],
    { state: "STATE_OPEN", ageHours: 1 },
  );
  const csvExport = overviewOrder(
    "60",
    "Support CSV export on the disputes table",
    [
      "Finance exports disputes to a spreadsheet by hand every week. Add a CSV export to the disputes table with the current filters applied.",
    ],
    { state: "STATE_OPEN", ageHours: 2 },
  );
  const exchangeRates = overviewOrder(
    "59",
    "Cache exchange rates for refund conversions",
    [
      "Every refund conversion calls the exchange-rate API, which is rate limited and slow. Cache rates for an hour and refresh them in the background.",
    ],
    { state: "STATE_OPEN", ageHours: 1 },
  );

  return [
    {
      order: auditLog,
      events: [
        opened(auditLog),
        stepEvent(auditLog, {
          line: "Backend",
          stepName: "Plan",
          atHours: 2,
          runId: "run-ov-63-plan",
          result: "passed",
        }),
        stepEvent(auditLog, { line: "Backend", stepName: "Build", atHours: 1, runId: "run-ov-63-build" }),
      ],
    },
    {
      order: goToolchain,
      events: [
        opened(goToolchain),
        stepEvent(goToolchain, {
          line: "Maintenance",
          stepName: "Build",
          atHours: 2,
          runId: "run-ov-62-build",
          result: "passed",
        }),
        stepEvent(goToolchain, { line: "Maintenance", stepName: "CI check", atHours: 1, runId: "run-ov-62-ci" }),
      ],
    },
    {
      order: csvExport,
      events: [
        opened(csvExport),
        stepEvent(csvExport, {
          line: "Frontend",
          stepName: "Build",
          atHours: 3,
          runId: "run-ov-60-build",
          result: "passed",
        }),
        stepEvent(csvExport, { line: "Frontend", stepName: "Review", atHours: 2, runId: "run-ov-60-review" }),
      ],
    },
    {
      order: exchangeRates,
      events: [
        opened(exchangeRates),
        stepEvent(exchangeRates, { line: "Backend", stepName: "Plan", atHours: 1, runId: "run-ov-59-plan" }),
      ],
    },
  ];
}

function shippedOrderFixtures(): OverviewOrderFixture[] {
  const expiredTokens = overviewOrder(
    "57",
    "Return clear errors for expired refund tokens",
    [
      "Expired refund tokens returned a generic 400, so users retried a request that could never succeed. Return a specific error code and a message that names the fix.",
    ],
    { state: "STATE_CLOSED", result: "RESULT_COMPLETED", ageHours: 2 },
  );
  const pagination = overviewOrder(
    "56",
    "Add pagination to the refunds list endpoint",
    [
      "The refunds list endpoint returns every row and times out for large accounts. Add cursor pagination with a default page size of 50.",
    ],
    { state: "STATE_OPEN", ageHours: 5 },
  );
  const dedupeEmails = overviewOrder(
    "53",
    "Dedupe customer notification emails",
    [
      "Customers received one email per retry attempt during the March incident. Dedupe notifications by refund id within a 24-hour window.",
    ],
    { state: "STATE_CLOSED", result: "RESULT_COMPLETED", ageHours: 24 },
  );
  const reconciliationJob = overviewOrder(
    "51",
    "Rewrite the ledger reconciliation job",
    [
      "Rewrite the nightly reconciliation job to stream ledger rows instead of loading them into memory. The current job exceeds its memory limit on month-end volumes.",
      "",
      "Closed as unsuccessful after three attempts: the streaming approach needs the ledger index migration to land first.",
    ],
    { state: "STATE_CLOSED", result: "RESULT_FAILED", ageHours: 48 },
  );
  const webhookLatency = overviewOrder(
    "49",
    "Log webhook delivery latency per provider",
    [
      "We cannot tell which payment provider delays webhook delivery. Log delivery latency per provider and expose it as a histogram for the on-call dashboard.",
    ],
    { state: "STATE_CLOSED", result: "RESULT_COMPLETED", ageHours: 72 },
  );

  const expiredTokensPr = prArtifact(expiredTokens, 3, {
    number: 482,
    title: "Return clear errors for expired refund tokens",
    repo: "superplane/superplane",
  });
  const paginationPr = prArtifact(pagination, 5, {
    number: 479,
    title: "Add cursor pagination to the refunds list endpoint",
    repo: "superplane/superplane",
  });
  const dedupeEmailsPr = prArtifact(dedupeEmails, 25, {
    number: 474,
    title: "Dedupe customer notification emails",
    repo: "superplane/notifications",
  });
  const webhookLatencyPr = prArtifact(webhookLatency, 73, {
    number: 468,
    title: "Log webhook delivery latency per provider",
    repo: "superplane/superplane",
  });

  return [
    {
      order: expiredTokens,
      events: [opened(expiredTokens), expiredTokensPr.event, closed(expiredTokens, 2, "completed")],
      artifacts: [expiredTokensPr.artifact],
    },
    {
      order: pagination,
      events: [
        opened(pagination),
        stepEvent(pagination, { line: "Backend", stepName: "Review", atHours: 6, runId: "run-ov-56-review" }),
        paginationPr.event,
      ],
      artifacts: [paginationPr.artifact],
    },
    {
      order: dedupeEmails,
      events: [opened(dedupeEmails), dedupeEmailsPr.event, closed(dedupeEmails, 24, "completed")],
      artifacts: [dedupeEmailsPr.artifact],
    },
    {
      order: reconciliationJob,
      events: [
        opened(reconciliationJob),
        stepEvent(reconciliationJob, { line: "Backend", stepName: "Build", atHours: 50, runId: "run-ov-51-build" }),
        stepEvent(reconciliationJob, {
          line: "Backend",
          stepName: "Build",
          atHours: 49,
          runId: "run-ov-51-build",
          result: "failed",
        }),
        closed(reconciliationJob, 48, "failed", { line: "Backend", stepName: "Build" }),
      ],
    },
    {
      order: webhookLatency,
      events: [opened(webhookLatency), webhookLatencyPr.event, closed(webhookLatency, 72, "completed")],
      artifacts: [webhookLatencyPr.artifact],
    },
  ];
}

const OVERVIEW_ORDER_FIXTURES = [...attentionOrderFixtures(), ...inFlightOrderFixtures(), ...shippedOrderFixtures()];

/** Default fixture plus a backing order (with activity) for every Overview redesign row. */
export const overviewRedesignFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  workOrdersByFactoryId: {
    ...defaultFactoriesFixture.workOrdersByFactoryId,
    [PRIMARY_FACTORY_ID]: [
      ...OVERVIEW_ORDER_FIXTURES.map((entry) => entry.order),
      ...(defaultFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? []),
    ],
  },
  eventsByOrderId: Object.fromEntries(OVERVIEW_ORDER_FIXTURES.map((entry) => [entry.order.id!, entry.events])),
  artifactsByOrderId: Object.fromEntries(
    OVERVIEW_ORDER_FIXTURES.filter((entry) => entry.artifacts).map((entry) => [entry.order.id!, entry.artifacts!]),
  ),
};
