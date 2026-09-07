import { describe, expect, it, vi, type Mock } from "vitest";

import { duplicateCanvas, type DuplicateCanvasDeps } from "./duplicateCanvas";

type TestDeps = DuplicateCanvasDeps & {
  createCanvas: Mock;
  describeCanvas: Mock;
  fetchSourceSpec: Mock;
  fetchConsoleYaml: Mock;
  putCanvasStaging: Mock;
  commitCanvasStaging: Mock;
};

function baseDeps(overrides: Partial<TestDeps> = {}): TestDeps {
  return {
    sourceCanvasId: "canvas-source",
    sourceName: "Refund Planner",
    sourceDescription: "Plans refunds",
    createCanvas: vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    }),
    describeCanvas: vi.fn().mockResolvedValue({
      data: {
        canvas: {
          metadata: { description: "from source" },
          spec: { nodes: [], edges: [] },
        },
      },
    }),
    fetchSourceSpec: vi.fn().mockResolvedValue({
      nodes: [{ id: "n1", name: "Node 1", component: "noop" }],
      edges: [{ sourceId: "n1", targetId: "n2", channel: "default" }],
    }),
    fetchConsoleYaml: vi.fn().mockResolvedValue(undefined),
    putCanvasStaging: vi.fn().mockResolvedValue({}),
    commitCanvasStaging: vi.fn().mockResolvedValue({}),
    ...overrides,
  };
}

