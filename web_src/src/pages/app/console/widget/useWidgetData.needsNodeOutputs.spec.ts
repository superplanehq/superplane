import { describe, it, expect } from "vitest";

import { renderNeedsRunNodeOutputs } from "./useWidgetData";
import type { WidgetChartRender, WidgetNumberRender, WidgetTableRender } from "./types";

describe("renderNeedsRunNodeOutputs", () => {
  it("returns false for an undefined render", () => {
    expect(renderNeedsRunNodeOutputs(undefined)).toBe(false);
  });

  it("returns false for a count KPI that only reads totalCount", () => {
    const render: WidgetNumberRender = { kind: "number", aggregation: "count" };
    expect(renderNeedsRunNodeOutputs(render)).toBe(false);
  });

  it("does not treat currency / label literals containing `$` as a node ref", () => {
    const render: WidgetNumberRender = {
      kind: "number",
      aggregation: "sum",
      field: "amount",
      prefix: "R$",
      suffix: " USD",
      label: "Total $",
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(false);
  });

  it("detects `$['node']` in a literal table column field", () => {
    const render: WidgetTableRender = {
      kind: "table",
      columns: [{ field: '$["deploy-prod"].outputs.url', format: "link" }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(true);
  });

  it("detects `$` inside a `{{ }}` CEL template column", () => {
    const render: WidgetTableRender = {
      kind: "table",
      columns: [{ field: "{{ $['deploy-prod'].data.url }}", format: "link" }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(true);
  });

  it("detects `$` with whitespace before the bracket", () => {
    const render: WidgetTableRender = {
      kind: "table",
      columns: [{ field: "$ ['deploy-prod'].state" }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(true);
  });

  it("detects `$` in a row-style condition field", () => {
    const render: WidgetTableRender = {
      kind: "table",
      columns: [{ field: "status" }],
      rowStyles: [{ field: '$["deploy-prod"].result', op: "eq", value: "RESULT_FAILED", tone: "red" }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(true);
  });

  it("detects `$` in a chart series field", () => {
    const render: WidgetChartRender = {
      kind: "chart",
      type: "bar",
      xField: "status",
      series: [{ field: '$["scoring"].outputs.score' }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(true);
  });

  it("returns false for a chart that only uses derived run fields", () => {
    const render: WidgetChartRender = {
      kind: "chart",
      type: "bar",
      xField: "status",
      series: [{ field: "durationMs" }],
    };
    expect(renderNeedsRunNodeOutputs(render)).toBe(false);
  });
});
