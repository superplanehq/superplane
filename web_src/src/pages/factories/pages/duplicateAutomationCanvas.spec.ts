import { describe, expect, it, vi, type Mock } from "vitest";

import {
  consoleYamlHasContent,
  duplicateAutomationCanvas,
  rematerializeDuplicateConsoleYaml,
  rewriteSelfCanvasRefs,
  type DuplicateAutomationCanvasDeps,
} from "./duplicateAutomationCanvas";

type TestDuplicateDeps = DuplicateAutomationCanvasDeps & {
  createCanvas: Mock;
  describeCanvas?: Mock;
  fetchConsoleYaml: Mock;
  putCanvasStaging: Mock;
  commitCanvasStaging: Mock;
};

function baseDeps(overrides: Partial<TestDuplicateDeps> = {}): TestDuplicateDeps {
  return {
    factoryId: "factory-1",
    app: { id: "canvas-source", name: "Refund Planner", description: "Plans refunds" },
    createCanvas: vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new" } } },
    }),
    fetchConsoleYaml: vi.fn().mockResolvedValue(undefined),
    putCanvasStaging: vi.fn().mockResolvedValue({}),
    commitCanvasStaging: vi.fn().mockResolvedValue({}),
    ...overrides,
  };
}

describe("rewriteSelfCanvasRefs", () => {
  it("rewrites runApp configuration.app and metadata.app to the clone id", () => {
    const rewritten = rewriteSelfCanvasRefs(
      [
        {
          id: "run-self",
          name: "Run Self",
          component: "runApp",
          configuration: { app: "canvas-source", node: "onrun-1" },
          metadata: { app: { id: "canvas-source", name: "Refund Planner" } },
        },
        {
          id: "run-other",
          name: "Run Other",
          component: "runApp",
          configuration: { app: "other-canvas", node: "onrun-2" },
          metadata: { app: { id: "other-canvas", name: "Other" } },
        },
      ],
      "canvas-source",
      "canvas-new",
      "Refund Planner copy",
    );

    expect(rewritten[0]?.configuration).toEqual({ app: "canvas-new", node: "onrun-1" });
    expect(rewritten[0]?.metadata).toEqual({
      app: { id: "canvas-new", name: "Refund Planner copy" },
    });
    expect(rewritten[1]?.configuration).toEqual({ app: "other-canvas", node: "onrun-2" });
  });
});

describe("rematerializeDuplicateConsoleYaml", () => {
  it("rewrites console metadata name and canvasId", () => {
    const next = rematerializeDuplicateConsoleYaml(
      "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\n  canvasId: canvas-source\nspec:\n  panels: []\n",
      "canvas-new",
      "Refund Planner copy",
    );
    expect(next).toContain("name: Refund Planner copy");
    expect(next).toContain("canvasId: canvas-new");
    expect(next).not.toContain("canvas-source");
  });
});

describe("consoleYamlHasContent", () => {
  it("treats empty panels/layout console shells as absent", () => {
    expect(
      consoleYamlHasContent(
        "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\n  canvasId: canvas-source\nspec:\n  panels: []\n  layout: []\n",
      ),
    ).toBe(false);
  });

  it("treats consoles with panels as present", () => {
    expect(
      consoleYamlHasContent(
        "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\nspec:\n  panels:\n    - id: p1\n      type: markdown\n      content:\n        body: hi\n  layout:\n    - i: p1\n      x: 0\n      'y': 0\n      w: 6\n      h: 4\n",
      ),
    ).toBe(true);
  });
});

