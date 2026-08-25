import { describe, expect, it, vi } from "vitest";

import { createIntakeAutomation } from "./createIntakeAutomation";

describe("createIntakeAutomation", () => {
  it("creates a factory app and commits the selected intake template", async () => {
    const createCanvas = vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-1" } } },
    });
    const stageCanvas = vi.fn().mockResolvedValue(undefined);
    const commitCanvas = vi.fn().mockResolvedValue(undefined);

    const canvasId = await createIntakeAutomation({
      factoryId: "factory-1",
      sourceId: "sentry-exceptions",
      confidencePct: 70,
      createCanvas,
      stageCanvas,
      commitCanvas,
    });

    expect(canvasId).toBe("canvas-1");
    expect(createCanvas).toHaveBeenCalledWith({
      name: "Sentry exceptions",
      description: "Analyze new Sentry exceptions and create work orders for suitable fixes.",
      factoryId: "factory-1",
      method: "template",
    });
    expect(stageCanvas).toHaveBeenCalledWith("canvas-1", expect.stringContaining("component: sentry.onIssue"));
    expect(commitCanvas).toHaveBeenCalledWith("canvas-1");
  });

  it("does not stage a template when canvas creation has no id", async () => {
    const stageCanvas = vi.fn();
    const commitCanvas = vi.fn();

    await expect(
      createIntakeAutomation({
        factoryId: "factory-1",
        sourceId: "github-issues",
        confidencePct: 65,
        createCanvas: vi.fn().mockResolvedValue({ data: {} }),
        stageCanvas,
        commitCanvas,
      }),
    ).rejects.toThrow("Failed to create intake automation");
    expect(stageCanvas).not.toHaveBeenCalled();
    expect(commitCanvas).not.toHaveBeenCalled();
  });

  it("deletes the empty canvas when staging fails", async () => {
    const deleteCanvas = vi.fn().mockResolvedValue(undefined);
    const commitCanvas = vi.fn();

    await expect(
      createIntakeAutomation({
        factoryId: "factory-1",
        sourceId: "github-issues",
        confidencePct: 65,
        createCanvas: vi.fn().mockResolvedValue({
          data: { canvas: { metadata: { id: "canvas-1" } } },
        }),
        stageCanvas: vi.fn().mockRejectedValue(new Error("stage failed")),
        commitCanvas,
        deleteCanvas,
      }),
    ).rejects.toThrow("stage failed");
    expect(deleteCanvas).toHaveBeenCalledWith("canvas-1");
    expect(commitCanvas).not.toHaveBeenCalled();
  });

  it("picks a free name when the intake name is taken", async () => {
    const createCanvas = vi
      .fn()
      .mockRejectedValueOnce(new Error("Canvas with the same name already exists"))
      .mockResolvedValueOnce({ data: { canvas: { metadata: { id: "canvas-2" } } } });

    const canvasId = await createIntakeAutomation({
      factoryId: "factory-1",
      sourceId: "github-issues",
      confidencePct: 65,
      createCanvas,
      stageCanvas: vi.fn().mockResolvedValue(undefined),
      commitCanvas: vi.fn().mockResolvedValue(undefined),
    });

    expect(canvasId).toBe("canvas-2");
    expect(createCanvas).toHaveBeenNthCalledWith(1, expect.objectContaining({ name: "GitHub issues" }));
    expect(createCanvas).toHaveBeenNthCalledWith(2, expect.objectContaining({ name: "GitHub issues (2)" }));
  });

  it("skips known names before calling the API", async () => {
    const createCanvas = vi.fn().mockResolvedValue({ data: { canvas: { metadata: { id: "canvas-3" } } } });

    await createIntakeAutomation({
      factoryId: "factory-1",
      sourceId: "github-issues",
      confidencePct: 65,
      createCanvas,
      existingCanvasNames: ["GitHub issues"],
      stageCanvas: vi.fn().mockResolvedValue(undefined),
      commitCanvas: vi.fn().mockResolvedValue(undefined),
    });

    expect(createCanvas).toHaveBeenCalledWith(expect.objectContaining({ name: "GitHub issues (2)" }));
  });

  it("deletes the canvas when the commit fails", async () => {
    const deleteCanvas = vi.fn().mockResolvedValue(undefined);

    await expect(
      createIntakeAutomation({
        factoryId: "factory-1",
        sourceId: "github-issues",
        confidencePct: 65,
        createCanvas: vi.fn().mockResolvedValue({
          data: { canvas: { metadata: { id: "canvas-1" } } },
        }),
        stageCanvas: vi.fn().mockResolvedValue(undefined),
        commitCanvas: vi.fn().mockRejectedValue(new Error("commit failed")),
        deleteCanvas,
      }),
    ).rejects.toThrow("commit failed");
    expect(deleteCanvas).toHaveBeenCalledWith("canvas-1");
  });
});
