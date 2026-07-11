import { describe, expect, it } from "vitest";

import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import {
  aggregateNumber,
  aggregateNumberPerSource,
  applySort,
  buildChartData,
  combinePartials,
  distinctSeriesKeys,
} from "./widgetData";

const entries: CanvasMemoryEntry[] = [
  { id: "1", namespace: "expenses", values: { amount: 10 }, source: "node" },
  { id: "2", namespace: "expenses", values: { amount: 30 }, source: "node" },
  { id: "3", namespace: "tests", values: { name: "a" }, source: "node" },
  { id: "4", namespace: "tests", values: { name: "b" }, source: "node" },
  { id: "5", namespace: "tests", values: { name: "c" }, source: "node" },
];

describe("aggregateNumber", () => {
  it("skips null and blank values instead of coercing them to 0", () => {
    const rows = [{ n: null }, { n: "" }, { n: 5 }, { n: 8 }];
    expect(aggregateNumber(rows, "first", "n")).toBe(5);
    expect(aggregateNumber(rows, "last", "n")).toBe(8);
    expect(aggregateNumber(rows, "sum", "n")).toBe(13);
  });

  it("returns null when every value is non-numeric", () => {
    expect(aggregateNumber([{ n: null }, { n: "" }, { n: "x" }], "sum", "n")).toBeNull();
  });
});

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

describe("buildChartData", () => {
  it("resolves literal field paths for x and series with one row per category", () => {
    const rows = [
      { status: "passed", durationMs: 120 },
      { status: "failed", durationMs: 800 },
    ];
    const data = buildChartData(rows, "status", [{ key: "duration", field: "durationMs" }, { key: "count" }]);
    expect(data).toEqual([
      { x: "passed", duration: 120, count: 1 },
      { x: "failed", duration: 800, count: 1 },
    ]);
  });

  it("buckets rows that share the same xField value and sums numeric series", () => {
    const rows = [
      { status: "passed", durationMs: 120 },
      { status: "failed", durationMs: 800 },
      { status: "passed", durationMs: 30 },
    ];
    const data = buildChartData(rows, "status", [{ key: "duration", field: "durationMs" }, { key: "count" }]);
    expect(data).toEqual([
      { x: "passed", duration: 150, count: 2 },
      { x: "failed", duration: 800, count: 1 },
    ]);
  });

  it("groups by a `{{ formatDate(..) }}` xField and sums durations within each day", () => {
    const day1morning = new Date(2026, 2, 15, 8, 0); // Mar 15, 2026 08:00 local
    const day1afternoon = new Date(2026, 2, 15, 14, 30); // Mar 15, 2026 14:30 local
    const day2 = new Date(2026, 2, 16, 17, 30); // Mar 16, 2026 17:30 local
    const rows = [
      { createdAt: day1morning.toISOString(), durationMs: 100 },
      { createdAt: day1afternoon.toISOString(), durationMs: 75 },
      { createdAt: day2.toISOString(), durationMs: 250 },
    ];
    const data = buildChartData(rows, '{{ formatDate(createdAt, "MM/dd") }}', [
      { key: "duration", field: "durationMs" },
    ]);
    expect(data).toEqual([
      { x: "03/15", duration: 175 },
      { x: "03/16", duration: 250 },
    ]);
  });

  it("evaluates `{{ expr }}` on a series field and aggregates per bucket", () => {
    const rows = [{ value: "12" }, { value: "8" }, { value: "12" }];
    const data = buildChartData(rows, "value", [{ key: "half", field: "{{ int(value) / 2 }}" }]);
    expect(data).toEqual([
      { x: "12", half: 12 },
      { x: "8", half: 4 },
    ]);
  });

  it("falls back to empty x when the xField resolves to undefined", () => {
    const data = buildChartData([{ foo: 1 }], "missing", [{ key: "count" }]);
    expect(data).toEqual([{ x: "", count: 1 }]);
  });

  it("ignores non-numeric series values when summing", () => {
    const rows = [
      { date: "2026-05-26", cost: 10 },
      { date: "2026-05-26", cost: "not-a-number" },
      { date: "2026-05-26", cost: 5 },
    ];
    const data = buildChartData(rows, "date", [{ key: "cost", field: "cost" }]);
    expect(data).toEqual([{ x: "2026-05-26", cost: 15 }]);
  });

  it("pivots long-format rows by `seriesField` into one series per distinct value", () => {
    const rows = [
      { date: "2026-05-26", service: "ec2", cost_usd: 58.45 },
      { date: "2026-05-26", service: "s3", cost_usd: 0.0034 },
      { date: "2026-05-27", service: "ec2", cost_usd: 60.0 },
      { date: "2026-05-27", service: "s3", cost_usd: 0.01 },
    ];
    const data = buildChartData(rows, "date", [{ key: "cost", field: "cost_usd" }], { seriesField: "service" });
    expect(data).toEqual([
      { x: "2026-05-26", ec2: 58.45, s3: 0.0034 },
      { x: "2026-05-27", ec2: 60.0, s3: 0.01 },
    ]);
  });

  it("sums values within the same `(x, seriesField)` bucket when pivoting", () => {
    const rows = [
      { date: "2026-05-26", service: "ec2", cost_usd: 10 },
      { date: "2026-05-26", service: "ec2", cost_usd: 5 },
      { date: "2026-05-26", service: "s3", cost_usd: 1 },
    ];
    const data = buildChartData(rows, "date", [{ key: "cost", field: "cost_usd" }], { seriesField: "service" });
    expect(data).toEqual([{ x: "2026-05-26", ec2: 15, s3: 1 }]);
  });

  it("uses (empty) as the series key when seriesField is missing on a row", () => {
    const rows = [
      { date: "2026-05-26", service: "ec2", cost_usd: 10 },
      { date: "2026-05-26", cost_usd: 3 },
      { date: "2026-05-27", service: "ec2", cost_usd: 5 },
    ];
    const data = buildChartData(rows, "date", [{ key: "cost", field: "cost_usd" }], { seriesField: "service" });
    expect(data).toEqual([
      { x: "2026-05-26", ec2: 10, "(empty)": 3 },
      { x: "2026-05-27", ec2: 5, "(empty)": 0 },
    ]);
  });

  it("counts rows per `(x, seriesField)` bucket when the value series omits `field`", () => {
    const rows = [
      { date: "2026-05-26", service: "ec2" },
      { date: "2026-05-26", service: "ec2" },
      { date: "2026-05-26", service: "s3" },
      { date: "2026-05-27", service: "ec2" },
    ];
    const data = buildChartData(rows, "date", [{ key: "count" }], { seriesField: "service" });
    expect(data).toEqual([
      { x: "2026-05-26", ec2: 2, s3: 1 },
      { x: "2026-05-27", ec2: 1, s3: 0 },
    ]);
  });
});

