import { CanvasesCanvasNodeQueueItem, ComponentsNode } from "@/api-client";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { formatRunTimestamp, resolveNodeIconSlug } from "../../canvasRunsUtils";
import { cn } from "@/lib/utils";
import { handleKeyboardActivation } from ".";
import { NodeIcon } from "./NodeIcon";
import { StatusBadge } from "./StatusBadge";
import { RUNS_CONSOLE_BADGE_COL } from "./constants";

export function QueueItemRow({
  item,
  node,
  componentIconMap,
  onNodeSelect,
}: {
  item: CanvasesCanvasNodeQueueItem;
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  onNodeSelect?: (nodeId: string) => void;
}) {
  const nodeName = node?.name || item.nodeId || "Unknown";
  const iconSrc = getHeaderIconSrc(node?.component?.name || node?.trigger?.name);
  const iconSlug = resolveNodeIconSlug(node, componentIconMap);

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
        onNodeSelect && "cursor-pointer hover:bg-gray-100 transition-colors",
      )}
      role={onNodeSelect ? "button" : undefined}
      tabIndex={onNodeSelect ? 0 : undefined}
      onClick={() => {
        if (onNodeSelect && item.nodeId) onNodeSelect(item.nodeId);
      }}
      onKeyDown={(e) => {
        if (onNodeSelect && item.nodeId) handleKeyboardActivation(e, () => onNodeSelect(item.nodeId!));
      }}
    >
      <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
        <NodeIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={nodeName} size={13} className="text-gray-400" />
      </div>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        <div className={RUNS_CONSOLE_BADGE_COL}>
          <StatusBadge status="queued" />
        </div>
        <span className="text-xs text-gray-700 truncate">{nodeName}</span>
      </div>
      <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">
        {item.createdAt ? formatRunTimestamp(item.createdAt) : ""}
      </span>
    </div>
  );
}
