import { describe, expect, it } from "vitest";

import { buildRankingData, parseWindowSeconds, type WidgetRankingRender } from "./rankingData";

/**
 * `buildRankingData` groups rows, aggregates a metric per group, ranks the
 * groups, and (optionally) compares each group's current metric against the
 * previous rolling window. Trend tests pin `nowSeconds` so the window split is
 * deterministic; timestamps are expressed relative to that fixed `now`.
 */
const NOW_SECONDS = 1_700_000_000;
const NOW_MS = NOW_SECONDS * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;

/** ISO timestamp `days` before the pinned `now`. */
function daysAgo(days: number): string {
  return new Date(NOW_MS - days * DAY_MS).toISOString();
}

describe("parseWindowSeconds", () => {
  it("parses unit suffixes", () => {
    expect(parseWindowSeconds("60s")).toBe(60);
    expect(parseWindowSeconds("30m")).toBe(1800);
    expect(parseWindowSeconds("24h")).toBe(86_400);
    expect(parseWindowSeconds("7d")).toBe(604_800);
    expect(parseWindowSeconds("2w")).toBe(1_209_600);
  });

  it("returns null for blank or malformed input", () => {
    expect(parseWindowSeconds(undefined)).toBeNull();
    expect(parseWindowSeconds("")).toBeNull();
    expect(parseWindowSeconds("7")).toBeNull();
    expect(parseWindowSeconds("7y")).toBeNull();
    expect(parseWindowSeconds("0d")).toBeNull();
  });
});

describe("buildRankingData grouping + aggregation", () => {
  const countRender: WidgetRankingRender = {
    kind: "ranking",
    groupField: "nodeName",
    aggregation: "count",
  };

  it("counts rows per group and ranks descending", () => {
    const rows = [
      { nodeName: "deploy" },
      { nodeName: "deploy" },
      { nodeName: "deploy" },
      { nodeName: "tests" },
      { nodeName: "build" },
      { nodeName: "build" },
    ];
    const result = buildRankingData(rows, countRender, NOW_SECONDS);
    expect(result.map((r) => [r.rank, r.group, r.value])).toEqual([
      [1, "deploy", 3],
      [2, "build", 2],
      [3, "tests", 1],
    ]);
  });

  it("sums a numeric valueField per group", () => {
    const rows = [
      { nodeName: "deploy", durationMs: 100 },
      { nodeName: "deploy", durationMs: 250 },
      { nodeName: "tests", durationMs: 400 },
    ];
    const result = buildRankingData(
      rows,
      { kind: "ranking", groupField: "nodeName", aggregation: "sum", valueField: "durationMs" },
      NOW_SECONDS,
    );
    expect(result).toEqual([
      { rank: 1, group: "tests", value: 400, previousValue: null, deltaPct: null, direction: "flat" },
      { rank: 2, group: "deploy", value: 350, previousValue: null, deltaPct: null, direction: "flat" },
    ]);
  });

  it("respects the top-N limit", () => {
    const rows = [
      { nodeName: "a" },
      { nodeName: "a" },
      { nodeName: "a" },
      { nodeName: "b" },
      { nodeName: "b" },
      { nodeName: "c" },
    ];
    const result = buildRankingData(rows, { ...countRender, limit: 2 }, NOW_SECONDS);
    expect(result.map((r) => r.group)).toEqual(["a", "b"]);
  });

  it("skips rows with non-numeric value fields when aggregating", () => {
    const rows = [
      { nodeName: "deploy", cost: "12.5" },
      { nodeName: "deploy", cost: "oops" },
      { nodeName: "deploy", cost: 7.5 },
    ];
    const result = buildRankingData(
      rows,
      { kind: "ranking", groupField: "nodeName", aggregation: "sum", valueField: "cost" },
      NOW_SECONDS,
    );
    expect(result).toEqual([
      { rank: 1, group: "deploy", value: 20, previousValue: null, deltaPct: null, direction: "flat" },
    ]);
  });
});

describe("buildRankingData trend", () => {
  const trendRender: WidgetRankingRender = {
    kind: "ranking",
    groupField: "nodeName",
    aggregation: "count",
    trend: { timestampField: "createdAt", window: "7d" },
  };

  it("marks a group up when the current window exceeds the previous", () => {
    const rows = [
      // current window (0-7d): 3 rows
      { nodeName: "deploy", createdAt: daysAgo(1) },
      { nodeName: "deploy", createdAt: daysAgo(2) },
      { nodeName: "deploy", createdAt: daysAgo(6) },
      // previous window (7-14d): 1 row
      { nodeName: "deploy", createdAt: daysAgo(9) },
    ];
    const [row] = buildRankingData(rows, trendRender, NOW_SECONDS);
    expect(row.value).toBe(3);
    expect(row.previousValue).toBe(1);
    expect(row.direction).toBe("up");
    expect(row.deltaPct).toBeCloseTo(2, 5);
  });

  it("marks a group down when the current window trails the previous", () => {
    const rows = [
      { nodeName: "tests", createdAt: daysAgo(3) },
      { nodeName: "tests", createdAt: daysAgo(8) },
      { nodeName: "tests", createdAt: daysAgo(9) },
      { nodeName: "tests", createdAt: daysAgo(10) },
    ];
    const [row] = buildRankingData(rows, trendRender, NOW_SECONDS);
    expect(row.value).toBe(1);
    expect(row.previousValue).toBe(3);
    expect(row.direction).toBe("down");
    expect(row.deltaPct).toBeCloseTo(-2 / 3, 5);
  });

  it("marks a group flat when both windows match", () => {
    const rows = [
      { nodeName: "build", createdAt: daysAgo(2) },
      { nodeName: "build", createdAt: daysAgo(3) },
      { nodeName: "build", createdAt: daysAgo(9) },
      { nodeName: "build", createdAt: daysAgo(10) },
    ];
    const [row] = buildRankingData(rows, trendRender, NOW_SECONDS);
    expect(row.value).toBe(2);
    expect(row.previousValue).toBe(2);
    expect(row.direction).toBe("flat");
    expect(row.deltaPct).toBe(0);
  });

  it("marks a group new when it has no previous baseline", () => {
    const rows = [
      { nodeName: "lint", createdAt: daysAgo(1) },
      { nodeName: "lint", createdAt: daysAgo(4) },
    ];
    const [row] = buildRankingData(rows, trendRender, NOW_SECONDS);
    expect(row.value).toBe(2);
    expect(row.previousValue).toBeNull();
    expect(row.direction).toBe("new");
    expect(row.deltaPct).toBeNull();
  });

  it("drops rows outside both windows", () => {
    const rows = [
      { nodeName: "deploy", createdAt: daysAgo(1) },
      // 20 days ago falls outside the previous window entirely
      { nodeName: "deploy", createdAt: daysAgo(20) },
    ];
    const [row] = buildRankingData(rows, trendRender, NOW_SECONDS);
    expect(row.value).toBe(1);
    expect(row.previousValue).toBeNull();
    expect(row.direction).toBe("new");
  });

  it("falls back to no trend when the window is malformed", () => {
    const rows = [
      { nodeName: "deploy", createdAt: daysAgo(1) },
      { nodeName: "deploy", createdAt: daysAgo(9) },
    ];
    const result = buildRankingData(
      rows,
      { ...trendRender, trend: { timestampField: "createdAt", window: "bogus" } },
      NOW_SECONDS,
    );
    expect(result[0].value).toBe(2);
    expect(result[0].previousValue).toBeNull();
    expect(result[0].direction).toBe("flat");
  });
});
