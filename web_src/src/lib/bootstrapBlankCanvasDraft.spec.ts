import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvas } from "@/api-client";

const {
  canvasesCreateDraftBranch,
  fetchCanvasRepositoryFileContent,
  putStaging,
  writeLastDraftBranch,
  buildCanvasYamlFromWorkflow,
  parseCanvasYamlToSpec,
} = vi.hoisted(() => ({
  canvasesCreateDraftBranch: vi.fn(),
  fetchCanvasRepositoryFileContent: vi.fn(),
  putStaging: vi.fn(),
  writeLastDraftBranch: vi.fn(),
  buildCanvasYamlFromWorkflow: vi.fn(),
  parseCanvasYamlToSpec: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api-client")>();
  return {
    ...actual,
    canvasesCreateDraftBranch,
  };
});

vi.mock("@/lib/withOrganizationHeader", () => ({
  withOrganizationHeader: (request: unknown) => request,
}));

vi.mock("@/hooks/useActiveDraftBranch", () => ({
  writeLastDraftBranch,
}));

vi.mock("@/pages/workflowv2/lib/canvas-repository-files", () => ({
  fetchCanvasRepositoryFileContent,
}));

vi.mock("@/pages/workflowv2/lib/canvas-yaml-staging", () => ({
  buildCanvasYamlFromWorkflow,
  parseCanvasYamlToSpec,
}));

vi.mock("@/lib/canvas-staging", () => ({
  CANVAS_YAML_PATH: "canvas.yaml",
  CONSOLE_YAML_PATH: "console.yaml",
  putStaging,
}));

import { bootstrapBlankCanvasDraft } from "./bootstrapBlankCanvasDraft";

describe("bootstrapBlankCanvasDraft", () => {
  const canvas: CanvasesCanvas = {
    metadata: { id: "canvas-1", name: "My App" },
    spec: { nodes: [], edges: [] },
  };

  beforeEach(() => {
    vi.clearAllMocks();
    canvasesCreateDraftBranch.mockResolvedValue({
      data: {
        branch: {
          branchName: "drafts/user-1",
          tipSha: "abc1234567890123456789012345678901234567890",
        },
      },
    });
    fetchCanvasRepositoryFileContent.mockImplementation(async (_canvasId, path) => {
      if (path === "canvas.yaml") {
        return "apiVersion: v1\nkind: Canvas\nspec:\n  nodes: []\n  edges: []";
      }
      return "";
    });
    parseCanvasYamlToSpec.mockReturnValue({ nodes: [], edges: [] });
    buildCanvasYamlFromWorkflow.mockReturnValue("staged-canvas-yaml");
    putStaging.mockResolvedValue(undefined);
  });

  it("creates a draft branch, stages the placeholder node, and returns the branch name", async () => {
    const branchName = await bootstrapBlankCanvasDraft(canvas);

    expect(branchName).toBe("drafts/user-1");
    expect(canvasesCreateDraftBranch).toHaveBeenCalledTimes(1);
    expect(putStaging).toHaveBeenCalledWith(
      expect.objectContaining({
        canvasId: "canvas-1",
        branch: "drafts/user-1",
        baseHeadSha: "abc1234567890123456789012345678901234567890",
        files: expect.objectContaining({
          "canvas.yaml": "staged-canvas-yaml",
        }),
      }),
    );
    expect(buildCanvasYamlFromWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        spec: expect.objectContaining({
          nodes: [
            expect.objectContaining({
              name: "New Component",
              type: "TYPE_ACTION",
            }),
          ],
        }),
      }),
    );
    expect(writeLastDraftBranch).toHaveBeenCalledWith("canvas-1", "drafts/user-1");
  });
});
