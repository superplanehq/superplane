import React, { useMemo } from "react";
import { AlertTriangle, FileText, Sparkles } from "lucide-react";

import type {
  CanvasesCanvasNodeExecution,
  CanvasesDescribeRunResponse,
  SuperplaneComponentsNode as ComponentsNode,
  SuperplaneMeUser,
} from "@/api-client";
import type { ApprovalActionName, ApprovalActionParams } from "@/pages/workflowv2/useApprovalActionHandler";
import { TimeAgo } from "@/components/TimeAgo";
import { formatDurationSeconds } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import {
  badgeColorForEventState,
  getStatusBadgeProps,
  resolveExecutionBadgeColor,
  resolveExecutionDisplayStatus,
  resolveExecutionEventState,
  resolveNodeIconSlug,
} from "@/pages/workflowv2/lib/canvas-runs";
import { type EventState } from "@/ui/componentBase";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { ReportMarkdown } from "./ReportMarkdown";
import { StepRibbon, type RibbonStep } from "./StepRibbon";

interface RunSummaryProps {
  runData: CanvasesDescribeRunResponse;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onPushThrough?: (nodeId: string, executionId: string) => void | Promise<void>;
  onCancelExecution?: (nodeId: string, executionId: string) => void | Promise<void>;
  /**
   * Cancels a queued item for a node before it turns into an execution.
   * Queue items live on a separate backend endpoint from executions
   * (DeleteNodeQueueItem vs CancelExecution), so the two cancel actions
   * must be wired separately even though they share a UI button.
   */
  onCancelQueueItem?: (nodeId: string, queueItemId: string) => void | Promise<void>;
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

interface Step {
  key: string;
  nodeId: string;
  executionId?: string;
  /**
   * Backend ID of the queue item backing this step, when the step is a
   * synthetic "queued" entry (no execution yet). Drives the cancel action
   * in Activity for queued items, which goes through DeleteNodeQueueItem
   * rather than CancelExecution.
   */
  queueItemId?: string;
  name: string;
  componentName?: string;
  /** Raw, possibly component-specific status label (e.g. "created", "deleted"). */
  status: string;
  /**
   * Canonical event state (success/failed/running/queued/...). Drives
   * semantics (isRunning?, isQueued?) and is the color fallback.
   */
  eventState: EventState;
  /**
   * Resolved tailwind bg class for this step, matching the canvas node
   * color. For executions this runs through the component's own
   * EventStateMap so component-specific colors (e.g. wait's amber
   * "pushed through") are honored. Other steps (trigger, queued) use the
   * canonical palette.
   */
  badgeColor: string;
  finished: boolean;
  startOffsetMs: number;
  durationMs: number;
  elapsedMs: number;
  startedAt?: string;
  finishedAt?: string;
  error?: string;
  reportEntry?: string;
  isTrigger: boolean;
}

const DEFAULT_PUSH_THROUGH_COMPONENTS = new Set(["wait", "time_gate", "timegate"]);

function isRunningState(state: EventState): boolean {
  return state === "running";
}

function formatMs(ms: number): string {
  if (ms <= 0) return "0s";
  return formatDurationSeconds(ms);
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
      status: "triggered",
      eventState: "triggered",
      badgeColor: badgeColorForEventState("triggered"),
      finished: true,
      startOffsetMs: 0,
      durationMs: 0,
      elapsedMs: 0,
      startedAt: rootEvent.createdAt || undefined,
      finishedAt: rootEvent.createdAt || undefined,
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

    const finished = exec.state === "STATE_FINISHED";
    steps.push({
      key: `exec:${exec.id || `${exec.nodeId}:${startMs}`}`,
      nodeId: exec.nodeId,
      executionId: exec.id,
      name: node?.name || exec.nodeId.slice(0, 8),
      componentName: node?.component?.name || node?.trigger?.name,
      status: resolveExecutionDisplayStatus(exec, nodes),
      eventState: resolveExecutionEventState(exec),
      badgeColor: resolveExecutionBadgeColor(exec, nodes),
      finished,
      startOffsetMs: Math.max(0, startMs - runStartMs),
      durationMs: dur,
      elapsedMs: elapsed,
      startedAt: exec.createdAt || undefined,
      finishedAt: finished ? exec.updatedAt || undefined : undefined,
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
      queueItemId: item.id || undefined,
      name: node?.name || item.nodeId.slice(0, 8),
      componentName: node?.component?.name || node?.trigger?.name,
      status: "queued",
      eventState: "queued",
      badgeColor: badgeColorForEventState("queued"),
      finished: false,
      startOffsetMs: Math.max(0, startMs - runStartMs),
      durationMs: 0,
      elapsedMs: 0,
      startedAt: item.createdAt || undefined,
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
  onCancelQueueItem,
  onApprovalAction,
}: {
  step: Step;
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  canPushThrough: boolean;
  /**
   * When true, a Cancel button is shown. The row picks the right handler
   * (execution vs queue item) based on what the step carries.
   */
  canCancel: boolean;
  canApprove: boolean;
  approvalIndex?: number;
  onOpenDetail?: (nodeId: string) => void;
  onPushThrough?: (nodeId: string, executionId: string) => void | Promise<void>;
  onCancelExecution?: (nodeId: string, executionId: string) => void | Promise<void>;
  onCancelQueueItem?: (nodeId: string, queueItemId: string) => void | Promise<void>;
  onApprovalAction?: (
    nodeId: string,
    executionId: string,
    actionName: ApprovalActionName,
    parameters: ApprovalActionParams,
  ) => void | Promise<void>;
}) {
  const eventState = step.eventState;
  const badge = getStatusBadgeProps(step.status, eventState, step.badgeColor);
  const accent = step.badgeColor;
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
        {step.startedAt ? (
          <span className="shrink-0 text-[11px] tabular-nums text-gray-400">
            <TimeAgo date={step.startedAt} />
          </span>
        ) : null}
        {(canPushThrough || canCancel || canApprove) && (
          <div className="ml-auto flex shrink-0 items-center gap-1">
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
                onClick={() => {
                  if (step.executionId) {
                    onCancelExecution?.(step.nodeId, step.executionId);
                  } else if (step.queueItemId) {
                    onCancelQueueItem?.(step.nodeId, step.queueItemId);
                  }
                }}
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
// Empty state for the Report card. Shown when a run produced no
// reportTemplate output at all (no entries and no errors). Intentionally
// airy -- the default card layout compresses down to a one-liner and
// feels like a missing section rather than a feature. This gives the
// user a clear "here's what this is for + here's how to opt in".
//
function ReportEmptyState() {
  return (
    <div className="relative flex flex-col items-center gap-3 overflow-hidden rounded-md border border-dashed border-slate-200 bg-gradient-to-b from-slate-50/60 to-white px-6 py-8 text-center">
      <span
        className="pointer-events-none absolute -right-6 -top-6 h-24 w-24 rounded-full bg-indigo-100/40 blur-2xl"
        aria-hidden
      />
      <span
        className="pointer-events-none absolute -bottom-8 -left-6 h-24 w-24 rounded-full bg-emerald-100/40 blur-2xl"
        aria-hidden
      />

      <div className="relative flex h-10 w-10 items-center justify-center rounded-full bg-white text-indigo-500 shadow-sm ring-1 ring-slate-200">
        <FileText className="h-5 w-5" />
        <Sparkles className="absolute -right-1 -top-1 h-3 w-3 text-amber-400" aria-hidden />
      </div>

      <div className="relative flex max-w-md flex-col gap-1">
        <p className="text-sm font-semibold text-gray-800">No report for this run</p>
        <p className="text-xs leading-relaxed text-gray-500">
          Reports surface the most important output of a run at a glance. Add a{" "}
          <code className="rounded bg-slate-100 px-1 py-[1px] font-mono text-[11px] text-slate-700">reportTemplate</code>{" "}
          to any trigger or component to populate this section.
        </p>
      </div>

      <div className="relative mt-1 w-full max-w-sm rounded-md border border-slate-200 bg-white p-3 text-left shadow-sm">
        <div className="mb-1.5 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wide text-gray-400">
          <FileText className="h-3 w-3" />
          Example
        </div>
        <pre className="overflow-x-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-snug text-slate-700">
{`reportTemplate: |
  ### Deploy to production
  - **Version**: \`{{ outputs.version }}\`
  - **Region**: {{ inputs.region }}`}
        </pre>
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
  onCancelQueueItem,
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
      steps.map((s) => {
        const node = nodeMap.get(s.nodeId);
        const iconSrc = getHeaderIconSrc(s.componentName);
        const iconSlug = resolveNodeIconSlug(node, componentIconMap || {});
        return {
          key: s.key,
          name: s.name,
          status: s.status,
          eventState: s.eventState,
          badgeColor: s.badgeColor,
          isTrigger: s.isTrigger,
          durationMs: s.durationMs,
          finished: s.finished,
          componentName: s.componentName,
          iconSrc: iconSrc || undefined,
          iconSlug: iconSlug || undefined,
          startedAt: s.startedAt,
          finishedAt: s.finishedAt,
          elapsedMs: s.elapsedMs,
          error: s.error,
        };
      }),
    [steps, nodeMap, componentIconMap],
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
                //
                // A step is cancellable either as an in-flight execution
                // (CancelExecution endpoint) or as a pending queue item
                // before it produces an execution (DeleteNodeQueueItem
                // endpoint). The two live on different backend paths so
                // the row needs a different handler depending on which.
                //
                const canCancelExec =
                  !!step.executionId &&
                  !!onCancelExecution &&
                  isRunningState(step.eventState);
                const canCancelQueue =
                  !step.executionId &&
                  !!step.queueItemId &&
                  !!onCancelQueueItem &&
                  step.eventState === "queued";
                const canCancel = canCancelExec || canCancelQueue;

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
                    onCancelQueueItem={onCancelQueueItem}
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
            <ReportEmptyState />
          )}
        </div>
      </div>
    </div>
  );
}