describe("duplicateAutomationCanvas", () => {
  it("creates a canvas and stages the source live graph", async () => {
    const deps = baseDeps({
      describeCanvas: vi.fn().mockResolvedValue({
        data: {
          canvas: {
            metadata: { description: "from source" },
            spec: {
              nodes: [{ id: "n1", name: "Node 1", component: "noop", type: "TYPE_ACTION" }],
              edges: [{ sourceId: "n1", targetId: "n2", channel: "default" }],
            },
          },
        },
      }),
    });

    const canvasId = await duplicateAutomationCanvas(deps);

    expect(canvasId).toBe("canvas-new");
    expect(deps.describeCanvas).toHaveBeenCalledWith("canvas-source");
    expect(deps.createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy",
      description: "Plans refunds",
      factoryId: "factory-1",
      method: "ui",
    });
    expect(deps.putCanvasStaging).toHaveBeenCalledTimes(1);
    expect(deps.putCanvasStaging.mock.calls[0]?.[0]).toBe("canvas-new");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("id: canvas-new");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("name: Refund Planner copy");
    expect(String(deps.putCanvasStaging.mock.calls[0]?.[1])).toContain("n1");
    expect(deps.putCanvasStaging.mock.calls[0]?.[2]).toBeUndefined();
    expect(deps.commitCanvasStaging).toHaveBeenCalledWith("canvas-new");
  });

  it("rewrites self canvas refs and stages console.yaml when present", async () => {
    const deps = baseDeps({
      describeCanvas: vi.fn().mockResolvedValue({
        data: {
          canvas: {
            spec: {
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
            },
          },
        },
      }),
      fetchConsoleYaml: vi
        .fn()
        .mockResolvedValue(
          "apiVersion: v1\nkind: Console\nmetadata:\n  name: Source\n  canvasId: canvas-source\nspec:\n  panels:\n    - id: p1\n      type: markdown\n      content:\n        body: hi\n  layout:\n    - i: p1\n      x: 0\n      'y': 0\n      w: 6\n      h: 4\n",
        ),
    });

    await duplicateAutomationCanvas(deps);

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
      describeCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { spec: { nodes: [], edges: [] } } },
      }),
      app: { id: "canvas-empty", name: "Empty" },
    });

    const canvasId = await duplicateAutomationCanvas(deps);

    expect(canvasId).toBe("canvas-empty-copy");
    expect(deps.putCanvasStaging).not.toHaveBeenCalled();
    expect(deps.commitCanvasStaging).not.toHaveBeenCalled();
  });

  it("rejects when the source automation id is missing", async () => {
    await expect(
      duplicateAutomationCanvas(
        baseDeps({
          app: { name: "No Id" },
          createCanvas: vi.fn(),
        }),
      ),
    ).rejects.toThrow("source automation id is required");
  });

  it("reuses pendingCanvasId on retry and skips CreateCanvas", async () => {
    const onCanvasCreated = vi.fn();
    const deps = baseDeps({
      createCanvas: vi.fn(),
      pendingCanvasId: "canvas-pending",
      onCanvasCreated,
      describeCanvas: vi.fn().mockResolvedValue({
        data: {
          canvas: {
            spec: {
              nodes: [{ id: "n1", name: "Node 1", component: "noop", type: "TYPE_ACTION" }],
              edges: [],
            },
          },
        },
      }),
    });

    const canvasId = await duplicateAutomationCanvas(deps);

    expect(canvasId).toBe("canvas-pending");
    expect(deps.createCanvas).not.toHaveBeenCalled();
    expect(onCanvasCreated).not.toHaveBeenCalled();
    expect(deps.putCanvasStaging.mock.calls[0]?.[0]).toBe("canvas-pending");
    expect(deps.commitCanvasStaging).toHaveBeenCalledWith("canvas-pending");
  });

  it("notifies onCanvasCreated before a stage/commit failure", async () => {
    const onCanvasCreated = vi.fn();
    const deps = baseDeps({
      createCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { metadata: { id: "canvas-orphan" } } },
      }),
      onCanvasCreated,
      describeCanvas: vi.fn().mockResolvedValue({
        data: {
          canvas: {
            spec: {
              nodes: [{ id: "n1", name: "Node 1", component: "noop", type: "TYPE_ACTION" }],
              edges: [],
            },
          },
        },
      }),
      putCanvasStaging: vi.fn().mockRejectedValue(new Error("stage failed")),
    });

    await expect(duplicateAutomationCanvas(deps)).rejects.toThrow("stage failed");
    expect(onCanvasCreated).toHaveBeenCalledWith("canvas-orphan", "Refund Planner copy");
  });

  it("picks a unique name when the copy name is already taken", async () => {
    const createCanvas = vi.fn().mockResolvedValue({
      data: { canvas: { metadata: { id: "canvas-new-2" } } },
    });
    const deps = baseDeps({
      createCanvas,
      existingCanvasNames: ["Refund Planner copy"],
      describeCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { spec: { nodes: [], edges: [] } } },
      }),
    });

    const canvasId = await duplicateAutomationCanvas(deps);

    expect(canvasId).toBe("canvas-new-2");
    expect(createCanvas).toHaveBeenCalledTimes(1);
    expect(createCanvas).toHaveBeenCalledWith({
      name: "Refund Planner copy (2)",
      description: "Plans refunds",
      factoryId: "factory-1",
      method: "ui",
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
      describeCanvas: vi.fn().mockResolvedValue({
        data: { canvas: { spec: { nodes: [], edges: [] } } },
      }),
    });

    const canvasId = await duplicateAutomationCanvas(deps);

    expect(canvasId).toBe("canvas-new-3");
    expect(createCanvas).toHaveBeenNthCalledWith(1, {
      name: "Refund Planner copy",
      description: "Plans refunds",
      factoryId: "factory-1",
      method: "ui",
    });
    expect(createCanvas).toHaveBeenNthCalledWith(2, {
      name: "Refund Planner copy (2)",
      description: "Plans refunds",
      factoryId: "factory-1",
      method: "ui",
    });
  });
});
