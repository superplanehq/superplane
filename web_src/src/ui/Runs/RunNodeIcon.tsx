import React from "react";
import { cn, resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";

export const RUN_NODE_ICON_SIZE = 14;

export function RunNodeIcon({
  componentName,
  iconSrc,
  iconSlug,
  alt,
  size = 16,
  className = "shrink-0 text-gray-500",
}: {
  componentName?: string;
  iconSrc?: string;
  iconSlug?: string;
  alt: string;
  size?: number;
  className?: string;
}) {
  const resolvedIconSrc = iconSrc || getHeaderIconSrc(componentName);
  const dimensionClass = size === RUN_NODE_ICON_SIZE ? "h-3.5 w-3.5" : "";

  if (resolvedIconSrc) {
    return (
      <img
        src={resolvedIconSrc}
        alt={alt}
        className={cn("shrink-0 object-contain", dimensionClass, className)}
        style={{ width: `${size}px`, height: `${size}px` }}
      />
    );
  }

  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size,
    className: cn("shrink-0", dimensionClass, className),
  });
}
