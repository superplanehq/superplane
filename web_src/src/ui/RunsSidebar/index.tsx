import React from "react";
import type { CanvasesCanvasEventWithExecutions, ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { getAggregateStatus, getStatusBadgeProps, resolveNodeIconSlug } from "@/pages/workflowv2/lib/canvas-runs";
import { getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { formatDuration } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { resolveIcon } from "@/lib/utils";
import { Link, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface RunsSidebarProps {
  events: CanvasesCanvasEventWithExecutions[];
  selectedEventId: string | null;
  onSelectRun: (eventId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  totalCount?: number;
}

function RunStatusBadge({ status }: { status: string }) {
  const { badgeColor, label } = getStatusBadgeProps(status);
  return (
    <span
      className={cn(
        "shrink-0 rounded px-[4px] py-[1px] text-[10px] font-semibold uppercase tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

function TriggerIcon({
  iconSrc,
  iconSlug,
  alt,
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
}) {
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} className="h-3.5 w-3.5 shrink-0 object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size: 14,
    className: "shrink-0 text-gray-400",
  });
}

function computeRunDuration(event: CanvasesCanvasEventWithExecutions): string | null {
  const executions = event.executions || [];
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
  return formatDuration(latestEndMs - startMs);
}

export function RunsSidebar({
  events,
  selectedEventId,
  onSelectRun,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  isLoading,
  workflowNodes,
  componentIconMap,
  totalCount,
}: RunsSidebarProps) {
  const nodeMap = React.useMemo(() => {
    const m = new Map<string, ComponentsNode>();
    for (const n of workflowNodes || []) {
      if (n.id) m.set(n.id, n);
    }
    return m;
  }, [workflowNodes]);

  return (
    <div className="flex w-80 shrink-0 flex-col border-r border-slate-200 bg-white">
      <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
        <span className="text-sm font-medium text-gray-700">Runs</span>
        {totalCount != null && totalCount > 0 && <span className="ml-1.5 text-xs text-gray-400">({totalCount})</span>}
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
          </div>
        ) : events.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-gray-400">No runs yet</div>
        ) : (
          <>
            {events.map((event) => {
              const executions = event.executions || [];
              const status = executions.length > 0 ? getAggregateStatus(executions) : "queued";
              const isSelected = event.id === selectedEventId;

              const triggerNode = event.nodeId ? nodeMap.get(event.nodeId) : undefined;
              const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";
              const iconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
              const iconSlug = resolveNodeIconSlug(triggerNode, componentIconMap || {});

              const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
              const eventInfo = buildEventInfo(event);
              const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };

              const duration = computeRunDuration(event);

              return (
                <button
                  key={event.id}
                  type="button"
                  onClick={() => event.id && onSelectRun(event.id)}
                  className={cn(
                    "group flex w-full flex-col gap-1 border-b border-slate-100 px-3 py-2.5 text-left transition-colors",
                    isSelected ? "bg-sky-50" : "hover:bg-gray-50",
                  )}
                >
                  {/* Row 1: Trigger icon + name + short ID + time */}
                  <div className="flex items-center gap-1.5">
                    <TriggerIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={triggerName} />
                    <span className="truncate text-xs text-gray-600">{triggerName}</span>
                    <span className="text-gray-300">&middot;</span>
                    <span className="shrink-0 font-mono text-[10px] text-gray-400">#{event.id?.slice(0, 4)}</span>
                    <button
                      type="button"
                      title="Copy link to run"
                      className="ml-auto hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:block"
                      onClick={(e) => {
                        e.stopPropagation();
                        const url = new URL(window.location.href);
                        url.searchParams.set("run", event.id || "");
                        navigator.clipboard.writeText(url.toString());
                        toast.success("Run link copied");
                      }}
                    >
                      <Link className="h-3 w-3" />
                    </button>
                    <span className="ml-auto shrink-0 text-[10px] tabular-nums text-gray-400">
                      {event.createdAt ? <TimeAgo date={event.createdAt} /> : ""}
                    </span>
                  </div>

                  {/* Row 2: Status badge + run title + duration */}
                  <div className="flex items-center gap-1.5">
                    <RunStatusBadge status={status} />
                    {title ? <span className="min-w-0 truncate text-xs font-medium text-gray-800">{title}</span> : null}
                    {duration && (
                      <span className="ml-auto shrink-0 font-mono text-[10px] text-gray-400">{duration}</span>
                    )}
                  </div>
                </button>
              );
            })}
            {hasNextPage && onLoadMore ? (
              <div className="px-3 py-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="w-full text-xs"
                  onClick={onLoadMore}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : null}
                  Load more
                </Button>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
