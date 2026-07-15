import { useLayoutEffect, useMemo, type ReactNode } from "react";
import { ThemeContext } from "../src/contexts/themeContextState";
import { applyResolvedThemeToDocument, type ResolvedTheme } from "../src/lib/themePreference";

export function StorybookThemeShell({ theme, children }: { theme: ResolvedTheme; children: ReactNode }) {
  useLayoutEffect(() => {
    applyResolvedThemeToDocument(theme);
  }, [theme]);

  const value = useMemo(
    () => ({
      preference: theme,
      resolvedTheme: theme,
      setPreference: () => {
        // Theme is controlled by the Storybook toolbar.
      },
    }),
    [theme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}
