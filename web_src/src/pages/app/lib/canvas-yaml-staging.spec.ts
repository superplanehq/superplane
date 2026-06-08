import { describe, expect, it } from "vitest";

import type { CanvasesCanvas } from "@/api-client";

import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "./canvas-yaml-staging";

const sampleWorkflow: CanvasesCanvas = {
  metadata: {
    id: "canvas-123",
    name: "Deploy Pipeline",
    description: "Production deploy flow",
    isTemplate: false,
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

  it("defaults missing node and edge lists to empty arrays", () => {
    const yamlText = `apiVersion: v1
kind: Canvas
metadata:
  name: Empty
spec:
  changeManagement:
    enabled: true
`;

    expect(parseCanvasYamlToSpec(yamlText)).toEqual({
      nodes: [],
      edges: [],
      changeManagement: { enabled: true },
    });
  });

  it("preserves changeManagement when present", () => {
    const workflow: CanvasesCanvas = {
      ...sampleWorkflow,
      spec: {
        ...sampleWorkflow.spec!,
        changeManagement: { enabled: true },
      },
    };

    const spec = parseCanvasYamlToSpec(buildCanvasYamlFromWorkflow(workflow));
    expect(spec?.changeManagement).toEqual({ enabled: true });
  });
});