describe("distinctSeriesKeys", () => {
  it("returns the first-seen distinct values of the seriesField", () => {
    const rows = [{ service: "ec2" }, { service: "s3" }, { service: "ec2" }, { service: "rds" }];
    expect(distinctSeriesKeys(rows, "service")).toEqual(["ec2", "s3", "rds"]);
  });

  it("uses (empty) for rows where the seriesField is missing", () => {
    const rows = [{ service: "ec2" }, { other: 1 }, { service: "s3" }];
    expect(distinctSeriesKeys(rows, "service")).toEqual(["ec2", "(empty)", "s3"]);
  });
});

describe("applySort", () => {
  it("returns the input array reference unchanged when sort is undefined", () => {
    const rows = [{ a: 3 }, { a: 1 }, { a: 2 }];
    expect(applySort(rows, undefined)).toBe(rows);
  });

  it("returns the input unchanged when sort.field is blank", () => {
    const rows = [{ a: 3 }, { a: 1 }];
    expect(applySort(rows, { field: "   " })).toBe(rows);
  });

  it("sorts numeric fields ascending by default", () => {
    const rows = [{ amount: 30 }, { amount: 10 }, { amount: 20 }];
    expect(applySort(rows, { field: "amount" })).toEqual([{ amount: 10 }, { amount: 20 }, { amount: 30 }]);
  });

  it("sorts numeric fields descending when order is desc", () => {
    const rows = [{ amount: 30 }, { amount: 10 }, { amount: 20 }];
    expect(applySort(rows, { field: "amount", order: "desc" })).toEqual([
      { amount: 30 },
      { amount: 20 },
      { amount: 10 },
    ]);
  });

  it("sorts ISO-string date fields chronologically", () => {
    const rows = [
      { createdAt: "2026-03-16T17:30:00.000Z" },
      { createdAt: "2026-03-14T08:00:00.000Z" },
      { createdAt: "2026-03-15T12:00:00.000Z" },
    ];
    expect(applySort(rows, { field: "createdAt" })).toEqual([
      { createdAt: "2026-03-14T08:00:00.000Z" },
      { createdAt: "2026-03-15T12:00:00.000Z" },
      { createdAt: "2026-03-16T17:30:00.000Z" },
    ]);
  });

  it("sends null/undefined values to the end regardless of order", () => {
    const rowsAsc = applySort(
      [{ a: null as number | null }, { a: 2 }, { a: undefined as number | undefined }, { a: 1 }],
      { field: "a" },
    );
    expect(rowsAsc.map((r) => r.a)).toEqual([1, 2, null, undefined]);

    const rowsDesc = applySort(
      [{ a: null as number | null }, { a: 2 }, { a: undefined as number | undefined }, { a: 1 }],
      { field: "a", order: "desc" },
    );
    expect(rowsDesc.map((r) => r.a)).toEqual([2, 1, null, undefined]);
  });

  it("supports `{{ expr }}` fields and pre-compiles them once per call", () => {
    const rows = [
      { a: 1, b: 5 }, // sum 6
      { a: 4, b: 1 }, // sum 5
      { a: 2, b: 2 }, // sum 4
    ];
    expect(applySort(rows, { field: "{{ a + b }}" })).toEqual([
      { a: 2, b: 2 },
      { a: 4, b: 1 },
      { a: 1, b: 5 },
    ]);
  });

  it("does not mutate the input array", () => {
    const rows = [{ n: 3 }, { n: 1 }, { n: 2 }];
    const snapshot = rows.map((r) => r.n);
    applySort(rows, { field: "n" });
    expect(rows.map((r) => r.n)).toEqual(snapshot);
  });
});
