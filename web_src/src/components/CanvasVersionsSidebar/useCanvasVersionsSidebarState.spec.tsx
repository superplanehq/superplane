import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useCanvasVersionsSidebarState } from "./useCanvasVersionsSidebarState";

describe("useCanvasVersionsSidebarState", () => {
  it("keeps the versions sidebar collapsed by default", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState());

    expect(result.current.isVersionsSidebarOpen).toBe(false);
  });

  it("toggles the open state without persisting it", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState());

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(true);

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(false);

    expect(localStorage.getItem("canvasVersionsSidebarOpen")).toBeNull();
  });
});
