import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";

import { refreshCachesAfterCommit, syncCommittedCanvasDraftState } from "./sync-committed-canvas-draft";
import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

vi.mock("./repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    fetchCanvasConsoleData: vi.fn(),
  };
});

describe("syncCommittedCanvasDraftState", () => {
  it("reloads committed canvas spec into staged and detail caches", async () => {
    const committedVersion: CanvasesCanvasVersion = {
      metadata: { id: "version-1" },
      spec: {
        nodes: [{ id: "node-1", name: "Trigger", type: "TYPE_TRIGGER" }],
        edges: [],
      },
    };
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(committedVersion);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(result).toEqual(committedVersion);
    expect(fetchCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-1", "version-1", false);
    expect(setQueryData).toHaveBeenCalledWith(
      canvasKeys.versionStagedDetail("canvas-1", "version-1"),
      committedVersion,
    );
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDetail("canvas-1", "version-1"), committedVersion);
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionList("canvas-1"), expect.any(Function));
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.detail("org-1", "canvas-1"), expect.any(Function));

    const detailKey = JSON.stringify(canvasKeys.detail("org-1", "canvas-1"));
    const updateCanvasDetail = setQueryData.mock.calls.find(([key]) => JSON.stringify(key) === detailKey)?.[1] as (
      current: CanvasesCanvas | undefined,
    ) => CanvasesCanvas | undefined;

    expect(
      updateCanvasDetail({
        metadata: { id: "canvas-1" },
        spec: {
          nodes: [{ id: "node-2", name: "New Component", type: "TYPE_ACTION" }],
          edges: [],
        },
      }),
    ).toEqual({
      metadata: { id: "canvas-1" },
      spec: committedVersion.spec,
    });
  });
});

describe("refreshCachesAfterCommit", () => {
  it("invalidates draft caches when post-commit sync fails", async () => {
    vi.mocked(fetchCanvasVersionWithSpec).mockRejectedValue(new Error("network error"));

    const invalidateQueries = vi.fn().mockResolvedValue(undefined);
    const queryClient = { setQueryData: vi.fn(), invalidateQueries } as unknown as QueryClient;

    await expect(
      refreshCachesAfterCommit({
        queryClient,
        organizationId: "org-1",
        canvasId: "canvas-1",
        versionId: "version-1",
      }),
    ).resolves.toBeUndefined();

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.versionDetail("canvas-1", "version-1"),
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.consoleStaged("canvas-1", "version-1"),
    });
    expect(fetchCanvasConsoleData).not.toHaveBeenCalled();
  });
});
