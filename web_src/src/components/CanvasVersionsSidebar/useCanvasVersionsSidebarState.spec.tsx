import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useCanvasVersionsSidebarState } from "./useCanvasVersionsSidebarState";

describe("useCanvasVersionsSidebarState", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("keeps the versions sidebar closed by default", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState());

    expect(result.current.isVersionsSidebarOpen).toBe(false);
  });

  it("restores persisted open state for the current canvas", () => {
    localStorage.setItem("canvasVersionsSidebarOpen:canvas-a", "true");

    const { result } = renderHook(() => useCanvasVersionsSidebarState("canvas-a"));

    expect(result.current.isVersionsSidebarOpen).toBe(true);
  });

  it("persists toggled state per canvas", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState("canvas-a"));

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(true);
    expect(localStorage.getItem("canvasVersionsSidebarOpen:canvas-a")).toBe("true");
    expect(localStorage.getItem("canvasVersionsSidebarOpen")).toBeNull();

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(false);
    expect(localStorage.getItem("canvasVersionsSidebarOpen:canvas-a")).toBe("false");
  });

  it("does not leak open state across canvases", () => {
    localStorage.setItem("canvasVersionsSidebarOpen:canvas-a", "true");
    localStorage.setItem("canvasVersionsSidebarOpen:canvas-b", "false");

    const { result, rerender } = renderHook(
      ({ canvasId }: { canvasId: string }) => useCanvasVersionsSidebarState(canvasId),
      {
        initialProps: { canvasId: "canvas-a" },
      },
    );

    expect(result.current.isVersionsSidebarOpen).toBe(true);

    rerender({ canvasId: "canvas-b" });
    expect(result.current.isVersionsSidebarOpen).toBe(false);

    rerender({ canvasId: "canvas-c" });
    expect(result.current.isVersionsSidebarOpen).toBe(false);
  });
});
