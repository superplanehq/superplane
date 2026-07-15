import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { useViewportLock, VIEWPORT_LOCK_STORAGE_KEY } from "@/hooks/useViewportLock";

describe("useViewportLock", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to unlocked when nothing is stored", () => {
    const { result } = renderHook(() => useViewportLock());
    expect(result.current[0]).toBe(false);
  });

  it("reads the persisted lock state on mount", () => {
    window.localStorage.setItem(VIEWPORT_LOCK_STORAGE_KEY, "true");
    const { result } = renderHook(() => useViewportLock());
    expect(result.current[0]).toBe(true);
  });

  it("toggles the lock and persists it so it survives a remount", () => {
    const first = renderHook(() => useViewportLock());
    expect(first.result.current[0]).toBe(false);

    act(() => first.result.current[1]());

    expect(first.result.current[0]).toBe(true);
    expect(window.localStorage.getItem(VIEWPORT_LOCK_STORAGE_KEY)).toBe("true");

    // A remount (as happens when switching canvas modes) restores the state.
    const second = renderHook(() => useViewportLock());
    expect(second.result.current[0]).toBe(true);
  });

  it("toggles back to unlocked", () => {
    window.localStorage.setItem(VIEWPORT_LOCK_STORAGE_KEY, "true");
    const { result } = renderHook(() => useViewportLock());

    act(() => result.current[1]());

    expect(result.current[0]).toBe(false);
    expect(window.localStorage.getItem(VIEWPORT_LOCK_STORAGE_KEY)).toBe("false");
  });

  it("stays in sync when another tab changes the stored preference", () => {
    const { result } = renderHook(() => useViewportLock());
    expect(result.current[0]).toBe(false);

    act(() => {
      window.localStorage.setItem(VIEWPORT_LOCK_STORAGE_KEY, "true");
      window.dispatchEvent(new StorageEvent("storage", { key: VIEWPORT_LOCK_STORAGE_KEY, newValue: "true" }));
    });

    expect(result.current[0]).toBe(true);
  });
});
