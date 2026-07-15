import { describe, expect, it } from "vitest";

import { isRunQueryStillSearching, pickMemoryRows, resolveMemoryVariable } from "./useMarkdownVariables";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import type { MarkdownMemoryVariableSource, MarkdownRunVariableSource } from "./markdownVariables";
import { WIDGET_MAX_EAGER_PAGES } from "./widget/widgetPagination";

function memorySource(extra: Partial<MarkdownMemoryVariableSource>): MarkdownMemoryVariableSource {
  return { kind: "memory", namespace: "ns", ...extra };
}

function runSource(extra: Partial<MarkdownRunVariableSource> = {}): MarkdownRunVariableSource {
  return { kind: "run", select: "latest", ...extra };
}

const rows = [{ name: "a" }, { name: "b" }, { name: "c" }];

describe("pickMemoryRows", () => {
  it("returns the first row in single mode (default)", () => {
    expect(pickMemoryRows(rows, memorySource({}))).toEqual({ name: "a" });
    expect(pickMemoryRows(rows, memorySource({ mode: "single" }))).toEqual({ name: "a" });
  });

  it("returns the full sorted array in list mode with no limit", () => {
    expect(pickMemoryRows(rows, memorySource({ mode: "list" }))).toEqual(rows);
  });

  it("respects an explicit limit in list mode", () => {
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: 2 }))).toEqual([{ name: "a" }, { name: "b" }]);
  });

  it("ignores limit when not in list mode", () => {
    // Single mode authors should keep getting the first row even if a stale
    // `limit` is still present on the source.
    expect(pickMemoryRows(rows, memorySource({ limit: 99 }))).toEqual({ name: "a" });
  });

  it("returns the full sorted array when limit is zero, negative, or fractional", () => {
    // Validation is layered above this helper - here we mirror the
    // production resolver's fail-soft behavior: any non-positive-integer
    // limit means "no cap", so the panel still renders the full list
    // instead of an empty one.
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: 0 }))).toEqual(rows);
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: -1 }))).toEqual(rows);
    // A fractional limit must not be silently floored by `slice` (1.5 -> 1 row).
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: 1.5 }))).toEqual(rows);
  });

  it("returns an empty array when the sorted set is empty in list mode", () => {
    expect(pickMemoryRows([], memorySource({ mode: "list" }))).toEqual([]);
  });

  it("returns undefined when the sorted set is empty in single mode", () => {
    // Mirrors how `sorted[0]` behaves; callers should still gate on the
    // outer `resolveMemoryVariable` empty-array branch instead of using
    // this helper for the no-rows path.
    expect(pickMemoryRows([], memorySource({}))).toBeUndefined();
  });
});

describe("resolveMemoryVariable loading state", () => {
  it("resolves a single-row variable to null while loading", () => {
    expect(resolveMemoryVariable([], memorySource({}), true)).toEqual({ value: null });
  });

  it("resolves a list-mode variable to null while loading (not an empty list)", () => {
    // Regression: returning `[]` here let `VariablePreview` skip its
    // `loading && value == null` guard and flash "List · 0 items" before the
    // memory query settled. Both modes must resolve to null mid-flight.
    expect(resolveMemoryVariable([], memorySource({ mode: "list" }), true)).toEqual({ value: null });
    expect(resolveMemoryVariable([], memorySource({ mode: "list", limit: 5 }), true)).toEqual({ value: null });
  });

  it("resolves a list-mode variable to an empty array once loading settles with no rows", () => {
    expect(resolveMemoryVariable([], memorySource({ mode: "list" }), false)).toEqual({ value: [] });
  });

  it("surfaces a no-rows error for a single-row variable once loading settles", () => {
    const result = resolveMemoryVariable([], memorySource({}), false);
    expect(result.value).toBeNull();
    expect(result.error).toMatch(/No memory rows/);
  });

  it("resolves the full list once rows arrive even if still flagged loading", () => {
    // Once matching entries exist the backing query has produced data, so the
    // list resolves normally regardless of the loading flag.
    const entries: CanvasMemoryEntry[] = [
      { id: "1", namespace: "ns", values: { name: "a" }, source: "node", createdAt: "2026-06-01T00:00:00Z" },
      { id: "2", namespace: "ns", values: { name: "b" }, source: "node", createdAt: "2026-06-02T00:00:00Z" },
    ];
    const result = resolveMemoryVariable(entries, memorySource({ mode: "list" }), true);
    expect(Array.isArray(result.value)).toBe(true);
    expect((result.value as unknown[]).length).toBe(2);
  });
});

describe("isRunQueryStillSearching", () => {
  const idleQuery = {
    isLoading: false,
    isFetchingNextPage: false,
    isError: false,
    isFetchNextPageError: false,
    hasNextPage: true as boolean | undefined,
    data: { pages: [{}] },
  };

  it("keeps searching while filtered pages remain under the eager page cap", () => {
    expect(isRunQueryStillSearching(idleQuery, runSource({ statuses: ["passed"] }))).toBe(true);
  });

  it("stops searching after a fetchNextPage failure so loading cannot stick", () => {
    // Regression: hasNextPage stays true when pageCount does not advance after
    // a failed next-page fetch — without the failure guard the variable stays
    // "searching" forever and the panel never surfaces an error.
    expect(
      isRunQueryStillSearching(
        { ...idleQuery, isFetchNextPageError: true },
        runSource({ statuses: ["passed"], triggers: ["deploy"] }),
      ),
    ).toBe(false);
  });

  it("stops searching after a page-1 query failure", () => {
    expect(isRunQueryStillSearching({ ...idleQuery, isError: true, data: undefined }, runSource({}))).toBe(false);
  });

  it("stops searching once the eager page cap is reached", () => {
    expect(
      isRunQueryStillSearching(
        {
          ...idleQuery,
          data: { pages: Array.from({ length: WIDGET_MAX_EAGER_PAGES }, () => ({})) },
        },
        runSource({ statuses: ["failed"] }),
      ),
    ).toBe(false);
  });

  it("stops searching for impossible select + status combinations", () => {
    // latest_passed hits a RESULT_PASSED bucket — filtering to failed can never match.
    expect(isRunQueryStillSearching(idleQuery, runSource({ select: "latest_passed", statuses: ["failed"] }))).toBe(
      false,
    );
    expect(isRunQueryStillSearching(idleQuery, runSource({ select: "latest_failed", statuses: ["passed"] }))).toBe(
      false,
    );
  });

  it("keeps searching while the node catalog is empty even if trigger refs do not resolve yet", () => {
    const unresolved = () => undefined;
    expect(
      isRunQueryStillSearching(idleQuery, runSource({ triggers: ["deploy"] }), unresolved, { nodeCatalogSize: 0 }),
    ).toBe(true);
    expect(
      isRunQueryStillSearching(idleQuery, runSource({ triggers: ["deploy"] }), unresolved, { nodeCatalogSize: 2 }),
    ).toBe(false);
  });
});
