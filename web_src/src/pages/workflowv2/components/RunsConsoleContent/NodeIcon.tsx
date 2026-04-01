import React from "react";
import { resolveIcon } from "@/lib/utils";

export function NodeIcon({
  iconSrc,
  iconSlug,
  alt,
  size = 14,
  className = "text-gray-500",
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
  size?: number;
  className?: string;
}) {
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} style={{ width: size, height: size }} className="object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "box"), { size, className });
}
