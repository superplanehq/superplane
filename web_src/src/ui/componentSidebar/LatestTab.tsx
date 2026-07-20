import { useMemo } from "react";
import { TextAlignStart } from "lucide-react";
import { RUNS_SIDEBAR_ROW_CLASS } from "@/components/CanvasToolSidebar/runsSidebarRowLayout";
import { cn } from "@/lib/utils";
import { CompactSidebarEventRow } from "./CompactSidebarEventRow";
import { SidebarEventItem } from "./SidebarEventItem";
import type { TabData } from "./SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "./types";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import type { EventState, EventStateMap } from "../componentBase";

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
  getTabData?: (event: SidebarEvent) => TabData | undefined;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  compact?: boolean;
  selectionNodeId?: string;
  resolveRunId?: (event: SidebarEvent) => string | null;
  fetchRunId?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRun?: (runId: string, options?: { nodeId?: string }) => void;
}

function SectionHeader({ label }: { label: string }) {
  return (
    <h2 className="flex h-9 shrink-0 items-center border-b border-b-slate-950/10 px-3 text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:border-gray-800/70 dark:text-gray-400">
      {label}
    </h2>
  );
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
  getTabData,
  onCancelQueueItem,
  onCancelExecution,
  onReEmit,
  getExecutionState,
  compact = false,
  selectionNodeId,
  resolveRunId,
  fetchRunId,
  onSelectRun,
}: LatestTabProps) => {
  const compactLatestEvents = useMemo(() => latestEvents.slice(0, 5), [latestEvents]);
  const compactQueueEvents = useMemo(() => nextInQueueEvents.slice(0, 5), [nextInQueueEvents]);
  const runIdByEventId = useMemo(() => {
    const map = new Map<string, string | null>();
    if (!resolveRunId) {
      return map;
    }

    for (const event of [...compactLatestEvents, ...compactQueueEvents]) {
      map.set(event.id, resolveRunId(event));
    }

    return map;
  }, [compactLatestEvents, compactQueueEvents, resolveRunId]);

  const handleSeeQueue = () => {
    onSeeQueue?.();
  };

  const handleSeeFullHistory = () => {
    onSeeFullHistory?.();
  };

  if (compact) {
    return (
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <SectionHeader label="Latest" />
        {latestEvents.length === 0 ? (
          <div className="px-3 py-4 text-center text-xs text-gray-500 dark:text-gray-400">No events found</div>
        ) : (
          <>
            {compactLatestEvents.map((event) => (
              <CompactSidebarEventRow
                key={event.id}
                event={event}
                selectionNodeId={selectionNodeId}
                runId={runIdByEventId.get(event.id) ?? null}
                fetchRunId={fetchRunId}
                onSelectRun={onSelectRun}
                onCancelExecution={onCancelExecution}
                onReEmit={onReEmit}
                getExecutionState={getExecutionState}
              />
            ))}
            {handleSeeFullHistory ? (
              <button
                type="button"
                onClick={handleSeeFullHistory}
                className={cn(
                  RUNS_SIDEBAR_ROW_CLASS,
                  "w-full text-xs font-medium text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
                )}
              >
                <TextAlignStart className="h-3.5 w-3.5 shrink-0" />
                See full history
              </button>
            ) : null}
          </>
        )}
        {!hideQueueEvents ? (
          <>
            <SectionHeader label="Queued" />
            {nextInQueueEvents.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-gray-500 dark:text-gray-400">Queue is empty</div>
            ) : (
              <>
                {compactQueueEvents.map((event) => (
                  <CompactSidebarEventRow
                    key={event.id}
                    event={event}
                    selectionNodeId={selectionNodeId}
                    runId={runIdByEventId.get(event.id) ?? null}
                    fetchRunId={fetchRunId}
                    onSelectRun={onSelectRun}
                    onCancelQueueItem={onCancelQueueItem}
                    onReEmit={onReEmit}
                    getExecutionState={getExecutionState}
                  />
                ))}
                {totalInQueueCount > 5 ? (
                  <button
                    type="button"
                    onClick={handleSeeQueue}
                    className={cn(
                      RUNS_SIDEBAR_ROW_CLASS,
                      "w-full text-xs font-medium text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
                    )}
                  >
                    <TextAlignStart className="h-3.5 w-3.5 shrink-0" />
                    {totalInQueueCount - 5} more in the queue
                  </button>
                ) : null}
              </>
            )}
          </>
        ) : null}
      </div>
    );
  }

  return (
    <div className="overflow-y-auto pb-20" style={{ maxHeight: "85vh" }}>
      <div className="p-4 border-b-1 border-border text-left">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-3">Latest</h2>
        <div className="flex flex-col">
          {latestEvents.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm font-medium">No events found</div>
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
                    isOpen={false}
                    onToggleOpen={onToggleOpen}
                    onEventClick={onEventClick}
                    tabData={getTabData?.(event)}
                    onCancelExecution={onCancelExecution}
                    onReEmit={onReEmit}
                    getExecutionState={getExecutionState}
                  />
                );
              })}
              {handleSeeFullHistory && (
                <button
                  onClick={handleSeeFullHistory}
                  className="text-sm text-gray-500 font-medium hover:text-gray-800 flex items-center gap-1 mt-4"
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
        <div className="p-4 text-left">
          <h2 className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-3">Queued</h2>
          <div className="flex flex-col">
            {nextInQueueEvents.length === 0 ? (
              <div className="text-center py-4 text-gray-500 text-sm font-medium">Queue is empty</div>
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
                      tabData={getTabData?.(event)}
                      onCancelQueueItem={onCancelQueueItem}
                      onReEmit={onReEmit}
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
    </div>
  );
};
