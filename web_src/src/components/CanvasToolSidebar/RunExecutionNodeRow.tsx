import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { cn } from "@/lib/utils";
import { EventStatusBadge } from "@/ui/EventStatusBadge";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { eventBadgeForExecution, eventBadgeForTriggeredTrigger } from "@/ui/Runs/runNodeDetailModel";

function ReplayLineage({ isReplay, sourceExecutionId }: { isReplay: boolean; sourceExecutionId?: string }) {
  if (!isReplay) return null;

  return (
    <span
      data-testid="replay-badge"
      title={sourceExecutionId ? `Replay of execution ${sourceExecutionId}` : "Replay"}
      className="shrink-0 rounded bg-violet-100 px-1.5 py-0.5 text-[10px] font-medium text-violet-700 dark:bg-violet-950 dark:text-violet-300"
    >
      Replay
    </span>
  );
}

interface RunExecutionNodeRowProps {
  nodeId: string;
  workflowNode?: ComponentsNode;
  componentIconMap: Record<string, string>;
  execution?: CanvasesCanvasNodeExecution;
  isTrigger: boolean;
  isSelected: boolean;
  onSelect: (nodeId: string) => void;
}

export function RunExecutionNodeRow({
  nodeId,
  workflowNode,
  componentIconMap,
  execution,
  isTrigger,
  isSelected,
  onSelect,
}: RunExecutionNodeRowProps) {
  const iconSrc = getHeaderIconSrc(workflowNode?.component);
  const iconSlug = workflowNode?.component ? componentIconMap[workflowNode.component] : undefined;
  const nodeName = workflowNode?.name || nodeId;

  const badge = isTrigger
    ? eventBadgeForTriggeredTrigger(workflowNode)
    : execution
      ? eventBadgeForExecution(workflowNode, execution)
      : null;

  //
  // Replay lineage lives on the run, but ListNodeExecutions surfaces it on every
  // execution it serializes, so the row can render the badge without a second
  // round trip. The source execution id is absent when the run that produced the
  // replay predates the pinned reference, so the badge stands on its own then.
  //
  const isReplay = Boolean(execution?.isReplay);

  return (
    <div
      data-testid="run-execution-node-row"
      role="button"
      tabIndex={0}
      onClick={() => onSelect(nodeId)}
      onKeyDown={(event) => {
        if (event.key !== "Enter" && event.key !== " ") return;
        event.preventDefault();
        onSelect(nodeId);
      }}
      className={cn(
        "flex w-full cursor-pointer items-center gap-2 border-b border-b-slate-950/10 px-3 py-2 text-left transition-colors dark:border-gray-800/70",
        isSelected ? "bg-sky-100 dark:bg-indigo-950" : "hover:bg-gray-50 dark:hover:bg-gray-800",
      )}
    >
      <RunNodeIcon
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        alt={nodeName}
        size={RUN_NODE_ICON_SIZE}
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          isSelected ? "text-gray-800 dark:text-gray-100" : "text-gray-500 dark:text-gray-400",
        )}
      />
      <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-gray-800 dark:text-gray-100">
        {nodeName}
      </span>
      <ReplayLineage isReplay={isReplay} sourceExecutionId={execution?.replaySourceExecutionId} />
      {badge ? <EventStatusBadge badgeColor={badge.badgeColor} label={badge.label} /> : null}
    </div>
  );
}
