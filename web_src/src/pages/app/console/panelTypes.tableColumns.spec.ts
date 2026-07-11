import { describe, expect, it } from "vitest";

import { normalizeTablePanelContent } from "./panelTypes";

describe("normalizeTablePanelContent — progress/trend columns", () => {
  it("preserves progress column options and drops unknown progressLabel values", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [
          { field: "done", format: "progress", progressTarget: "total", progressLabel: "number" },
          { field: "score", format: "progress", progressTarget: "100", progressLabel: "fraction" },
        ],
      },
    });
    expect(normalized.render.columns[0]).toEqual({
      field: "done",
      format: "progress",
      progressTarget: "total",
      progressLabel: "number",
    });
    expect(normalized.render.columns[1]).toEqual({
      field: "score",
      format: "progress",
      progressTarget: "100",
      progressLabel: undefined,
    });
  });

  it("preserves trend column options and drops invalid enum values", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "runs" },
      render: {
        kind: "table",
        columns: [
          {
            field: "durationMs",
            label: "Trend",
            format: "trend",
            trendBetter: "down",
            trendDisplay: "value",
          },
          {
            field: "score",
            label: "Bad",
            format: "trend",
            trendBetter: "sideways",
            trendDisplay: "chart",
          },
        ],
      },
    });
    expect(normalized.render.columns[0]).toMatchObject({
      field: "durationMs",
      format: "trend",
      trendBetter: "down",
      trendDisplay: "value",
    });
    expect(normalized.render.columns[1].trendBetter).toBeUndefined();
    expect(normalized.render.columns[1].trendDisplay).toBeUndefined();
  });
});
