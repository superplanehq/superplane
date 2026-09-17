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

  it("defaults to the Clarity summary open and the plan closed", () => {
    expect(readStoredRefineLayout()).toEqual({ openSummary: "clarity", planOpen: false });

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.openSummary).toBe("clarity");
    expect(result.current.planOpen).toBe(false);
  });

  it("reads a stored layout on init", () => {
    window.localStorage.setItem(
      REFINE_LAYOUT_STORAGE_KEY,
      JSON.stringify({ openSummary: "confidence", planOpen: true }),
    );

    const { result } = renderHook(() => useRefineLayoutPreference());
    expect(result.current.openSummary).toBe("confidence");
    expect(result.current.planOpen).toBe(true);
  });

  it("reads a stored closed summary", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: null, planOpen: false }));

    expect(readStoredRefineLayout()).toEqual({ openSummary: null, planOpen: false });
  });

  it("migrates the legacy clarityExpanded flag", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: false, planOpen: true }));
    expect(readStoredRefineLayout()).toEqual({ openSummary: null, planOpen: true });

    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: true, planOpen: false }));
    expect(readStoredRefineLayout()).toEqual({ openSummary: "clarity", planOpen: false });
  });

  it("keeps the other field when only one value is valid", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: "bogus", planOpen: "yes" }));

    expect(readStoredRefineLayout()).toEqual({ openSummary: "clarity", planOpen: false });
  });

  it("falls back when the stored value is not JSON", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, "not-json");
    expect(readStoredRefineLayout()).toEqual({ openSummary: "clarity", planOpen: false });
  });

  it("opens one summary at a time and persists each pane", () => {
    const { result } = renderHook(() => useRefineLayoutPreference());

    act(() => {
      result.current.toggleSummary("confidence");
    });
    expect(result.current.openSummary).toBe("confidence");
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      openSummary: "confidence",
      planOpen: false,
    });

    act(() => {
      result.current.toggleSummary("confidence");
    });
    expect(result.current.openSummary).toBeNull();

    act(() => {
      result.current.toggleSummary("clarity");
    });
    expect(result.current.openSummary).toBe("clarity");

    act(() => {
      result.current.togglePlan();
    });
    expect(result.current.planOpen).toBe(true);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      openSummary: "clarity",
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
