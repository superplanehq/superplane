import { useEffect } from "react";

/**
 * Toggles the `theme-factories` class on `document.body` for the lifetime
 * of the story so components rendered outside of `FactoriesLayout` (e.g.
 * individual layout parts, dialogs, or the design-system swatches) pick up
 * the v3 design tokens defined in `App.css`.
 *
 * Kept as a plain component here (rather than a Storybook decorator) so
 * fast-refresh can hot-reload the file. The decorator that wraps this
 * component lives in `factoriesStoryTheme.ts`.
 */
export function FactoriesThemeStoryShell({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    document.body.classList.add("theme-factories");
    return () => {
      document.body.classList.remove("theme-factories");
    };
  }, []);

  return <div className="min-h-svh bg-background text-foreground">{children}</div>;
}
