import type { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { syncCommittedCanvasDraftState } from "./sync-committed-canvas-draft";
import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

vi.mock("./repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: vi.fn(),
}));

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

    const updateCanvasDetail = setQueryData.mock.calls.find(
      ([key]) => key === canvasKeys.detail("org-1", "canvas-1"),
    )?.[1] as (current: CanvasesCanvas | undefined) => CanvasesCanvas | undefined;

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
