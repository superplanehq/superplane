import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  CanvasesCanvasEventWithExecutions,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { getAggregateRunStatus, getStatusBadgeProps, resolveNodeIconSlug } from "@/pages/workflowv2/lib/canvas-runs";
import { getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { formatDuration } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { Filter, Link as LinkIcon, Loader2, Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Checkbox } from "@/ui/checkbox";
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
  size = 14,
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
  size?: number;
}) {
  if (iconSrc) {
    return (
      <img
        src={iconSrc}
        alt={alt}
        className="shrink-0 object-contain"
        style={{ width: `${size}px`, height: `${size}px` }}
      />
    );
  }
  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size,
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

type TriggerOption = {
  id: string;
  name: string;
  iconSrc: string | undefined;
  iconSlug: string | undefined;
};

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
  const nodeMap = useMemo(() => {
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

  const [search, setSearch] = useState("");
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(new Set());
  const [isFilterOpen, setIsFilterOpen] = useState(false);

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

  const triggerOptions = useMemo<TriggerOption[]>(() => {
    const opts: TriggerOption[] = [];
    for (const n of workflowNodes || []) {
      if (!n.id || !n.trigger) continue;
      const name = n.name || n.trigger.name || "Trigger";
      const iconSrc = getHeaderIconSrc(n.trigger.name);
      const iconSlug = resolveNodeIconSlug(n, componentIconMap || {}) || undefined;
      opts.push({ id: n.id, name, iconSrc, iconSlug });
    }
    opts.sort((a, b) => a.name.localeCompare(b.name));
    return opts;
  }, [workflowNodes, componentIconMap]);

  type DecoratedEvent = {
    event: CanvasesCanvasEventWithExecutions;
    triggerName: string;
    title: string;
    haystack: string;
  };

  const decoratedEvents = useMemo<DecoratedEvent[]>(() => {
    return events.map((event) => {
      const triggerNode = event.nodeId ? nodeMap.get(event.nodeId) : undefined;
      const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";
      const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
      const eventInfo = buildEventInfo(event);
      const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };
      const shortId = event.id ? event.id.slice(0, 8) : "";
      const haystack = `${title} ${triggerName} ${shortId}`.toLowerCase();
      return { event, triggerName, title: title || "", haystack };
    });
  }, [events, nodeMap]);

  const searchTrimmed = search.trim().toLowerCase();
  const hasSearch = searchTrimmed.length > 0;
  const hasTriggerFilter = selectedTriggerIds.size > 0;
  const hasAnyFilter = hasSearch || hasTriggerFilter;

  const filteredEvents = useMemo(() => {
    if (!hasAnyFilter) return decoratedEvents;
    return decoratedEvents.filter(({ event, haystack }) => {
      if (hasSearch && !haystack.includes(searchTrimmed)) return false;
      if (hasTriggerFilter) {
        if (!event.nodeId || !selectedTriggerIds.has(event.nodeId)) return false;
      }
      return true;
    });
  }, [decoratedEvents, hasAnyFilter, hasSearch, searchTrimmed, hasTriggerFilter, selectedTriggerIds]);

  const handleToggleTrigger = useCallback((id: string) => {
    setSelectedTriggerIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleClearFilters = useCallback(() => {
    setSearch("");
    setSelectedTriggerIds(new Set());
  }, []);

  const handleClearTriggerFilter = useCallback(() => {
    setSelectedTriggerIds(new Set());
  }, []);

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

      <div className="flex shrink-0 items-center gap-1.5 border-b border-slate-200 px-2 py-1.5">
        <Popover open={isFilterOpen} onOpenChange={setIsFilterOpen}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn(
                "relative h-7 w-7 shrink-0 p-0",
                hasTriggerFilter && "bg-sky-50 text-sky-700 hover:bg-sky-100",
              )}
              aria-label="Filter runs"
              title="Filter runs"
            >
              <Filter className="h-3.5 w-3.5" />
              {hasTriggerFilter ? (
                <span className="absolute -right-0.5 -top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-sky-500 px-1 text-[9px] font-semibold text-white">
                  {selectedTriggerIds.size}
                </span>
              ) : null}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="start" className="w-64 p-0" sideOffset={6}>
            <div className="flex items-center justify-between border-b border-slate-200 px-3 py-2">
              <span className="text-[12px] font-medium text-gray-700">Filter by trigger</span>
              <button
                type="button"
                onClick={handleClearTriggerFilter}
                disabled={!hasTriggerFilter}
                className={cn(
                  "text-[11px]",
                  hasTriggerFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-300",
                )}
              >
                Clear
              </button>
            </div>
            <div className="max-h-64 overflow-y-auto py-1">
              {triggerOptions.length === 0 ? (
                <div className="px-3 py-4 text-center text-[11px] text-gray-400">No triggers in this canvas</div>
              ) : (
                triggerOptions.map((opt) => {
                  const checked = selectedTriggerIds.has(opt.id);
                  return (
                    <label
                      key={opt.id}
                      className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
                    >
                      <Checkbox
                        checked={checked}
                        onCheckedChange={() => handleToggleTrigger(opt.id)}
                        className="h-3.5 w-3.5"
                      />
                      <TriggerIcon iconSrc={opt.iconSrc} iconSlug={opt.iconSlug} alt={opt.name} size={12} />
                      <span className="min-w-0 truncate">{opt.name}</span>
                    </label>
                  );
                })
              )}
            </div>
          </PopoverContent>
        </Popover>

        <InputGroup className="h-7 flex-1 border border-slate-200 shadow-none !ring-0 focus-within:!ring-0 focus-within:ring-offset-0 [&_[data-slot=input-group-control]]:!text-[12px]">
          <InputGroupAddon className="!text-[12px]">
            <Search className="h-3.5 w-3.5 text-gray-500" />
          </InputGroupAddon>
          <InputGroupInput
            placeholder="Search runs..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-6 !text-[12px] border-0 shadow-none focus:ring-0 focus-visible:ring-0 focus-visible:border-0"
          />
          {hasSearch ? (
            <InputGroupAddon>
              <button
                type="button"
                aria-label="Clear search"
                onClick={() => setSearch("")}
                className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
              >
                <X className="h-3 w-3" />
              </button>
            </InputGroupAddon>
          ) : null}
        </InputGroup>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
          </div>
        ) : events.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-gray-400">No runs yet</div>
        ) : filteredEvents.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-gray-400">
            <span>No runs match your filters</span>
            <button
              type="button"
              onClick={handleClearFilters}
              className="text-[11px] text-sky-600 hover:text-sky-800"
            >
              Clear filters
            </button>
          </div>
        ) : (
          <>
            {filteredEvents.map(({ event, triggerName, title }) => {
              const executions = event.executions || [];
              const pendingQueueCount = (event.queueItems || []).length;
              const status = getAggregateRunStatus(executions, pendingQueueCount > 0);
              const isSelected = event.id === selectedEventId;

              const triggerNode = event.nodeId ? nodeMap.get(event.nodeId) : undefined;
              const iconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
              const iconSlug = resolveNodeIconSlug(triggerNode, componentIconMap || {});

              //
              // Only show duration for terminal runs. If there are pending
              // queue items or a running execution, the run isn't actually
              // done -- the numeric duration would lie.
              //
              const duration =
                pendingQueueCount === 0 && status !== "running" ? computeRunDuration(event) : null;

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

      {hasAnyFilter && events.length > 0 ? (
        <div className="flex shrink-0 items-center justify-between gap-2 border-t border-slate-200 bg-slate-50 px-3 py-1.5 text-[11px] text-gray-500">
          <span>
            Showing {filteredEvents.length} of {events.length} loaded
          </span>
          <button
            type="button"
            onClick={handleClearFilters}
            className="shrink-0 text-sky-600 hover:text-sky-800"
          >
            Clear filters
          </button>
        </div>
      ) : null}

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
