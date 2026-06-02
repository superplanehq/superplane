import { describe, expect, it } from "vitest";

import { dashboardToYaml, parseDashboardYaml } from "./dashboardYaml";

describe("dashboardToYaml / parseDashboardYaml — multi-number panels", () => {
  it("round-trips a multi-number panel with independently-configured metrics", () => {
    const panels = [
      {
        id: "kpis",
        type: "number",
        content: {
          title: "Pipeline KPIs",
          metrics: [
            {
              dataSource: { kind: "runs" },
              render: { kind: "number", aggregation: "count", label: "Total runs" },
            },
            {
              dataSource: { kind: "memory", namespace: "costs" },
              render: {
                kind: "number",
                aggregation: "sum",
                field: "cost",
                label: "Total cost",
                format: "number",
                prefix: "R$",
              },
            },
          ],
        },
      },
    ];
    const layout = [{ i: "kpis", x: 0, y: 0, w: 8, h: 3 }];
    const text = dashboardToYaml({ panels, layout });
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(true);
    if (!result.ok) throw new Error(result.error);
    expect(result.data.spec.panels).toEqual(panels);
  });

  it("rejects a multi-number panel with an empty metrics array", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: kpis
      type: number
      content:
        metrics: []
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/metrics must be a non-empty array/);
  });

  it("rejects a multi-number metric using a composite data source", () => {
    const text = `apiVersion: v1
kind: Console
metadata: {}
spec:
  panels:
    - id: kpis
      type: number
      content:
        metrics:
          - dataSource:
              kind: memory
              combine: sum
              sources:
                - namespace: a
                  aggregation: count
            render:
              kind: number
              aggregation: count
  layout: []
`;
    const result = parseDashboardYaml(text);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/metrics\[0\]\.dataSource must be a single-source/);
  });
});
