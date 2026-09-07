import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { saveLastLocation } = vi.hoisted(() => ({
  saveLastLocation: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("./useLastLocation", () => ({ saveLastLocation }));

import { readLastVisitedLocation } from "@/lib/lastVisitedLocation";
import { useRecordLastLocation } from "./useRecordLastLocation";

describe("useRecordLastLocation", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    window.localStorage.clear();
    saveLastLocation.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("records the path to local storage immediately", () => {
    renderHook(() => useRecordLastLocation("acme", "account-1", "/acme/apps/deploy?run=1"));

    expect(readLastVisitedLocation("account-1", "acme")).toBe("/acme/apps/deploy?run=1");
    expect(saveLastLocation).not.toHaveBeenCalled();
  });

  it("saves to the backend after the debounce window", () => {
    renderHook(() => useRecordLastLocation("acme", "account-1", "/acme/apps/deploy?run=1"));

    vi.runAllTimers();

    expect(saveLastLocation).toHaveBeenCalledWith("acme", "/acme/apps/deploy?run=1");
  });

  it("debounces rapid path changes into a single backend save", () => {
    const { rerender } = renderHook(({ path }) => useRecordLastLocation("acme", "account-1", path), {
      initialProps: { path: "/acme/apps/deploy?run=1" },
    });

    rerender({ path: "/acme/apps/deploy?run=1&node=approve-1" });
    rerender({ path: "/acme/apps/deploy?run=1&node=approve-2" });

    vi.runAllTimers();

    expect(saveLastLocation).toHaveBeenCalledTimes(1);
    expect(saveLastLocation).toHaveBeenCalledWith("acme", "/acme/apps/deploy?run=1&node=approve-2");
    expect(readLastVisitedLocation("account-1", "acme")).toBe("/acme/apps/deploy?run=1&node=approve-2");
  });

  it("does nothing without an organization route or account id", () => {
    renderHook(() => useRecordLastLocation(null, "account-1", "/acme"));
    renderHook(() => useRecordLastLocation("acme", null, "/acme"));

    vi.runAllTimers();

    expect(saveLastLocation).not.toHaveBeenCalled();
  });

  it("flushes the latest path when the hook unmounts", () => {
    const { unmount } = renderHook(() => useRecordLastLocation("acme", "account-1", "/acme/apps/deploy?run=1"));

    unmount();

    expect(saveLastLocation).toHaveBeenCalledWith("acme", "/acme/apps/deploy?run=1");
  });
});
