import type { FactoriesWorkOrder, FactoriesWorkOrderResult } from "@/api-client";

import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";

/**
 * Backing work orders for every row the Overview redesign mock shows, so
 * clicking a row opens a real work order detail page instead of bouncing
 * back to the Work Orders list (the detail route redirects when it cannot
 * resolve the order number).
 */

function hoursAgo(hours: number): string {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

function overviewOrder(
  number: string,
  title: string,
  description: string,
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
    description,
    state: options.state,
    result: options.result ?? "RESULT_UNSPECIFIED",
    createdAt: hoursAgo(options.ageHours + 24),
    updatedAt: hoursAgo(options.ageHours),
    assignees: [],
    executions: [],
  };
}

/** Titles and numbers mirror `overviewRedesignMocks.ts`; keep the two in sync. */
const OVERVIEW_ORDERS: FactoriesWorkOrder[] = [
  // Needs attention
  overviewOrder("61", "Migrate refund webhooks to the new event schema", "Waits for plan approval.", {
    state: "STATE_OPEN",
    ageHours: 4,
  }),
  overviewOrder("58", "Add retry limits to the payment poller", "The agent asked a question during the build step.", {
    state: "STATE_OPEN",
    ageHours: 1,
  }),
  overviewOrder("54", "Fix flaky checkout E2E test on slow networks", "The CI check run failed.", {
    state: "STATE_OPEN",
    ageHours: 1,
  }),
  // In flight
  overviewOrder("63", "Add audit log entries for refund overrides", "Build step in progress.", {
    state: "STATE_OPEN",
    ageHours: 1,
  }),
  overviewOrder("62", "Bump Go toolchain and fix deprecations", "CI check in progress.", {
    state: "STATE_OPEN",
    ageHours: 1,
  }),
  overviewOrder("60", "Support CSV export on the disputes table", "Review step in progress.", {
    state: "STATE_OPEN",
    ageHours: 2,
  }),
  overviewOrder("59", "Cache exchange rates for refund conversions", "Plan step in progress.", {
    state: "STATE_OPEN",
    ageHours: 1,
  }),
  // Recently shipped
  overviewOrder("57", "Return clear errors for expired refund tokens", "PR #482 merged.", {
    state: "STATE_CLOSED",
    result: "RESULT_COMPLETED",
    ageHours: 2,
  }),
  overviewOrder("56", "Add pagination to the refunds list endpoint", "PR #479 is in review.", {
    state: "STATE_OPEN",
    ageHours: 5,
  }),
  overviewOrder("53", "Dedupe customer notification emails", "PR #474 merged.", {
    state: "STATE_CLOSED",
    result: "RESULT_COMPLETED",
    ageHours: 24,
  }),
  overviewOrder("51", "Rewrite the ledger reconciliation job", "Closed as unsuccessful after 3 attempts.", {
    state: "STATE_CLOSED",
    result: "RESULT_FAILED",
    ageHours: 48,
  }),
  overviewOrder("49", "Log webhook delivery latency per provider", "PR #468 merged.", {
    state: "STATE_CLOSED",
    result: "RESULT_COMPLETED",
    ageHours: 72,
  }),
];

/** Default fixture plus a backing order for every Overview redesign row. */
export const overviewRedesignFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  workOrdersByFactoryId: {
    ...defaultFactoriesFixture.workOrdersByFactoryId,
    [PRIMARY_FACTORY_ID]: [
      ...OVERVIEW_ORDERS,
      ...(defaultFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? []),
    ],
  },
};
