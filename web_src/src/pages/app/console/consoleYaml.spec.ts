import { describe, expect, it } from "vitest";

import { consoleToYaml, parseConsoleYaml, validateConsoleContent, MAX_CONSOLE_PANELS } from "./consoleYaml";

describe("consoleToYaml / parseConsoleYaml", () => {
  it("round-trips an empty console", () => {
    const text = consoleToYaml({ panels: [], layout: [], canvasId: "abc", canvasName: "My Canvas" });
    expect(text).toContain("apiVersion: v1");
    expect(text).toContain("kind: Console");
    expect(text).toContain("canvasId: abc");
    expect(text).toContain("name: My Canvas");

    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.spec.panels).toEqual([]);
      expect(result.data.spec.layout).toEqual([]);
    }
  });

  it("rejects legacy `kind: Dashboard` on import", () => {
    const text = `apiVersion: v1
kind: Dashboard
metadata: {}
spec:
  panels: []
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (result.ok) throw new Error("expected parseConsoleYaml to fail for kind: Dashboard");
    expect(result.error).toMatch(/Unsupported kind/);
  });

  it("round-trips a populated console", () => {
    const text = consoleToYaml({
      panels: [{ id: "intro", type: "markdown", content: { body: "# Hi" } }],
      layout: [{ i: "intro", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }],
    });

    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error("parseConsoleYaml should succeed");
    expect(result.data.spec.panels).toEqual([{ id: "intro", type: "markdown", content: { body: "# Hi" } }]);
    expect(result.data.spec.layout).toEqual([{ i: "intro", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }]);
  });

  it("round-trips all typed panel kinds", () => {
    const panels = [
      { id: "doc", type: "markdown", content: { title: "Intro", body: "# Hi" } },
      { id: "deploy", type: "node", content: { node: "deploy-prod", showRun: true } },
      {
        id: "key-nodes",
        type: "nodes",
        content: {
          title: "Key Nodes",
          nodes: [
            { node: "deploy-prod", description: "Promotes the latest build", showRun: true },
            { node: "rollback", label: "Rollback" },
          ],
        },
      },
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
            sort: { field: "createdAt", order: "desc" },
          },
        },
      },
      {
        id: "perf",
        type: "chart",
        content: {
          dataSource: { kind: "executions", limit: 100 },
          render: {
            kind: "chart",
            type: "bar",
            xField: "status",
            series: [{ label: "Count" }],
            sort: { field: "createdAt", order: "asc" },
          },
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
      {
        id: "papercuts",
        type: "scorecard",
        content: {
          title: "Open UX papercuts",
          dataSource: { kind: "memory", namespace: "ux_papercuts" },
          render: {
            kind: "scorecard",
            aggregation: "last",
            field: "openCount",
            format: "number",
            label: "Open UX papercuts",
            better: "down",
            target: "80",
            showProgress: true,
            sparklineField: "openCount",
            showChange: "both",
            changeCaption: "vs start of range",
          },
        },
      },
    ];
    const layout = panels.map((p, i) => ({ i: p.id, x: 0, y: i * 4, w: 12, h: 4 }));
    const text = consoleToYaml({ panels, layout });
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
    expect(result.data.spec.layout).toEqual(layout);
  });

  it("rejects render.sort when field is missing", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: runs
      type: table
      content:
        dataSource: { kind: executions, limit: 25 }
        render:
          kind: table
          columns:
            - field: status
          sort:
            order: asc
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.sort\.field/);
  });

  it("rejects render.sort.order with an unknown value", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: perf
      type: chart
      content:
        dataSource: { kind: executions, limit: 100 }
        render:
          kind: chart
          type: bar
          xField: status
          series:
            - label: Count
          sort:
            field: createdAt
            order: random
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.sort\.order/);
  });

  it("rejects a nodes panel entry without a node reference", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: key-nodes
      type: nodes
      content:
        nodes:
          - description: missing node
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/content\.nodes\[0\]\.node must be a non-empty string/);
  });

  it("rejects a nodes panel where nodes is not an array", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: key-nodes
      type: nodes
      content:
        nodes:
          oops: true
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/content\.nodes must be an array/);
  });

  it("rejects a node panel whose node field is not a string", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: bad-node
      type: node
      content:
        node: 123
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/content\.node must be a string/);
  });

  it("accepts a table panel without columns (configured via the form)", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
  });

  it("rejects empty input", () => {
    expect(parseConsoleYaml("")).toEqual({ ok: false, error: expect.any(String) });
  });

  it("rejects unknown root keys", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels: []
  layout: []
extra: 1
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("Unknown top-level field");
  });

  it("rejects unsupported panel types", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p1
      type: timeline
      content: {}
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("unsupported type");
  });

  it("rejects duplicate panel ids", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("Duplicate panel id");
  });

  it("rejects non-string body", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: p
      type: markdown
      content:
        body: 42
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toContain("body must be a string");
  });

  it("rejects layout referring to missing panel", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
  });

  it("rejects wrong apiVersion", () => {
    const text = `apiVersion: v2
kind: Console
metadata: {}
spec:
  panels: []
  layout: []
`;
    const result = parseConsoleYaml(text);
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
    const text = consoleToYaml({ panels, layout });
    const result = parseConsoleYaml(text);
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
    const text = consoleToYaml({ panels, layout });
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("rejects a composite memory number panel when render.aggregation is set", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.aggregation must not be set/);
  });

  it("rejects a composite memory number panel when only render.field is set", () => {
    const text = `apiVersion: v1
kind: Console
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
          field: cost
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.field must not be set/);
  });

  it("rejects a composite memory number panel with an unknown combine operator", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/dataSource\.combine must be one of/);
  });

  it("rejects a composite memory source missing a field for non-count aggregation", () => {
    const text = `apiVersion: v1
kind: Console
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
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/dataSource\.sources\[0\]\.field is required/);
  });

  it("round-trips a scorecard panel with target, sparkline, and change caption", () => {
    const panels = [
      {
        id: "papercuts",
        type: "scorecard",
        content: {
          title: "Open UX papercuts",
          dataSource: { kind: "memory", namespace: "ux_papercuts" },
          render: {
            kind: "scorecard",
            aggregation: "last",
            field: "openCount",
            format: "number",
            label: "Open UX papercuts",
            better: "down",
            target: "80",
            showProgress: true,
            sparklineField: "openCount",
            showChange: "both",
            changeCaption: "vs start of range",
          },
        },
      },
    ];
    const layout = [{ i: "papercuts", x: 0, y: 0, w: 6, h: 4 }];
    const text = consoleToYaml({ panels, layout });
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("rejects a scorecard panel with an unknown better value", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: papercuts
      type: scorecard
      content:
        dataSource: { kind: memory, namespace: ux_papercuts }
        render:
          kind: scorecard
          aggregation: count
          better: sideways
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.better must be one of/);
  });

  it("rejects a scorecard panel without a field for non-count aggregation", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: papercuts
      type: scorecard
      content:
        dataSource: { kind: memory, namespace: ux_papercuts }
        render:
          kind: scorecard
          aggregation: last
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.field is required/);
  });
});

describe("validateConsoleContent", () => {
  it("flags too many panels", () => {
    const panels = Array.from({ length: MAX_CONSOLE_PANELS + 1 }, (_, i) => ({
      id: `p${i}`,
      type: "markdown",
      content: {},
    }));
    expect(validateConsoleContent(panels, [])).toContain("Too many panels");
  });

  it("flags layout with non-positive size", () => {
    expect(
      validateConsoleContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: 0, y: 0, w: 0, h: 1 }]),
    ).toContain("positive width and height");
  });

  it("flags negative position", () => {
    expect(
      validateConsoleContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: -1, y: 0, w: 1, h: 1 }]),
    ).toContain("non-negative");
  });

  it("accepts a valid console", () => {
    expect(
      validateConsoleContent(
        [{ id: "p", type: "markdown", content: { body: "ok" } }],
        [{ i: "p", x: 0, y: 0, w: 1, h: 1 }],
      ),
    ).toBeNull();
  });
});
