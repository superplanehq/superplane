import { resolveIcon } from "@/lib/utils";
import React from "react";
import { showErrorToast, showSuccessToast } from "@/utils/toast";

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
  className = "px-2 py-1.5 border-b border-slate-950/20 text-gray-500 flex flex-col gap-1",
  iconSize = 16,
  underlined = false,
}) => {
  const handleCopy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      showSuccessToast("Copied to clipboard");
    } catch (_err) {
      showErrorToast("Failed to copy text");
    }
  };

  if (!items || items.length === 0) {
    return null;
  }

  return (
    <div className={className}>
      {items.map((item, index) => {
        const Icon = resolveIcon(item.icon);
        const labelText = typeof item.label === "string" ? item.label : null;
        const isCopyableUrl = !!labelText && /^https?:\/\//.test(labelText);

        return (
          <div key={index} className="flex items-center min-w-0">
            <div className="w-4 h-4 mr-2">
              <Icon size={iconSize} className="flex-shrink-0" />
            </div>
            <span
              className={
                "text-[13px] font-medium font-inter truncate" +
                (underlined ? " underline underline-offset-3 decoration-dotted decoration-1" : "") +
                (isCopyableUrl ? " cursor-pointer" : "")
              }
              onClick={
                isCopyableUrl
                  ? () => {
                      void handleCopy(labelText);
                    }
                  : undefined
              }
              title={isCopyableUrl ? "Click to copy" : undefined}
            >
              {item.label}
            </span>
          </div>
        );
      })}
    </div>
  );
};
