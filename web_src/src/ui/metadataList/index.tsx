import { nodeCanvasMetadataSectionClassName } from "@/lib/nodeCanvasSections";
import { resourceRefLabel } from "@/lib/integrationResource";
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
  className = nodeCanvasMetadataSectionClassName,
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

/**
 * Guards against rendering a raw data object as a React child. A metadata label
 * can accidentally be an unresolved integration-resource object ({ id, name,
 * type }) instead of a string; rendering it directly throws "Objects are not
 * valid as a React child" and crashes the whole canvas. Coerce such objects to
 * their display name so a single bad node degrades gracefully instead.
 */
function renderLabel(label: string | React.ReactNode): React.ReactNode {
  if (label !== null && typeof label === "object" && !React.isValidElement(label)) {
    return resourceRefLabel(label) ?? "";
  }

  return label;
}

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
        {renderLabel(item.label)}
      </span>
    </div>
  );
}
