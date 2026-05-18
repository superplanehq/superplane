import React from "react";
import { resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";

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
  if (resolvedIconSrc) {
    return (
      <img
        src={resolvedIconSrc}
        alt={alt}
        className="shrink-0 object-contain"
        style={{ width: `${size}px`, height: `${size}px` }}
      />
    );
  }

  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size,
    className,
  });
}
