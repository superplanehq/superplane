import { describe, expect, it } from "vitest";

import { makeCanvas } from "@/test/factories";

import { resolveCanvasForView, shouldSyncLoadedVersionToCanvasDetail } from "./resolve-canvas-for-view";

describe("resolveCanvasForView", () => {
  const liveCanvas = makeCanvas({
    metadata: { id: "canvas-1", name: "Live" },
    spec: { nodes: [{ id: "live-node" }], edges: [] },
  });
  const stagedSpec = { nodes: [{ id: "staged-node" }], edges: [] };

  it("renders staged draft spec while editing the live version", () => {
    const canvas = resolveCanvasForView({
      isEditing: true,
      isViewingCurrentLiveVersion: true,
      liveCanvas,
      draftSpecToRender: stagedSpec,
      selectedCanvasVersion: null,
      canvasId: "canvas-1",
    });

    expect(canvas?.spec).toEqual(stagedSpec);
  });

  it("keeps committed live canvas when not editing even if a version overlay is present", () => {
    const canvas = resolveCanvasForView({
      isEditing: false,
      isViewingCurrentLiveVersion: true,
      liveCanvas,
      draftSpecToRender: stagedSpec,
      selectedCanvasVersion: {
        metadata: { id: "live-version" },
        spec: stagedSpec,
      },
      canvasId: "canvas-1",
    });

    expect(canvas?.spec).toEqual(liveCanvas.spec);
  });

  it("keeps the live canvas visible while the staged draft is still loading in edit mode", () => {
    const canvas = resolveCanvasForView({
      isEditing: true,
      isViewingCurrentLiveVersion: true,
      liveCanvas,
      draftSpecToRender: null,
      selectedCanvasVersion: null,
      canvasId: "canvas-1",
    });

    expect(canvas?.spec).toEqual(liveCanvas.spec);
  });

  it("overlays a committed historical version when previewing outside edit mode", () => {
    const historicalSpec = { nodes: [{ id: "old-node" }], edges: [] };

    const canvas = resolveCanvasForView({
      isEditing: false,
      isViewingCurrentLiveVersion: false,
      liveCanvas,
      draftSpecToRender: null,
      selectedCanvasVersion: {
        metadata: { id: "old-version" },
        spec: historicalSpec,
      },
      canvasId: "canvas-1",
    });

    expect(canvas?.spec).toEqual(historicalSpec);
  });
});

describe("shouldSyncLoadedVersionToCanvasDetail", () => {
  it("skips syncing staged reads for the live version while editing", () => {
    expect(
      shouldSyncLoadedVersionToCanvasDetail({
        activeCanvasVersionId: "live-version",
        loadedCanvasVersionId: "live-version",
        hasLocalSaveActivity: false,
        isEditing: true,
        isViewingCurrentLiveVersion: true,
      }),
    ).toBe(false);
  });

  it("skips syncing version overlays onto the live canvas detail cache", () => {
    expect(
      shouldSyncLoadedVersionToCanvasDetail({
        activeCanvasVersionId: "live-version",
        loadedCanvasVersionId: "live-version",
        hasLocalSaveActivity: false,
        isEditing: false,
        isViewingCurrentLiveVersion: true,
      }),
    ).toBe(false);
  });

  it("allows syncing committed historical version previews", () => {
    expect(
      shouldSyncLoadedVersionToCanvasDetail({
        activeCanvasVersionId: "old-version",
        loadedCanvasVersionId: "old-version",
        hasLocalSaveActivity: false,
        isEditing: false,
        isViewingCurrentLiveVersion: false,
      }),
    ).toBe(true);
  });
});
