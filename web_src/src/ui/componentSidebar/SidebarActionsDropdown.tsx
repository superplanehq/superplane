import { resolveIcon } from "@/lib/utils";
import { EllipsisVertical } from "lucide-react";
import { useState, useRef, useEffect } from "react";

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
  onDuplicate?: () => void;
  onDocs?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
}

export const SidebarActionsDropdown = ({
  onRun,
  onDuplicate,
  onDocs,
  onDeactivate,
  onToggleView,
  onDelete,
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
      id: "deactivate",
      label: "Deactivate",
      icon: "octagon-pause",
      onAction: onDeactivate,
    },
    {
      id: "toggle-view",
      label: isCompactView ? "Detailed view" : "Compact view",
      icon: isCompactView ? "expand" : "minimize",
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

  // Filter out actions that don't have an onAction function
  const availableActions = actions.filter(action => action.onAction);

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
    <div className="relative" ref={dropdownRef}>
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
            {availableActions.length > 0 && availableActions.map((action) => {
              const Icon = resolveIcon(action.icon);
              return (
                <div key={action.id} className={"px-2 " + (action.hasBorder ? "border-b-1 border-t-1 border-gray-200 my-1 hover:bg-gray-100 transition-colors" : "")}>
                  <button
                    onClick={() => {
                      action.onAction?.();
                      setIsOpen(false);
                    }}
                    className={`w-full px-3 py-2 text-left flex items-center rounded-md gap-2 hover:bg-gray-100 text-sm text-gray-700 transition-colors ${action.hoverBackground} ${action.hoverColor} transition-colors`}
                  >
                    <Icon size={16} />
                    <span>{action.label}</span>
                  </button>
                </div>
              );
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