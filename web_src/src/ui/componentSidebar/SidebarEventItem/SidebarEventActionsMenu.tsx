import { resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { EllipsisVertical } from "lucide-react";
import React from "react";
import type { ChildEventsState } from "../../composite";

interface SidebarEventActionsMenuProps {
  eventId: string;
  onCancelQueueItem?: (id: string) => void;
  onPassThrough?: (executionId: string) => void;
  supportsPassThrough?: boolean;
  eventState: ChildEventsState;
}

export const SidebarEventActionsMenu: React.FC<SidebarEventActionsMenuProps> = ({
  eventId,
  onCancelQueueItem,
  onPassThrough,
  supportsPassThrough,
  eventState,
}) => {
  const isProcessed = eventState === "processed";
  const isDiscarded = eventState === "discarded";
  const isWaiting = eventState === "waiting";

  const showPassThrough = supportsPassThrough && !(isProcessed || isDiscarded || isWaiting);
  const showCancel = !(isProcessed || isDiscarded);
  const showReEmit = isProcessed || isDiscarded;

  const handleReEmit = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      // Implement re-emit logic here
      // TODO: Add re-emit handler prop if needed
    },
    [onPassThrough, eventId],
  );

  const handlePassThrough = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (onPassThrough) {
        onPassThrough(eventId);
      }
    },
    [onPassThrough, eventId],
  );

  const handleCancelQueueItem = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (onCancelQueueItem) {
        onCancelQueueItem(eventId);
      }
    },
    [onCancelQueueItem, eventId],
  );

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
        {showCancel && (
          <DropdownMenuItem onClick={handleCancelQueueItem} className="gap-2">
            {React.createElement(resolveIcon("x-circle"), { size: 16 })}
            Cancel
          </DropdownMenuItem>
        )}

        {showPassThrough && (
          <DropdownMenuItem onClick={handlePassThrough} className="gap-2">
            {React.createElement(resolveIcon("fast-forward"), { size: 16 })}
            Push Through
          </DropdownMenuItem>
        )}

        {showReEmit && (
          <DropdownMenuItem onClick={handleReEmit} className="gap-2">
            {React.createElement(resolveIcon("rotate-ccw"), { size: 16 })}
            Re-emit
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
