import React from "react";
import { cn } from "@/lib/utils";
import { getIconThemeTreatment } from "./integrationIconMaps";

export interface AppLogoProps {
  /** Resolved logo src (e.g. from getHeaderIconSrc / getIntegrationIconSrc). */
  src: string;
  alt?: string;
  className?: string;
  style?: React.CSSProperties;
}

/**
 * Renders an integration/component logo that adapts to the active theme.
 *
 * Recoloring is driven entirely by Tailwind's `.dark` class (no JS theme
 * threading, no flash): monochrome logos are inverted on dark surfaces, and
 * brand-colored logos with a dedicated dark asset are swapped via pure CSS.
 * Theme-agnostic logos render unchanged.
 *
 * This is the single place that knows how to theme a logo `<img>`, so adding a
 * new theme-aware icon only means one entry in `integrationIconMaps.ts`.
 */
export function AppLogo({ src, alt = "", className, style }: AppLogoProps): React.ReactElement {
  const { invertInDark, darkSrc } = getIconThemeTreatment(src);

  if (darkSrc) {
    return (
      <>
        <img src={src} alt={alt} className={cn(className, "dark:hidden")} style={style} />
        <img src={darkSrc} alt={alt} className={cn(className, "hidden dark:block")} style={style} />
      </>
    );
  }

  return (
    <img src={src} alt={alt} className={cn(className, invertInDark && "dark:brightness-0 dark:invert")} style={style} />
  );
}
