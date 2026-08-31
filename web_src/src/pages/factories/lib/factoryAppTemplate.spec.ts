import { describe, expect, it } from "vitest";
import type { CanvasesCanvas } from "@/api-client";

import { hasFactoryAppDefaults, resolveFactoryAppTemplate } from "./factoryAppTemplate";

function canvasWith(nodes: NonNullable<NonNullable<CanvasesCanvas["spec"]>["nodes"]>): CanvasesCanvas {
  return { metadata: { id: "app-1" }, spec: { nodes, edges: [] } };
}

describe("resolveFactoryAppTemplate", () => {
  it.each([
    ["onrun-create-plan", "line-planning"],
    ["onrun-implement", "line-implementation"],
    ["on-pr-closed", "pr-closure"],
  ])("matches %s to %s", (nodeId, templateId) => {
    expect(resolveFactoryAppTemplate(canvasWith([{ id: nodeId }]))?.id).toBe(templateId);
  });

  it("returns null for an unknown canvas", () => {
    expect(resolveFactoryAppTemplate(canvasWith([{ id: "custom" }]))).toBeNull();
  });
});

describe("hasFactoryAppDefaults", () => {
  it("recognizes explicit backend template metadata", () => {
    expect(
      hasFactoryAppDefaults(
        canvasWith([
          {
            id: "entrypoint",
            metadata: { factoryTemplate: { id: "line-planning", version: 1 } },
          },
        ]),
      ),
    ).toBe(true);
  });

  it("recognizes generated factory intakes", () => {
    expect(hasFactoryAppDefaults(canvasWith([{ id: "trigger" }, { id: "create-work-order" }]))).toBe(true);
  });

  it("recognizes the generated Backlog scoring canvas", () => {
    expect(hasFactoryAppDefaults(canvasWith([{ id: "trigger", component: "onWorkOrder" }]))).toBe(true);
  });

  it("recognizes legacy backlog automations", () => {
    expect(hasFactoryAppDefaults(canvasWith([{ id: "on-issue-labeled" }, { id: "create-work-order" }]))).toBe(true);
  });

  it("rejects custom canvases", () => {
    expect(hasFactoryAppDefaults(canvasWith([{ id: "custom" }]))).toBe(false);
  });
});
