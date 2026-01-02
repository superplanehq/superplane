import { resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { EllipsisVertical } from "lucide-react";
import React from "react";
import type { ChildEventsState } from "../../composite";

interface SidebarEventActionsMenuProps {
  eventId: string;
  executionId?: string;
  onCancelQueueItem?: (id: string) => void;
  onPushThrough?: (executionId: string) => void;
  onCancelExecution?: (executionId: string) => void;
  supportsPushThrough?: boolean;
  eventState: ChildEventsState;
  onReEmit?: () => void;
  kind: "queue" | "execution" | "trigger";
  onOpenChange?: (open: boolean) => void;
}

export const SidebarEventActionsMenu: React.FC<SidebarEventActionsMenuProps> = ({
  eventId,
  executionId,
  onCancelQueueItem,
  onPushThrough,
  onCancelExecution,
  supportsPushThrough,
  eventState,
  onReEmit,
  kind,
  onOpenChange,
}) => {
  const isProcessed = eventState === "triggered";
  const isDiscarded = eventState === "discarded";
  const isWaiting = eventState === "waiting";
  const isQueued = eventState === "queued";
  const isRunning = eventState === "running";

  const showPushThrough = supportsPushThrough && !!executionId && isRunning;
  const showCancel = (kind === "queue" && isQueued) || (kind === "execution" && (isRunning || isWaiting));
  const showReEmit = (isProcessed || isDiscarded) && kind === "trigger";
  const showDropdown = showPushThrough || showCancel || showReEmit;

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
    [onCancelExecution, executionId, eventId],
  );

  const handlePushThrough = React.useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (!executionId) {
        console.warn("No executionId provided for push-through action");
        return;
      }

      if (onPushThrough) {
        onPushThrough(executionId);
      }
    },
    [onPushThrough, executionId, eventId],
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
    <DropdownMenu onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <button
          className="h-6 w-6 flex items-center justify-center rounded text-gray-500"
          aria-label="Open actions"
          onClick={(e) => e.stopPropagation()}
        >
          <EllipsisVertical size={16} />
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

        {showPushThrough && (
          <DropdownMenuItem onClick={handlePushThrough} className="gap-2" data-testid="push-through-item">
            {React.createElement(resolveIcon("fast-forward"), { size: 16 })}
            Push Through
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
