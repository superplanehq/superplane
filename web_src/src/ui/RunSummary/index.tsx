import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesDescribeRunResponse,
  ComponentsNode,
} from "@/api-client";
import { formatDuration } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { TimeAgo } from "@/components/TimeAgo";
import { getAggregateStatus, resolveNodeIconSlug } from "@/pages/workflowv2/lib/canvas-runs";
import { getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { Check, AlertTriangle, Clock, Circle, X } from "lucide-react";
import { useMemo } from "react";

interface RunSummaryProps {
  runData: CanvasesDescribeRunResponse;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
}

type StepStatus = "success" | "error" | "running" | "queued" | "cancelled";

interface Step {
  nodeId: string;
  name: string;
  status: StepStatus;
  startOffsetMs: number;
  durationMs: number;
  error?: string;
  reportEntry?: string;
  isTrigger: boolean;
}

const STATUS_STYLES: Record<StepStatus, { badge: string; label: string; dot: string; icon: string }> = {
  success: {
    badge: "bg-emerald-500 text-white",
    label: "Success",
    dot: "bg-emerald-500",
    icon: "bg-emerald-500",
  },
  error: {
    badge: "bg-red-500 text-white",
    label: "Failed",
    dot: "bg-red-500",
    icon: "bg-red-500",
  },
  running: {
    badge: "bg-amber-500 text-white",
    label: "Running",
    dot: "bg-amber-500",
    icon: "bg-amber-500",
  },
  queued: {
    badge: "bg-gray-400 text-white",
    label: "Queued",
    dot: "bg-gray-400",
    icon: "bg-gray-400",
  },
  cancelled: {
    badge: "bg-slate-500 text-white",
    label: "Cancelled",
    dot: "bg-slate-500",
    icon: "bg-slate-500",
  },
};

function StatusIcon({ status }: { status: StepStatus }) {
  const style = STATUS_STYLES[status];
  return (
    <div className={cn("flex h-8 w-8 shrink-0 items-center justify-center rounded-full", style.icon)}>
      {status === "success" ? (
        <Check className="h-4.5 w-4.5 text-white" />
      ) : status === "error" ? (
        <AlertTriangle className="h-4 w-4 text-white" />
      ) : status === "running" ? (
        <Clock className="h-4 w-4 text-white" />
      ) : status === "cancelled" ? (
        <Circle className="h-4 w-4 text-white" />
      ) : (
        <Circle className="h-4 w-4 text-white" />
      )}
    </div>
  );
}

function resolveStepStatus(exec: CanvasesCanvasNodeExecution | CanvasesCanvasNodeExecutionRef): StepStatus {
  if (exec.state === "STATE_PENDING") return "queued";
  if (exec.state === "STATE_STARTED") return "running";
  if (exec.result === "RESULT_FAILED") return "error";
  if (exec.result === "RESULT_CANCELLED") return "cancelled";
  if (exec.result === "RESULT_PASSED") return "success";
  if (exec.state === "STATE_FINISHED") return "success";
  return "queued";
}

function execDurationMs(exec: { createdAt?: string; updatedAt?: string; state?: string }): number {
  if (!exec.createdAt || !exec.updatedAt) return 0;
  if (exec.state !== "STATE_FINISHED") return 0;
  return Math.max(0, new Date(exec.updatedAt).getTime() - new Date(exec.createdAt).getTime());
}

function formatMs(ms: number): string {
  if (ms === 0) return "0s";
  return formatDuration(ms);
}

function formatAbsoluteTime(dateStr: string): string {
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(dateStr));
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
  const px = `${size}px`;
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} className="shrink-0 object-contain" style={{ height: px, width: px }} />;
  }
  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size,
    className: "shrink-0 text-gray-400",
  });
}

function buildSteps(
  runData: CanvasesDescribeRunResponse,
  nodeMap: Map<string, ComponentsNode>,
): { steps: Step[]; totalDurationMs: number; runStartMs: number } {
  const rootEvent = runData.run;
  const executions = runData.executions || [];
  const runStartMs = rootEvent?.createdAt ? new Date(rootEvent.createdAt).getTime() : 0;

  const steps: Step[] = [];

  if (rootEvent?.nodeId) {
    const triggerNode = nodeMap.get(rootEvent.nodeId);
    steps.push({
      nodeId: rootEvent.nodeId,
      name: triggerNode?.name || "Trigger",
      status: "success",
      startOffsetMs: 0,
      durationMs: 0,
      reportEntry: rootEvent.reportEntry || undefined,
      isTrigger: true,
    });
  }

  const sortedExecs = [...executions].sort((a, b) => {
    const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return ta - tb;
  });

  for (const exec of sortedExecs) {
    if (!exec.nodeId) continue;
    const node = nodeMap.get(exec.nodeId);
    const startMs = exec.createdAt ? new Date(exec.createdAt).getTime() : runStartMs;
    const dur = execDurationMs(exec);

    steps.push({
      nodeId: exec.nodeId,
      name: node?.name || exec.nodeId.slice(0, 8),
      status: resolveStepStatus(exec),
      startOffsetMs: Math.max(0, startMs - runStartMs),
      durationMs: dur,
      error: exec.result === "RESULT_FAILED" && exec.resultMessage ? exec.resultMessage : undefined,
      reportEntry: exec.reportEntry || undefined,
      isTrigger: false,
    });
  }

  let totalDurationMs = 0;
  for (const s of steps) {
    const end = s.startOffsetMs + s.durationMs;
    if (end > totalDurationMs) totalDurationMs = end;
  }

  if (totalDurationMs === 0) totalDurationMs = 1000;

  return { steps, totalDurationMs, runStartMs };
}

