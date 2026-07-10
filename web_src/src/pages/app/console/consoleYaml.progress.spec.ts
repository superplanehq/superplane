import { describe, expect, it } from "vitest";

import { consoleToYaml, parseConsoleYaml } from "./consoleYaml";

describe("consoleYaml — progress columns", () => {
  it("round-trips a table panel with a progress column", () => {
    const panels = [
      {
        id: "checklist",
        type: "table",
        content: {
          dataSource: { kind: "memory", namespace: "tasks" },
          render: {
            kind: "table",
            columns: [
              {
                field: "done",
                label: "Progress",
                format: "progress",
                progressTarget: "total",
                progressLabel: "number",
              },
            ],
          },
        },
      },
    ];
    const layout = [{ i: "checklist", x: 0, y: 0, w: 12, h: 4 }];
    const text = consoleToYaml({ panels, layout });
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("rejects a progress column without progressTarget", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: table-1
      type: table
      content:
        dataSource: { kind: memory, namespace: env }
        render:
          kind: table
          columns:
            - field: done
              format: progress
  layout: []
`;
    const result = parseConsoleYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/render\.columns\[0\]\.progressTarget/);
  });
});
