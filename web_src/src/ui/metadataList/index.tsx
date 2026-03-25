import { resolveIcon } from "@/lib/utils";
import React from "react";

export interface MetadataItem {
  icon: string;
  label: string | React.ReactNode;
}

export interface MetadataListProps {
  items: MetadataItem[];
  className?: string;
  iconSize?: number;
  underlined?: boolean;
  /** When set, only the first N items are shown. When omitted, all items are shown. */
  maxVisibleItems?: number;
}

export const MetadataList: React.FC<MetadataListProps> = ({
  items,
  className = "px-2 py-1.5 border-b border-slate-950/20 text-gray-500 flex flex-col gap-1",
  iconSize = 16,
  underlined = false,
  maxVisibleItems,
}) => {
  if (!items || items.length === 0) {
    return null;
  }

  const visibleItems =
    maxVisibleItems != null && Number.isFinite(maxVisibleItems) ? items.slice(0, Math.max(0, maxVisibleItems)) : items;

  return (
    <div className={className}>
      {visibleItems.map((item, index) => renderMetadataItem(item, index, iconSize, underlined))}
    </div>
  );
};

function renderMetadataItem(item: MetadataItem, index: number, iconSize: number, underlined: boolean) {
  const Icon = resolveIcon(item.icon);

  return (
    <div key={index} className="flex items-center min-w-0">
      <div className="w-4 h-4 mr-2">
        <Icon size={iconSize} className="flex-shrink-0" />
      </div>
      <span
        className={
          "text-[13px] font-medium font-inter truncate" +
          (underlined ? " underline underline-offset-3 decoration-dotted decoration-1" : "")
        }
      >
        {item.label}
      </span>
    </div>
  );
}
