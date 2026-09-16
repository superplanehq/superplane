import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";
import type { FactoriesWorkOrder } from "@/api-client";

import {
  LINE_COLUMN_FILTER_LABELS,
  LINE_COLUMN_SORT_LABELS,
  LINE_COLUMN_SORT_STORAGE_KEY,
  LINE_COLUMN_SORTS,
  allowedFiltersForColumn,
  allowedSortsForColumn,
  compareLineOrders,
  compareLinePhaseRuns,
  confidenceScoresFromCheckQueries,
  filterLineColumnOrders,
  lineColumnSortDirectionLabels,
  readStoredLineColumnSorts,
  resolveLineColumnFilter,
  resolveLineColumnSort,
  useLineColumnSortPreference,
} from "./lineColumnSort";

function order(overrides: Partial<FactoriesWorkOrder> & { id: string }): FactoriesWorkOrder {
  return {
    title: overrides.id,
    state: "STATE_DRAFT",
    ...overrides,
  };
}

describe("lineColumnSort", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("limits options by column type", () => {
    expect(allowedSortsForColumn("backlog")).toEqual(LINE_COLUMN_SORTS.backlog);
    expect(allowedSortsForColumn("done")).toEqual(LINE_COLUMN_SORTS.done);
    expect(allowedSortsForColumn("verify")).toEqual(LINE_COLUMN_SORTS.verify);
    expect(allowedSortsForColumn("phase-0")).toEqual(LINE_COLUMN_SORTS.phase);
    expect(allowedSortsForColumn("phase-2")).toEqual(["updated", "created"]);
    expect(LINE_COLUMN_SORT_LABELS.updated).toBe("Newest activity");
    expect(allowedFiltersForColumn("done")).toEqual(["all", "completed", "failed", "rejected"]);
    expect(allowedFiltersForColumn("backlog")).toEqual(["all"]);
    expect(LINE_COLUMN_FILTER_LABELS.completed).toBe("Completed");
    expect(lineColumnSortDirectionLabels("created")).toEqual({ desc: "Newest first", asc: "Oldest first" });
    expect(lineColumnSortDirectionLabels("confidence")).toEqual({ desc: "High to low", asc: "Low to high" });
  });

  it("falls back to newest activity for unknown or disallowed ids", () => {
    expect(resolveLineColumnSort("backlog", undefined)).toBe("updated");
    expect(resolveLineColumnSort("backlog", "compact")).toBe("updated");
    expect(resolveLineColumnSort("done", "confidence")).toBe("updated");
    expect(resolveLineColumnSort("phase-1", "result")).toBe("updated");
    expect(resolveLineColumnSort("backlog", "confidence")).toBe("confidence");
    expect(resolveLineColumnFilter("done", "completed")).toBe("completed");
    expect(resolveLineColumnFilter("backlog", "completed")).toBe("all");
    expect(resolveLineColumnFilter("done", "compact")).toBe("all");
  });

  it("sorts created time from work order createdAt newest first", () => {
    const older = order({
      id: "wo-old",
      createdAt: "2026-08-11T10:00:00.000Z",
      updatedAt: "2026-08-11T18:00:00.000Z",
    });
    const newer = order({
      id: "wo-new",
      createdAt: "2026-08-11T12:00:00.000Z",
      updatedAt: "2026-08-11T13:00:00.000Z",
    });

    expect(compareLineOrders("created", older, newer)).toBeGreaterThan(0);
    expect(compareLineOrders("updated", older, newer)).toBeLessThan(0);
    expect(compareLineOrders("created", older, newer, undefined, "asc")).toBeLessThan(0);
  });

  it("sorts completed time from closed updatedAt newest first", () => {
    const earlier = order({
      id: "wo-early",
      state: "STATE_CLOSED",
      updatedAt: "2026-08-11T10:00:00.000Z",
    });
    const later = order({
      id: "wo-late",
      state: "STATE_CLOSED",
      updatedAt: "2026-08-11T12:00:00.000Z",
    });

    expect(compareLineOrders("completed", earlier, later)).toBeGreaterThan(0);
  });

  it("sorts Done results Failed, Rejected, then Completed", () => {
    const failed = order({ id: "wo-failed", state: "STATE_CLOSED", result: "RESULT_FAILED" });
    const rejected = order({ id: "wo-rejected", state: "STATE_CLOSED", result: "RESULT_REJECTED" });
    const completed = order({ id: "wo-completed", state: "STATE_CLOSED", result: "RESULT_COMPLETED" });

    const sorted = [completed, failed, rejected].sort((left, right) => compareLineOrders("result", left, right));
    expect(sorted.map((entry) => entry.id)).toEqual(["wo-failed", "wo-rejected", "wo-completed"]);

    const reversed = [completed, failed, rejected].sort((left, right) =>
      compareLineOrders("result", left, right, undefined, "asc"),
    );
    expect(reversed.map((entry) => entry.id)).toEqual(["wo-completed", "wo-rejected", "wo-failed"]);
  });

  it("filters Done orders by result", () => {
    const failed = order({ id: "wo-failed", state: "STATE_CLOSED", result: "RESULT_FAILED" });
    const rejected = order({ id: "wo-rejected", state: "STATE_CLOSED", result: "RESULT_REJECTED" });
    const completed = order({ id: "wo-completed", state: "STATE_CLOSED", result: "RESULT_COMPLETED" });

    expect(filterLineColumnOrders([failed, rejected, completed], "all").map((entry) => entry.id)).toEqual([
      "wo-failed",
      "wo-rejected",
      "wo-completed",
    ]);
    expect(filterLineColumnOrders([failed, rejected, completed], "completed").map((entry) => entry.id)).toEqual([
      "wo-completed",
    ]);
  });

  it("sorts confidence high to low and puts missing scores last", () => {
    const high = order({ id: "wo-high" });
    const low = order({ id: "wo-low" });
    const missing = order({ id: "wo-missing" });
    const scores = new Map<string, number | undefined>([
      ["wo-high", 5],
      ["wo-low", 1],
      ["wo-missing", undefined],
    ]);

    const sorted = [missing, low, high].sort((left, right) => compareLineOrders("confidence", left, right, scores));
    expect(sorted.map((entry) => entry.id)).toEqual(["wo-high", "wo-low", "wo-missing"]);

    const lowFirst = [missing, high, low].sort((left, right) =>
      compareLineOrders("confidence", left, right, scores, "asc"),
    );
    expect(lowFirst.map((entry) => entry.id)).toEqual(["wo-low", "wo-high", "wo-missing"]);
  });

  it("sorts phase created time from the work order, not the execution", () => {
    const olderOrder = {
      executionId: "exec-new",
      order: { id: "wo-old", createdAt: "2026-08-11T09:00:00.000Z" },
      execution: { createdAt: "2026-08-11T16:00:00.000Z", updatedAt: "2026-08-11T16:00:00.000Z" },
    };
    const newerOrder = {
      executionId: "exec-old",
      order: { id: "wo-new", createdAt: "2026-08-11T12:00:00.000Z" },
      execution: { createdAt: "2026-08-11T10:00:00.000Z", updatedAt: "2026-08-11T10:00:00.000Z" },
    };

    expect(compareLinePhaseRuns("created", olderOrder, newerOrder)).toBeGreaterThan(0);
    expect(compareLinePhaseRuns("updated", olderOrder, newerOrder)).toBeLessThan(0);
    expect(compareLinePhaseRuns("created", olderOrder, newerOrder, "asc")).toBeLessThan(0);
  });

  it("keeps updated order while confidence queries are pending", () => {
    expect(
      confidenceScoresFromCheckQueries(["wo-1"], [{ data: [{ name: "Confidence score", score: 5 }], isPending: true }]),
    ).toBeUndefined();
    expect(
      confidenceScoresFromCheckQueries(
        ["wo-1"],
        [{ data: [{ name: "Confidence score", score: 5 }], isPending: false }],
      ),
    ).toEqual(new Map([["wo-1", 5]]));
  });

  it("ignores corrupt storage and omits the default sort", () => {
    window.localStorage.setItem(LINE_COLUMN_SORT_STORAGE_KEY, "{not-json");
    expect(readStoredLineColumnSorts("line-1")).toEqual({});

    window.localStorage.setItem(
      LINE_COLUMN_SORT_STORAGE_KEY,
      JSON.stringify({ "line-1": { backlog: "created", done: "updated", verify: "confidence" } }),
    );
    expect(readStoredLineColumnSorts("line-1")).toEqual({
      backlog: { sort: "created", direction: "desc", filter: "all" },
    });
  });

  it("stores non-default sorts per line and column", () => {
    const { result } = renderHook(() => useLineColumnSortPreference("line-1"));

    expect(result.current.sortFor("backlog")).toBe("updated");
    expect(result.current.viewFor("backlog")).toEqual({ sort: "updated", direction: "desc", filter: "all" });
    expect(window.localStorage.getItem(LINE_COLUMN_SORT_STORAGE_KEY)).toBeNull();

    act(() => {
      result.current.setSort("backlog", "confidence");
      result.current.setSort("done", "result");
    });
    expect(result.current.sortFor("backlog")).toBe("confidence");
    expect(result.current.sortFor("done")).toBe("result");
    expect(JSON.parse(window.localStorage.getItem(LINE_COLUMN_SORT_STORAGE_KEY) ?? "{}")).toEqual({
      "line-1": { backlog: { sort: "confidence" }, done: { sort: "result" } },
    });

    act(() => {
      result.current.setSort("backlog", "updated");
      result.current.setDirection("done", "asc");
      result.current.setFilter("done", "completed");
    });
    expect(result.current.sortFor("backlog")).toBe("updated");
    expect(result.current.viewFor("done")).toEqual({ sort: "result", direction: "asc", filter: "completed" });
    expect(JSON.parse(window.localStorage.getItem(LINE_COLUMN_SORT_STORAGE_KEY) ?? "{}")).toEqual({
      "line-1": { done: { sort: "result", direction: "asc", filter: "completed" } },
    });
  });
});
