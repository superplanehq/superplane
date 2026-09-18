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

  it("defaults to dim when nothing is stored", () => {
    expect(readStoredLineBoardColumnColorView()).toBe("dim");
  });

  it("reads a stored borders view", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "borders");
    expect(readStoredLineBoardColumnColorView()).toBe("borders");
  });

  it("maps the previous fill view to dim", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "fill");
    expect(readStoredLineBoardColumnColorView()).toBe("dim");
  });

  it("falls back to dim when the stored value is invalid", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "pastel");
    expect(readStoredLineBoardColumnColorView()).toBe("dim");
    expect(isLineBoardColumnColorView("pastel")).toBe(false);
    expect(isLineBoardColumnColorView("borders")).toBe(true);
    expect(isLineBoardColumnColorView("vivid")).toBe(true);
  });

  it("stores a non-default view and clears storage when dim is selected", () => {
    const { result } = renderHook(() => useLineBoardColumnColorViewPreference());

    expect(result.current.view).toBe("dim");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBeNull();

    act(() => {
      result.current.setView("vivid");
    });
    expect(result.current.view).toBe("vivid");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBe("vivid");

    act(() => {
      result.current.setView("dim");
    });
    expect(result.current.view).toBe("dim");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBeNull();
  });
});
