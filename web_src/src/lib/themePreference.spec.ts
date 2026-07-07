import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import {
  applyResolvedThemeToDocument,
  getSystemPrefersDark,
  readStoredThemePreference,
  resolveTheme,
  persistThemePreference,
  THEME_PREFERENCE_STORAGE_KEY,
} from "./themePreference";

describe("themePreference", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    document.documentElement.style.colorScheme = "";
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to system when storage is empty", () => {
    expect(readStoredThemePreference()).toBe("system");
  });

  it("reads stored preference", () => {
    localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, "dark");
    expect(readStoredThemePreference()).toBe("dark");
  });

  it("falls back to system for invalid stored values", () => {
    localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, "invalid");
    expect(readStoredThemePreference()).toBe("system");
  });

  it("resolves system preference from prefers-color-scheme", () => {
    expect(resolveTheme("system", true)).toBe("dark");
    expect(resolveTheme("system", false)).toBe("light");
  });

  it("resolves explicit light and dark preferences", () => {
    expect(resolveTheme("light", true)).toBe("light");
    expect(resolveTheme("dark", false)).toBe("dark");
  });

  it("persists preference to localStorage", () => {
    persistThemePreference("dark");
    expect(localStorage.getItem(THEME_PREFERENCE_STORAGE_KEY)).toBe("dark");
  });

  it("applies dark class and color-scheme to document", () => {
    applyResolvedThemeToDocument("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe("dark");

    applyResolvedThemeToDocument("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("reads system prefers dark from matchMedia", () => {
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: vi.fn().mockReturnValue({
        matches: true,
        media: "(prefers-color-scheme: dark)",
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      } as unknown as MediaQueryList),
    });

    expect(getSystemPrefersDark()).toBe(true);
  });
});
