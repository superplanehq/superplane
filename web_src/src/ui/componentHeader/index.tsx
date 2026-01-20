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
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  title,
  onDoubleClick,
  isCompactView = false,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div
      data-testid={toTestId(`node-${title}-header`)}
      className={
        "canvas-node-drag-handle text-left text-lg w-full px-2 py-1.5 flex items-center flex-col rounded-t-md items-center relative" +
        (isCompactView ? "" : " border-b border-slate-400")
      }
      onDoubleClick={onDoubleClick}
    >
      <div className="w-full flex items-center">
        <div className="w-4 h-4 overflow-hidden flex items-center justify-center mr-2">
          {iconSrc ? (
            <img src={iconSrc} alt={title} className="max-w-5 max-h-5 object-contain" />
          ) : (
            <Icon size={16} className={iconColor} />
          )}
        </div>
        <h2 className="font-semibold text-sm">{title}</h2>
      </div>
    </div>
  );
};
