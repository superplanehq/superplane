import { describe, expect, it } from "vitest";

import type { CanvasesCanvas } from "@/api-client";

import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "./canvas-yaml-staging";

const sampleWorkflow: CanvasesCanvas = {
  metadata: {
    id: "canvas-123",
    name: "Deploy Pipeline",
    description: "Production deploy flow",
  },
  spec: {
    nodes: [
      {
        id: "trigger-1",
        name: "On Push",
        type: "TYPE_TRIGGER",
        component: "github.on_push",
        position: { x: 100, y: 200 },
      },
      {
        id: "deploy-1",
        name: "Deploy",
        type: "TYPE_ACTION",
        component: "deploy",
        position: { x: 400, y: 200 },
        isCollapsed: true,
      },
    ],
    edges: [{ sourceId: "trigger-1", targetId: "deploy-1" }],
  },
};

describe("parseCanvasYamlToSpec / buildCanvasYamlFromWorkflow", () => {
  it("round-trips a populated canvas", () => {
    const yamlText = buildCanvasYamlFromWorkflow(sampleWorkflow);
    expect(yamlText).toContain("kind: Canvas");
    expect(yamlText).toContain("Deploy Pipeline");

    const spec = parseCanvasYamlToSpec(yamlText);
    expect(spec).not.toBeNull();
    expect(spec?.nodes).toHaveLength(2);
    expect(spec?.edges).toHaveLength(1);
    expect(spec?.nodes?.[0]?.id).toBe("trigger-1");
    expect(spec?.nodes?.[1]?.isCollapsed).toBe(true);
  });

  it("returns null for empty or invalid yaml", () => {
    expect(parseCanvasYamlToSpec("")).toBeNull();
    expect(parseCanvasYamlToSpec("not: [valid")).toBeNull();
    expect(parseCanvasYamlToSpec("kind: Workflow\nspec:\n  nodes: []")).toBeNull();
  });

  it("defaults missing action node type when component is present", () => {
    const yamlText = `apiVersion: v1
kind: Canvas
metadata:
  name: Wait
spec:
  nodes:
    - id: wait-1
      name: Wait
      component: wait
      configuration:
        mode: interval
  edges: []
`;

    expect(parseCanvasYamlToSpec(yamlText)?.nodes?.[0]).toMatchObject({
      id: "wait-1",
      name: "Wait",
      component: "wait",
      type: "TYPE_ACTION",
    });
  });

  it("defaults missing node and edge lists to empty arrays", () => {
    const yamlText = `apiVersion: v1
kind: Canvas
metadata:
  name: Empty
spec:
`;

    expect(parseCanvasYamlToSpec(yamlText)).toEqual({
      nodes: [],
      edges: [],
    });
  });

  it("quotes position y keys when exporting yaml", () => {
    const workflow: CanvasesCanvas = {
      ...sampleWorkflow,
      spec: {
        nodes: [
          {
            id: "node-1",
            name: "Positioned",
            type: "TYPE_ACTION",
            component: "noop",
            position: { x: 500, y: 200 },
          },
        ],
        edges: [],
      },
    };

    const yamlText = buildCanvasYamlFromWorkflow(workflow);
    expect(yamlText).toMatch(/['"]y['"]: 200/);

    const spec = parseCanvasYamlToSpec(yamlText);
    expect(spec?.nodes?.[0]?.position).toEqual({ x: 500, y: 200 });
  });

  it("normalizes snake_case aliases without keeping duplicate fields", () => {
    const yamlText = `apiVersion: v1
kind: Canvas
metadata:
  name: Alias test
spec:
  nodes:
    - id: node-1
      name: Node 1
      component: noop
      is_collapsed: true
      position:
        x: 120
        y: 80
  edges:
    - source_id: node-1
      target_id: node-1
`;

    const spec = parseCanvasYamlToSpec(yamlText);
    expect(spec?.nodes?.[0]?.isCollapsed).toBe(true);
    expect(spec?.edges?.[0]?.sourceId).toBe("node-1");
    expect(spec?.edges?.[0]?.targetId).toBe("node-1");
    expect("is_collapsed" in ((spec?.nodes?.[0] as Record<string, unknown>) || {})).toBe(false);
    expect("source_id" in ((spec?.edges?.[0] as Record<string, unknown>) || {})).toBe(false);
    expect("target_id" in ((spec?.edges?.[0] as Record<string, unknown>) || {})).toBe(false);

    const rebuilt = buildCanvasYamlFromWorkflow({
      metadata: { id: "id", name: "Alias test", description: "" },
      spec: { nodes: spec?.nodes ?? [], edges: spec?.edges ?? [] },
    });
    expect(rebuilt).not.toContain("is_collapsed:");
    expect(rebuilt).not.toContain("source_id:");
    expect(rebuilt).not.toContain("target_id:");
  });
});
