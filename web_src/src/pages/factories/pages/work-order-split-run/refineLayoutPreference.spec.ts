import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { REFINE_LAYOUT_STORAGE_KEY, readStoredRefineLayout, useRefineLayoutPreference } from "./refineLayoutPreference";

describe("refineLayoutPreference", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to the plan closed", () => {
    expect(readStoredRefineLayout()).toEqual({ planOpen: false });

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.planOpen).toBe(false);
  });

  it("reads a stored layout on init", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ planOpen: true }));

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.planOpen).toBe(true);
  });

  it("ignores the legacy summary keys", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: false, planOpen: true }));
    expect(readStoredRefineLayout()).toEqual({ planOpen: true });

    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: "clarity", planOpen: false }));
    expect(readStoredRefineLayout()).toEqual({ planOpen: false });
  });

  it("falls back when the stored value is invalid", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ planOpen: "yes" }));
    expect(readStoredRefineLayout()).toEqual({ planOpen: false });

    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, "not-json");
    expect(readStoredRefineLayout()).toEqual({ planOpen: false });
  });

  it("persists the plan pane toggle", () => {
    const { result } = renderHook(() => useRefineLayoutPreference());

    act(() => {
      result.current.togglePlan();
    });
    expect(result.current.planOpen).toBe(true);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({ planOpen: true });

    act(() => {
      result.current.togglePlan();
    });
    expect(result.current.planOpen).toBe(false);
  });

  it("keeps state when localStorage.setItem throws", () => {
    const spy = vi.spyOn(window.localStorage.__proto__, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });

    const { result } = renderHook(() => useRefineLayoutPreference());
    act(() => {
      result.current.togglePlan();
    });

    expect(result.current.planOpen).toBe(true);
    spy.mockRestore();
  });
});
