import { resolveIcon } from "@/lib/utils";
import React from "react";
import { toTestId } from "../../utils/testID";
import { ComponentActionsProps } from "../types/componentActions";

export interface ComponentHeaderProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  title: string;
  onDoubleClick?: () => void;
  statusBadgeColor?: string;
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  title,
  onDoubleClick,
  statusBadgeColor,
  isCompactView = false,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div
      data-testid={toTestId(`node-${title}-header`)}
      data-view-mode={isCompactView ? "compact" : "expanded"}
      className={
        "canvas-node-drag-handle text-left text-lg w-full px-2 py-1.5 flex items-center flex-col rounded-t-md items-center relative" +
        (isCompactView ? "" : " border-b border-slate-950/20")
      }
      onDoubleClick={onDoubleClick}
    >
      <div className="flex w-full items-center justify-between">
        <div className="flex items-center">
          <div className="mr-2 flex h-4 w-4 items-center justify-center overflow-hidden">
            {iconSrc ? (
              <img src={iconSrc} alt={title} className="h-4 w-4 shrink-0 object-contain" />
            ) : (
              <Icon size={16} className={iconColor} />
            )}
          </div>
          <h2 className="text-sm font-semibold">{title}</h2>
        </div>
        {isCompactView && statusBadgeColor ? (
          <span className={`h-2.5 w-2.5 rounded-full ${statusBadgeColor}`} aria-hidden="true" />
        ) : null}
      </div>
    </div>
  );
};
