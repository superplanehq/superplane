export const THEME_PREFERENCE_STORAGE_KEY = "superplane-theme";

export type ThemePreference = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === "light" || value === "dark" || value === "system";
}

export function readStoredThemePreference(): ThemePreference {
  if (typeof window === "undefined") {
    return "system";
  }

  try {
    const stored = window.localStorage.getItem(THEME_PREFERENCE_STORAGE_KEY);
    return isThemePreference(stored) ? stored : "system";
  } catch {
    return "system";
  }
}

export function resolveTheme(preference: ThemePreference, prefersDark: boolean): ResolvedTheme {
  if (preference === "system") {
    return prefersDark ? "dark" : "light";
  }

  return preference;
}

export function getSystemPrefersDark(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

/** SVG favicons per resolved theme; keep in sync with the FOUC script in index.html. */
export const FAVICON_HREF: Record<ResolvedTheme, string> = {
  light: "/favicon.svg",
  dark: "/favicon-dark.svg",
};

export function applyFaviconForTheme(resolvedTheme: ResolvedTheme): void {
  if (typeof document === "undefined") {
    return;
  }

  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"][type="image/svg+xml"]');
  if (link) {
    link.href = FAVICON_HREF[resolvedTheme];
  }
}

export function applyResolvedThemeToDocument(resolvedTheme: ResolvedTheme): void {
  if (typeof document === "undefined") {
    return;
  }

  document.documentElement.classList.toggle("dark", resolvedTheme === "dark");
  document.documentElement.style.colorScheme = resolvedTheme;
  applyFaviconForTheme(resolvedTheme);
}

export function persistThemePreference(preference: ThemePreference): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(THEME_PREFERENCE_STORAGE_KEY, preference);
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}
