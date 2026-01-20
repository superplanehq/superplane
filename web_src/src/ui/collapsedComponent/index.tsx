import React from "react";
import { resolveIcon } from "@/lib/utils";
import { ComponentActionsProps } from "../types/componentActions";

export interface CollapsedComponentProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  title: string;
  collapsedBackground?: string;
  shape?: "rounded" | "circle";
  children?: React.ReactNode;
  onDoubleClick?: () => void;
}

export const CollapsedComponent: React.FC<CollapsedComponentProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  title,
  collapsedBackground,
  shape = "rounded",
  children,
  onDoubleClick,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onDuplicate,
  onEdit,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  const containerClass = shape === "circle" ? "rounded-full" : "rounded-md";

  return (
    <div className="relative w-20 h-20" onDoubleClick={onDoubleClick}>
      <div
        className={`canvas-node-drag-handle flex h-20 w-20 items-center justify-center outline-1 outline-slate-400 ${containerClass} ${collapsedBackground || ""}`}
      >
        {iconSrc ? (
          <div className="w-16 h-16 rounded-full overflow-hidden flex items-center justify-center">
            <img src={iconSrc} alt={title} className="h-12 w-12 object-contain" />
          </div>
        ) : (
          <Icon size={30} className={iconColor} />
        )}
      </div>
      <h2 className="absolute top-full left-1/2 transform -translate-x-1/2 text-base font-semibold text-neutral-900 pt-1 whitespace-nowrap">
        {title}
      </h2>
      {children && (
        <div className="absolute top-full left-1/2 transform -translate-x-1/2 mt-8 text-center flex flex-col flex-wrap w-[400px] items-center justify-center">
          {children}
        </div>
      )}
    </div>
  );
};
