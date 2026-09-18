import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";

import {
  LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY,
  isLineBoardColumnColorView,
  readStoredLineBoardColumnColorView,
  useLineBoardColumnColorViewPreference,
} from "./lineBoardColumnColorViewPreference";

describe("lineBoardColumnColorViewPreference", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to fill when nothing is stored", () => {
    expect(readStoredLineBoardColumnColorView()).toBe("fill");
  });

  it("reads a stored borders view", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "borders");
    expect(readStoredLineBoardColumnColorView()).toBe("borders");
  });

  it("falls back to fill when the stored value is invalid", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "pastel");
    expect(readStoredLineBoardColumnColorView()).toBe("fill");
    expect(isLineBoardColumnColorView("pastel")).toBe(false);
    expect(isLineBoardColumnColorView("borders")).toBe(true);
  });

  it("stores a non-default view and clears storage when fill is selected", () => {
    const { result } = renderHook(() => useLineBoardColumnColorViewPreference());

    expect(result.current.view).toBe("fill");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBeNull();

    act(() => {
      result.current.setView("off");
    });
    expect(result.current.view).toBe("off");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBe("off");

    act(() => {
      result.current.setView("fill");
    });
    expect(result.current.view).toBe("fill");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBeNull();
  });
});
