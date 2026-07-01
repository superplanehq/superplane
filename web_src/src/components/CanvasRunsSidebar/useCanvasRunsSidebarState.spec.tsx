import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useCanvasRunsSidebarState } from "./useCanvasRunsSidebarState";

describe("useCanvasRunsSidebarState", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("opens the runs sidebar by default", () => {
    const { result } = renderHook(() => useCanvasRunsSidebarState());

    expect(result.current.isRunsSidebarOpen).toBe(true);
  });

  it("restores persisted open state", () => {
    localStorage.setItem("canvasRunsSidebarOpen", "false");

    const { result } = renderHook(() => useCanvasRunsSidebarState());

    expect(result.current.isRunsSidebarOpen).toBe(false);
  });

  it("persists toggled state", () => {
    const { result } = renderHook(() => useCanvasRunsSidebarState());

    act(() => {
      result.current.handleRunsSidebarToggle();
    });

    expect(result.current.isRunsSidebarOpen).toBe(false);
    expect(localStorage.getItem("canvasRunsSidebarOpen")).toBe("false");
  });
});
