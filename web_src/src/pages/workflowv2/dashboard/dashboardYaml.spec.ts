import { describe, expect, it } from "vitest";

import { dashboardToYaml, parseDashboardYaml, validateDashboardContent, MAX_DASHBOARD_PANELS } from "./dashboardYaml";

describe("dashboardToYaml / parseDashboardYaml", () => {
  it("round-trips an empty dashboard", () => {
    const text = dashboardToYaml({ panels: [], layout: [], canvasId: "abc", canvasName: "My Canvas" });
    expect(text).toContain("apiVersion: v1");
    expect(text).toContain("kind: Dashboard");
    expect(text).toContain("canvasId: abc");
    expect(text).toContain("name: My Canvas");

    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.spec.panels).toEqual([]);
      expect(result.data.spec.layout).toEqual([]);
    }
  });

  it("round-trips a populated dashboard", () => {
    const text = dashboardToYaml({
      panels: [{ id: "intro", type: "markdown", content: { body: "# Hi" } }],
      layout: [{ i: "intro", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }],
    });

    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error("parseDashboardYaml should succeed");
    expect(result.data.spec.panels).toEqual([{ id: "intro", type: "markdown", content: { body: "# Hi" } }]);
    expect(result.data.spec.layout).toEqual([{ i: "intro", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }]);
  });

  it("round-trips all typed panel kinds", () => {
    const panels = [
      { id: "doc", type: "markdown", content: { title: "Intro", body: "# Hi" } },
      { id: "deploy", type: "node", content: { node: "deploy-prod", showRun: true } },
      {
        id: "runs",
        type: "table",
        content: {
          dataSource: { kind: "executions", limit: 25 },
          render: {
            kind: "table",
            columns: [
              { field: "status", label: "Status", format: "status" },
              { field: "createdAt", label: "Started", format: "datetime" },
            ],
          },
        },
      },
      {
        id: "perf",
        type: "chart",
        content: {
          dataSource: { kind: "executions", limit: 100 },
          render: { kind: "chart", type: "bar", xField: "status", series: [{ label: "Count" }] },
        },
      },
      {
        id: "total",
        type: "number",
        content: {
          dataSource: { kind: "runs" },
          render: { kind: "number", aggregation: "count" },
        },
      },
    ];
    const layout = panels.map((p, i) => ({ i: p.id, x: 0, y: i * 4, w: 12, h: 4 }));
    const text = dashboardToYaml({ panels, layout });
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
    expect(result.data.spec.layout).toEqual(layout);
  });

  it("rejects a node panel whose node field is not a string", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: bad-node
      type: node
      content:
        node: 123
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/content\.node must be a string/);
  });

  it("accepts a table panel without columns (configured via the form)", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: table-1
      type: table
      content:
        dataSource: { kind: memory, namespace: environments }
        render:
          kind: table
          columns: []
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
  });

  it("rejects empty input", () => {
    expect(parseDashboardYaml("")).toEqual({ ok: false, error: expect.any(String) });
  });

  it("rejects unknown root keys", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels: []
  layout: []
extra: 1
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("Unknown top-level field");
  });

  it("rejects unsupported panel types", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: p1
      type: timeline
      content: {}
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("unsupported type");
  });

  it("rejects duplicate panel ids", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: p
      type: markdown
      content: {}
    - id: p
      type: markdown
      content: {}
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("Duplicate panel id");
  });

  it("rejects non-string body", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: p
      type: markdown
      content:
        body: 42
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("body must be a string");
  });

  it("rejects layout referring to missing panel", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: p
      type: markdown
      content: {}
  layout:
    - i: other
      x: 0
      y: 0
      w: 1
      h: 1
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
  });

  it("rejects wrong apiVersion", () => {
    const text = `apiVersion: v2
kind: Dashboard
metadata: {}
spec:
  panels: []
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("Unsupported apiVersion");
  });

  it("round-trips a number panel with prefix and suffix symbols", () => {
    const panels = [
      {
        id: "spend",
        type: "number",
        content: {
          dataSource: { kind: "memory", namespace: "expenses" },
          render: {
            kind: "number",
            aggregation: "sum",
            field: "amount",
            format: "number",
            prefix: "R$",
            suffix: " /mo",
          },
        },
      },
    ];
    const layout = [{ i: "spend", x: 0, y: 0, w: 6, h: 3 }];
    const text = dashboardToYaml({ panels, layout });
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("round-trips a composite memory number panel with heterogeneous sources", () => {
    const panels = [
      {
        id: "score",
        type: "number",
        content: {
          dataSource: {
            kind: "memory",
            combine: "sum",
            sources: [
              { namespace: "a", aggregation: "sum", field: "cost" },
              { namespace: "b", aggregation: "count" },
            ],
          },
          render: { kind: "number", format: "number", prefix: "R$" },
        },
      },
    ];
    const layout = [{ i: "score", x: 0, y: 0, w: 4, h: 3 }];
    const text = dashboardToYaml({ panels, layout });
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("rejects a composite memory number panel when render.aggregation is set", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: score
      type: number
      content:
        dataSource:
          kind: memory
          combine: sum
          sources:
            - namespace: a
              aggregation: sum
              field: cost
        render:
          kind: number
          aggregation: sum
          field: cost
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.aggregation must not be set/);
  });

  it("rejects a composite memory number panel with an unknown combine operator", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: score
      type: number
      content:
        dataSource:
          kind: memory
          combine: median
          sources:
            - namespace: a
              aggregation: sum
              field: cost
        render:
          kind: number
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/dataSource\.combine must be one of/);
  });

  it("rejects a composite memory source missing a field for non-count aggregation", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: score
      type: number
      content:
        dataSource:
          kind: memory
          combine: sum
          sources:
            - namespace: a
              aggregation: sum
        render:
          kind: number
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/dataSource\.sources\[0\]\.field is required/);
  });
});

describe("validateDashboardContent", () => {
  it("flags too many panels", () => {
    const panels = Array.from({ length: MAX_DASHBOARD_PANELS + 1 }, (_, i) => ({
      id: `p${i}`,
      type: "markdown",
      content: {},
    }));
    expect(validateDashboardContent(panels, [])).toContain("Too many panels");
  });

  it("flags layout with non-positive size", () => {
    expect(
      validateDashboardContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: 0, y: 0, w: 0, h: 1 }]),
    ).toContain("positive width and height");
  });

  it("flags negative position", () => {
    expect(
      validateDashboardContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: -1, y: 0, w: 1, h: 1 }]),
    ).toContain("non-negative");
  });

  it("accepts a valid dashboard", () => {
    expect(
      validateDashboardContent(
        [{ id: "p", type: "markdown", content: { body: "ok" } }],
        [{ i: "p", x: 0, y: 0, w: 1, h: 1 }],
      ),
    ).toBeNull();
  });
});
