import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { canvasKeys } from "@/hooks/useCanvasData";

import { executeCommitStaging } from "./commit-staging-flow";
import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

vi.mock("./repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: vi.fn(),
}));

describe("executeCommitStaging", () => {
  it("returns true and clears local draft state when post-commit cache sync fails", async () => {
    vi.mocked(fetchCanvasVersionWithSpec).mockRejectedValue(new Error("sync failed"));

    const commitCanvasStagingMutation = { mutateAsync: vi.fn().mockResolvedValue({}) };
    const draftCanvasSpecsRef = { current: new Map([["version-1", { nodes: [], edges: [] }]]) };
    const setDraftCanvasSpec = vi.fn();
    const setStagingResetNonce = vi.fn();
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);
    const queryClient = { invalidateQueries } as unknown as QueryClient;

    const result = await executeCommitStaging({
      organizationId: "org-1",
      canvasId: "canvas-1",
      activeCanvasVersionId: "version-1",
      commitMessage: "Update workflow",
      queryClient,
      commitCanvasStagingMutation,
      consoleMutationGenerationRef: { current: 0 },
      draftCanvasSpecsRef,
      setDraftCanvasSpec,
      setStagingResetNonce,
      ensureVersionActionDraftReady: vi.fn().mockResolvedValue(true),
    });

    expect(result).toBe(true);
    expect(commitCanvasStagingMutation.mutateAsync).toHaveBeenCalledTimes(1);
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.versionDetail("canvas-1", "version-1"),
    });
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: canvasKeys.repository("canvas-1") });
    expect(draftCanvasSpecsRef.current.has("version-1")).toBe(false);
    expect(setDraftCanvasSpec).toHaveBeenCalledWith(null);
    expect(setStagingResetNonce).toHaveBeenCalled();
  });
});
