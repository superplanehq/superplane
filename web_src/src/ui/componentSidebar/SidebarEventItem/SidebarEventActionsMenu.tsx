import { resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { EllipsisVertical } from "lucide-react";
import React from "react";
import type { ChildEventsState } from "../../composite";

interface SidebarEventActionsMenuProps {
  eventId: string;
  executionId?: string;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  eventState: ChildEventsState;
  onReEmit?: () => void;
  kind: "queue" | "execution" | "trigger";
  onOpenChange?: (open: boolean) => void;
}

export const SidebarEventActionsMenu: React.FC<SidebarEventActionsMenuProps> = ({
  eventId,
  executionId,
  onCancelQueueItem,
  onCancelExecution,
  eventState,
  onReEmit,
  kind,
  onOpenChange,
}) => {
  const isWaiting = eventState === "waiting";
  const isQueued = eventState === "queued";
  const isRunning = eventState === "running";

  const canCancelQueueItem = kind === "queue" && isQueued && !!onCancelQueueItem;
  const canCancelExecution = kind === "execution" && (isRunning || isWaiting) && !!onCancelExecution && !!executionId;
  const showCancel = canCancelQueueItem || canCancelExecution;
  const showReEmit = kind === "trigger" && !!onReEmit;
  const showDropdown = showCancel || showReEmit;

  const handleReEmit = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onReEmit?.();
    },
    [onReEmit],
  );

  const handleCancelExecution = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (!executionId || !onCancelExecution) return;
      onCancelExecution(executionId);
    },
    [onCancelExecution, executionId],
  );

  const handleCancelQueueItem = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (!onCancelQueueItem) return;
      onCancelQueueItem(eventId);
    },
    [onCancelQueueItem, eventId],
  );

  if (!showDropdown) return null;

  return (
    <DropdownMenu onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <button
          className="flex h-6 w-6 items-center justify-center rounded border border-slate-950/10 bg-white/70 text-gray-600 shadow-xs hover:bg-gray-100 hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950/20"
          aria-label="Open actions"
          onClick={(e) => e.stopPropagation()}
        >
          <EllipsisVertical size={16} />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={6} className="min-w-[11rem]">
        {showCancel && (
          <DropdownMenuItem
            onClick={canCancelQueueItem ? handleCancelQueueItem : handleCancelExecution}
            className="gap-2"
            data-testid="cancel-queue-item"
          >
            {React.createElement(resolveIcon("x-circle"), { size: 16 })}
            Cancel
          </DropdownMenuItem>
        )}

        {showReEmit && (
          <DropdownMenuItem onClick={handleReEmit} className="gap-2" data-testid="reemit-item">
            {React.createElement(resolveIcon("rotate-ccw"), { size: 16 })}
            Re-emit
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
