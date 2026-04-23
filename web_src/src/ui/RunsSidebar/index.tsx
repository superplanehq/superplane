import React, { useCallback, useEffect, useRef, useState } from "react";
import type { CanvasesCanvasEventWithExecutions, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { getAggregateStatus, getStatusBadgeProps, resolveNodeIconSlug } from "@/pages/workflowv2/lib/canvas-runs";
import { getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { formatDuration } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { Link as LinkIcon, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

export const RUNS_SIDEBAR_WIDTH_STORAGE_KEY = "runs-sidebar-width";
const RUNS_SIDEBAR_MIN_WIDTH = 260;
const RUNS_SIDEBAR_MAX_WIDTH = 640;
const RUNS_SIDEBAR_DEFAULT_WIDTH = 320;

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

  const sidebarRef = useRef<HTMLDivElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = typeof window !== "undefined" ? localStorage.getItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY) : null;
    const parsed = saved ? parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) return RUNS_SIDEBAR_DEFAULT_WIDTH;
    return Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;
      const rect = sidebarRef.current?.getBoundingClientRect();
      const left = rect?.left ?? 0;
      const newWidth = e.clientX - left;
      const clampedWidth = Math.max(RUNS_SIDEBAR_MIN_WIDTH, Math.min(RUNS_SIDEBAR_MAX_WIDTH, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    localStorage.setItem(RUNS_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!isResizing) return;
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  return (
    <div
      ref={sidebarRef}
      className="relative flex shrink-0 flex-col border-r border-slate-200 bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
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
                <div
                  key={event.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => event.id && onSelectRun(event.id)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      event.id && onSelectRun(event.id);
                    }
                  }}
                  className={cn(
                    "group flex w-full cursor-pointer flex-col gap-1 border-b border-slate-100 px-3 py-2.5 text-left transition-colors",
                    isSelected ? "bg-sky-50" : "hover:bg-gray-50",
                  )}
                >
                  <div className="flex items-center gap-1.5">
                    <TriggerIcon iconSrc={iconSrc} iconSlug={iconSlug || undefined} alt={triggerName} />
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
                      <LinkIcon className="h-3 w-3" />
                    </button>
                    <span className="ml-auto shrink-0 text-[10px] tabular-nums text-gray-400">
                      {event.createdAt ? <TimeAgo date={event.createdAt} /> : ""}
                    </span>
                  </div>

                  <div className="flex items-center gap-1.5">
                    <RunStatusBadge status={status} />
                    {title ? <span className="min-w-0 truncate text-xs font-medium text-gray-800">{title}</span> : null}
                    {duration && (
                      <span className="ml-auto shrink-0 font-mono text-[10px] text-gray-400">{duration}</span>
                    )}
                  </div>
                </div>
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

      <div
        onMouseDown={handleMouseDown}
        className={cn(
          "absolute right-0 top-0 bottom-0 z-30 flex w-4 cursor-ew-resize items-center justify-center transition-colors hover:bg-gray-100 group",
          isResizing && "bg-blue-50",
        )}
        style={{ marginRight: "-8px" }}
        aria-label="Resize runs sidebar"
        role="separator"
      >
        <div
          className={cn(
            "h-14 w-2 rounded-full bg-gray-300 transition-colors group-hover:bg-gray-800",
            isResizing && "bg-blue-500",
          )}
        />
      </div>
    </div>
  );
}
