import { useMemo } from "react";
import { resolveShowNodeRuntimeActivity } from "./resolveShowNodeRuntimeActivity";

export function useVisibleNodeRuntimeMaps<TExec, TQueue, TEvent>({
  showLiveActivity,
  factoryEmbed,
  isRunInspectionMode,
  nodeExecutionsMap,
  nodeQueueItemsMap,
  nodeEventsMap,
}: {
  showLiveActivity: boolean;
  factoryEmbed: boolean;
  isRunInspectionMode: boolean;
  nodeExecutionsMap: Record<string, TExec>;
  nodeQueueItemsMap: Record<string, TQueue>;
  nodeEventsMap: Record<string, TEvent>;
}) {
  const showNodeRuntimeActivity = resolveShowNodeRuntimeActivity({
    showLiveActivity,
    factoryEmbed,
    isRunInspectionMode,
  });

  const visibleNodeExecutionsMap = useMemo(
    () => (showNodeRuntimeActivity ? nodeExecutionsMap : {}),
    [showNodeRuntimeActivity, nodeExecutionsMap],
  );
  const visibleNodeQueueItemsMap = useMemo(
    () => (showNodeRuntimeActivity ? nodeQueueItemsMap : {}),
    [showNodeRuntimeActivity, nodeQueueItemsMap],
  );
  const visibleNodeEventsMap = useMemo(
    () => (showNodeRuntimeActivity ? nodeEventsMap : {}),
    [showNodeRuntimeActivity, nodeEventsMap],
  );

  return {
    showNodeRuntimeActivity,
    visibleNodeExecutionsMap,
    visibleNodeQueueItemsMap,
    visibleNodeEventsMap,
  };
}