describe("duplicateCanvas", () => {
  it("creates a canvas and stages+commits the source graph", async () => {
    const deps = baseDeps();

    const canvasId = await duplicateCanvas(deps);

    expect(canvasId).toBe("canvas-new");
    expect(deps.describeCanvas).toHaveBeenCalledWith("canvas-source");
    expect(deps.fetchSourceSpec).toHaveBeenCalledWith("canvas-source");
    expect(deps.createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy",
      description: "Plans refunds",
    });
    expect(deps.putCanvasStaging).toHaveBeenCalledTimes(1);
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[0])).toBe("canvas-new");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("id: canvas-new");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("name: Refund Planner copy");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("n1");
    expect(deps.putCanvasStaging.mock.calls[0]?.[2]).toBeUndefined();
    expect(deps.commitCanvasStaging).toHaveBeenCalledWith("canvas-new");
  });

  it("uses sourceDescription over source canvas metadata description", async () => {
    const deps = baseDeps({
      sourceDescription: "custom description",
    });

    await duplicateCanvas(deps);

    expect(deps.createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy",
      description: "custom description",
    });
  });

  it("falls back to source canvas description when sourceDescription is not provided", async () => {
    const deps = baseDeps({
      sourceDescription: undefined,
    });

    await duplicateCanvas(deps);

    expect(deps.createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy",
      description: "from source",
    });
  });

  it("rewrites self canvas refs and stages console.yaml when present", async () => {
    const deps = baseDeps({
      fetchSourceSpec: vi.fn().mockResolvedValue({
        nodes: [
          {
            id: "run-self",
            name: "Run Self",
            component: "runApp",
            configuration: { app: "canvas-source", node: "onrun-1" },
            metadata: { app: { id: "canvas-source", name: "Refund Planner" } },
          },
        ],
        edges: [],
      }),
      fetchConsoleYaml: vi
        .fn()
        .mockResolvedValue(
          "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\n  canvasId: canvas-source\nspec:\n  panels:\n    - id: p1\n      type: markdown\n      content:\n        body: hi\n  layout:\n    - i: p1\n      x: 0\n      'y': 0\n      w: 6\n      h: 4\n",
        ),
    });

    await duplicateCanvas(deps);

    const stagedCanvasYaml = String(deps.putCanvasStaging.mock.calls[0]?.[1]);
    const stagedConsoleYaml = String(deps.putCanvasStaging.mock.calls[0]?.[2]);
    expect(stagedCanvasYaml).toContain("app: canvas-new");
    expect(stagedCanvasYaml).not.toContain("app: canvas-source");
    expect(stagedConsoleYaml).toContain("canvasId: canvas-new");
    expect(stagedConsoleYaml).toContain("name: Refund Planner copy");
  });

  it("skips staging when the source graph and console are empty", async () => {
    const deps = baseDeps({
      createCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { metadata: { id: "canvas-empty-copy" } } },
      }),
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
    });

    const canvasId = await duplicateCanvas(deps);

    expect(canvasId).toBe("canvas-empty-copy");
    expect(deps.putCanvasStaging).not.toHaveBeenCalled();
    expect(deps.commitCanvasStaging).not.toHaveBeenCalled();
  });

  it("stages when console.yaml has content even if graph is empty", async () => {
    const deps = baseDeps({
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
      fetchConsoleYaml: vi
        .fn()
        .mockResolvedValue(
          "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\nspec:\n  panels:\n    - id: p1\n      type: markdown\n      content:\n        body: hi\n  layout:\n    - i: p1\n      x: 0\n      'y': 0\n      w: 6\n      h: 4\n",
        ),
    });

    await duplicateCanvas(deps);

    expect(deps.putCanvasStaging).toHaveBeenCalledTimes(1);
    expect(deps.commitCanvasStaging).toHaveBeenCalledTimes(1);
  });

  it("picks a unique name when the copy name is already taken", async () => {
    const createCanvas = vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new-2" } } },
    });
    const deps = baseDeps({
      createCanvas,
      existingCanvasNames: ["Refund Planner copy"],
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
    });

    const canvasId = await duplicateCanvas(deps);

    expect(canvasId).toBe("canvas-new-2");
    expect(createCanvas).toHaveBeenCalledTimes(1);
    expect(createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy (2)",
      description: "Plans refunds",
    });
  });

  it("retries CreateCanvas when the server reports a name collision", async () => {
    const createCanvas = vi
      .fn()
      .mockRejectedValueOnce(new Error("Canvas with the same name already exists"))
      .mockResolvedValueOnce({
        data: { canvas: { metadata: { id: "canvas-new-3" } } },
      });
    const deps = baseDeps({
      createCanvas,
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
    });

    const canvasId = await duplicateCanvas(deps);

    expect(canvasId).toBe("canvas-new-3");
    expect(createCanvas).toHaveBeenNthCalledWith(1, {
      name: "Refund Planner copy",
      description: "Plans refunds",
    });
    expect(createCanvas).toHaveBeenNthCalledWith(2, {
      name: "Refund Planner copy (2)",
      description: "Plans refunds",
    });
  });

  it("propagates non-name-collision errors from createCanvas", async () => {
    const deps = baseDeps({
      createCanvas: vi.fn().mockRejectedValue(new Error("server down")),
    });

    await expect(duplicateCanvas(deps)).rejects.toThrow("server down");
  });

  it("propagates stage/commit errors", async () => {
    const deps = baseDeps({
      putCanvasStaging: vi.fn().mockRejectedValue(new Error("stage failed")),
    });

    await expect(duplicateCanvas(deps)).rejects.toThrow("stage failed");
    expect(deps.commitCanvasStaging).not.toHaveBeenCalled();
  });

  it("throws when createCanvas returns no ID", async () => {
    const deps = baseDeps({
      createCanvas: vi.fn().mockResolvedValue({ data: { canvas: { metadata: {} } } }),
    });

    await expect(duplicateCanvas(deps)).rejects.toThrow("Failed to create canvas");
  });

  it("reuses pending canvas id on retry (skips createCanvas)", async () => {
    const deps = baseDeps({
      pendingCanvasId: "canvas-pending",
      pendingCanvasName: "Refund Planner copy",
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
    });

    const canvasId = await duplicateCanvas(deps);

    expect(canvasId).toBe("canvas-pending");
    expect(deps.createCanvas).not.toHaveBeenCalled();
  });

  it("calls onCanvasCreated after creating a new canvas", async () => {
    const onCanvasCreated = vi.fn();
    const deps = baseDeps({
      onCanvasCreated,
      fetchSourceSpec: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
    });

    await duplicateCanvas(deps);

    expect(onCanvasCreated).toHaveBeenCalledWith("canvas-new", "Refund Planner copy");
  });

  it("handles missing console.yaml gracefully (404)", async () => {
    const deps = baseDeps({
      fetchConsoleYaml: vi.fn().mockResolvedValue(undefined),
    });

    await duplicateCanvas(deps);

    expect(deps.putCanvasStaging.mock.calls[0]?.[2]).toBeUndefined();
  });
});
