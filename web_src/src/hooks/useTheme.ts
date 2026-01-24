import { useEffect, useState, useCallback } from "react";

export type ThemePreference = "system" | "light" | "dark";

const STORAGE_KEY = "superplane-theme-preference";

function getStoredPreference(): ThemePreference {
  if (typeof window === "undefined") return "light";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "light";
}

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function useTheme() {
  const [preference, setPreferenceState] = useState<ThemePreference>(getStoredPreference);
  const [systemTheme, setSystemTheme] = useState<"light" | "dark">(getSystemTheme);

  // Compute effective theme based on preference
  const effectiveTheme: "light" | "dark" = preference === "system" ? systemTheme : preference;

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");

    const handleChange = (e: MediaQueryListEvent) => {
      setSystemTheme(e.matches ? "dark" : "light");
    };

    mediaQuery.addEventListener("change", handleChange);
    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  // Apply/remove dark class to document root
  useEffect(() => {
    const root = document.documentElement;
    if (effectiveTheme === "dark") {
      root.classList.add("dark");
    } else {
      root.classList.remove("dark");
    }
  }, [effectiveTheme]);

  // Save preference to localStorage
  const setPreference = useCallback((newPreference: ThemePreference) => {
    setPreferenceState(newPreference);
    localStorage.setItem(STORAGE_KEY, newPreference);
  }, []);

  return {
    preference,
    effectiveTheme,
    setPreference,
    isDark: effectiveTheme === "dark",
  };
}
