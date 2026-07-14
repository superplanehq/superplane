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

/**
 * Coerce a metadata label into something React can safely render.
 *
 * Mappers build labels from node configuration/metadata, which can hold a raw
 * resource reference object (e.g. an IntegrationResourceRef `{ id, name, type }`)
 * instead of the expected string. Rendering a plain object as a React child throws
 * "Objects are not valid as a React child" and — when it happens during a
 * synchronous commit flush — can crash the entire canvas. Rather than trust every
 * caller, we normalize at the render boundary: primitives and React elements pass
 * through, and object references fall back to their human-readable name/id.
 */
function toRenderableLabel(label: MetadataItem["label"]): React.ReactNode {
  if (label === null || label === undefined || typeof label === "boolean") {
    return null;
  }

  if (typeof label === "string" || typeof label === "number" || React.isValidElement(label)) {
    return label;
  }

  if (typeof label === "object") {
    const record = label as unknown as Record<string, unknown>;
    for (const key of ["name", "label", "title", "id"]) {
      const candidate = record[key];
      if (typeof candidate === "string" && candidate.length > 0) {
        return candidate;
      }
      if (typeof candidate === "number") {
        return String(candidate);
      }
    }
    return null;
  }

  return String(label);
}
