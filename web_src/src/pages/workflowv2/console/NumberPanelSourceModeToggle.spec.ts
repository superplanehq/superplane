import { describe, expect, it } from "vitest";

import { convertNumberPanelMode, countNonMemoryMetrics, detectMode } from "./numberPanelSourceMode";
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

describe("countNonMemoryMetrics — guards lossy multi → composite switch", () => {
  it("returns 0 for non-multi content (single / composite panels)", () => {
    const single: NumberPanelContent = {
      title: "Single",
      dataSource: { kind: "runs", limit: 100 },
      render: { kind: "number", aggregation: "count" },
    };
    expect(countNonMemoryMetrics(single)).toBe(0);

    const composite: NumberPanelContent = {
      title: "Composite",
      dataSource: { kind: "memory", combine: "sum", sources: [{ namespace: "a", aggregation: "count" }] },
      render: { kind: "number" },
    };
    expect(countNonMemoryMetrics(composite)).toBe(0);
  });

  it("returns 0 when every metric is memory-backed", () => {
    const allMemory: NumberPanelContent = {
      title: "All memory",
      metrics: [
        { dataSource: { kind: "memory", namespace: "aws" }, render: { kind: "number", aggregation: "count" } },
        { dataSource: { kind: "memory", namespace: "gcp" }, render: { kind: "number", aggregation: "count" } },
      ],
    };
    expect(countNonMemoryMetrics(allMemory)).toBe(0);
  });

  it("treats a metric missing its dataSource as non-memory without throwing", () => {
    // Mirrors a YAML draft where a `metrics` entry has no `dataSource` yet.
    const draft = {
      title: "Draft",
      metrics: [
        { dataSource: { kind: "memory", namespace: "aws" }, render: { kind: "number", aggregation: "count" } },
        { render: { kind: "number", aggregation: "count" } },
        {},
      ],
    } as unknown as NumberPanelContent;
    expect(() => countNonMemoryMetrics(draft)).not.toThrow();
    expect(countNonMemoryMetrics(draft)).toBe(2);
  });

  it("counts every runs/executions metric that composite mode cannot represent", () => {
    const mixed: NumberPanelContent = {
      title: "Mixed",
      metrics: [
        { dataSource: { kind: "memory", namespace: "aws" }, render: { kind: "number", aggregation: "count" } },
        { dataSource: { kind: "runs", limit: 100 }, render: { kind: "number", aggregation: "count" } },
        { dataSource: { kind: "executions", limit: 50 }, render: { kind: "number", aggregation: "count" } },
      ],
    };
    expect(countNonMemoryMetrics(mixed)).toBe(2);
  });

  it("still drops non-memory metrics if the conversion is forced (documents why the UI blocks it)", () => {
    const mixed: NumberPanelContent = {
      title: "Mixed",
      metrics: [
        {
          dataSource: { kind: "memory", namespace: "aws" },
          render: { kind: "number", aggregation: "sum", field: "cost" },
        },
        { dataSource: { kind: "runs", limit: 100 }, render: { kind: "number", aggregation: "count" } },
      ],
    };
    // The conversion itself is inherently lossy (composite is memory-only), so
    // the toggle disables it whenever countNonMemoryMetrics > 0.
    const forced = convertNumberPanelMode("composite", mixed);
    expect(isCompositeMemoryDataSource(forced.dataSource) ? forced.dataSource.sources.length : 0).toBe(1);
    expect(countNonMemoryMetrics(mixed)).toBe(1);
  });
});
