import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import { canvasKeys } from "@/hooks/useCanvasData";

import { executeCommitStaging } from "./commit-staging-flow";

describe("executeCommitStaging", () => {
  it("exits edit before invalidating caches and clears local editor state", async () => {
    const committedVersion = {
      metadata: { id: "version-2" },
      spec: { nodes: [], edges: [] },
    };
    const commitCanvasStagingMutation = {
      mutateAsync: vi.fn().mockResolvedValue({ version: committedVersion }),
    };
    const draftCanvasSpecsRef = { current: new Map([["version-1", { nodes: [], edges: [] }]]) };
    const setDraftCanvasSpec = vi.fn();
    const setStagingResetNonce = vi.fn();
    const callOrder: string[] = [];
    const onCommittedVersionId = vi.fn(() => {
      callOrder.push("exit-edit");
    });
    const invalidateQueries = vi.fn().mockImplementation(async () => {
      callOrder.push("invalidate");
    });
    const setQueryData = vi.fn();
    const cancelQueries = vi.fn().mockResolvedValue(undefined);
    const removeQueries = vi.fn();
    const queryClient = { invalidateQueries, setQueryData, cancelQueries, removeQueries } as unknown as QueryClient;

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
      onCommittedVersionId,
    });

    expect(result).toBe(true);
    expect(onCommittedVersionId).toHaveBeenCalledWith("version-2");
    expect(callOrder[0]).toBe("exit-edit");
    expect(callOrder.indexOf("invalidate")).toBeGreaterThan(0);
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.detail("org-1", "canvas-1"), expect.any(Function));
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.detail("org-1", "canvas-1"),
      refetchType: "all",
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.console("canvas-1", undefined),
      refetchType: "all",
    });
    expect(draftCanvasSpecsRef.current.has("version-1")).toBe(false);
    expect(setDraftCanvasSpec).toHaveBeenCalledWith(null);
    expect(setStagingResetNonce).toHaveBeenCalled();
  });
});
