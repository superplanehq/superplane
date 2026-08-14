import { describe, expect, it, vi } from "vitest";

import { duplicateAutomationCanvas } from "./duplicateAutomationCanvas";

describe("duplicateAutomationCanvas", () => {
  it("creates a canvas and stages the source live graph", async () => {
    const createCanvas = vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    });
    const describeCanvas = vi.fn().mockResolvedValue({
      data: {
        canvas: {
          metadata: { description: "from source" },
          spec: {
            nodes: [{ id: "n1", name: "Node 1", component: "noop", type: "TYPE_ACTION" }],
            edges: [{ sourceId: "n1", targetId: "n2", channel: "default" }],
          },
        },
      },
    });
    const putCanvasStaging = vi.fn().mockResolvedValue({});
    const commitCanvasStaging = vi.fn().mockResolvedValue({});

    const canvasId = await duplicateAutomationCanvas({
      factoryId: "factory-1",
      app: { id: "canvas-source", name: "Refund Planner", description: "Plans refunds" },
      createCanvas,
      describeCanvas,
      putCanvasStaging,
      commitCanvasStaging,
    });

    expect(canvasId).toBe("canvas-new");
    expect(describeCanvas).toHaveBeenCalledWith("canvas-source");
    expect(createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy",
      description: "Plans refunds",
      factoryId: "factory-1",
      method: "ui",
    });
    expect(putCanvasStaging).toHaveBeenCalledTimes(1);
    expect(putCanvasStaging.mock.calls[0]?.[0]).toBe("canvas-new");
    expect(String(putCanvasStaging.mock.calls[0]?.[1])).toContain("id: canvas-new");
    expect(String(putCanvasStaging.mock.calls[0]?.[1])).toContain("name: Refund Planner copy");
    expect(String(putCanvasStaging.mock.calls[0]?.[1])).toContain("n1");
    expect(commitCanvasStaging).toHaveBeenCalledWith("canvas-new");
  });

  it("skips staging when the source graph is empty", async () => {
    const createCanvas = vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-empty-copy" } } },
    });
    const putCanvasStaging = vi.fn();
    const commitCanvasStaging = vi.fn();

    const canvasId = await duplicateAutomationCanvas({
      factoryId: "factory-1",
      app: { id: "canvas-empty", name: "Empty" },
      createCanvas,
      describeCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { spec: { nodes: [], edges: [] } } },
      }),
      putCanvasStaging,
      commitCanvasStaging,
    });

    expect(canvasId).toBe("canvas-empty-copy");
    expect(putCanvasStaging).not.toHaveBeenCalled();
    expect(commitCanvasStaging).not.toHaveBeenCalled();
  });

  it("rejects when the source automation id is missing", async () => {
    await expect(
      duplicateAutomationCanvas({
        factoryId: "factory-1",
        app: { name: "No Id" },
        createCanvas: vi.fn(),
      }),
    ).rejects.toThrow("source automation id is required");
  });
});
