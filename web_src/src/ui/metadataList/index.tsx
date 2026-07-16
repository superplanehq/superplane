import { nodeCanvasMetadataSectionClassName } from "@/lib/nodeCanvasSections";
import { integrationResourceDisplayLabel } from "@/lib/integrationResourceLabel";
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

function coerceMetadataLabel(label: MetadataItem["label"]): React.ReactNode {
  if (
    label === null ||
    label === undefined ||
    typeof label === "string" ||
    typeof label === "number" ||
    typeof label === "boolean" ||
    React.isValidElement(label) ||
    Array.isArray(label)
  ) {
    return label;
  }

  // Guard against IntegrationResourceRef-like objects reaching JSX children.
  return integrationResourceDisplayLabel(label) ?? null;
}

function renderMetadataItem(item: MetadataItem, index: number, iconSize: number, underlined: boolean) {
  const Icon = resolveIcon(item.icon);
  const label = coerceMetadataLabel(item.label);

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
        {label}
      </span>
    </div>
  );
}
