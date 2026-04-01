import { CanvasesCanvasNodeExecution, ComponentsNode } from "@/api-client";
import { cn } from "@/lib/utils";
import { handleKeyboardActivation } from ".";
import { RUNS_CONSOLE_BADGE_COL } from ".";
import { StatusBadge } from "./StatusBadge";
import { NodeIcon } from "./NodeIcon";
import {
  computeDuration,
  formatRunTimestamp,
  resolveExecutionDisplayStatus,
  resolveNodeIconSlug,
} from "../../canvasRunsUtils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";

export function ExecutionRow({
  execution,
  node,
  componentIconMap,
  nodes,
  onSelect,
}: {
  execution: CanvasesCanvasNodeExecution;
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  nodes: ComponentsNode[];
  onSelect?: () => void;
}) {
  const { nodeName, iconSrc, iconSlug, status, duration, errorMessage, timestamp } = resolveExecutionRowData(
    execution,
    node,
    componentIconMap,
    nodes,
  );

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
        onSelect && "cursor-pointer hover:bg-gray-100 transition-colors",
      )}
      role={onSelect ? "button" : undefined}
      tabIndex={onSelect ? 0 : undefined}
      onClick={onSelect}
      onKeyDown={onSelect ? (e) => handleKeyboardActivation(e, onSelect) : undefined}
    >
      <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
        <NodeIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={nodeName} size={13} className="text-gray-400" />
      </div>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        <div className={RUNS_CONSOLE_BADGE_COL}>
          <StatusBadge status={status} />
        </div>
        <span className="text-xs text-gray-700 truncate">{nodeName}</span>
        {duration && <span className="text-xs text-gray-500 tabular-nums">{duration}</span>}
      </div>
      {errorMessage && <span className="text-xs text-red-600 truncate max-w-[300px]">{errorMessage}</span>}
      {timestamp && <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">{timestamp}</span>}
    </div>
  );
}

function resolveExecutionRowData(
  execution: CanvasesCanvasNodeExecution,
  node: ComponentsNode | undefined,
  componentIconMap: Record<string, string>,
  nodes: ComponentsNode[],
) {
  return {
    nodeName: node?.name || execution.nodeId || "Unknown",
    iconSrc: getHeaderIconSrc(node?.component?.name || node?.trigger?.name),
    iconSlug: resolveNodeIconSlug(node, componentIconMap),
    status: resolveExecutionDisplayStatus(execution, nodes),
    duration: computeDuration(execution),
    errorMessage: execution.result === "RESULT_FAILED" ? execution.resultMessage : undefined,
    timestamp: execution.createdAt ? formatRunTimestamp(execution.createdAt) : undefined,
  };
}
