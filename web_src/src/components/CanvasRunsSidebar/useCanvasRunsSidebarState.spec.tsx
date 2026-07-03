import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useCanvasRunsSidebarState, writeCanvasRunsSidebarOpen } from "./useCanvasRunsSidebarState";

describe("useCanvasRunsSidebarState", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("opens the runs sidebar by default", () => {
    const { result } = renderHook(() => useCanvasRunsSidebarState());

    expect(result.current.isRunsSidebarOpen).toBe(true);
  });

  it("falls back to the legacy global preference when a canvas does not have one", () => {
    localStorage.setItem("canvasRunsSidebarOpen", "false");

    const { result } = renderHook(() => useCanvasRunsSidebarState("canvas-a"));

    expect(result.current.isRunsSidebarOpen).toBe(false);
  });

  it("restores persisted open state for the current canvas", () => {
    localStorage.setItem("canvasRunsSidebarOpen:canvas-a", "true");

    const { result } = renderHook(() => useCanvasRunsSidebarState("canvas-a"));

    expect(result.current.isRunsSidebarOpen).toBe(true);
  });

  it("persists toggled state per canvas", () => {
    const { result } = renderHook(() => useCanvasRunsSidebarState("canvas-a"));

    act(() => {
      result.current.handleRunsSidebarToggle();
    });

    expect(result.current.isRunsSidebarOpen).toBe(false);
    expect(localStorage.getItem("canvasRunsSidebarOpen:canvas-a")).toBe("false");
    expect(localStorage.getItem("canvasRunsSidebarOpen")).toBeNull();
  });

  it("does not leak open state across canvases", () => {
    localStorage.setItem("canvasRunsSidebarOpen:canvas-a", "true");
    localStorage.setItem("canvasRunsSidebarOpen:canvas-b", "false");

    const { result, rerender } = renderHook(
      ({ canvasId }: { canvasId: string }) => useCanvasRunsSidebarState(canvasId),
      {
        initialProps: { canvasId: "canvas-a" },
      },
    );

    expect(result.current.isRunsSidebarOpen).toBe(true);

    rerender({ canvasId: "canvas-b" });
    expect(result.current.isRunsSidebarOpen).toBe(false);

    rerender({ canvasId: "canvas-c" });
    expect(result.current.isRunsSidebarOpen).toBe(true);
  });

  it("keeps a new canvas closed when it has an explicit per-canvas preference", () => {
    localStorage.setItem("canvasRunsSidebarOpen", "true");
    writeCanvasRunsSidebarOpen("new-canvas", false);

    const { result } = renderHook(() => useCanvasRunsSidebarState("new-canvas"));

    expect(result.current.isRunsSidebarOpen).toBe(false);
  });
});
