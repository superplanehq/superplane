import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useCanvasRunsSidebarState } from "./useCanvasRunsSidebarState";

describe("useCanvasRunsSidebarState", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("keeps the runs sidebar closed by default", () => {
    const { result } = renderHook(() => useCanvasRunsSidebarState());

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

    expect(result.current.isRunsSidebarOpen).toBe(true);
    expect(localStorage.getItem("canvasRunsSidebarOpen:canvas-a")).toBe("true");
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
    expect(result.current.isRunsSidebarOpen).toBe(false);
  });
});
