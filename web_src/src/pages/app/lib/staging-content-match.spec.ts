import { beforeEach, describe, expect, it, vi } from "vitest";

import { canvasesDescribeCanvasVersion } from "@/api-client";

import {
  committedCanvasMatchesYaml,
  committedConsoleMatchesYaml,
  matchesCommittedCanvasYaml,
  matchesCommittedConsoleYaml,
} from "./staging-content-match";

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesDescribeCanvasVersion: vi.fn(),
  };
});

const emptyConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels: []\n  layout: []\n";

const reorderedCanvasYaml = `apiVersion: v1
kind: Canvas
metadata:
  name: demo
spec:
  edges: []
  nodes:
    - id: node-1
      name: Trigger
      type: trigger
`;

describe("staging-content-match", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(canvasesDescribeCanvasVersion).mockResolvedValue({
      data: {
        version: {
          metadata: { id: "version-1" },
          spec: {
            nodes: [{ id: "node-1", name: "Trigger", type: "trigger" as never }],
            edges: [],
            panels: [],
            layout: [],
          },
        },
      },
    } as never);
  });

  it("treats semantically identical canvas yaml as committed", async () => {
    await expect(matchesCommittedCanvasYaml("canvas-1", "version-1", reorderedCanvasYaml)).resolves.toBe(true);
    expect(
      committedCanvasMatchesYaml(
        {
          nodes: [{ id: "node-1", name: "Trigger", type: "trigger" as never }],
          edges: [],
        },
        reorderedCanvasYaml,
      ),
    ).toBe(true);
    expect(canvasesDescribeCanvasVersion).toHaveBeenCalled();
  });

  it("treats semantically identical console yaml as committed", async () => {
    await expect(matchesCommittedConsoleYaml("canvas-1", "version-1", emptyConsoleYaml)).resolves.toBe(true);
    expect(committedConsoleMatchesYaml("canvas-1", { panels: [], layout: [] }, emptyConsoleYaml)).toBe(true);
  });
});
