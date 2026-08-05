import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrder } from "@/api-client";

import {
  countActiveWorkOrders,
  filterWorkOrdersByStatus,
  getWorkOrderDisplayStatus,
  type WorkOrderStatusFilter,
} from "./workOrderProgress";

function order(overrides: Partial<FactoriesWorkOrder>): FactoriesWorkOrder {
  return {
    id: `wo-${overrides.state ?? "open"}-${overrides.result ?? "unspec"}`,
    title: "test order",
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    executions: [],
    ...overrides,
  };
}

function idsFilteredBy(orders: FactoriesWorkOrder[], statusFilter: WorkOrderStatusFilter) {
  return filterWorkOrdersByStatus(orders, statusFilter).map((o) => o.id);
}

describe("filterWorkOrdersByStatus", () => {
  const draft = order({ state: "STATE_DRAFT" });
  const open = order({ state: "STATE_OPEN" });
  const running = order({
    state: "STATE_OPEN",
    id: "wo-running",
    executions: [{ id: "e1", step: "s", state: "STATE_STARTED", result: "RESULT_UNKNOWN" }],
  });
  const failed = order({
    state: "STATE_OPEN",
    id: "wo-failed",
    executions: [{ id: "e1", step: "s", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
  });
  const closedCompleted = order({ state: "STATE_CLOSED", result: "RESULT_COMPLETED", id: "wo-completed" });
  const closedRejected = order({ state: "STATE_CLOSED", result: "RESULT_REJECTED", id: "wo-rejected" });
  const closedFailed = order({ state: "STATE_CLOSED", result: "RESULT_FAILED", id: "wo-closed-failed" });

  const all = [draft, open, running, failed, closedCompleted, closedRejected, closedFailed];

  it("`active` returns every non-closed order (draft/open/running/failed)", () => {
    expect(idsFilteredBy(all, "active")).toEqual([draft.id, open.id, running.id, failed.id]);
  });

  it("`failed` includes both in-flight failed and closed-as-failed orders", () => {
    expect(idsFilteredBy(all, "failed")).toEqual([failed.id, closedFailed.id]);
  });

  it("`all` returns everything", () => {
    expect(idsFilteredBy(all, "all")).toEqual(all.map((o) => o.id));
  });

  it("specific display statuses (draft/completed/rejected) match by that status only", () => {
    expect(idsFilteredBy(all, "draft")).toEqual([draft.id]);
    expect(idsFilteredBy(all, "completed")).toEqual([closedCompleted.id]);
    expect(idsFilteredBy(all, "rejected")).toEqual([closedRejected.id]);
  });

  it("countActiveWorkOrders matches the size of the default `active` filter", () => {
    expect(countActiveWorkOrders(all)).toBe(filterWorkOrdersByStatus(all, "active").length);
    expect(countActiveWorkOrders(all)).toBe(4);
  });
});

describe("getWorkOrderDisplayStatus", () => {
  it("returns `failed` for an open order whose latest attempt failed", () => {
    const failedAt = "2026-08-04T12:00:00.000Z";
    const created = "2026-08-04T11:00:00.000Z";
    const failedOpen: FactoriesWorkOrder = {
      id: "wo-failed-current",
      title: "current failure",
      state: "STATE_OPEN",
      result: "RESULT_UNSPECIFIED",
      createdAt: created,
      updatedAt: created,
      executions: [
        {
          id: "e1",
          step: "s",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          updatedAt: failedAt,
        },
      ],
    };
    expect(getWorkOrderDisplayStatus(failedOpen)).toBe("failed");
  });

  it("resets to `open` after a reopen (updatedAt bumped past the last failure)", () => {
    const failedAt = "2026-08-04T12:00:00.000Z";
    const reopenAt = "2026-08-04T13:00:00.000Z";
    const reopened: FactoriesWorkOrder = {
      id: "wo-reopened",
      title: "reopened after failure",
      state: "STATE_OPEN",
      result: "RESULT_UNSPECIFIED",
      createdAt: "2026-08-04T10:00:00.000Z",
      updatedAt: reopenAt,
      executions: [
        {
          id: "e1",
          step: "s",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          updatedAt: failedAt,
        },
      ],
    };
    expect(getWorkOrderDisplayStatus(reopened)).toBe("open");
  });

  it("clears the failed pill when a subsequent retry finishes successfully", () => {
    //
    // The latest finished execution passed, so it supersedes the older
    // failure regardless of the `updatedAt` fence, and the display
    // returns to "open".
    //
    const orderCreated = "2026-08-04T10:00:00.000Z";
    const failedAt = "2026-08-04T11:00:00.000Z";
    const passedAt = "2026-08-04T12:00:00.000Z";
    const passedAfterFail: FactoriesWorkOrder = {
      id: "wo-passed-after-fail",
      title: "retry succeeded",
      state: "STATE_OPEN",
      result: "RESULT_UNSPECIFIED",
      createdAt: orderCreated,
      updatedAt: orderCreated,
      executions: [
        {
          id: "e1",
          step: "s",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          updatedAt: failedAt,
        },
        {
          id: "e2",
          step: "s",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          updatedAt: passedAt,
        },
      ],
    };
    expect(getWorkOrderDisplayStatus(passedAfterFail)).toBe("open");
  });
});
