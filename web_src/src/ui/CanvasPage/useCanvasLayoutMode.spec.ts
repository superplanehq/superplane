import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  CANVAS_LAYOUT_MODE_STORAGE_KEY,
  isCanvasLayoutMode,
  readStoredCanvasLayoutMode,
  useCanvasLayoutMode,
} from "./useCanvasLayoutMode";

describe("useCanvasLayoutMode", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to freeform when nothing is stored", () => {
    const { result } = renderHook(() => useCanvasLayoutMode());
    expect(result.current.layoutMode).toBe("freeform");
  });

  it("reads a persisted vertical preference", () => {
    window.localStorage.setItem(CANVAS_LAYOUT_MODE_STORAGE_KEY, "vertical");
    const { result } = renderHook(() => useCanvasLayoutMode());
    expect(result.current.layoutMode).toBe("vertical");
  });

  it("toggles between freeform and vertical and persists the choice", () => {
    const { result } = renderHook(() => useCanvasLayoutMode());

    act(() => result.current.toggleLayoutMode());
    expect(result.current.layoutMode).toBe("vertical");
    expect(window.localStorage.getItem(CANVAS_LAYOUT_MODE_STORAGE_KEY)).toBe("vertical");

    act(() => result.current.toggleLayoutMode());
    expect(result.current.layoutMode).toBe("freeform");
    expect(window.localStorage.getItem(CANVAS_LAYOUT_MODE_STORAGE_KEY)).toBe("freeform");
  });

  it("sets a specific mode and persists it", () => {
    const { result } = renderHook(() => useCanvasLayoutMode());
    act(() => result.current.setLayoutMode("vertical"));
    expect(result.current.layoutMode).toBe("vertical");
    expect(readStoredCanvasLayoutMode()).toBe("vertical");
  });

  it("ignores invalid stored values", () => {
    window.localStorage.setItem(CANVAS_LAYOUT_MODE_STORAGE_KEY, "diagonal");
    expect(readStoredCanvasLayoutMode()).toBe("freeform");
    expect(isCanvasLayoutMode("diagonal")).toBe(false);
    expect(isCanvasLayoutMode("vertical")).toBe(true);
  });
});
