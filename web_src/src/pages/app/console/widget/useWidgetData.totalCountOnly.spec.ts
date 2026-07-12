import { describe, expect, it } from "vitest";

import { makeRunsFlightKey, runsRenderIsTotalCountOnly } from "./runsWidgetQuery";
import type { WidgetDataSource, WidgetNumberRender, WidgetScorecardRender } from "./types";

const runsSource: WidgetDataSource = { kind: "runs", limit: 1000 };
const executionsSource: WidgetDataSource = { kind: "executions", limit: 100 };
const memorySource: WidgetDataSource = { kind: "memory", namespace: "deploys" };

const numberCount: WidgetNumberRender = { kind: "number", aggregation: "count" };
const numberSum: WidgetNumberRender = { kind: "number", aggregation: "sum", field: "value" };
const numberCountWithFilters: WidgetNumberRender = {
  kind: "number",
  aggregation: "count",
  filters: ["result == 'RESULT_PASSED'"],
};
const numberCountWithSparkline: WidgetNumberRender = {
  kind: "number",
  aggregation: "count",
  sparklineField: "value",
};
const scorecardCount: WidgetScorecardRender = { kind: "scorecard", aggregation: "count" };
const scorecardSum: WidgetScorecardRender = { kind: "scorecard", aggregation: "sum", field: "value" };

describe("runsRenderIsTotalCountOnly", () => {
  it("returns true for runs + number count with no filters or sparkline", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, numberCount)).toBe(true);
  });

  it("returns true for runs + scorecard count with no filters or sparkline", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, scorecardCount)).toBe(true);
  });

  it("returns false when the aggregation is not count", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, numberSum)).toBe(false);
    expect(runsRenderIsTotalCountOnly(runsSource, scorecardSum)).toBe(false);
  });

  it("returns false when the render carries row filters", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, numberCountWithFilters)).toBe(false);
  });

  it("returns false when the render needs a sparkline (rows still required)", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, numberCountWithSparkline)).toBe(false);
  });

  it("returns false for non-runs data sources (executions totalCount != run totalCount)", () => {
    expect(runsRenderIsTotalCountOnly(executionsSource, numberCount)).toBe(false);
    expect(runsRenderIsTotalCountOnly(memorySource, numberCount)).toBe(false);
  });

  it("returns false for table/chart renders that need rows", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, { kind: "table", columns: [] })).toBe(false);
    expect(
      runsRenderIsTotalCountOnly(runsSource, {
        kind: "chart",
        type: "bar",
        xField: "day",
        series: [],
      }),
    ).toBe(false);
  });

  it("returns false when render is undefined", () => {
    expect(runsRenderIsTotalCountOnly(runsSource, undefined)).toBe(false);
  });
});

describe("makeRunsFlightKey", () => {
  it("produces the same key for identical filter shapes", () => {
    expect(makeRunsFlightKey("canvas-1", {})).toBe(makeRunsFlightKey("canvas-1", {}));
  });

  it("produces distinct keys per canvasId", () => {
    expect(makeRunsFlightKey("canvas-1", {})).not.toBe(makeRunsFlightKey("canvas-2", {}));
  });

  it("produces distinct keys per filter shape", () => {
    expect(makeRunsFlightKey("canvas-1", {})).not.toBe(makeRunsFlightKey("canvas-1", { states: ["STATE_STARTED"] }));
    expect(makeRunsFlightKey("canvas-1", { states: ["STATE_STARTED"] })).not.toBe(
      makeRunsFlightKey("canvas-1", { results: ["RESULT_PASSED"] }),
    );
  });
});
