import { TextAlignStart } from "lucide-react";
import { SidebarEventItem } from "./SidebarEventItem";
import { TabData } from "./SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "./types";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import { EventState, EventStateMap } from "../componentBase";

interface LatestTabProps {
  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  totalInQueueCount: number;
  hideQueueEvents?: boolean;
  openEventIds: Set<string>;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  onSeeFullHistory?: () => void;
  onSeeQueue?: () => void;
  onSeeExecutionChain?: (eventId: string, triggerEvent?: SidebarEvent) => void;
  getTabData?: (event: SidebarEvent) => TabData | undefined;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onPushThrough?: (executionId: string) => void;
  supportsPushThrough?: boolean;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<unknown[]>;
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };
}

export const LatestTab = ({
  latestEvents,
  nextInQueueEvents,
  totalInQueueCount,
  hideQueueEvents = false,
  openEventIds,
  onToggleOpen,
  onEventClick,
  onSeeFullHistory,
  onSeeQueue,
  onSeeExecutionChain,
  getTabData,
  onCancelQueueItem,
  onCancelExecution,
  onPushThrough,
  supportsPushThrough,
  onReEmit,
  loadExecutionChain,
  getExecutionState,
}: LatestTabProps) => {

  const handleSeeQueue = () => {
    onSeeQueue?.();
  };

  const handleSeeFullHistory = () => {
    onSeeFullHistory?.();
  };

  const handleTriggerNavigate = (event: SidebarEvent) => {
    const eventId = event.triggerEventId || event.id;
    onSeeExecutionChain?.(eventId, event);
  };

  return (
    <>
      <div className="px-3 py-1 border-b-1 border-border pb-3 text-left">
        <h2 className="text-xs font-semibold uppercase text-gray-500 my-2">Latest</h2>
        <div className="flex flex-col">
          {latestEvents.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm">No events found</div>
          ) : (
            <>
              {latestEvents.slice(0, 5).map((event, index) => {
                const totalItems = Math.min(latestEvents.length, 5);
                return (
                  <SidebarEventItem
                    key={event.id}
                    event={event}
                    index={index}
                    totalItems={totalItems}
                    variant="latest"
                    isOpen={openEventIds.has(event.id) || event.isOpen}
                    onToggleOpen={onToggleOpen}
                    onEventClick={onEventClick}
                    onTriggerNavigate={handleTriggerNavigate}
                    tabData={getTabData?.(event)}
                    onPushThrough={onPushThrough}
                    onCancelExecution={onCancelExecution}
                    supportsPushThrough={supportsPushThrough}
                    onReEmit={onReEmit}
                    loadExecutionChain={loadExecutionChain}
                    getExecutionState={getExecutionState}
                  />
                );
              })}
              {handleSeeFullHistory && (
                <button
                  onClick={handleSeeFullHistory}
                  className="text-sm text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                >
                  <TextAlignStart size={16} />
                  See full history
                </button>
              )}
            </>
          )}
        </div>
      </div>
      {!hideQueueEvents && (
        <div className="px-3 py-1 pb-3 text-left">
          <h2 className="text-xs font-semibold uppercase text-gray-500 my-2">Queued</h2>
          <div className="flex flex-col">
            {nextInQueueEvents.length === 0 ? (
              <div className="text-center py-4 text-gray-500 text-sm">Queue is empty</div>
            ) : (
              <>
                {nextInQueueEvents.slice(0, 5).map((event, index) => {
                  const totalItems = Math.min(nextInQueueEvents.length, 5);
                  return (
                    <SidebarEventItem
                      key={event.id}
                      event={event}
                      index={index}
                      totalItems={totalItems}
                      variant="queue"
                      isOpen={openEventIds.has(event.id) || event.isOpen}
                      onToggleOpen={onToggleOpen}
                      onEventClick={onEventClick}
                      onTriggerNavigate={handleTriggerNavigate}
                      tabData={getTabData?.(event)}
                      onCancelQueueItem={onCancelQueueItem}
                      onPushThrough={onPushThrough}
                      supportsPushThrough={supportsPushThrough}
                      onReEmit={onReEmit}
                      loadExecutionChain={loadExecutionChain}
                      getExecutionState={getExecutionState}
                    />
                  );
                })}
                {totalInQueueCount > 5 && (
                  <button
                    onClick={handleSeeQueue}
                    className="text-xs font-medium text-gray-500 hover:underline flex items-center gap-1 px-2 py-1"
                  >
                    <TextAlignStart size={16} />
                    {totalInQueueCount - 5} more in the queue
                  </button>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </>
  );
};
