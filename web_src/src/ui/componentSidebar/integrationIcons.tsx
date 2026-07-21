import { cn, resolveIcon } from "@/lib/utils";
import React from "react";

import { getIntegrationIconSrc } from "./integrationIconMaps";

const DEFAULT_ICON_SIZE = 16;

/** Monochrome SVG logos that need inversion on dark surfaces. */
const INTEGRATION_LOGO_INVERT_IN_DARK = new Set(["github", "opencode", "open-code", "anthropic", "ollama"]);

interface IntegrationIconProps {
  integrationName: string | undefined;
  /** Fallback Lucide icon slug when no custom logo exists */
  iconSlug?: string;
  className?: string;
  size?: number;
}

/**
 * Renders the integration's custom logo when available, otherwise a Lucide icon.
 * Use next to integration names (Settings tab) and in the component header for consistency.
 */
export function IntegrationIcon({
  integrationName,
  iconSlug,
  className = "h-4 w-4",
  size = DEFAULT_ICON_SIZE,
}: IntegrationIconProps): React.ReactElement {
  const logoSrc = getIntegrationIconSrc(integrationName);
  if (logoSrc) {
    const invertInDark = INTEGRATION_LOGO_INVERT_IN_DARK.has(integrationName?.toLowerCase() ?? "");

    return (
      <span className={cn("inline-block flex-shrink-0", className)}>
        <img
          src={logoSrc}
          alt=""
          className={cn("h-full w-full object-contain", invertInDark && "dark:brightness-0 dark:invert")}
        />
      </span>
    );
  }
  const IconComponent = resolveIcon(iconSlug);
  return <IconComponent className={className} size={size} />;
}
