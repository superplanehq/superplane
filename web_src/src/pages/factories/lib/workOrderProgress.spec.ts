import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrder } from "@/api-client";

import {
  countActiveWorkOrders,
  filterWorkOrdersByStatus,
  getWorkOrderDisplayKey,
  getWorkOrderDisplayStatus,
  groupWorkOrdersByLane,
  type WorkOrderStatusFilter,
} from "./workOrderProgress";

function order(overrides: Partial<FactoriesWorkOrder>): FactoriesWorkOrder {
  return {
    id: `wo-${overrides.state ?? "open"}-${overrides.result ?? "unspec"}`,
    title: "test order",
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    lineDispatches: [],
    ...overrides,
  };
}

/** One line dispatch with a single step execution — enough to drive
 * `hasActiveLineDispatch`/`getWorkOrderDisplayStatus` in these tests. */
function activeDispatch(): FactoriesWorkOrder["lineDispatches"] {
  return [
    {
      id: "d1",
      state: "STATE_ACTIVE",
      stepExecutions: [{ id: "e1", step: "s", state: "STATE_STARTED", result: "RESULT_UNKNOWN" }],
    },
  ];
}

function idsFilteredBy(orders: FactoriesWorkOrder[], statusFilter: WorkOrderStatusFilter) {
  return filterWorkOrdersByStatus(orders, statusFilter).map((o) => o.id);
}

describe("getWorkOrderDisplayStatus", () => {
  it("draft orders are draft", () => {
    expect(getWorkOrderDisplayStatus(order({ state: "STATE_DRAFT" }))).toBe("draft");
  });

  it("open orders with an active execution are running", () => {
    expect(
      getWorkOrderDisplayStatus(
        order({
          state: "STATE_OPEN",
          lineDispatches: activeDispatch(),
        }),
      ),
    ).toBe("running");
  });

  it("open orders without an active execution are waiting", () => {
    expect(getWorkOrderDisplayStatus(order({ state: "STATE_OPEN" }))).toBe("waiting");
    // A previously failed step no longer changes the pill — the row shows
    // the last line separately.
    expect(
      getWorkOrderDisplayStatus(
        order({
          state: "STATE_OPEN",
          lineDispatches: [
            {
              id: "d1",
              state: "STATE_FINISHED",
              result: "RESULT_FAILED",
              stepExecutions: [{ id: "e1", step: "s", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
            },
          ],
        }),
      ),
    ).toBe("waiting");
  });

  it.each([
    ["RESULT_COMPLETED", "completed"] as const,
    ["RESULT_FAILED", "failed"] as const,
    ["RESULT_REJECTED", "cancelled"] as const,
  ])("closed orders with %s map to %s", (result, expected) => {
    expect(getWorkOrderDisplayStatus(order({ state: "STATE_CLOSED", result }))).toBe(expected);
  });
});

describe("filterWorkOrdersByStatus", () => {
  const draft = order({ state: "STATE_DRAFT" });
  const waiting = order({ state: "STATE_OPEN" });
  const running = order({
    state: "STATE_OPEN",
    id: "wo-running",
    lineDispatches: activeDispatch(),
  });
  const closedCompleted = order({ state: "STATE_CLOSED", result: "RESULT_COMPLETED", id: "wo-completed" });
  const closedCancelled = order({ state: "STATE_CLOSED", result: "RESULT_REJECTED", id: "wo-cancelled" });
  const closedFailed = order({ state: "STATE_CLOSED", result: "RESULT_FAILED", id: "wo-failed" });

  const all = [draft, waiting, running, closedCompleted, closedCancelled, closedFailed];

  it("`active` returns every non-closed order", () => {
    expect(idsFilteredBy(all, "active")).toEqual([draft.id, waiting.id, running.id]);
  });

  it("`all` returns everything", () => {
    expect(idsFilteredBy(all, "all")).toEqual(all.map((o) => o.id));
  });

  it("specific display statuses match by that status only", () => {
    expect(idsFilteredBy(all, "draft")).toEqual([draft.id]);
    expect(idsFilteredBy(all, "waiting")).toEqual([waiting.id]);
    expect(idsFilteredBy(all, "running")).toEqual([running.id]);
    expect(idsFilteredBy(all, "completed")).toEqual([closedCompleted.id]);
    expect(idsFilteredBy(all, "failed")).toEqual([closedFailed.id]);
    expect(idsFilteredBy(all, "cancelled")).toEqual([closedCancelled.id]);
  });

  it("countActiveWorkOrders matches the size of the default `active` filter", () => {
    expect(countActiveWorkOrders(all)).toBe(filterWorkOrdersByStatus(all, "active").length);
    expect(countActiveWorkOrders(all)).toBe(3);
  });
});

describe("groupWorkOrdersByLane", () => {
  it("splits orders across the four board lanes", () => {
    const orders = [
      order({ id: "d", state: "STATE_DRAFT" }),
      order({ id: "w", state: "STATE_OPEN" }),
      order({
        id: "r",
        state: "STATE_OPEN",
        lineDispatches: activeDispatch(),
      }),
      order({ id: "c", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
      order({ id: "f", state: "STATE_CLOSED", result: "RESULT_FAILED" }),
      order({ id: "x", state: "STATE_CLOSED", result: "RESULT_REJECTED" }),
    ];

    const grouped = groupWorkOrdersByLane(orders);
    const byLane = Object.fromEntries(grouped.map((entry) => [entry.lane.id, entry.orders.map((o) => o.id)]));

    expect(byLane).toEqual({
      backlog: ["d"],
      running: ["r"],
      review: ["w"],
      done: ["c", "f", "x"],
    });
  });
});

describe("getWorkOrderDisplayKey", () => {
  it("prefers the server-provided key", () => {
    expect(getWorkOrderDisplayKey(order({ key: "SP-42" }))).toBe("SP-42");
  });

  it("composes the key from the factory key + number when the server did not send one", () => {
    expect(getWorkOrderDisplayKey(order({ id: "abcdef1234567890", number: "7" }), "SP")).toBe("SP-7");
  });

  it("falls back to a UUID prefix when nothing else is available", () => {
    expect(getWorkOrderDisplayKey(order({ id: "abcdef1234567890" }))).toBe("abcdef12");
  });
});
