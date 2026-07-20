import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";

import { syncCommittedCanvasDraftState, syncCommittedConsoleCaches } from "./sync-committed-canvas-draft";
import { fetchCommittedCanvasVersionWithSpec, fetchLiveCommittedCanvasVersionWithSpec } from "./repository-spec-files";

vi.mock("./repository-spec-files", () => ({
  fetchCommittedCanvasVersionWithSpec: vi.fn(),
  fetchLiveCommittedCanvasVersionWithSpec: vi.fn(),
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

describe("syncCommittedCanvasDraftState", () => {
  it("reloads committed canvas spec into staged and detail caches", async () => {
    const committedVersion: CanvasesCanvasVersion = {
      metadata: { id: "version-1" },
      spec: {
        nodes: [{ id: "node-1", name: "Trigger", type: "TYPE_TRIGGER" }],
        edges: [],
      },
    };
    vi.mocked(fetchCommittedCanvasVersionWithSpec).mockResolvedValue(committedVersion);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-1",
    });

    expect(result).toEqual(committedVersion);
    expect(fetchCommittedCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-1", "version-1");
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.stagedCanvasSpec("canvas-1"), committedVersion);
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

  it("prepends a new committed version when the version list cache is empty", async () => {
    const committedVersion: CanvasesCanvasVersion = {
      metadata: { id: "version-2" },
      spec: { nodes: [], edges: [] },
    };
    vi.mocked(fetchCommittedCanvasVersionWithSpec).mockResolvedValue(committedVersion);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "version-2",
    });

    const updateVersionList = setQueryData.mock.calls.find(
      ([key]) => JSON.stringify(key) === JSON.stringify(canvasKeys.versionList("canvas-1")),
    )?.[1] as (current: CanvasesCanvasVersion[] | undefined) => CanvasesCanvasVersion[] | undefined;

    expect(updateVersionList(undefined)).toEqual([committedVersion]);
  });

  it("loads the live committed version when the requested version id is stale", async () => {
    const committedVersion: CanvasesCanvasVersion = {
      metadata: { id: "live-version-2" },
      spec: {
        nodes: [{ id: "node-1", name: "Trigger", type: "TYPE_TRIGGER" }],
        edges: [],
      },
    };
    vi.mocked(fetchLiveCommittedCanvasVersionWithSpec).mockResolvedValue(committedVersion);

    const setQueryData = vi.fn();
    const queryClient = { setQueryData } as unknown as QueryClient;

    const result = await syncCommittedCanvasDraftState({
      queryClient,
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionId: "stale-version-1",
      resolveLiveVersion: true,
    });

    expect(result).toEqual(committedVersion);
    expect(fetchLiveCommittedCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-1");
    expect(fetchCommittedCanvasVersionWithSpec).not.toHaveBeenCalled();
    expect(setQueryData).toHaveBeenCalledWith(canvasKeys.versionDetail("canvas-1", "live-version-2"), committedVersion);
  });
});

describe("syncCommittedConsoleCaches", () => {
  it("invalidates staged console cache when committed console.yaml is missing or unparsable", async () => {
    vi.mocked(fetchCanvasConsoleData).mockResolvedValue(undefined);

    const invalidateQueries = vi.fn().mockResolvedValue(undefined);
    const setQueryData = vi.fn();
    const queryClient = { invalidateQueries, setQueryData } as unknown as QueryClient;

    await syncCommittedConsoleCaches({
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
