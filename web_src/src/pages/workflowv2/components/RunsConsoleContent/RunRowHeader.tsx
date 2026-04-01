import { useMemo } from "react";
import {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  ComponentsNode,
} from "@/api-client";
import { getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { computeDuration, getAggregateStatus, resolveNodeIconSlug } from "@/pages/workflowv2/canvasRunsUtils";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { RUNS_CONSOLE_BADGE_COL } from ".";
import { StatusBadge } from "./StatusBadge";
import { NodeIcon } from "./NodeIcon";
import { formatRunTimestamp } from "../../canvasRunsUtils";

export function RunRowHeader({
  event,
  triggerNode,
  componentIconMap,
  executions,
  queueItemCount,
  isExpanded,
  onToggle,
}: {
  event: CanvasesCanvasEventWithExecutions;
  triggerNode: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  executions: CanvasesCanvasNodeExecutionRef[];
  queueItemCount: number;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = buildEventInfo(event);
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "Run" };
  const aggregateStatus = executions.length > 0 ? getAggregateStatus(executions, []) : "queued";
  const totalSteps = executions.length + queueItemCount;

  const errorAckLabel = useMemo(() => {
    const failedExecs = executions.filter((e) => e.result === "RESULT_FAILED");
    if (failedExecs.length === 0) return null;
    return failedExecs.every((e) => e.resultReason === "RESULT_REASON_ERROR_RESOLVED") ? "acknowledged" : "new";
  }, [executions]);

  const totalDuration = useMemo(() => {
    if (!event.createdAt || executions.length === 0) return null;
    if (!executions.every((e) => e.state === "STATE_FINISHED")) return null;
    const startMs = new Date(event.createdAt).getTime();
    let latestEndMs = startMs;
    for (const exec of executions) {
      if (exec.updatedAt) {
        const endMs = new Date(exec.updatedAt).getTime();
        if (endMs > latestEndMs) latestEndMs = endMs;
      }
    }
    if (latestEndMs <= startMs) return null;
    return computeDuration({
      createdAt: event.createdAt,
      updatedAt: new Date(latestEndMs).toISOString(),
      state: "STATE_FINISHED",
    } as CanvasesCanvasNodeExecution);
  }, [event.createdAt, executions]);

  const triggerIconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
  const triggerIconSlug = resolveNodeIconSlug(triggerNode, componentIconMap);
  const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";

  return (
    <button
      type="button"
      onClick={onToggle}
      className="flex h-8 w-full items-center gap-3 px-4 text-left hover:bg-gray-50 transition-colors"
      aria-expanded={isExpanded}
    >
      <div className="text-gray-400">
        {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
      </div>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        <div className={RUNS_CONSOLE_BADGE_COL}>
          <StatusBadge status={aggregateStatus} />
          {errorAckLabel && (
            <span
              className={cn(
                "shrink-0 text-[10px] font-medium rounded px-1.5 py-0.5",
                errorAckLabel === "new" ? "bg-red-50 text-red-600" : "bg-gray-100 text-gray-500",
              )}
            >
              {errorAckLabel === "new" ? "New" : "Acknowledged"}
            </span>
          )}
        </div>
        <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
          <NodeIcon iconSrc={triggerIconSrc} iconSlug={triggerIconSlug || "bolt"} alt={triggerName} />
        </div>
        <span className="text-xs text-gray-600 truncate flex-shrink-0">{triggerName}</span>
        <span className="text-gray-300">·</span>
        <span className="font-mono text-xs text-gray-500">#{event.id?.slice(0, 4)}</span>
        <span className="text-xs font-medium text-gray-900 truncate">{title}</span>
        {totalSteps > 0 && (
          <span className="text-xs text-gray-400 whitespace-nowrap">
            {totalSteps} {totalSteps === 1 ? "step" : "steps"}
            {totalDuration && <span className="ml-1">· {totalDuration}</span>}
          </span>
        )}
      </div>
      <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
        {event.createdAt ? formatRunTimestamp(event.createdAt) : ""}
      </span>
    </button>
  );
}
