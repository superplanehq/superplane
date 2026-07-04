import { describe, expect, it, vi } from "vitest";

import { processCanvasLifecycleEvent } from "./canvas-version-lifecycle";

describe("processCanvasLifecycleEvent", () => {
  it("marks the canvas as remotely deleted", () => {
    const setCanvasDeletedRemotely = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_deleted",
      canvasId: "canvas-1",
      activeCanvasVersionId: "",
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasVersionData: vi.fn(),
      resyncDraftToCommitted: vi.fn(),
      setCanvasDeletedRemotely,
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(setCanvasDeletedRemotely).toHaveBeenCalledWith(true);
  });

  it("skips canvas_updated invalidation when the echo is consumed", () => {
    const invalidateCanvasVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      activeCanvasVersionId: "version-1",
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => true,
      invalidateCanvasVersionData,
      resyncDraftToCommitted: vi.fn(),
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(false);
    expect(invalidateCanvasVersionData).not.toHaveBeenCalled();
  });

  it("resyncs the active draft when canvas_updated arrives during an edit session", () => {
    const invalidateCanvasVersionData = vi.fn();
    const resyncDraftToCommitted = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      activeCanvasVersionId: "version-1",
      editSessionActive: true,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasVersionData,
      resyncDraftToCommitted,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(invalidateCanvasVersionData).toHaveBeenCalledWith("canvas-1", "version-1");
    expect(resyncDraftToCommitted).toHaveBeenCalledWith("version-1");
  });

  it("flags a pending remote update when local save activity is in flight", () => {
    const setRemoteCanvasUpdatePending = vi.fn();
    const invalidateCanvasVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      activeCanvasVersionId: "version-1",
      editSessionActive: true,
      hasLocalSaveActivity: true,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasVersionData,
      resyncDraftToCommitted: vi.fn(),
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending,
    });

    expect(result).toBe(true);
    expect(setRemoteCanvasUpdatePending).toHaveBeenCalledWith(true);
    expect(invalidateCanvasVersionData).toHaveBeenCalledWith("canvas-1", "version-1");
  });
});
