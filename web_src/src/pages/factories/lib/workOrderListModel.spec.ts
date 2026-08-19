import { describe, expect, it } from "vitest";

import type {
  FactoriesFactory,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";

import {
  EMPTY_WORK_ORDER_FILTERS,
  UNASSIGNED_FILTER_VALUE,
  applyWorkOrderFilters,
  applyWorkOrderOrdering,
  applyWorkOrderScope,
  applyWorkOrderSearch,
  buildWorkOrderListEntries,
  buildWorkOrderListEntry,
  groupWorkOrderEntriesByLane,
} from "./workOrderListModel";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";

const factory: FactoriesFactory = { id: "f", name: "Refunds", key: "RF" };

/** Fixture-only shape: a step execution plus which line it ran on, before
 * it's grouped into a dispatch by `order()` below. */
type TestExecution = FactoriesWorkOrderExecution & { line?: FactoriesLineRef };

interface OrderOverrides extends Omit<Partial<FactoriesWorkOrder>, "lineDispatches"> {
  /** Convenience: a flat execution list, grouped into one dispatch per
   * distinct line so most test cases don't need to hand-build dispatches. */
  executions?: TestExecution[];
}

function order(overrides: OrderOverrides = {}): FactoriesWorkOrder {
  const { executions, ...rest } = overrides;
  return {
    id: overrides.id ?? "wo-1",
    title: "Order title",
    state: "STATE_OPEN",
    result: "RESULT_UNSPECIFIED",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: executions ? dispatchesFromExecutions(executions) : [],
    ...rest,
  };
}

function dispatchesFromExecutions(executions: TestExecution[]): FactoriesWorkOrderLineDispatch[] {
  const byLineId = new Map<string, FactoriesWorkOrderLineDispatch>();
  for (const { line, ...execution } of executions) {
    const lineId = line?.id ?? "unknown";
    const dispatch = byLineId.get(lineId);
    if (dispatch) {
      dispatch.stepExecutions = [...(dispatch.stepExecutions ?? []), execution];
      continue;
    }
    byLineId.set(lineId, {
      id: `dispatch-${lineId}`,
      line,
      state: isActiveWorkOrderExecution(execution) ? "STATE_ACTIVE" : "STATE_FINISHED",
      stepExecutions: [execution],
    });
  }
  return [...byLineId.values()];
}

describe("buildWorkOrderListEntry", () => {
  it("derives the display key, status, and searchable haystack", () => {
    const entry = buildWorkOrderListEntry(
      order({
        id: "wo-1",
        title: "Reconcile refunds",
        description: "Duplicate refund entries after retries.",
        number: "42",
        key: "RF-42",
        assignees: [{ id: "u1", name: "Alex Reviewer" }],
      }),
      factory,
    );

    expect(entry.displayKey).toBe("RF-42");
    expect(entry.displayStatus).toBe("waiting");
    expect(entry.assigneeIds).toEqual(["u1"]);
    expect(entry.assigneeNames).toEqual(["Alex Reviewer"]);
    expect(entry.searchHaystack).toContain("rf-42");
    expect(entry.searchHaystack).toContain("reconcile refunds");
    expect(entry.searchHaystack).toContain("alex reviewer");
    expect(entry.isDispatchable).toBe(true);
  });

  it("picks the latest execution and counts distinct lines", () => {
    const entry = buildWorkOrderListEntry(
      order({
        executions: [
          {
            id: "e1",
            step: "plan",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            line: { id: "line-a", name: "plan-and-implement" },
            updatedAt: "2024-06-01T09:00:00Z",
          },
          {
            id: "e2",
            step: "verify",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
            line: { id: "line-b", name: "hotfix" },
            updatedAt: "2024-06-02T10:00:00Z",
          },
        ],
      }),
      factory,
    );

    expect(entry.latestLineName).toBe("hotfix");
    expect(entry.latestStepName).toBe("verify");
    expect(entry.distinctLineCount).toBe(2);
    expect(entry.lineIds).toEqual(["line-a", "line-b"]);
    expect(entry.displayStatus).toBe("running");
    expect(entry.searchHaystack).toContain("plan-and-implement");
  });

  it("builds the shared line and step caption", () => {
    const single = buildWorkOrderListEntry(
      order({
        executions: [{ id: "e1", step: "plan", state: "STATE_STARTED", line: { id: "line-a", name: "hotfix" } }],
      }),
      factory,
    );
    expect(single.lineStepLabel).toBe("hotfix · plan");

    const twoLines = buildWorkOrderListEntry(
      order({
        executions: [
          { id: "e1", step: "plan", state: "STATE_FINISHED", line: { id: "line-a", name: "review" } },
          {
            id: "e2",
            step: "verify",
            state: "STATE_STARTED",
            line: { id: "line-b", name: "hotfix" },
            updatedAt: "2024-06-03T00:00:00Z",
          },
        ],
      }),
      factory,
    );
    expect(twoLines.lineStepLabel).toBe("hotfix · verify · +1 more line");

    expect(buildWorkOrderListEntry(order(), factory).lineStepLabel).toBe("");
  });

  it("prefers an in-flight execution when timestamps tie", () => {
    const sameMoment = "2024-06-02T10:00:00Z";
    const entry = buildWorkOrderListEntry(
      order({
        executions: [
          {
            id: "e1",
            step: "plan",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            line: { id: "line-a", name: "plan-and-implement" },
            updatedAt: sameMoment,
          },
          {
            id: "e2",
            step: "verify",
            state: "STATE_CANCELLING",
            result: "RESULT_UNKNOWN",
            line: { id: "line-b", name: "hotfix" },
            updatedAt: sameMoment,
          },
        ],
      }),
      factory,
    );

    // A cancelling step is still active, so it must win the tie and agree
    // with the Running status pill.
    expect(entry.latestStepName).toBe("verify");
    expect(entry.latestLineName).toBe("hotfix");
    expect(entry.displayStatus).toBe("running");
  });

  it("aggregates usage across executions when order totals are absent", () => {
    const entry = buildWorkOrderListEntry(
      order({
        executions: [
          {
            id: "e1",
            step: "plan",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            totalTokens: "1200",
            costCents: "45",
          },
          {
            id: "e2",
            step: "run",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            totalTokens: "500",
            costCents: "15",
          },
        ],
      }),
      factory,
    );

    expect(entry.totalTokens).toBe(1700);
    expect(entry.totalCostCents).toBe(60);
    expect(entry.usageLabel).toBe("$0.60 · 2k tokens");
  });

  it("keeps closed orders out of the dispatchable set", () => {
    const entry = buildWorkOrderListEntry(order({ state: "STATE_CLOSED", result: "RESULT_COMPLETED" }), factory);
    expect(entry.isDispatchable).toBe(false);
  });
});

describe("scope + filter + search + ordering", () => {
  const meAssigned = order({ id: "mine-1", assignees: [{ id: "me", name: "You" }], updatedAt: "2024-06-05T00:00:00Z" });
  const unassigned = order({ id: "u-1", assignees: [], updatedAt: "2024-06-06T00:00:00Z" });
  const others = order({ id: "o-1", assignees: [{ id: "alex", name: "Alex" }], updatedAt: "2024-06-07T00:00:00Z" });
  const closed = order({
    id: "c-1",
    state: "STATE_CLOSED",
    result: "RESULT_COMPLETED",
    updatedAt: "2024-05-01T00:00:00Z",
  });

  const entries = buildWorkOrderListEntries([meAssigned, unassigned, others, closed], factory);

  it("scope=my only keeps orders assigned to the current user", () => {
    expect(applyWorkOrderScope(entries, "my", "me").map((e) => e.id)).toEqual(["mine-1"]);
  });

  it("scope=active drops completed, failed, and cancelled orders", () => {
    expect(applyWorkOrderScope(entries, "active").map((e) => e.id)).toEqual(["mine-1", "u-1", "o-1"]);
  });

  it("scope=all keeps everything", () => {
    expect(applyWorkOrderScope(entries, "all")).toHaveLength(4);
  });

  it("status filter narrows down to the selected display statuses", () => {
    const only = applyWorkOrderFilters(entries, { ...EMPTY_WORK_ORDER_FILTERS, statuses: ["waiting"] }).map(
      (e) => e.id,
    );
    expect(only).toContain("mine-1");
    expect(only).not.toContain("c-1");
  });

  it("assignee filter matches people and the unassigned sentinel", () => {
    const byPerson = applyWorkOrderFilters(entries, { ...EMPTY_WORK_ORDER_FILTERS, assigneeIds: ["alex"] });
    expect(byPerson.map((e) => e.id)).toEqual(["o-1"]);

    const nobody = applyWorkOrderFilters(entries, {
      ...EMPTY_WORK_ORDER_FILTERS,
      assigneeIds: [UNASSIGNED_FILTER_VALUE],
    });
    expect(nobody.map((e) => e.id)).toEqual(["u-1", "c-1"]);
  });

  it("line filter keeps orders that ran on any selected line", () => {
    const withLines = buildWorkOrderListEntries(
      [
        order({ id: "on-a", executions: [{ id: "e1", line: { id: "line-a", name: "Bug fixer" } }] }),
        order({ id: "on-b", executions: [{ id: "e2", line: { id: "line-b", name: "Hotfix" } }] }),
      ],
      factory,
    );
    const onA = applyWorkOrderFilters(withLines, { ...EMPTY_WORK_ORDER_FILTERS, lineIds: ["line-a"] });
    expect(onA.map((e) => e.id)).toEqual(["on-a"]);
  });

  it("filters across dimensions narrow together", () => {
    const none = applyWorkOrderFilters(entries, {
      ...EMPTY_WORK_ORDER_FILTERS,
      statuses: ["waiting"],
      assigneeIds: ["nobody-here"],
    });
    expect(none).toEqual([]);
  });

  it("search matches on title, description, line, and assignee names", () => {
    const alex = applyWorkOrderSearch(entries, "alex").map((e) => e.id);
    expect(alex).toEqual(["o-1"]);
  });

  it("ordering=updated puts the newest updated first", () => {
    expect(applyWorkOrderOrdering(entries, "updated").map((e) => e.id)).toEqual(["o-1", "u-1", "mine-1", "c-1"]);
  });

  it("ordering=status follows the display status order", () => {
    const mixed = buildWorkOrderListEntries(
      [
        order({ id: "done", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
        order({ id: "draft", state: "STATE_DRAFT" }),
        order({ id: "waiting" }),
      ],
      factory,
    );
    expect(applyWorkOrderOrdering(mixed, "status").map((e) => e.id)).toEqual(["draft", "waiting", "done"]);
  });

  it("ordering=spend puts the most expensive order first", () => {
    const spenders = buildWorkOrderListEntries(
      [
        order({ id: "cheap", totalCostCents: "10" }),
        order({ id: "pricey", totalCostCents: "900" }),
        order({ id: "mid", totalCostCents: "120" }),
      ],
      factory,
    );
    expect(applyWorkOrderOrdering(spenders, "spend").map((e) => e.id)).toEqual(["pricey", "mid", "cheap"]);
  });

  it("ordering=key sorts identifiers numerically, newest first", () => {
    const numbered = buildWorkOrderListEntries(
      [
        order({ id: "wo-1", key: "RF-1", number: "1" }),
        order({ id: "wo-10", key: "RF-10", number: "10" }),
        order({ id: "wo-2", key: "RF-2", number: "2" }),
      ],
      factory,
    );
    expect(applyWorkOrderOrdering(numbered, "key").map((e) => e.displayKey)).toEqual(["RF-10", "RF-2", "RF-1"]);
  });
});

describe("groupWorkOrderEntriesByLane", () => {
  it("maps every display status to its board lane", () => {
    const grouped = groupWorkOrderEntriesByLane(
      buildWorkOrderListEntries(
        [
          order({ id: "draft", state: "STATE_DRAFT" }),
          order({ id: "waiting" }),
          order({ id: "completed", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
          order({ id: "failed", state: "STATE_CLOSED", result: "RESULT_FAILED" }),
          order({ id: "cancelled", state: "STATE_CLOSED", result: "RESULT_REJECTED" }),
        ],
        factory,
      ),
    );

    expect(grouped.get("backlog")?.map((entry) => entry.id)).toEqual(["draft"]);
    expect(grouped.get("running")).toEqual([]);
    expect(grouped.get("review")?.map((entry) => entry.id)).toEqual(["waiting"]);
    expect(grouped.get("done")?.map((entry) => entry.id)).toEqual(["completed", "failed", "cancelled"]);
  });
});
