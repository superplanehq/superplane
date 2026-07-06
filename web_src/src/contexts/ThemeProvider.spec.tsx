import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "./ThemeProvider";
import { useTheme } from "./useTheme";
import { THEME_PREFERENCE_STORAGE_KEY } from "@/lib/themePreference";

function ThemeProbe() {
  const { preference, resolvedTheme } = useTheme();
  return <div data-testid="theme-state">{`${preference}:${resolvedTheme}`}</div>;
}

function renderThemeProvider() {
  return render(
    <ThemeProvider>
      <ThemeProbe />
    </ThemeProvider>,
  );
}

function mockPrefersDark(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn().mockReturnValue({
      matches,
      media: "(prefers-color-scheme: dark)",
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    } as unknown as MediaQueryList),
  });
}

function dispatchThemeStorageEvent(newValue: string | null, key: string | null = THEME_PREFERENCE_STORAGE_KEY) {
  act(() => {
    window.dispatchEvent(
      new StorageEvent("storage", {
        key,
        newValue,
        storageArea: window.localStorage,
      }),
    );
  });
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    document.documentElement.style.colorScheme = "";
    mockPrefersDark(false);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("syncs theme preference changes from other tabs", () => {
    renderThemeProvider();

    expect(screen.getByTestId("theme-state")).toHaveTextContent("system:light");

    dispatchThemeStorageEvent("dark");

    expect(screen.getByTestId("theme-state")).toHaveTextContent("dark:dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("resets to system when another tab clears the stored preference", () => {
    localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, "dark");
    renderThemeProvider();

    expect(screen.getByTestId("theme-state")).toHaveTextContent("dark:dark");

    dispatchThemeStorageEvent(null);

    expect(screen.getByTestId("theme-state")).toHaveTextContent("system:light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("ignores storage changes for unrelated keys", () => {
    renderThemeProvider();

    dispatchThemeStorageEvent("dark", "other-key");

    expect(screen.getByTestId("theme-state")).toHaveTextContent("system:light");
  });
});
