import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  CANVAS_AUTO_FOCUS_STORAGE_KEY,
  readStoredCanvasAutoFocusEnabled,
  useCanvasAutoFocusPreference,
} from "./useCanvasAutoFocusPreference";

describe("useCanvasAutoFocusPreference", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to enabled when nothing is stored", () => {
    const { result } = renderHook(() => useCanvasAutoFocusPreference());

    expect(result.current.isAutoFocusEnabled).toBe(true);
  });

  it("reads a persisted disabled value on init", () => {
    window.localStorage.setItem(CANVAS_AUTO_FOCUS_STORAGE_KEY, "false");

    const { result } = renderHook(() => useCanvasAutoFocusPreference());

    expect(result.current.isAutoFocusEnabled).toBe(false);
  });

  it("reads a persisted enabled value on init", () => {
    window.localStorage.setItem(CANVAS_AUTO_FOCUS_STORAGE_KEY, "true");

    const { result } = renderHook(() => useCanvasAutoFocusPreference());

    expect(result.current.isAutoFocusEnabled).toBe(true);
  });

  it("falls back to enabled when the stored value is not JSON", () => {
    window.localStorage.setItem(CANVAS_AUTO_FOCUS_STORAGE_KEY, "not-json");

    expect(readStoredCanvasAutoFocusEnabled()).toBe(true);
  });

  it("falls back to enabled when the stored value is not a boolean", () => {
    window.localStorage.setItem(CANVAS_AUTO_FOCUS_STORAGE_KEY, JSON.stringify("yes"));

    expect(readStoredCanvasAutoFocusEnabled()).toBe(true);
  });

  it("falls back to enabled when localStorage.getItem throws", () => {
    const spy = vi.spyOn(window.localStorage.__proto__, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });

    expect(readStoredCanvasAutoFocusEnabled()).toBe(true);
    spy.mockRestore();
  });

  it("toggles and persists the new value", () => {
    const { result } = renderHook(() => useCanvasAutoFocusPreference());

    act(() => {
      result.current.handleToggleAutoFocus();
    });

    expect(result.current.isAutoFocusEnabled).toBe(false);
    expect(window.localStorage.getItem(CANVAS_AUTO_FOCUS_STORAGE_KEY)).toBe("false");

    act(() => {
      result.current.handleToggleAutoFocus();
    });

    expect(result.current.isAutoFocusEnabled).toBe(true);
    expect(window.localStorage.getItem(CANVAS_AUTO_FOCUS_STORAGE_KEY)).toBe("true");
  });

  it("keeps state stable when localStorage.setItem throws", () => {
    const spy = vi.spyOn(window.localStorage.__proto__, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });

    const { result } = renderHook(() => useCanvasAutoFocusPreference());

    act(() => {
      result.current.handleToggleAutoFocus();
    });

    expect(result.current.isAutoFocusEnabled).toBe(false);
    spy.mockRestore();
  });
});
