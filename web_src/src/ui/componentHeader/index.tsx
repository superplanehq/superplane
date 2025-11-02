import React from "react";
import { resolveIcon } from "@/lib/utils";
import { SidebarActionsDropdown } from "../componentSidebar/SidebarActionsDropdown";
import { ComponentActionsProps } from "../types/componentActions";

export interface ComponentHeaderProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  description?: string;
  onDoubleClick?: () => void;
}

export const ComponentHeader: React.FC<ComponentHeaderProps> = ({
  iconSrc,
  iconSlug,
  iconBackground,
  iconColor,
  headerColor,
  title,
  description,
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
}) => {
  const Icon = React.useMemo(() => {
    return resolveIcon(iconSlug);
  }, [iconSlug]);

  return (
    <div className={"text-left text-lg w-full px-2 flex flex-col border-b p-2 gap-2 rounded-t items-center relative " + headerColor} onDoubleClick={onDoubleClick}>
      <div className="w-full flex items-center gap-2">
        <div className={`w-6 h-6 rounded-full overflow-hidden flex items-center justify-center ${iconBackground || ''}`}>
          {iconSrc ? <img src={iconSrc} alt={title} className="w-5 h-5 " /> : <Icon size={20} className={iconColor} />}
        </div>
        <h2 className="font-semibold">{title}</h2>
        <div className="absolute top-2 right-2">
          <SidebarActionsDropdown
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
      </div>
      {description && <p className="w-full text-gray-500 pl-8">{description}</p>}
    </div>
  );
};
