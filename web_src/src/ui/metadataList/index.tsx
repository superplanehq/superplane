import { resolveIcon } from "@/lib/utils";
import React from "react";

export interface MetadataItem {
  icon: string;
  label: string;
}

export interface MetadataListProps {
  items: MetadataItem[];
  className?: string;
  iconSize?: number;
}

export const MetadataList: React.FC<MetadataListProps> = ({
  items,
  className = "px-2 py-3 border-b text-gray-500 flex flex-col gap-1.5",
  iconSize = 16,
}) => {
  if (!items || items.length === 0) {
    return null;
  }

  return (
    <div className={className}>
      {items.map((item, index) => {
        const Icon = resolveIcon(item.icon);
        return (
          <div key={index} className="flex items-center gap-2 min-w-0">
            <Icon size={iconSize} className="flex-shrink-0" />
            <span className="text-sm truncate">{item.label}</span>
          </div>
        );
      })}
    </div>
  );
};
