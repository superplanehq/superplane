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
}

export const MetadataList: React.FC<MetadataListProps> = ({
  items,
  className = "px-2 py-3 border-b-2 border-slate-300 text-gray-500 flex flex-col gap-1.5",
  iconSize = 16,
  underlined = false,
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
            <span
              className={
                "text-sm font-medium font-inter truncate" +
                (underlined ? " underline underline-offset-3 decoration-dotted decoration-1" : "")
              }
            >
              {item.label}
            </span>
          </div>
        );
      })}
    </div>
  );
};
