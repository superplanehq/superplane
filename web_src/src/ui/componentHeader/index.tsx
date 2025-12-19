import { resolveIcon } from "@/lib/utils";
import React from "react";
import { toTestId } from "../../utils/testID";
import { SidebarActionsDropdown } from "../componentSidebar/SidebarActionsDropdown";
import { ComponentActionsProps } from "../types/componentActions";

export interface ComponentHeaderProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  onDoubleClick?: () => void;
  hideActionsButton?: boolean;
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconBackground,
  iconColor,
  headerColor,
  title,
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
  isCompactView = false,
  hideActionsButton = false,
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div
      data-testid={toTestId(`node-${title}-header`)}
      className={
        "canvas-node-drag-handle text-left text-lg w-full px-2 py-1.5 flex items-center flex-col border-b border-slate-400 rounded-t-md items-center relative " +
        headerColor
      }
      onDoubleClick={onDoubleClick}
    >
      <div className="w-full flex items-center">
        <div
          className={`w-4 h-4 rounded-full overflow-hidden flex items-center justify-center mr-2 ${iconBackground || ""}`}
        >
          {iconSrc ? (
            <img src={iconSrc} alt={title} className="max-w-5 max-h-5 object-contain" />
          ) : (
            <Icon size={16} className={iconColor} />
          )}
        </div>
        <h2 className="font-semibold text-sm">{title}</h2>
        {!hideActionsButton && (
          <div className="absolute top-1 right-1 rounded flex items-center justify-center hover:bg-slate-950/5 h-6 w-6 leading-none nodrag">
            <SidebarActionsDropdown
              dataTestId={toTestId(`node-${title}-header-dropdown`)}
              onRun={onRun}
              runDisabled={runDisabled}
              runDisabledTooltip={runDisabledTooltip}
              onDuplicate={onDuplicate}
              onEdit={onEdit}
              onConfigure={onConfigure}
              onDeactivate={onDeactivate}
              onToggleView={onToggleView}
              onDelete={onDelete}
              isCompactView={isCompactView}
            />
          </div>
        )}
      </div>
    </div>
  );
};
