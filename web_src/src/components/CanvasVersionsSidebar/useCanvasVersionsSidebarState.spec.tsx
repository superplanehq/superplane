import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useCanvasVersionsSidebarState } from "./useCanvasVersionsSidebarState";

describe("useCanvasVersionsSidebarState", () => {
  it("opens the versions sidebar by default", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState());

    expect(result.current.isVersionsSidebarOpen).toBe(true);
  });

  it("supports a collapsed default state", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState({ defaultOpen: false }));

    expect(result.current.isVersionsSidebarOpen).toBe(false);
  });

  it("toggles the open state without persisting it", () => {
    const { result } = renderHook(() => useCanvasVersionsSidebarState());

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(false);

    act(() => {
      result.current.handleVersionsSidebarToggle();
    });
    expect(result.current.isVersionsSidebarOpen).toBe(true);

    expect(localStorage.getItem("canvasVersionsSidebarOpen")).toBeNull();
  });
});
