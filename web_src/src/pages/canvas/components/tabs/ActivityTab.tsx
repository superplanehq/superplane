import { ExecutionWithEvent, StageWithEventQueue } from "../../store/types";
import { ExecutionTimeline } from '../ExecutionTimeline';
import { SuperplaneStageEvent } from "@/api-client";
import MessageItem from '../MessageItem';

interface ActivityTabProps {
  selectedStage: StageWithEventQueue;
  pendingEvents: SuperplaneStageEvent[];
  waitingEvents: SuperplaneStageEvent[];
  allExecutions: ExecutionWithEvent[];
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  executionRunning: boolean;
  onChangeTab: (tab: string) => void;
}

export const ActivityTab = ({
  selectedStage,
  pendingEvents,
  waitingEvents,
  allExecutions,
  approveStageEvent,
  executionRunning,
  onChangeTab
}: ActivityTabProps) => {
  const queueCount = pendingEvents.length + waitingEvents.length;

  return (
    <div className="p-6 space-y-6">
      {/* Recent Runs Section */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-bold text-sm text-gray-500 uppercase tracking-wide">Recent Runs</h3>
          <button className="text-xs text-blue-600 hover:text-blue-800 font-medium"
            onClick={() => onChangeTab('history')}
          >
            View all
          </button>
        </div>
        <ExecutionTimeline
          executions={allExecutions.slice(0, 3)}
        />
      </div>

      {/* Queue Section */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="font-bold text-sm text-gray-500 uppercase tracking-wide">
            Queue ({queueCount})
          </h3>
          {queueCount > 0 && (
            <button className="text-xs text-blue-600 hover:text-blue-800 font-medium">
              Manage queue
            </button>
          )}
        </div>

        <div className="space-y-3">
          {/* All queue events using MessageItem */}
          {[...pendingEvents, ...waitingEvents].length === 0 ? (
            <div className="bg-gray-50 rounded-lg p-8 text-center">
              <div className="material-symbols-outlined text-4xl text-gray-400 mb-2">queue</div>
              <div className="text-sm text-gray-500">No items in queue</div>
            </div>
          ) : (
            [...pendingEvents, ...waitingEvents]
              .sort((a, b) => new Date(b.createdAt || '').getTime() - new Date(a.createdAt || '').getTime())
              .map((event) => (
                <MessageItem
                  key={event.id}
                  event={event}
                  selectedStage={selectedStage}
                  onApprove={event.state === 'STATE_WAITING' ? (eventId) => approveStageEvent(eventId, selectedStage.metadata!.id!) : undefined}
                  executionRunning={executionRunning}
                />
              ))
          )}
        </div>
      </div>
    </div>
  );
};