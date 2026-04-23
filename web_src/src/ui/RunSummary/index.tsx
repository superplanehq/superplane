import React, { useMemo } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";

import type {
  CanvasesCanvasNodeExecution,
  CanvasesDescribeRunResponse,
  SuperplaneComponentsNode as ComponentsNode,
  SuperplaneMeUser,
} from "@/api-client";
import type { ApprovalActionName, ApprovalActionParams } from "@/pages/workflowv2/useApprovalActionHandler";
import { formatDuration } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import {
  getStatusBadgeProps,
  resolveExecutionDisplayStatus,
  resolveNodeIconSlug,
} from "@/pages/workflowv2/lib/canvas-runs";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { ReportMarkdown } from "./ReportMarkdown";
import { StepRibbon, type RibbonStep, type RibbonStepStatus } from "./StepRibbon";

interface RunSummaryProps {
  runData: CanvasesDescribeRunResponse;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onPushThrough?: (nodeId: string, executionId: string) => void | Promise<void>;
  onCancelExecution?: (nodeId: string, executionId: string) => void | Promise<void>;
  /**
   * Opens the per-node detail modal (Details / Payload / Config tabs). Used
   * from Activity rows so the user can reach richer interactions (like
   * approvals) without switching to Canvas mode.
   */
  onOpenNodeDetail?: (nodeId: string) => void;
  /**
   * Invokes approve/reject on an approval execution. When supplied, any
   * waiting approval in Activity that the current user can act on shows
   * inline Approve/Reject buttons.
   */
  onApprovalAction?: (
    nodeId: string,
    executionId: string,
    actionName: ApprovalActionName,
    parameters: ApprovalActionParams,
  ) => void | Promise<void>;
  /** Current user; used to decide whether inline approve/reject is available. */
  currentUser?: SuperplaneMeUser | null;
  /** Components (by `component.name` / `trigger.name`) that support the pushThrough action. */
  pushThroughComponentNames?: Set<string>;
}

//
// Shape that matches the approval execution metadata emitted by
// pkg/components/approval. We redeclare the pieces we need here instead of
// importing from the mapper to keep RunSummary self-contained.
//
interface ApprovalRecordLike {
  index: number;
  state: string;
  type: string;
  user?: { id?: string; email?: string };
  roleRef?: { name?: string };
  groupRef?: { name?: string };
}

//
// Returns the record index that the given user can currently act on, or
// undefined if there's nothing for them to do. Mirrors the rules used by
// the full Approval UI (`canCurrentUserActOnRecord` +
// `hasUserGivenInputInAnyRecord` from the approval mapper): a user can act
// on a pending record they match (by user/role/group/"anyone") as long as
// they haven't already responded anywhere on this execution.
//
function findActionableApprovalIndex(
  execution: CanvasesCanvasNodeExecution | undefined,
  me: SuperplaneMeUser | null | undefined,
): number | undefined {
  if (!execution?.metadata || !me) return undefined;
  const metadata = execution.metadata as { records?: ApprovalRecordLike[] } | undefined;
  const records = metadata?.records;
  if (!Array.isArray(records)) return undefined;

  //
  // If the user already responded (approved or rejected) anywhere, they
  // don't get another turn. Matches hasUserGivenInputInAnyRecord.
  //
  const hasResponded = records.some(
    (r) =>
      r.state !== "pending" &&
      (r.user?.id === me.id || (!!me.email && r.user?.email === me.email)),
  );
  if (hasResponded) return undefined;

  const myRoles = me.roles || [];
  const myGroups = me.groups || [];

  for (const record of records) {
    if (record.state !== "pending") continue;
    switch (record.type) {
      case "anyone":
        return record.index;
      case "user":
        if (
          (record.user?.id && record.user.id === me.id) ||
          (record.user?.email && me.email && record.user.email === me.email)
        ) {
          return record.index;
        }
        break;
      case "role":
        if (record.roleRef?.name && myRoles.includes(record.roleRef.name)) {
          return record.index;
        }
        break;
      case "group":
        if (record.groupRef?.name && myGroups.includes(record.groupRef.name)) {
          return record.index;
        }
        break;
    }
  }
  return undefined;
}

