import { describe, expect, it } from "vitest";

import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { aggregateNumberPerSource, combinePartials } from "./widgetData";

const entries: CanvasMemoryEntry[] = [
  { id: "1", namespace: "expenses", values: { amount: 10 } },
  { id: "2", namespace: "expenses", values: { amount: 30 } },
  { id: "3", namespace: "tests", values: { name: "a" } },
  { id: "4", namespace: "tests", values: { name: "b" } },
  { id: "5", namespace: "tests", values: { name: "c" } },
];

describe("aggregateNumberPerSource", () => {
  it("sums a numeric field across rows of the same namespace", () => {
    const value = aggregateNumberPerSource(
      entries,
      { namespace: "expenses", aggregation: "sum", field: "amount" },
      undefined,
    );
    expect(value).toBe(40);
  });

  it("counts rows in a namespace when aggregation is count", () => {
    expect(aggregateNumberPerSource(entries, { namespace: "tests", aggregation: "count" }, undefined)).toBe(3);
  });

  it("returns null when no rows match the namespace and aggregation is not count", () => {
    expect(
      aggregateNumberPerSource(entries, { namespace: "missing", aggregation: "sum", field: "amount" }, undefined),
    ).toBe(null);
  });

  it("returns 0 when no rows match the namespace and aggregation is count", () => {
    expect(aggregateNumberPerSource(entries, { namespace: "missing", aggregation: "count" }, undefined)).toBe(0);
  });

  it("applies shared widget filters before aggregating each source", () => {
    const value = aggregateNumberPerSource(entries, { namespace: "expenses", aggregation: "sum", field: "amount" }, [
      "row.amount > 10",
    ]);
    expect(value).toBe(30);
  });
});

describe("combinePartials", () => {
  it("sums available partials and skips nulls", () => {
    expect(combinePartials([40, null, 3], "sum")).toBe(43);
  });

  it("returns null when every partial is null", () => {
    expect(combinePartials([null, null], "sum")).toBe(null);
  });

  it("picks the minimum of non-null partials", () => {
    expect(combinePartials([5, 2, null, 8], "min")).toBe(2);
  });

  it("picks the maximum of non-null partials", () => {
    expect(combinePartials([5, 2, null, 8], "max")).toBe(8);
  });

  it("averages non-null partials (unweighted across sources)", () => {
    expect(combinePartials([2, 4, null], "avg")).toBe(3);
  });
});
