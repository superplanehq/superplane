import { resolveIcon } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/ui/tooltip";
import { EllipsisVertical } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export interface SidebarAction {
  id: string;
  label: string;
  icon: string;
  onAction?: () => void;
  hoverBackground?: string;
  hoverColor?: string;
  hasBorder?: boolean;
}

interface SidebarActionsDropdownProps {
  onRun?: () => void;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onDuplicate?: () => void;
  onDocs?: () => void;
  onEdit?: () => void;
  onConfigure?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
  dataTestId?: string;
}

export const SidebarActionsDropdown = ({
  onRun,
  runDisabled,
  runDisabledTooltip,
  onDuplicate,
  onDocs,
  onEdit,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
  dataTestId,
  isCompactView = false,
}: SidebarActionsDropdownProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const actions: SidebarAction[] = [
    {
      id: "run",
      label: "Run",
      icon: "play",
      onAction: onRun,
    },
    {
      id: "duplicate",
      label: "Duplicate",
      icon: "copy",
      onAction: onDuplicate,
    },
    {
      id: "docs",
      label: "Docs",
      icon: "book-text",
      onAction: onDocs,
    },
    {
      id: "edit",
      label: "Edit",
      icon: "pencil",
      onAction: onEdit,
    },
    {
      id: "configure",
      label: "Configure",
      icon: "settings-2",
      onAction: onConfigure,
    },
    {
      id: "deactivate",
      label: "Deactivate",
      icon: "octagon-pause",
      onAction: onDeactivate,
    },
    {
      id: "toggle-view",
      label: isCompactView ? "Detailed view" : "Compact view",
      icon: isCompactView ? "list-chevrons-up-down" : "list-chevrons-down-up",
      onAction: onToggleView,
      hasBorder: true,
    },
    {
      id: "delete",
      label: "Delete",
      icon: "trash-2",
      onAction: onDelete,
      hoverBackground: "hover:bg-red-100",
      hoverColor: "hover:text-red-700",
    },
  ];

  // Filter out actions that don't have an onAction function,
  // but keep Run even if disabled (so we can show tooltip/disabled state)
  const availableActions = actions.filter((action) => {
    if (action.id === "run") return !!onRun; // keep if run is supported
    return Boolean(action.onAction);
  });

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  return (
    <div className="relative" ref={dropdownRef} data-testid={dataTestId}>
      <button
        onClick={(e) => {
          e.stopPropagation();
          setIsOpen(!isOpen);
        }}
        className="ml-auto"
        aria-label="More actions"
      >
        <EllipsisVertical size={16} />
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-1 w-53 bg-white border border-gray-200 rounded-md shadow-lg z-50">
          <div className="py-1">
            {availableActions.length > 0 &&
              availableActions.map((action) => {
                const Icon = resolveIcon(action.icon);
                const isRun = action.id === "run";
                const disabled = isRun && runDisabled;
                const content = (
                  <div
                    key={action.id}
                    className={
                      "px-2 " +
                      (action.hasBorder
                        ? "border-b-1 border-t-1 border-gray-200 my-1 hover:bg-gray-100 transition-colors"
                        : "")
                    }
                  >
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        if (!disabled) {
                          action.onAction?.();
                          setIsOpen(false);
                        }
                      }}
                      disabled={disabled}
                      aria-disabled={disabled}
                      className={`w-full px-3 py-2 text-left flex items-center rounded-md gap-2 text-sm transition-colors ${
                        action.hoverBackground || ""
                      } ${action.hoverColor || ""} ${
                        disabled
                          ? "text-gray-300 cursor-not-allowed bg-white hover:bg-white hover:opacity-100"
                          : "text-gray-700 hover:bg-gray-100"
                      }`}
                    >
                      <Icon size={16} />
                      <span>{action.label}</span>
                    </button>
                  </div>
                );

                if (isRun && disabled && runDisabledTooltip) {
                  return (
                    <TooltipProvider delayDuration={150} key={action.id}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          {/* wrap content */}
                          <div>{content}</div>
                        </TooltipTrigger>
                        <TooltipContent side="left">{runDisabledTooltip}</TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  );
                }

                return content;
              })}
            {availableActions.length === 0 && (
              <div className="w-full px-3 py-2 text-left flex items-center gap-2 hover:bg-gray-50 text-sm text-gray-700 transition-colors">
                <span>No actions available</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
