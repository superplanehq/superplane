import { describe, expect, it } from "vitest";

import type { CanvasesCanvas } from "@/api-client";

import { resolveEditableWorkflowSnapshot } from "./editable-workflow-snapshot";

const stagedWorkflow = {
  metadata: { id: "canvas-1", name: "Canvas" },
  spec: { nodes: [{ id: "staged-note", name: "TAB1-DRAFT" }], edges: [] },
} as CanvasesCanvas;

const committedWorkflow = {
  metadata: { id: "canvas-1", name: "Canvas" },
  spec: { nodes: [], edges: [] },
} as CanvasesCanvas;

describe("resolveEditableWorkflowSnapshot", () => {
  it("prefers the rendered draft workflow while editing", () => {
    expect(
      resolveEditableWorkflowSnapshot({
        isEditing: true,
        renderedWorkflow: stagedWorkflow,
        detailWorkflow: committedWorkflow,
      }),
    ).toBe(stagedWorkflow);
  });

  it("uses the detail workflow in view mode", () => {
    expect(
      resolveEditableWorkflowSnapshot({
        isEditing: false,
        renderedWorkflow: stagedWorkflow,
        detailWorkflow: committedWorkflow,
      }),
    ).toBe(committedWorkflow);
  });

  it("falls back to the rendered workflow when detail is missing", () => {
    expect(
      resolveEditableWorkflowSnapshot({
        isEditing: false,
        renderedWorkflow: committedWorkflow,
        detailWorkflow: undefined,
      }),
    ).toBe(committedWorkflow);
  });
});
