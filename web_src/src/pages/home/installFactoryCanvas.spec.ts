import { describe, expect, it, vi, type Mock } from "vitest";

import { ensureFactoryCanvas, type CreateFactoryCanvasFn } from "./installFactoryCanvas";
import type { FactoryDefinition } from "./factories";

const definition = {
  id: "pr-closure",
  title: "PR Closure",
  description: "Closes work orders when their pull request merges",
} as FactoryDefinition;

type CreateCanvasMock = CreateFactoryCanvasFn & Mock;

function createCanvasMock(response: unknown): CreateCanvasMock {
  return vi.fn().mockResolvedValue(response) as unknown as CreateCanvasMock;
}

function ensureCanvas(createCanvas: CreateCanvasMock) {
  return ensureFactoryCanvas({
    pending: null,
    definition,
    workspaceFactoryId: "factory-1",
    createCanvas,
    updateCanvasFolderMembership: vi.fn(),
  });
}

describe("ensureFactoryCanvas", () => {
  it("lets the server pick a free name and keeps the name it assigned", async () => {
    const createCanvas = createCanvasMock({
      data: { canvas: { metadata: { id: "canvas-1", name: "PR Closure (21)" } } },
    });

    const created = await ensureCanvas(createCanvas);

    expect(createCanvas).toHaveBeenCalledTimes(1);
    expect(createCanvas).toHaveBeenCalledWith({
      name: "PR Closure",
      description: definition.description,
      factoryId: "factory-1",
      uniqueName: true,
      method: "ui",
    });
    expect(created).toEqual({ canvasId: "canvas-1", canvasName: "PR Closure (21)" });
  });

  it("falls back to the title when the response carries no name", async () => {
    const createCanvas = createCanvasMock({ data: { canvas: { metadata: { id: "canvas-1" } } } });

    await expect(ensureCanvas(createCanvas)).resolves.toEqual({
      canvasId: "canvas-1",
      canvasName: "PR Closure",
    });
  });

  it("fails when the response carries no canvas id", async () => {
    const createCanvas = createCanvasMock({ data: {} });

    await expect(ensureCanvas(createCanvas)).rejects.toThrow("Failed to create factory canvas");
  });
});
