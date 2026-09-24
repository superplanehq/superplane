import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  WORK_ORDER_FULL_PAGE_STORAGE_KEY,
  readStoredWorkOrderFullPage,
  useWorkOrderFullPagePreference,
} from "./workOrderFullPagePreference";

describe("workOrderFullPagePreference", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to the compact popup", () => {
    expect(readStoredWorkOrderFullPage()).toBe(false);

    const { result } = renderHook(() => useWorkOrderFullPagePreference());
    expect(result.current.fullPage).toBe(false);
  });

  it("reads a stored full-page preference on init", () => {
    window.localStorage.setItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY, "1");

    const { result } = renderHook(() => useWorkOrderFullPagePreference());
    expect(result.current.fullPage).toBe(true);
  });

  it("toggles and persists the preference", () => {
    const { result } = renderHook(() => useWorkOrderFullPagePreference());

    act(() => {
      result.current.toggleFullPage();
    });
    expect(result.current.fullPage).toBe(true);
    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("1");

    act(() => {
      result.current.toggleFullPage();
    });
    expect(result.current.fullPage).toBe(false);
    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("0");
  });

  it("keeps state when localStorage.setItem throws", () => {
    const spy = vi.spyOn(window.localStorage.__proto__, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });

    const { result } = renderHook(() => useWorkOrderFullPagePreference());
    act(() => {
      result.current.toggleFullPage();
    });

    expect(result.current.fullPage).toBe(true);
    spy.mockRestore();
  });
});
