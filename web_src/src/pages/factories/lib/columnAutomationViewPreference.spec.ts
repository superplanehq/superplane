import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  COLUMN_AUTOMATION_VIEW_STORAGE_KEY,
  isColumnAutomationView,
  readStoredColumnAutomationView,
  useColumnAutomationViewPreference,
} from "./columnAutomationViewPreference";

describe("columnAutomationViewPreference", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to names when nothing is stored", () => {
    expect(readStoredColumnAutomationView()).toBe("names");
  });

  it("reads a stored icons view", () => {
    window.localStorage.setItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY, "icons");
    expect(readStoredColumnAutomationView()).toBe("icons");
  });

  it("falls back to names when the stored value is invalid", () => {
    window.localStorage.setItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY, "compact");
    expect(readStoredColumnAutomationView()).toBe("names");
    expect(isColumnAutomationView("compact")).toBe(false);
  });

  it("stores icons only after an explicit switch", () => {
    const { result } = renderHook(() => useColumnAutomationViewPreference());

    expect(result.current.view).toBe("names");
    expect(window.localStorage.getItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY)).toBeNull();

    act(() => {
      result.current.setView("icons");
    });
    expect(result.current.view).toBe("icons");
    expect(window.localStorage.getItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY)).toBe("icons");

    act(() => {
      result.current.setView("names");
    });
    expect(result.current.view).toBe("names");
    expect(window.localStorage.getItem(COLUMN_AUTOMATION_VIEW_STORAGE_KEY)).toBeNull();
  });
});
