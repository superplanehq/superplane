import React from "react";
import { resolveIcon } from "@/lib/utils";

export interface CollapsedComponentProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  title: string;
  collapsedBackground?: string;
  shape?: "rounded" | "circle";
  children?: React.ReactNode;
}

export const CollapsedComponent: React.FC<CollapsedComponentProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  title,
  collapsedBackground,
  shape = "rounded",
  children,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const containerClass = shape === "circle" ? "rounded-full" : "rounded-md";

  return (
    <div className="flex w-fit flex-col items-center">
      <div className={`flex h-20 w-20 items-center justify-center border border-border ${containerClass} ${collapsedBackground || ''}`}>
        {iconSrc ? (
          <div className={`w-16 h-16 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
            <img src={iconSrc} alt={title} className="h-12 w-12 object-contain" />
          </div>
        ) : (
          <Icon size={30} className={iconColor} />
        )}
      </div>
      <h2 className="text-base font-semibold text-neutral-900 pt-1">{title}</h2>
      {children && (
        <div className="mt-2">
          {children}
        </div>
      )}
    </div>
  );
};