import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  REFINE_LAYOUT_STORAGE_KEY,
  readStoredRefineLayout,
  useRefineLayoutPreference,
} from "./refineLayoutPreference";

describe("refineLayoutPreference", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to Clarity open and the plan closed", () => {
    expect(readStoredRefineLayout()).toEqual({ clarityExpanded: true, planOpen: false });

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.clarityExpanded).toBe(true);
    expect(result.current.planOpen).toBe(false);
  });

  it("reads a stored layout on init", () => {
    window.localStorage.setItem(
      REFINE_LAYOUT_STORAGE_KEY,
      JSON.stringify({ clarityExpanded: false, planOpen: true }),
    );

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.clarityExpanded).toBe(false);
    expect(result.current.planOpen).toBe(true);
  });

  it("keeps the other field when only one value is valid", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: false, planOpen: "yes" }));

    expect(readStoredRefineLayout()).toEqual({ clarityExpanded: false, planOpen: false });
  });

  it("falls back when the stored value is not JSON", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, "not-json");
    expect(readStoredRefineLayout()).toEqual({ clarityExpanded: true, planOpen: false });
  });

  it("toggles and persists each pane", () => {
    const { result } = renderHook(() => useRefineLayoutPreference());

    act(() => {
      result.current.toggleClarity();
    });
    expect(result.current.clarityExpanded).toBe(false);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      clarityExpanded: false,
      planOpen: false,
    });

    act(() => {
      result.current.togglePlan();
    });
    expect(result.current.planOpen).toBe(true);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      clarityExpanded: false,
      planOpen: true,
    });
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
