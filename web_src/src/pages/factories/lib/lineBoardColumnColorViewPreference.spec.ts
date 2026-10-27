import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "bun:test";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useTheme } from "@/contexts/useTheme";
import { THEME_PREFERENCE_STORAGE_KEY } from "@/lib/themePreference";

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

  it.each([
    { theme: "light", expected: "vivid" },
    { theme: "dark", expected: "dim" },
  ])("defaults to $expected in $theme mode", ({ theme, expected }) => {
    window.localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, theme);
    const { result } = renderHook(() => useLineBoardColumnColorViewPreference(), { wrapper: ThemeProvider });
    expect(result.current.view).toBe(expected);
  });

  it("follows theme changes until a view is explicitly selected, then preserves the choice after remount", () => {
    window.localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, "light");
    const mount = () =>
      renderHook(() => ({ ...useLineBoardColumnColorViewPreference(), theme: useTheme() }), {
        wrapper: ThemeProvider,
      });
    const { result, unmount } = mount();
    act(() => result.current.theme.setPreference("dark"));
    expect(result.current.view).toBe("dim");
    act(() => result.current.theme.setPreference("light"));
    expect(result.current.view).toBe("vivid");
    act(() => result.current.setView("vivid"));
    act(() => result.current.theme.setPreference("dark"));
    expect(result.current.view).toBe("vivid");
    unmount();
    expect(mount().result.current.view).toBe("vivid");
  });

  it("leaves the default to the theme when nothing is stored", () => {
    expect(readStoredLineBoardColumnColorView()).toBeNull();
  });

  it("reads a stored borders view", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "borders");
    expect(readStoredLineBoardColumnColorView()).toBe("borders");
  });

  it("maps the previous fill view to dim", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "fill");
    expect(readStoredLineBoardColumnColorView()).toBe("dim");
  });

  it("leaves the default to the theme when the stored value is invalid", () => {
    window.localStorage.setItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY, "pastel");
    expect(readStoredLineBoardColumnColorView()).toBeNull();
    expect(isLineBoardColumnColorView("pastel")).toBe(false);
    expect(isLineBoardColumnColorView("borders")).toBe(true);
    expect(isLineBoardColumnColorView("vivid")).toBe(true);
  });

  it("stores explicit selections including vivid", () => {
    window.localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, "light");
    const { result } = renderHook(() => useLineBoardColumnColorViewPreference(), { wrapper: ThemeProvider });

    expect(result.current.view).toBe("vivid");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBeNull();

    act(() => {
      result.current.setView("dim");
    });
    expect(result.current.view).toBe("dim");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBe("dim");

    act(() => {
      result.current.setView("vivid");
    });
    expect(result.current.view).toBe("vivid");
    expect(window.localStorage.getItem(LINE_BOARD_COLUMN_COLOR_VIEW_STORAGE_KEY)).toBe("vivid");
  });
});
