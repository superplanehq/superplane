import React, { useMemo } from "react";
import { Clock } from "lucide-react";
import type {
  CanvasesCanvasNodeExecutionRef,
  CanvasesDescribeRunResponse,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDurationSeconds } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { getAggregateStatus, resolveNodeIconSlug } from "@/pages/workflowv2/lib/canvas-runs";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";

export type RunViewMode = "summary" | "canvas";

interface RunContextHeaderProps {
  runData: CanvasesDescribeRunResponse | null;
  isLoading?: boolean;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  viewMode: RunViewMode;
  onChangeViewMode: (mode: RunViewMode) => void;
}

type HeaderStatus = "success" | "error" | "running" | "queued" | "cancelled";

const STATUS_DOT_CLASS: Record<HeaderStatus, string> = {
  success: "bg-emerald-500",
  error: "bg-red-500",
  running: "bg-amber-500 animate-pulse",
  queued: "bg-gray-400",
  cancelled: "bg-slate-500",
};

const STATUS_LABEL: Record<HeaderStatus, string> = {
  success: "Succeeded",
  error: "Failed",
  running: "Running",
  queued: "Queued",
  cancelled: "Cancelled",
};

function aggregateToHeaderStatus(aggregate: string): HeaderStatus {
  if (aggregate === "completed") return "success";
  if (aggregate === "error") return "error";
  if (aggregate === "running") return "running";
  if (aggregate === "cancelled") return "cancelled";
  return "queued";
}

function formatAbsoluteTime(dateStr: string): string {
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(dateStr));
}

function computeRunDurationMs(runData: CanvasesDescribeRunResponse): number | null {
  const run = runData.run;
  if (!run?.createdAt) return null;
  const executions = runData.executions || [];
  if (executions.length === 0) return null;

  const allFinished = executions.every((e) => e.state === "STATE_FINISHED");
  const startMs = new Date(run.createdAt).getTime();
  let latestEndMs = startMs;
  for (const exec of executions) {
    const ts = exec.updatedAt || exec.createdAt;
    if (!ts) continue;
    const endMs = new Date(ts).getTime();
    if (endMs > latestEndMs) latestEndMs = endMs;
  }
  if (!allFinished) {
    const nowMs = Date.now();
    if (nowMs > latestEndMs) latestEndMs = nowMs;
  }
  if (latestEndMs <= startMs) return null;
  return latestEndMs - startMs;
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

function ViewTabs({ value, onChange }: { value: RunViewMode; onChange: (v: RunViewMode) => void }) {
  return (
    <Tabs
      value={value}
      onValueChange={(v) => {
        if (v === "summary" || v === "canvas") onChange(v);
      }}
      className="inline-flex w-auto"
      aria-label="Run view"
    >
      <TabsList className="h-7 w-fit gap-0 rounded-sm border border-slate-300 bg-white/80 p-0">
        <TabsTrigger
          value="summary"
          aria-label="Summary"
          className="rounded-sm rounded-br-none rounded-tr-none border-none px-3 py-0.5 text-xs text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Summary
        </TabsTrigger>
        <div className="h-full w-px bg-slate-300" />
        <TabsTrigger
          value="canvas"
          aria-label="Canvas"
          className="rounded-sm rounded-bl-none rounded-tl-none border-none px-3 py-0.5 text-xs text-slate-600 transition-colors data-[state=active]:bg-sky-50 data-[state=active]:text-sky-700 data-[state=active]:shadow-none"
        >
          Canvas
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

export function RunContextHeader({
  runData,
  isLoading,
  workflowNodes,
  componentIconMap,
  viewMode,
  onChangeViewMode,
}: RunContextHeaderProps) {
  const nodeMap = useMemo(() => {
    const m = new Map<string, ComponentsNode>();
    for (const n of workflowNodes || []) {
      if (n.id) m.set(n.id, n);
    }
    return m;
  }, [workflowNodes]);

  const rootEvent = runData?.run;
  const executionRefs: CanvasesCanvasNodeExecutionRef[] = rootEvent?.executions || [];
  const executions = runData?.executions || [];

  const aggregate =
    executionRefs.length > 0
      ? getAggregateStatus(executionRefs)
      : executions.length > 0
        ? "completed"
        : "queued";
  const status = aggregateToHeaderStatus(aggregate);

  const triggerNode = rootEvent?.nodeId ? nodeMap.get(rootEvent.nodeId) : undefined;
  const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";
  const iconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
  const iconSlug = resolveNodeIconSlug(triggerNode, componentIconMap || {});
  const shortId = rootEvent?.id ? rootEvent.id.slice(0, 8) : "";

  const durationMs = runData ? computeRunDurationMs(runData) : null;
  const durationLabel = durationMs != null ? formatDurationSeconds(durationMs) : null;

  // customName surfaces user-labeled events (e.g. re-run markers); keep as a subtle pill
  const customName = rootEvent?.customName;

  const showSkeleton = isLoading && !runData;

  return (
    <div className="pointer-events-auto relative z-20 flex h-10 shrink-0 items-center gap-3 border-b border-slate-200 bg-white px-3">
      {showSkeleton ? (
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <div className="h-2 w-2 animate-pulse rounded-full bg-slate-300" />
          <div className="h-3 w-20 animate-pulse rounded bg-slate-200" />
          <div className="h-3 w-32 animate-pulse rounded bg-slate-200" />
        </div>
      ) : (
        <div className="flex min-w-0 flex-1 items-center gap-2 text-xs text-gray-600">
          <span
            className={cn("h-2 w-2 shrink-0 rounded-full", STATUS_DOT_CLASS[status])}
            title={STATUS_LABEL[status]}
            aria-label={STATUS_LABEL[status]}
          />
          <TriggerIcon iconSrc={iconSrc} iconSlug={iconSlug || undefined} alt={triggerName} />
          <span className="max-w-[180px] truncate font-medium text-gray-800">{triggerName}</span>
          {shortId ? (
            <>
              <span className="text-gray-300">&middot;</span>
              <span className="shrink-0 font-mono text-[11px] text-gray-400">#{shortId}</span>
            </>
          ) : null}
          {rootEvent?.createdAt ? (
            <>
              <span className="text-gray-300">&middot;</span>
              <span className="shrink-0 whitespace-nowrap text-gray-500">{formatAbsoluteTime(rootEvent.createdAt)}</span>
              <span className="shrink-0 whitespace-nowrap text-gray-400">
                (<TimeAgo date={rootEvent.createdAt} />)
              </span>
            </>
          ) : null}
          {customName ? (
            <span className="ml-1 shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500">
              {customName}
            </span>
          ) : null}
        </div>
      )}

      <div className="flex shrink-0 items-center gap-2">
        {durationLabel ? (
          <span className="flex items-center gap-1 text-xs tabular-nums text-gray-500" title="Run duration">
            <Clock className="h-3 w-3 text-gray-400" />
            {durationLabel}
          </span>
        ) : null}
        <ViewTabs value={viewMode} onChange={onChangeViewMode} />
      </div>
    </div>
  );
}
