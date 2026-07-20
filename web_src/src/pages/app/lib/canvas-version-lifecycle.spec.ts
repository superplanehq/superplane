import { describe, expect, it, vi } from "vitest";

import { processCanvasLifecycleEvent } from "./canvas-version-lifecycle";

describe("processCanvasLifecycleEvent", () => {
  it("marks the canvas as remotely deleted", () => {
    const setCanvasDeletedRemotely = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_deleted",
      canvasId: "canvas-1",
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasStaging: vi.fn(),
      invalidateLiveVersionData: vi.fn(),
      setCanvasDeletedRemotely,
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(setCanvasDeletedRemotely).toHaveBeenCalledWith(true);
  });

  it("skips canvas_updated invalidation when the echo is consumed", () => {
    const invalidateCanvasStaging = vi.fn();
    const invalidateLiveVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => true,
      invalidateCanvasStaging,
      invalidateLiveVersionData,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(false);
    expect(invalidateCanvasStaging).not.toHaveBeenCalled();
    expect(invalidateLiveVersionData).not.toHaveBeenCalled();
  });

  it("refreshes staging metadata only when canvas_updated arrives during an edit session", () => {
    const invalidateCanvasStaging = vi.fn();
    const invalidateLiveVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      editSessionActive: true,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasStaging,
      invalidateLiveVersionData,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(invalidateCanvasStaging).toHaveBeenCalledWith("canvas-1");
    expect(invalidateLiveVersionData).not.toHaveBeenCalled();
  });

  it("refreshes live metadata when canvas_updated arrives outside an edit session", () => {
    const invalidateCanvasStaging = vi.fn();
    const invalidateLiveVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasStaging,
      invalidateLiveVersionData,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(invalidateLiveVersionData).toHaveBeenCalledWith("canvas-1");
    expect(invalidateCanvasStaging).not.toHaveBeenCalled();
  });

  it("flags a pending remote update when local save activity is in flight", () => {
    const setRemoteCanvasUpdatePending = vi.fn();
    const invalidateCanvasStaging = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1" },
      eventName: "canvas_updated",
      canvasId: "canvas-1",
      editSessionActive: true,
      hasLocalSaveActivity: true,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      invalidateCanvasStaging,
      invalidateLiveVersionData: vi.fn(),
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending,
    });

    expect(result).toBe(true);
    expect(setRemoteCanvasUpdatePending).toHaveBeenCalledWith(true);
    expect(invalidateCanvasStaging).toHaveBeenCalledWith("canvas-1");
  });
});