type StepStatus = RibbonStepStatus;

interface Step {
  key: string;
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

const DEFAULT_PUSH_THROUGH_COMPONENTS = new Set(["wait", "time_gate", "timegate"]);

//
// Accent colors match DEFAULT_EVENT_STATE_MAP so Run View stays visually
// consistent with the canvas/node surfaces. "queued" covers both the raw
// queued state and "waiting" (e.g. approvals), matching STATUS_TO_EVENT_STATE.
//
const STATUS_ACCENT: Record<StepStatus, string> = {
  success: "bg-emerald-500",
  error: "bg-red-500",
  running: "bg-amber-500",
  queued: "bg-yellow-500",
  cancelled: "bg-gray-500",
};

function normalizeStatus(raw: string): StepStatus {
  if (raw === "success" || raw === "completed" || raw === "passed") return "success";
  if (raw === "error" || raw === "failed") return "error";
  if (raw === "running" || raw === "pending" || raw === "started") return "running";
  if (raw === "cancelled") return "cancelled";
  return "queued";
}

function formatMs(ms: number): string {
  if (ms <= 0) return "0s";
  return formatDuration(ms);
}

function execDurationMs(exec: { createdAt?: string; updatedAt?: string; state?: string }): number {
  if (!exec.createdAt || !exec.updatedAt) return 0;
  if (exec.state !== "STATE_FINISHED") return 0;
  return Math.max(0, new Date(exec.updatedAt).getTime() - new Date(exec.createdAt).getTime());
}

function buildSteps(
  runData: CanvasesDescribeRunResponse,
  nodeMap: Map<string, ComponentsNode>,
  nodes: ComponentsNode[],
): { steps: Step[]; totalDurationMs: number } {
  const rootEvent = runData.run;
  const executions = runData.executions || [];
  const runStartMs = rootEvent?.createdAt ? new Date(rootEvent.createdAt).getTime() : 0;

  const steps: Step[] = [];

  if (rootEvent?.nodeId) {
    const triggerNode = nodeMap.get(rootEvent.nodeId);
    steps.push({
      key: `trigger:${rootEvent.id || rootEvent.nodeId}`,
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

  const sorted = [...executions].sort((a, b) => {
    const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return ta - tb;
  });

  const nowMs = Date.now();
  for (const exec of sorted) {
    if (!exec.nodeId) continue;
    const node = nodeMap.get(exec.nodeId);
    const startMs = exec.createdAt ? new Date(exec.createdAt).getTime() : runStartMs;
    const dur = execDurationMs(exec);
    const elapsed = exec.createdAt ? Math.max(0, nowMs - new Date(exec.createdAt).getTime()) : 0;

    steps.push({
      key: `exec:${exec.id || `${exec.nodeId}:${startMs}`}`,
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

  //
  // Queue items represent components that are scheduled to run for this
  // triggering event but don't have an execution record yet (waiting for a
  // previous stage, paused nodes, approval gates, etc.). Without these the
  // Activity section would silently miss anything that hasn't produced an
  // execution yet, and the run would look "idle" when it isn't.
  //
  const queueItems = runData.queueItems || [];
  const executedNodeIds = new Set(
    sorted.map((exec) => exec.nodeId).filter((id): id is string => !!id),
  );

  for (const item of queueItems) {
    if (!item.nodeId) continue;
    //
    // A node can have a queue item AND an execution at once (e.g. batching
    // multiple inputs). If we already have an execution step for that node,
    // trust the execution -- it carries real state. Otherwise render a
    // synthetic "queued" step so the user sees there's pending work.
    //
    if (executedNodeIds.has(item.nodeId)) continue;

    const node = nodeMap.get(item.nodeId);
    const startMs = item.createdAt ? new Date(item.createdAt).getTime() : runStartMs;
    steps.push({
      key: `queue:${item.id || `${item.nodeId}:${startMs}`}`,
      nodeId: item.nodeId,
      name: node?.name || item.nodeId.slice(0, 8),
      componentName: node?.component?.name || node?.trigger?.name,
      status: "queued",
      finished: false,
      startOffsetMs: Math.max(0, startMs - runStartMs),
      durationMs: 0,
      elapsedMs: 0,
      isTrigger: false,
    });
  }

  let totalDurationMs = 0;
  for (const s of steps) {
    const end = s.isTrigger ? 0 : s.startOffsetMs + (s.durationMs || s.elapsedMs);
    if (end > totalDurationMs) totalDurationMs = end;
  }

  return { steps, totalDurationMs };
}

function StepIcon({
  iconSrc,
  iconSlug,
  alt,
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
}) {
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} className="h-4 w-4 shrink-0 object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "bolt"), {
    size: 16,
    className: "shrink-0 text-gray-500",
  });
}

//
// Activity surfaces steps that currently need attention: in-flight executions
// and waiters that the user can act on (push through, cancel, approve). The
// whole point is to save the user from hunting for these nodes on the canvas.
// Clicking the row opens the NodeDetailPanel modal where richer interactions
// (e.g. the approval UI) live.
//
function ActivityRow({
  step,
  iconSrc,
  iconSlug,
  canPushThrough,
  canCancel,
  canApprove,
  approvalIndex,
  onOpenDetail,
  onPushThrough,
  onCancelExecution,
  onApprovalAction,
}: {
  step: Step;
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  canPushThrough: boolean;
  canCancel: boolean;
  canApprove: boolean;
  approvalIndex?: number;
  onOpenDetail?: (nodeId: string) => void;
  onPushThrough?: (nodeId: string, executionId: string) => void | Promise<void>;
  onCancelExecution?: (nodeId: string, executionId: string) => void | Promise<void>;
  onApprovalAction?: (
    nodeId: string,
    executionId: string,
    actionName: ApprovalActionName,
    parameters: ApprovalActionParams,
  ) => void | Promise<void>;
}) {
  const normalized = normalizeStatus(step.status);
  const badge = getStatusBadgeProps(step.status);
  const accent = STATUS_ACCENT[normalized];
  const isRunning = normalized === "running";
  const clickable = !!onOpenDetail;

  const handleRowClick = (event: React.MouseEvent<HTMLDivElement>) => {
    if (!clickable) return;
    const target = event.target as HTMLElement;
    if (target.closest("button")) return;
    onOpenDetail?.(step.nodeId);
  };

  return (
    <div
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      onClick={handleRowClick}
      onKeyDown={
        clickable
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onOpenDetail?.(step.nodeId);
              }
            }
          : undefined
      }
      className={cn(
        "group/activity relative flex overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm transition-colors",
        clickable && "cursor-pointer hover:border-slate-300 hover:bg-slate-50",
      )}
    >
      <div className={cn("w-[3px] shrink-0", accent)} />
      <div className="flex min-w-0 flex-1 items-center gap-2 px-3 py-2 text-xs">
        <span
          className={cn(
            "shrink-0 rounded px-1.5 py-[1px] text-[10px] font-semibold uppercase tracking-wide text-white",
            badge.badgeColor,
          )}
        >
          {badge.label}
        </span>
        <StepIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={step.name} />
        <span className="min-w-0 truncate text-sm font-medium text-gray-800">{step.name}</span>
        {step.executionId ? (
          <span className="shrink-0 font-mono text-[10px] text-gray-400">#{step.executionId.slice(0, 6)}</span>
        ) : null}
        <div className="ml-auto flex shrink-0 items-center gap-1.5 text-[11px] tabular-nums">
          {isRunning ? (
            <span className="flex items-center gap-1 text-amber-600">
              <Loader2 className="h-3 w-3 animate-spin" />
              running
            </span>
          ) : normalized === "error" ? (
            <span className="text-red-600">failed</span>
          ) : normalized === "queued" ? (
            <span className="text-yellow-700">
              {step.status === "waiting" ? "waiting" : "queued"}
            </span>
          ) : null}
        </div>
        {(canPushThrough || canCancel || canApprove) && (
          <div className="ml-1 flex shrink-0 items-center gap-1">
            {canApprove && approvalIndex != null && (
              <>
                <button
                  type="button"
                  className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-600 transition-colors hover:bg-red-50 hover:text-red-600"
                  onClick={() => {
                    if (!step.executionId) return;
                    //
                    // The backend requires a non-empty reason on reject.
                    // A lightweight prompt is enough for the Activity row --
                    // users who want a richer flow can open the node detail
                    // and use the full approval UI there.
                    //
                    const reason = window.prompt("Reason for rejection?");
                    if (reason == null) return;
                    const trimmed = reason.trim();
                    if (!trimmed) return;
                    onApprovalAction?.(step.nodeId, step.executionId, "reject", {
                      index: approvalIndex,
                      reason: trimmed,
                    });
                  }}
                >
                  Reject
                </button>
                <button
                  type="button"
                  className="rounded bg-gray-900 px-2 py-0.5 text-[11px] font-medium text-white transition-colors hover:bg-gray-700"
                  onClick={() => {
                    if (!step.executionId) return;
                    onApprovalAction?.(step.nodeId, step.executionId, "approve", {
                      index: approvalIndex,
                    });
                  }}
                >
                  Approve
                </button>
              </>
            )}
            {canPushThrough && (
              <button
                type="button"
                className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-600 transition-colors hover:bg-slate-100 hover:text-gray-800"
                onClick={() => onPushThrough?.(step.nodeId, step.executionId!)}
              >
                Push Through
              </button>
            )}
            {canCancel && (
              <button
                type="button"
                className="rounded px-1.5 py-0.5 text-[11px] font-medium text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600"
                onClick={() => onCancelExecution?.(step.nodeId, step.executionId!)}
              >
                Cancel
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

//
// A single step's slice inside the consolidated Report. Shows a compact
// byline (icon, name, optional duration) above the markdown so the reader
// knows which step a given report chunk came from without needing a full
// row header.
//
function ReportEntry({
  step,
  iconSrc,
  iconSlug,
  hasError,
}: {
  step: Step;
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  hasError: boolean;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center gap-2 text-[11px] text-gray-500">
        <StepIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={step.name} />
        <span className="truncate font-medium text-gray-700">{step.name}</span>
        {step.isTrigger ? (
          <span className="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500">
            Trigger
          </span>
        ) : null}
        {!step.isTrigger && step.durationMs > 0 ? (
          <span className="shrink-0 tabular-nums text-gray-400">{formatMs(step.durationMs)}</span>
        ) : null}
      </div>
      {hasError && step.error ? (
        <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span className="min-w-0 break-words">{step.error}</span>
        </div>
      ) : null}
      {step.reportEntry ? (
        <ReportMarkdown className="prose prose-sm prose-gray max-w-none text-sm text-gray-700 [&_p]:my-1 [&_ul]:my-1 [&_ol]:my-1 [&_li]:my-0 [&_h1]:text-base [&_h1]:font-semibold [&_h1]:mb-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:mb-1 [&_h3]:text-xs [&_h3]:font-semibold [&_h3]:mb-1">
          {step.reportEntry}
        </ReportMarkdown>
      ) : null}
    </div>
  );
}

export function RunSummary({
  runData,
  workflowNodes,
  componentIconMap,
  onPushThrough,
  onCancelExecution,
  onOpenNodeDetail,
  onApprovalAction,
  currentUser,
  pushThroughComponentNames = DEFAULT_PUSH_THROUGH_COMPONENTS,
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

  const ribbonSteps: RibbonStep[] = useMemo(
    () =>
      steps.map((s) => ({
        key: s.key,
        name: s.name,
        status: normalizeStatus(s.status),
        isTrigger: s.isTrigger,
        durationMs: s.durationMs,
        finished: s.finished,
      })),
    [steps],
  );

  //
  // Activity surfaces anything still in-flight. The rule is simple: if an
  // execution has not finished, it belongs here -- running, queued, waiting
  // for approval, held at a push-through gate, whatever. The trigger step
  // is always a terminal "done" in our model, so skip it.
  //
  const activeSteps = useMemo(
    () => steps.filter((s) => !s.isTrigger && !s.finished),
    [steps],
  );

  const reportSteps = useMemo(() => steps.filter((s) => !!s.reportEntry || !!s.error), [steps]);
  const hasAnyReport = reportSteps.some((s) => !!s.reportEntry);

  //
  // Map executions by id so the Activity row can look up an execution's
  // metadata (needed for approvals) without re-scanning the whole list per
  // row.
  //
  const executionMap = useMemo(() => {
    const m = new Map<string, CanvasesCanvasNodeExecution>();
    for (const exec of runData.executions || []) {
      if (exec.id) m.set(exec.id, exec);
    }
    return m;
  }, [runData.executions]);

  return (
    <div className="pointer-events-auto flex h-full w-full flex-col overflow-y-auto bg-slate-50 px-6 py-5">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <StepRibbon steps={ribbonSteps} totalDurationMs={totalDurationMs} />
        </div>

        {activeSteps.length > 0 ? (
          <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
            <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
              <span>Activity</span>
              <span className="rounded bg-slate-100 px-1.5 py-[1px] text-[10px] font-medium tracking-normal text-gray-600">
                {activeSteps.length}
              </span>
            </div>
            <div className="flex flex-col gap-1.5">
              {activeSteps.map((step) => {
                const iconSrc = getHeaderIconSrc(step.componentName);
                const node = nodeMap.get(step.nodeId);
                const iconSlug = resolveNodeIconSlug(node, componentIconMap || {});
                const canPushThrough =
                  !!step.executionId &&
                  !!onPushThrough &&
                  !!step.componentName &&
                  pushThroughComponentNames.has(step.componentName);
                const canCancel =
                  !!step.executionId &&
                  !!onCancelExecution &&
                  normalizeStatus(step.status) === "running";

                const execution = step.executionId ? executionMap.get(step.executionId) : undefined;
                const approvalIndex =
                  step.componentName === "approval"
                    ? findActionableApprovalIndex(execution, currentUser)
                    : undefined;
                const canApprove =
                  approvalIndex != null && !!step.executionId && !!onApprovalAction;

                return (
                  <ActivityRow
                    key={step.key}
                    step={step}
                    iconSrc={iconSrc}
                    iconSlug={iconSlug || undefined}
                    canPushThrough={canPushThrough}
                    canCancel={canCancel}
                    canApprove={canApprove}
                    approvalIndex={approvalIndex}
                    onOpenDetail={onOpenNodeDetail}
                    onPushThrough={onPushThrough}
                    onCancelExecution={onCancelExecution}
                    onApprovalAction={onApprovalAction}
                  />
                );
              })}
            </div>
          </div>
        ) : null}

        <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
          <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">Report</div>
          {hasAnyReport || reportSteps.some((s) => !!s.error) ? (
            <div className="flex flex-col divide-y divide-slate-100">
              {reportSteps.map((step, idx) => {
                const iconSrc = getHeaderIconSrc(step.componentName);
                const node = nodeMap.get(step.nodeId);
                const iconSlug = resolveNodeIconSlug(node, componentIconMap || {});
                return (
                  <div key={step.key} className={cn(idx === 0 ? "pt-0 pb-3" : "py-3", "last:pb-0")}>
                    <ReportEntry
                      step={step}
                      iconSrc={iconSrc}
                      iconSlug={iconSlug || undefined}
                      hasError={!!step.error}
                    />
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-xs text-gray-500">
              No report entries yet. Add a <code className="rounded bg-gray-100 px-0.5">reportTemplate</code> to your
              triggers or components to populate the report.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
