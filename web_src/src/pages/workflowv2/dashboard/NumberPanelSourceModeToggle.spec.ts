import { describe, expect, it } from "vitest";

import { convertNumberPanelMode, detectMode } from "./numberPanelSourceMode";
import { isCompositeMemoryDataSource, type NumberPanelContent } from "./panelTypes";

describe("convertNumberPanelMode — composite ↔ multi", () => {
  const composite: NumberPanelContent = {
    title: "Spend",
    dataSource: {
      kind: "memory",
      combine: "sum",
      sources: [
        { namespace: "aws", aggregation: "sum", field: "cost", fieldPath: "billing" },
        { namespace: "gcp", aggregation: "count" },
        { namespace: "azure", aggregation: "max", field: "spike" },
      ],
    },
    render: {
      kind: "number",
      label: "Total spend",
      format: "number",
      prefix: "R$",
      suffix: " /mo",
      sparklineField: "trend",
    },
  };

  it("maps every composite source into its own metric (no sources dropped)", () => {
    const result = convertNumberPanelMode("multi", composite);
    expect(result.metrics).toHaveLength(3);
    expect(result.metrics?.map((m) => (m.dataSource.kind === "memory" ? m.dataSource.namespace : null))).toEqual([
      "aws",
      "gcp",
      "azure",
    ]);
    expect(result.dataSource).toBeUndefined();
  });

  it("carries each source's aggregation/field and fieldPath onto its metric", () => {
    const result = convertNumberPanelMode("multi", composite);
    expect(result.metrics?.[0].render.aggregation).toBe("sum");
    expect(result.metrics?.[0].render.field).toBe("cost");
    expect(result.metrics?.[0].dataSource).toMatchObject({ kind: "memory", namespace: "aws", fieldPath: "billing" });
    expect(result.metrics?.[1].render.aggregation).toBe("count");
    expect(result.metrics?.[1].render.field).toBeUndefined();
    expect(result.metrics?.[2].render.aggregation).toBe("max");
    expect(result.metrics?.[2].render.field).toBe("spike");
  });

  it("preserves top-level presentation options on every metric", () => {
    const result = convertNumberPanelMode("multi", composite);
    for (const metric of result.metrics ?? []) {
      expect(metric.render).toMatchObject({
        label: "Total spend",
        format: "number",
        prefix: "R$",
        suffix: " /mo",
        sparklineField: "trend",
      });
    }
  });

  it("round-trips multi → composite → multi without losing namespaces", () => {
    const multi = convertNumberPanelMode("multi", composite);
    const backToComposite = convertNumberPanelMode("composite", multi);
    expect(detectMode(backToComposite)).toBe("composite");
    expect(
      isCompositeMemoryDataSource(backToComposite.dataSource) ? backToComposite.dataSource.sources.length : 0,
    ).toBe(3);

    const backToMulti = convertNumberPanelMode("multi", backToComposite);
    expect(backToMulti.metrics?.map((m) => (m.dataSource.kind === "memory" ? m.dataSource.namespace : null))).toEqual([
      "aws",
      "gcp",
      "azure",
    ]);
  });

  it("falls back to a single seed metric when the composite has no sources", () => {
    const empty: NumberPanelContent = {
      title: "Empty",
      dataSource: { kind: "memory", combine: "sum", sources: [] },
      render: { kind: "number" },
    };
    const result = convertNumberPanelMode("multi", empty);
    expect(result.metrics).toHaveLength(1);
  });
});
