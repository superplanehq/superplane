import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";

import { syncCanvasDraftState, syncConsoleCaches } from "./sync-canvas-draft";
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

beforeEach(() => {
  vi.clearAllMocks();
});

describe("syncCanvasDraftState", () => {
  it("reloads canvas spec into staged and detail caches", async () => {
    const version: CanvasesCanvasVersion = {
      metadata: { id: "version-1" },
      spec: {
        nodes: [{ id: "node-1", name: "Trigger", type: "TYPE_TRIGGER" }],
        edges: [],
      },
    };
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(version);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(result).toEqual(version);
    expect(fetchCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-1", "version-1");
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.stagedCanvasSpec("canvas-1"), version);
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDetail("canvas-1", "version-1"), version);
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDescribe("canvas-1", "version-1"), version);
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
      spec: version.spec,
    });
  });

  it("updates the version describe cache for the requested version id", async () => {
    const version: CanvasesCanvasVersion = {
      metadata: { id: "version-2" },
      spec: { nodes: [], edges: [] },
    };
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(version);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    await syncCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-2",
    });

    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDescribe("canvas-1", "version-2"), version);
  });
});

describe("syncConsoleCaches", () => {
  it("invalidates staged console cache when console.yaml is missing or unparsable", async () => {
    vi.mocked(fetchCanvasConsoleData).mockResolvedValue(undefined);

    const invalidateQueries = vi.fn().mockResolvedValue(undefined);
    const setQueryData = vi.fn();
    const queryClient = { invalidateQueries, setQueryData } as unknown as QueryClient;

    await syncConsoleCaches({
      queryClient,
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(fetchCanvasConsoleData).toHaveBeenCalledWith("canvas-1", "version-1", false);
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.stagedConsole("canvas-1"),
    });
    expect(setQueryData).not.toHaveBeenCalled();
  });
});
