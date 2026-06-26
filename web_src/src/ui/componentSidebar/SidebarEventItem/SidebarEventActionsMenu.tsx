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
  triggerVariant?: "icon" | "labeled";
}

export const SidebarEventActionsMenu: React.FC<SidebarEventActionsMenuProps> = ({
  eventId,
  executionId,
  onCancelQueueItem,
  onCancelExecution,
  eventState,
  onReEmit,
  kind,
  triggerVariant = "icon",
}) => {
  const isWaiting = eventState === "waiting";
  const isQueued = eventState === "queued";
  const isRunning = eventState === "running";

  const showQueueCancel = kind === "queue" && isQueued && !!onCancelQueueItem;
  const showExecutionCancel = kind === "execution" && (isRunning || isWaiting) && !!executionId && !!onCancelExecution;
  const showCancel = showQueueCancel || showExecutionCancel;
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

      if (!executionId) {
        console.warn("No executionId provided for cancel action");
        return;
      }

      if (onCancelExecution) {
        onCancelExecution(executionId);
      }
    },
    [onCancelExecution, executionId],
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

  if (!showDropdown) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className={
            triggerVariant === "labeled"
              ? "inline-flex h-7 items-center gap-1 rounded border border-slate-900/15 bg-white/90 px-2 text-[11px] font-medium text-gray-700 hover:bg-white"
              : "h-6 w-6 flex items-center justify-center rounded border border-slate-900/15 bg-white/90 text-gray-600 hover:bg-white"
          }
          aria-label="Open actions"
          onClick={(e) => e.stopPropagation()}
          data-testid="sidebar-event-actions-trigger"
        >
          <EllipsisVertical size={16} />
          {triggerVariant === "labeled" ? <span>Actions</span> : null}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={6} className="min-w-[11rem]">
        {showCancel && (
          <DropdownMenuItem
            onClick={kind === "queue" ? handleCancelQueueItem : handleCancelExecution}
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
