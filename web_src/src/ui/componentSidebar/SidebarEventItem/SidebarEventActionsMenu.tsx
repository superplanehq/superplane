import React from "react";
import { resolveIcon } from "@/lib/utils";
import { EllipsisVertical } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import type { ChildEventsState } from "../../composite";

interface SidebarEventActionsMenuProps {
  eventId: string;
  onCancelQueueItem?: (id: string) => void;
  eventState: ChildEventsState;
}

export const SidebarEventActionsMenu: React.FC<SidebarEventActionsMenuProps> = ({
  eventId,
  onCancelQueueItem,
  eventState,
}) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className="ml-1 h-6 w-6 flex items-center justify-center rounded hover:bg-black/5 text-gray-600"
          aria-label="Open actions"
          onClick={(e) => e.stopPropagation()}
        >
          <EllipsisVertical size={16} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="min-w-[11rem]">
        {/* Cancel when not finished (queued or running) */}
        {!(eventState === "processed" || eventState === "discarded") && (
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              onCancelQueueItem?.(eventId);
            }}
            className="gap-2"
          >
            {React.createElement(resolveIcon("x-circle"), { size: 16 })}
            Cancel
          </DropdownMenuItem>
        )}

        {/* Push Through only when running (not finished and not queued) */}
        {!(eventState === "processed" || eventState === "discarded" || eventState === "waiting") && (
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
            }}
            className="gap-2"
          >
            {React.createElement(resolveIcon("chevrons-right"), { size: 16 })}
            Push Through
          </DropdownMenuItem>
        )}

        {/* Re-emit for finished or running; not for queued */}
        {eventState !== "waiting" && (
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
            }}
            className="gap-2"
          >
            {React.createElement(resolveIcon("rotate-ccw"), { size: 16 })}
            Re-emit
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
