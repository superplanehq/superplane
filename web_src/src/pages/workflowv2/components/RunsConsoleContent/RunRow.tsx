import type {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsNode,
} from "@/api-client";
import { findNode } from "@/pages/workflowv2/lib/canvas-runs";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { useCallback, useMemo } from "react";
import { buildTriggerSidebarEvent } from "../../utils";
import { useQuery } from "@tanstack/react-query";
import { eventExecutionsQueryOptions } from "@/hooks/useCanvasData";
import { RunRowHeader } from "./RunRowHeader";
import { ExecutionRow } from "./ExecutionRow";
import { ExternalLink, Loader2 } from "lucide-react";
import { QueueItemRow } from "./QueueItemRow";
import { Button } from "@/components/ui/button";

export function RunRow({
  event,
  nodes,
  componentIconMap = {},
  queueItems = [],
  isExpanded,
  onToggle,
  onNodeSelect,
  onExecutionSelect,
  onOpenInRunView,
}: {
  event: CanvasesCanvasEventWithExecutions;
  nodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  queueItems?: CanvasesCanvasNodeQueueItem[];
  isExpanded: boolean;
  onToggle: () => void;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onOpenInRunView?: (eventId: string) => void;
}) {
  const canvasId = event.canvasId || "";
  const eventId = event.id || "";
  const enabled = isExpanded && !!event.canvasId && !!event.id;

  const triggerNode = useMemo(() => nodes.find((n) => n.id === event.nodeId), [nodes, event.nodeId]);
  const triggerSidebarEvent = useMemo(() => buildTriggerSidebarEvent(event, triggerNode), [event, triggerNode]);
  const executionRefs = useMemo(() => event.executions || [], [event.executions]);
  const executionDetailsQuery = useQuery({
    ...eventExecutionsQueryOptions(canvasId, eventId),
    enabled,
  });

  const detailedExecutions = useMemo(
    () => executionDetailsQuery.data?.executions || [],
    [executionDetailsQuery.data?.executions],
  );

  const makeExecutionSelectHandler = useCallback(
    (execution: CanvasesCanvasNodeExecution) => {
      if (onExecutionSelect && execution.id && event.id && execution.nodeId) {
        return () =>
          onExecutionSelect({
            nodeId: execution.nodeId!,
            eventId: event.id!,
            executionId: execution.id!,
            triggerEvent: triggerSidebarEvent,
          });
      }
      if (onNodeSelect && execution.nodeId) {
        return () => onNodeSelect(execution.nodeId!);
      }
      return undefined;
    },
    [onExecutionSelect, onNodeSelect, event.id, triggerSidebarEvent],
  );

  return (
    <div className="border-b border-gray-200 last:border-b-0">
      <RunRowHeader
        event={event}
        triggerNode={triggerNode}
        componentIconMap={componentIconMap}
        executions={executionRefs}
        queueItemCount={queueItems.length}
        isExpanded={isExpanded}
        onToggle={onToggle}
      />
      {isExpanded && (executionRefs.length > 0 || queueItems.length > 0) && (
        <div className="bg-gray-50">
          {onOpenInRunView && event.id ? (
            <div className="flex items-center border-t border-gray-200 px-4 py-1.5 pl-11">
              <Button
                variant="ghost"
                size="sm"
                className="h-6 gap-1 text-xs text-blue-600 hover:text-blue-700"
                onClick={() => onOpenInRunView(event.id!)}
              >
                <ExternalLink className="h-3 w-3" />
                Open in Run View
              </Button>
            </div>
          ) : null}
          {executionDetailsQuery.isPending && executionRefs.length > 0 && (
            <div className="flex items-center gap-2 px-4 py-2 pl-11 text-xs text-gray-500 border-t border-gray-200">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Loading run details...</span>
            </div>
          )}
          {executionDetailsQuery.isError && executionRefs.length > 0 && (
            <div className="px-4 py-2 pl-11 text-xs text-red-600 border-t border-gray-200">
              Could not load run details.
            </div>
          )}
          {detailedExecutions.map((execution) => (
            <ExecutionRow
              key={execution.id}
              execution={execution}
              node={findNode(nodes, execution.nodeId)}
              componentIconMap={componentIconMap}
              nodes={nodes}
              onSelect={makeExecutionSelectHandler(execution)}
            />
          ))}
          {queueItems.map((item) => (
            <QueueItemRow
              key={`q-${item.id}`}
              item={item}
              node={findNode(nodes, item.nodeId)}
              componentIconMap={componentIconMap}
              onNodeSelect={onNodeSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}
