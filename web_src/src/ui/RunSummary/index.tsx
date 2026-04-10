import React, { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ReportMarkdown } from "./ReportMarkdown";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesDescribeRunResponse,
  ComponentsNode,
} from "@/api-client";
import { canvasesInvokeNodeExecutionAction } from "@/api-client";
import { formatDuration } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { TimeAgo } from "@/components/TimeAgo";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useMe } from "@/hooks/useMe";
import {
  getAggregateStatus,
  getStatusBadgeProps,
  resolveExecutionDisplayStatus,
  resolveNodeIconSlug,
} from "@/pages/workflowv2/lib/canvas-runs";
import { getStateMap, getTriggerRenderer } from "@/pages/workflowv2/mappers";
import { buildEventInfo } from "@/pages/workflowv2/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { NodeIcon } from "@/pages/workflowv2/components/RunsConsoleContent/NodeIcon";
import { Check, AlertTriangle, Clock, Circle, X, Loader2 } from "lucide-react";
import { useMemo } from "react";

interface RunSummaryProps {
  runData: CanvasesDescribeRunResponse;
  canvasId?: string;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onPushThrough?: (nodeId: string, executionId: string) => void;
  onCancelExecution?: (nodeId: string, executionId: string) => void;
  supportsPushThrough?: (nodeId: string) => boolean;
}

type StepStatus = "success" | "error" | "running" | "queued" | "cancelled";

