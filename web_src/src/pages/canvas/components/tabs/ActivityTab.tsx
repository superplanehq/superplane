import { Stage } from "../../store/types";
import { ExecutionTimeline } from '../ExecutionTimeline';
import { SuperplaneStageEvent, SuperplaneExecution } from "@/api-client";
import MessageItem from '../MessageItem';

interface ActivityTabProps {
  selectedStage: Stage;
  pendingEvents: SuperplaneStageEvent[];
  waitingEvents: SuperplaneStageEvent[];
  partialExecutions: SuperplaneExecution[];
  approveStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
  discardStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
  cancelStageExecution: (executionId: string, stageId: string) => Promise<void>;
  executionRunning: boolean;
  onChangeTab: (tab: string) => void;
  organizationId: string;
  isLoading: boolean;
}

export const ActivityTab = ({
  selectedStage,
  pendingEvents,
  waitingEvents,
  partialExecutions,
  approveStageEvent,
  discardStageEvent,
  cancelStageExecution,
  onChangeTab,
  organizationId,
  isLoading
}: ActivityTabProps) => {
  const queueCount = pendingEvents.length + waitingEvents.length;

  if (isLoading) {
    return (
      <div className="p-6">
        <div className="text-center py-8">
          <div className="inline-flex items-center justify-center w-16 h-16 mb-3">
            <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent"></div>
          </div>
          <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading activity...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Recent Runs Section */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-bold text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">Recent Runs</h3>
          <button className="text-xs text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium"
            onClick={() => onChangeTab('executions')}
          >
            View all
          </button>
        </div>
        <ExecutionTimeline
          executions={partialExecutions.slice(0, 3)}
          organizationId={organizationId}
          onCancel={(executionId) => cancelStageExecution(executionId, selectedStage.metadata!.id!)}
        />
      </div>

      {/* Queue Section */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-bold text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
            Queue ({queueCount})
          </h3>
        </div>

        <div className="space-y-3">
          {/* All queue events using MessageItem */}
          {[...pendingEvents, ...waitingEvents].length === 0 ? (
            <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
              <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3 " aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>queue</span>
              <p data-slot="text" className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No items in queue</p>
            </div>
          ) : (
            [...pendingEvents, ...waitingEvents]
              .sort((a, b) => new Date(b.createdAt || '').getTime() - new Date(a.createdAt || '').getTime())
              .map((event) => {
                const sourceEvent = event.triggerEvent;

                return (
                  <MessageItem
                    key={event.id}
                    event={event}
                    selectedStage={selectedStage}
                    onApprove={event.state === 'STATE_WAITING' ? (eventId) => approveStageEvent(eventId, selectedStage.metadata!.id!) : undefined}
                    onCancel={(eventId) => discardStageEvent(eventId, selectedStage.metadata!.id!)}
                    sourceEvent={sourceEvent}
                  />
                );
              })
          )}
        </div>
      </div>
    </div>
  );
};