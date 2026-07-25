import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { canvasesDescribeCanvasVersion } from "@/api-client";

import { matchesCommittedCanvasSpec, matchesCommittedCanvasYaml, matchesCommittedConsoleYaml } from "./staging-content-match";
import { dematerializeCanvasSpec } from "./workflow-spec-files";

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api-client")>();
  return {
    ...actual,
    canvasesDescribeCanvasVersion: vi.fn(),
  };
});

const fetchCanvasVersionWithSpecMock = vi.hoisted(() => vi.fn());

vi.mock("./repository-spec-files", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./repository-spec-files")>();
  return {
    ...actual,
    fetchCanvasVersionWithSpec: fetchCanvasVersionWithSpecMock,
  };
});

const emptyConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels: []\n  layout: []\n";

const sampleCanvasYaml = `apiVersion: v1
kind: Canvas
metadata:
  name: demo
spec:
  nodes:
    - id: node-1
      name: Trigger
      type: trigger
  edges: []
`;

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
    vi.mocked(canvasesDescribeCanvasVersion).mockResolvedValue({
      data: {
        version: {
          metadata: { id: "version-1" },
          spec: { panels: [], layout: [] },
        },
      },
      error: undefined,
    });
    fetchCanvasVersionWithSpecMock.mockResolvedValue({
      metadata: { id: "version-1" },
      spec: dematerializeCanvasSpec(sampleCanvasYaml),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("treats semantically identical canvas specs as committed", () => {
    expect(
      matchesCommittedCanvasSpec(
        { metadata: { id: "version-1" }, spec: dematerializeCanvasSpec(sampleCanvasYaml) },
        reorderedCanvasYaml,
      ),
    ).toBe(true);
  });

  it("loads committed canvas through DescribeCanvasVersion", async () => {
    await expect(matchesCommittedCanvasYaml("canvas-1", "version-1", reorderedCanvasYaml)).resolves.toBe(true);
    expect(fetchCanvasVersionWithSpecMock).toHaveBeenCalledWith("canvas-1", "version-1");
  });

  it("treats semantically identical console yaml as committed", async () => {
    await expect(matchesCommittedConsoleYaml("canvas-1", "version-1", emptyConsoleYaml)).resolves.toBe(true);
  });
});