interface Step {
  nodeId: string;
  executionId?: string;
  name: string;
  componentName?: string;
  status: string;
  finished: boolean;
  startOffsetMs: number;
  durationMs: number;
  elapsedMs: number;
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

interface ApprovalRecord {
  index: number;
  state: string;
  type: string;
  user?: { id?: string; email?: string; name?: string };
  role?: string;
  group?: string;
}

function findActionableApprovalIndex(
  execution: CanvasesCanvasNodeExecution | undefined,
  me: { id?: string; email?: string; roles?: string[] } | null | undefined,
): number | undefined {
  if (!execution?.metadata || !me) return undefined;
  const records = execution.metadata.records as ApprovalRecord[] | undefined;
  if (!Array.isArray(records)) return undefined;

  const hasApproved = records.some(
    (r) => r.state === "approved" && (r.user?.id === me.id || (me.email && r.user?.email === me.email)),
  );
  if (hasApproved) return undefined;

  for (const record of records) {
    if (record.state !== "pending") continue;
    if (record.type === "anyone") return record.index;
    if (record.type === "user" && (record.user?.id === me.id || (me.email && record.user?.email === me.email))) {
      return record.index;
    }
    if (record.type === "role" && record.role && me.roles?.includes(record.role)) {
      return record.index;
    }
  }
  return undefined;
}

function ReportSection({
  step,
  iconSrc,
  iconSlug,
}: {
  step: Step;
  iconSrc: string | undefined;
  iconSlug: string | undefined;
}) {
  const [hovered, setHovered] = useState(false);
  const componentStateMap = step.componentName ? getStateMap(step.componentName) : undefined;
  const borderColor = componentStateMap?.[step.status]?.badgeColor ?? getStatusBadgeProps(step.status).badgeColor;

  return (
    <div
      className="group/section relative flex"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div className={cn("w-[3px] shrink-0 rounded-full", borderColor, !hovered && "opacity-60")} />
      <div
        className={cn(
          "min-w-0 flex-1 py-1.5 pl-3 pr-10 transition-colors duration-100",
          hovered && "bg-slate-50 rounded-r-md",
        )}
      >
        <ReportMarkdown className="prose prose-sm prose-gray max-w-none text-sm text-gray-700 [&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_h1]:text-base [&_h1]:font-semibold [&_h1]:mb-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:mb-1 [&_h3]:text-xs [&_h3]:font-semibold [&_h3]:mb-1">
          {step.reportEntry!}
        </ReportMarkdown>
      </div>
      <div
        className={cn(
          "absolute right-0 top-1.5 flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2 py-1 shadow-sm transition-opacity duration-100",
          hovered ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      >
        <NodeIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={step.name} size={13} className="text-gray-400" />
        <span className="max-w-32 truncate text-[11px] font-medium text-gray-600">{step.name}</span>
      </div>
    </div>
  );
}

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
  nodes: ComponentsNode[],
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
      componentName: triggerNode?.trigger?.name,
      status: "success",
      finished: true,
      startOffsetMs: 0,
      durationMs: 0,
      elapsedMs: 0,
      reportEntry: rootEvent.reportEntry || undefined,
      isTrigger: true,
    });
  }

  const sortedExecs = [...executions].sort((a, b) => {
    const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return ta - tb;
  });

  const nowMs = Date.now();
  for (const exec of sortedExecs) {
    if (!exec.nodeId) continue;
    const node = nodeMap.get(exec.nodeId);
    const startMs = exec.createdAt ? new Date(exec.createdAt).getTime() : runStartMs;
    const dur = execDurationMs(exec);
    const elapsed = exec.createdAt ? Math.max(0, nowMs - new Date(exec.createdAt).getTime()) : 0;

    steps.push({
      nodeId: exec.nodeId,
      executionId: exec.id,
      name: node?.name || exec.nodeId.slice(0, 8),
      componentName: node?.component?.name || node?.trigger?.name,
      status: resolveExecutionDisplayStatus(exec, nodes),
      finished: exec.state === "STATE_FINISHED",
      startOffsetMs: Math.max(0, startMs - runStartMs),
      durationMs: dur,
      elapsedMs: elapsed,
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

export function RunSummary({
  runData,
  canvasId,
  workflowNodes,
  componentIconMap,
  onPushThrough,
  onCancelExecution,
  supportsPushThrough,
}: RunSummaryProps) {
  const nodeMap = useMemo(() => {
    const m = new Map<string, ComponentsNode>();
    for (const n of workflowNodes || []) {
      if (n.id) m.set(n.id, n);
    }
    return m;
  }, [workflowNodes]);

  const nodes = workflowNodes || [];
  const { steps, totalDurationMs } = useMemo(() => buildSteps(runData, nodeMap, nodes), [runData, nodeMap, nodes]);

  const { data: me } = useMe();
  const queryClient = useQueryClient();

  const executionMap = useMemo(() => {
    const m = new Map<string, CanvasesCanvasNodeExecution>();
    for (const e of runData.executions || []) {
      if (e.id) m.set(e.id, e);
    }
    return m;
  }, [runData.executions]);

  const handleApprovalAction = useCallback(
    async (executionId: string, actionName: "approve" | "reject", recordIndex: number) => {
      if (!canvasId) return;
      try {
        await canvasesInvokeNodeExecutionAction(
          withOrganizationHeader({
            path: { canvasId, executionId, actionName },
            body: { parameters: { index: recordIndex } },
          }),
        );
        queryClient.invalidateQueries({ queryKey: canvasKeys.run(canvasId, runData.run?.id || "") });
      } catch {
        // Action failed silently; the UI will reflect stale state until next poll
      }
    },
    [canvasId, queryClient, runData.run?.id],
  );

  const executions = runData.executions || [];
  const rootEvent = runData.run;
  const executionRefs: CanvasesCanvasNodeExecutionRef[] = rootEvent?.executions || [];
  const aggregateStatus =
    executionRefs.length > 0 ? getAggregateStatus(executionRefs) : executions.length > 0 ? "completed" : "queued";
  const headerStatus = (aggregateStatus as StepStatus) in STATUS_STYLES ? (aggregateStatus as StepStatus) : "queued";

  const stepCount = new Set(executions.map((e) => e.nodeId).filter(Boolean)).size;
  const successCount = useMemo(() => {
    return (runData.executions || []).filter((e) => e.state === "STATE_FINISHED" && e.result === "RESULT_PASSED")
      .length;
  }, [runData.executions]);
  const errorCount = useMemo(() => {
    return (runData.executions || []).filter((e) => e.result === "RESULT_FAILED").length;
  }, [runData.executions]);

  const activeSteps = useMemo(() => steps.filter((s) => !s.finished && !s.isTrigger), [steps]);

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

        {/* Activity */}
        {activeSteps.length > 0 && (
          <div className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
            <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">Activity</h3>
            <div className="flex flex-col">
              {activeSteps.map((step) => {
                const iconSrc = getHeaderIconSrc(step.componentName);
                const node = nodeMap.get(step.nodeId);
                const iconSlug = resolveNodeIconSlug(node, componentIconMap || {});
                const badge = getStatusBadgeProps(step.status);
                const canPushThrough = step.executionId && supportsPushThrough?.(step.nodeId);
                const canCancel = !!step.executionId;
                const exec = step.executionId ? executionMap.get(step.executionId) : undefined;
                const isApproval = step.componentName === "approval";
                const approvalIndex = isApproval ? findActionableApprovalIndex(exec, me) : undefined;
                const canApprove = approvalIndex != null && !!step.executionId && !!canvasId;
                const hasActions = canPushThrough || canCancel || canApprove;
                return (
                  <div
                    key={step.nodeId}
                    className="group/row flex items-center gap-2.5 border-t border-slate-100 py-2 first:border-t-0"
                  >
                    <div className="flex h-4 w-4 shrink-0 items-center justify-center">
                      <NodeIcon
                        iconSrc={iconSrc}
                        iconSlug={iconSlug}
                        alt={step.name}
                        size={14}
                        className="text-gray-400"
                      />
                    </div>
                    <div
                      className={cn(
                        "shrink-0 rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
                        badge.badgeColor,
                      )}
                    >
                      {badge.label}
                    </div>
                    <span className="min-w-0 truncate text-xs font-medium text-gray-700">{step.name}</span>
                    {!step.finished && step.elapsedMs > 0 && (
                      <span className="shrink-0 text-xs tabular-nums text-gray-400">{formatMs(step.elapsedMs)}</span>
                    )}
                    {hasActions && (
                      <div className="ml-auto flex shrink-0 items-center gap-1.5">
                        {canApprove && (
                          <>
                            <button
                              type="button"
                              className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600"
                              onClick={() => handleApprovalAction(step.executionId!, "reject", approvalIndex)}
                            >
                              Reject
                            </button>
                            <button
                              type="button"
                              className="rounded bg-gray-900 px-2 py-0.5 text-[11px] font-medium text-white transition-colors hover:bg-gray-700"
                              onClick={() => handleApprovalAction(step.executionId!, "approve", approvalIndex)}
                            >
                              Approve
                            </button>
                          </>
                        )}
                        <div
                          className={cn(
                            "flex items-center gap-1.5",
                            !canApprove && "opacity-0 transition-opacity group-hover/row:opacity-100",
                          )}
                        >
                          {canPushThrough && onPushThrough && (
                            <button
                              type="button"
                              className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700"
                              onClick={() => onPushThrough(step.nodeId, step.executionId!)}
                            >
                              Push Through
                            </button>
                          )}
                          {canCancel && onCancelExecution && (
                            <button
                              type="button"
                              className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                              onClick={() => onCancelExecution(step.nodeId, step.executionId!)}
                            >
                              Cancel
                            </button>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Report */}
        <div className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">Report</h3>
          {steps.some((s) => s.reportEntry) ? (
            <div className="flex flex-col gap-1">
              {steps
                .filter((s) => s.reportEntry)
                .map((step) => {
                  const sIconSrc = getHeaderIconSrc(step.componentName);
                  const sNode = nodeMap.get(step.nodeId);
                  const sIconSlug = resolveNodeIconSlug(sNode, componentIconMap || {});
                  return <ReportSection key={step.nodeId} step={step} iconSrc={sIconSrc} iconSlug={sIconSlug} />;
                })}
            </div>
          ) : null}

          {activeSteps.length > 0 && (
            <div
              className={cn(
                "flex items-center gap-2 text-xs text-gray-400",
                steps.some((s) => s.reportEntry) && "mt-4 border-t border-slate-100 pt-3",
              )}
            >
              <Loader2 className="h-3.5 w-3.5 animate-spin text-amber-500" />
              <span>
                {activeSteps.length} {activeSteps.length === 1 ? "component" : "components"} still running
              </span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
