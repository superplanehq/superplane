import { resolveIcon } from "@/lib/utils";
import React from "react";
import type { ChainItemData, ChildExecution } from "./types";

type ChainItemIconSource = Pick<ChainItemData, "nodeIcon" | "nodeIconSlug" | "nodeIconSrc"> &
  Pick<ChildExecution, "componentIcon" | "componentIconSrc">;

interface ChainItemIconProps {
  item: Partial<ChainItemIconSource>;
  size?: number;
  className?: string;
}

export function ChainItemIcon({
  item,
  size = 16,
  className = "text-gray-800",
}: ChainItemIconProps): React.ReactElement | null {
  const iconSrc = item.nodeIconSrc || item.componentIconSrc;
  const iconSlug = item.nodeIconSlug || item.nodeIcon || item.componentIcon;
  if (!iconSrc && !iconSlug) {
    return null;
  }

  return (
    <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
      {iconSrc ? (
        <img src={iconSrc} alt="" className="h-4 w-4 object-contain" />
      ) : (
        React.createElement(resolveIcon(iconSlug), {
          size,
          className,
        })
      )}
    </div>
  );
}
