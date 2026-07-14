import { nodeCanvasMetadataSectionClassName } from "@/lib/nodeCanvasSections";
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
        {toRenderableLabel(item.label)}
      </span>
    </div>
  );
}

// A label is normally a string or a React element, but mappers derive labels
// from backend data that can occasionally be a plain object (e.g. an integration
// resource reference like { id, name, type }). React throws "Objects are not
// valid as a React child" for such values, which would crash the whole canvas.
// Coerce those to a display string so a single bad label can never take the app down.
function toRenderableLabel(label: string | React.ReactNode): React.ReactNode {
  if (label === null || label === undefined || typeof label === "string" || typeof label === "number") {
    return label;
  }

  if (React.isValidElement(label) || Array.isArray(label)) {
    return label;
  }

  if (typeof label === "object") {
    const ref = label as { name?: unknown; id?: unknown };
    if (typeof ref.name === "string" && ref.name) {
      return ref.name;
    }
    if (typeof ref.id === "string" && ref.id) {
      return ref.id;
    }
  }

  return String(label);
}
