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
          render: { kind: "chart", type: "bar", xField: "status", series: [{ field: "count", label: "Count" }] },
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

  it("rejects a table panel without columns", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels:
    - id: bad-table
      type: table
      content:
        dataSource: { kind: executions }
        render:
          kind: table
          columns: []
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.columns must be a non-empty array/);
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