export function RunSummary({ runData, workflowNodes, componentIconMap }: RunSummaryProps) {
  const nodeMap = useMemo(() => {
    const m = new Map<string, ComponentsNode>();
    for (const n of workflowNodes || []) {
      if (n.id) m.set(n.id, n);
    }
    return m;
  }, [workflowNodes]);

  const { steps, totalDurationMs } = useMemo(() => buildSteps(runData, nodeMap), [runData, nodeMap]);

  const executions = runData.executions || [];
  const rootEvent = runData.run;
  const executionRefs: CanvasesCanvasNodeExecutionRef[] = rootEvent?.executions || [];
  const aggregateStatus =
    executionRefs.length > 0 ? getAggregateStatus(executionRefs) : executions.length > 0 ? "completed" : "queued";
  const headerStatus = (aggregateStatus as StepStatus) in STATUS_STYLES ? (aggregateStatus as StepStatus) : "queued";

  const stepCount = new Set(executions.map((e) => e.nodeId).filter(Boolean)).size;
  const successCount = steps.filter((s) => s.status === "success" && !s.isTrigger).length;
  const errorCount = steps.filter((s) => s.status === "error").length;

  const slowestStep = useMemo(() => {
    const execSteps = steps.filter((s) => !s.isTrigger && s.durationMs > 0);
    if (execSteps.length === 0) return null;
    return execSteps.reduce((a, b) => (a.durationMs >= b.durationMs ? a : b));
  }, [steps]);

  const triggerNode = rootEvent?.nodeId ? nodeMap.get(rootEvent.nodeId) : undefined;
  const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";
  const triggerIconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
  const triggerIconSlug = resolveNodeIconSlug(triggerNode, componentIconMap || {});

  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = rootEvent ? buildEventInfo(rootEvent) : undefined;
  const { title: runTitle } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };

  return (
    <div className="flex h-full w-full flex-col overflow-y-auto bg-slate-50 p-6">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
        {/* Header Card */}
        <div className="rounded-lg border border-slate-200 bg-white shadow-sm">
          <div className="p-5">
            {/* Identity row: status icon + title + metadata */}
            <div className="flex items-start gap-3.5">
              <StatusIcon status={headerStatus} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  {runTitle ? (
                    <span className="truncate text-base font-semibold text-gray-900">{runTitle}</span>
                  ) : (
                    <span className="text-base font-semibold text-gray-900">{STATUS_STYLES[headerStatus].label}</span>
                  )}
                  {rootEvent?.customName ? (
                    <span className="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500">
                      {rootEvent.customName}
                    </span>
                  ) : null}
                </div>
                <div className="mt-1 flex items-center gap-1.5 text-xs text-gray-400">
                  <TriggerIcon iconSrc={triggerIconSrc} iconSlug={triggerIconSlug} alt={triggerName} />
                  <span className="text-gray-500">{triggerName}</span>
                  <span className="text-gray-300">&middot;</span>
                  <span className="font-mono">#{rootEvent?.id?.slice(0, 8)}</span>
                  {rootEvent?.createdAt ? (
                    <>
                      <span className="text-gray-300">&middot;</span>
                      <span>{formatAbsoluteTime(rootEvent.createdAt)}</span>
                      <span className="text-gray-300">
                        (<TimeAgo date={rootEvent.createdAt} />)
                      </span>
                    </>
                  ) : null}
                </div>
              </div>
            </div>
          </div>

          {/* Metrics row */}
          <div className="flex items-center gap-6 border-t border-slate-100 px-5 py-3 text-xs">
            <div className="flex items-center gap-1.5">
              <span className="font-medium uppercase tracking-wider text-gray-400">Duration</span>
              <span className="font-semibold tabular-nums text-gray-800">{formatMs(totalDurationMs)}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="font-medium uppercase tracking-wider text-gray-400">Steps</span>
              <span className="font-semibold tabular-nums text-gray-800">{stepCount}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Check className="h-3 w-3 text-emerald-500" />
              <span className="font-semibold tabular-nums text-emerald-600">{successCount}</span>
            </div>
            {errorCount > 0 && (
              <div className="flex items-center gap-1.5">
                <X className="h-3 w-3 text-red-500" />
                <span className="font-semibold tabular-nums text-red-600">{errorCount}</span>
              </div>
            )}
            {slowestStep && (
              <div className="ml-auto flex items-center gap-1.5 text-gray-400">
                <span className="uppercase tracking-wider">Slowest</span>
                <span className="max-w-28 truncate font-medium text-gray-600">{slowestStep.name}</span>
                <span className="font-semibold tabular-nums text-gray-800">{formatMs(slowestStep.durationMs)}</span>
              </div>
            )}
          </div>
        </div>

        {/* Report */}
        <div className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">Report</h3>
          {steps.some((s) => s.reportEntry) ? (
            <div className="prose prose-sm prose-gray max-w-none text-sm text-gray-700 [&_a]:text-blue-600 [&_a]:underline [&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_code]:rounded [&_code]:bg-gray-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h1]:text-base [&_h1]:font-semibold [&_h1]:mb-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:mb-1 [&_h3]:text-xs [&_h3]:font-semibold [&_h3]:mb-1">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {steps
                  .filter((s) => s.reportEntry)
                  .map((s) => s.reportEntry)
                  .join("\n\n")}
              </ReactMarkdown>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
