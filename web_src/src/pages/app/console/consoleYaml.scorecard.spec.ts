import { describe, expect, it } from "vitest";

import { consoleToYaml, parseConsoleYaml } from "./consoleYaml";

describe("console YAML — scorecard panels", () => {
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
