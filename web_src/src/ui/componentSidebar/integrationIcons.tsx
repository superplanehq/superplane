import { cn, resolveIcon } from "@/lib/utils";
import React from "react";

import { AppLogo } from "./AppLogo";
import { getIntegrationIconSrc } from "./integrationIconMaps";

const DEFAULT_ICON_SIZE = 16;

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
 * Logos automatically adapt to the active theme via {@link AppLogo}.
 */
export function IntegrationIcon({
  integrationName,
  iconSlug,
  className = "h-4 w-4",
  size = DEFAULT_ICON_SIZE,
}: IntegrationIconProps): React.ReactElement {
  const logoSrc = getIntegrationIconSrc(integrationName);
  if (logoSrc) {
    return (
      <span className={cn("inline-block flex-shrink-0", className)}>
        <AppLogo src={logoSrc} className="h-full w-full object-contain" />
      </span>
    );
  }
  const IconComponent = resolveIcon(iconSlug);
  return <IconComponent className={className} size={size} />;
}
