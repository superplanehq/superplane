import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { syncCanvasDraftState } from "./sync-canvas-draft";
import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

vi.mock("./repository-spec-files", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./repository-spec-files")>();
  return {
    ...actual,
    fetchCanvasVersionWithSpec: vi.fn(),
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
        panels: [{ id: "panel-1", type: "markdown", content: { title: "Hello" } }],
        layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      },
    };
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(version);

    const setQueryData = vi.fn();
    const invalidateQueries = vi.fn();
    const queryClient = { setQueryData, invalidateQueries } as unknown as QueryClient;

    const result = await syncCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(result).toEqual(version);
    expect(fetchCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-1", "version-1");
    expect(setQueryData).toHaveBeenCalledWith(
      canvasKeys.canvasStaging("canvas-1"),
      expect.objectContaining({
        stagingSummary: { hasStaging: false, stagedPaths: [] },
        spec: version.spec,
      }),
    );
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.version("canvas-1", "version-1"), version);
    expect(setQueryData).toHaveBeenCalledTimes(3);
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

  it("updates the version cache for the requested version id", async () => {
    const version: CanvasesCanvasVersion = {
      metadata: { id: "version-2" },
      spec: { nodes: [], edges: [], panels: [], layout: [] },
    };
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(version);

    const setQueryData = vi.fn();
    const invalidateQueries = vi.fn();
    const queryClient = { setQueryData, invalidateQueries } as unknown as QueryClient;

    await syncCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-2",
    });

    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.version("canvas-1", "version-2"), version);
  });
});
