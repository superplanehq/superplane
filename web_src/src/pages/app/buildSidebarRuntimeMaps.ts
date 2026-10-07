type SidebarNodeRuntimeData<TExecution, TQueueItem, TEvent> = {
  executions: TExecution[];
  queueItems: TQueueItem[];
  events: TEvent[];
  totalInHistoryCount: number;
  totalInQueueCount: number;
};

/** Build per-node maps for the component sidebar from live store data. */
export function buildSidebarRuntimeMaps<TExecution, TQueueItem, TEvent>({
  nodeId,
  showNodeRuntimeActivity,
  nodeData,
  fallbackEvents,
}: {
  nodeId: string;
  showNodeRuntimeActivity: boolean;
  nodeData: SidebarNodeRuntimeData<TExecution, TQueueItem, TEvent>;
  fallbackEvents: TEvent[] | undefined;
}) {
  if (!showNodeRuntimeActivity) {
    return {
      executionsMap: {} as Record<string, TExecution[]>,
      queueItemsMap: {} as Record<string, TQueueItem[]>,
      eventsMap: {} as Record<string, TEvent[]>,
      totalHistoryCount: 0,
      totalQueueCount: 0,
    };
  }

  return {
    executionsMap: nodeData.executions.length === 0 ? {} : { [nodeId]: nodeData.executions },
    queueItemsMap: nodeData.queueItems.length === 0 ? {} : { [nodeId]: nodeData.queueItems.slice().reverse() },
    eventsMap:
      nodeData.events.length === 0
        ? {}
        : { [nodeId]: nodeData.events.length > 0 ? nodeData.events : (fallbackEvents ?? []) },
    totalHistoryCount: nodeData.totalInHistoryCount,
    totalQueueCount: nodeData.totalInQueueCount,
  };
}
