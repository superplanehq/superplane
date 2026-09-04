import { useEffect, type ReactNode } from "react";

export type FactoriesSettingsPalette = "cursor" | "github";

const PALETTE_CLASS: Record<FactoriesSettingsPalette, string | null> = {
  cursor: null,
  github: "theme-factories-palette-github",
};

/**
 * Applies an optional factories color overlay on `document.body` for Storybook.
 * Cursor uses the shipped warm tokens. GitHub cools the whole surface so
 * field fills stay in the same hue family as the canvas.
 */
export function FactoriesPaletteStoryShell({
  palette,
  children,
}: {
  palette: FactoriesSettingsPalette;
  children: ReactNode;
}) {
  useEffect(() => {
    const className = PALETTE_CLASS[palette];
    if (!className) {
      return;
    }
    document.body.classList.add(className);
    return () => {
      document.body.classList.remove(className);
    };
  }, [palette]);

  return children;
}
