import React from 'react';
import { SuperplaneStageEvent } from '@/api-client';
import { ApprovalQueueItem, WaitingQueueItem, PendingQueueItem, EmptyQueueItem } from './QueueItems';

interface StageQueueSectionProps {
  lastWaitingEvent: SuperplaneStageEvent | null;
  lastPendingEvent: SuperplaneStageEvent | null;
  eventsMoreCount: number;
  onApproveEvent: (eventId: string, stageId: string) => void;
  stageId: string;
}

export const StageQueueSection: React.FC<StageQueueSectionProps> = ({
  lastWaitingEvent,
  lastPendingEvent,
  eventsMoreCount,
  onApproveEvent,
  stageId
}) => {
  return (
    <div className="px-3 pt-2 pb-2 w-full">
      <div className="w-full text-left flex justify-between text-xs font-bold text-gray-900 dark:text-gray-100 uppercase tracking-wide mb-1">
        Next in queue
        {eventsMoreCount > 0 && (
          <span className="text-xs text-gray-400 font-medium">+{eventsMoreCount} more</span>
        )}
      </div>

      {/* Approval Events */}
      {lastWaitingEvent && lastWaitingEvent.stateReason === "STATE_REASON_APPROVAL" && (
        <ApprovalQueueItem
          event={lastWaitingEvent}
          onApprove={onApproveEvent}
          stageId={stageId}
        />
      )}

      {/* Time Window Events */}
      {lastWaitingEvent && lastWaitingEvent.stateReason === "STATE_REASON_TIME_WINDOW" && (
        <WaitingQueueItem
          event={lastWaitingEvent}
          label="Waiting"
        />
      )}

      {/* Other Waiting Events */}
      {lastWaitingEvent && !["STATE_REASON_APPROVAL", "STATE_REASON_TIME_WINDOW"].includes(lastWaitingEvent.stateReason || '') && (
        <WaitingQueueItem
          event={lastWaitingEvent}
          label="Waiting"
        />
      )}

      {/* Pending Events */}
      {lastPendingEvent && (
        <PendingQueueItem event={lastPendingEvent} />
      )}

      {/* Empty State */}
      {!lastPendingEvent && !lastWaitingEvent && (
        <EmptyQueueItem />
      )}
    </div>
  );
};