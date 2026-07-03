import { AlertTriangle, type LucideIcon } from "lucide-react";
import type { ReactNode, Ref } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { RUN_STATUS_META, type RunStatusKey } from "./runPresentation";
import { formatRunDuration, RUN_STEP_FILTERS, type RunStepFilter, type RunStepSummary } from "./runSummary";

function humanizeReason(reason?: string) {
  if (!reason) return undefined;
  const cleaned = reason
    .replace(/^RESULT_REASON_/, "")
    .replace(/_/g, " ")
    .toLowerCase();
  return cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
}

function RunStatusPill({ status }: { status: RunStatusKey }) {
  const meta = RUN_STATUS_META[status];
  const Icon: LucideIcon = meta.icon;
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset",
        meta.badgeClassName,
      )}
    >
      <Icon className="h-3.5 w-3.5" />
      {meta.label}
    </span>
  );
}

/**
 * Run-level primary action shown beside the summary meta line: "Stop" while the
 * run is still going, otherwise "Rerun" to restart the whole run.
 */
function RunLevelActionButton({ status }: { status: RunStatusKey }) {
  const isRunning = status === "running";
  const label = isRunning ? "Stop" : "Rerun";
  const tooltip = isRunning
    ? "Stop all running steps and cancel queued ones"
    : "Restart this whole run from trigger event";

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className={cn(
            "shrink-0 rounded border px-2 py-0.5 text-xs font-medium transition-colors",
            isRunning
              ? "border-red-200 bg-white text-red-600 hover:bg-red-50"
              : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50 hover:text-slate-900",
          )}
        >
          {label}
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{tooltip}</TooltipContent>
    </Tooltip>
  );
}

/**
 * The run's headline: status pill inline with the title, and a single muted meta
 * line (when, duration, step count, and an error marker when present). No trigger
 * line or version - the trigger is the first step and version is a URL-only id.
 */
export function IdentityHeader({
  run,
  title,
  status,
  stepCount,
  errorCount,
  contentClass,
  isFull,
}: {
  run: CanvasesCanvasRun;
  title: string;
  status: RunStatusKey;
  stepCount: number;
  errorCount: number;
  contentClass: string;
  isFull: boolean;
}) {
  const duration = formatRunDuration(run);

  const metaItems: { key: string; content: ReactNode }[] = [];
  if (run.createdAt) {
    metaItems.push({
      key: "when",
      content: (
        <span title={new Date(run.createdAt).toLocaleString()}>
          <TimeAgo date={run.createdAt} />
        </span>
      ),
    });
  }
  if (duration) metaItems.push({ key: "duration", content: duration });
  metaItems.push({
    key: "steps",
    content: `${stepCount} ${stepCount === 1 ? "step" : "steps"}`,
  });
  if (errorCount > 0) {
    metaItems.push({
      key: "errors",
      content: (
        <span className="font-medium text-red-600">
          {errorCount} {errorCount === 1 ? "error" : "errors"}
        </span>
      ),
    });
  }

  return (
    <div className={cn("flex flex-col gap-1.5 border-b border-slate-950/10 py-4", contentClass)}>
      <div className="flex flex-wrap items-center gap-2">
        <RunStatusPill status={status} />
        <h1 className={cn("min-w-0 font-semibold leading-tight text-gray-900", isFull ? "text-xl" : "text-base")}>
          {title}
        </h1>
      </div>
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-gray-500">
          {metaItems.map((item, index) => (
            <span key={item.key} className="inline-flex items-center gap-x-1.5">
              {index > 0 ? <span className="text-gray-300">&middot;</span> : null}
              {item.content}
            </span>
          ))}
        </div>
        <RunLevelActionButton status={status} />
      </div>
    </div>
  );
}

/** A single red callout for one errored step. */
function ErrorBannerItem({
  execution,
  workflowNodes,
  onJump,
}: {
  execution: CanvasesCanvasNodeExecution;
  workflowNodes: ComponentsNode[];
  onJump: (nodeId: string) => void;
}) {
  const node = workflowNodes.find((item) => item.id === execution.nodeId);
  const name = node?.name || execution.nodeId || "a step";
  const reason = humanizeReason(execution.resultReason);

  return (
    <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2.5">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600" />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-semibold text-red-800">Errored at &ldquo;{name}&rdquo;</p>
        {execution.resultMessage ? (
          <p className="mt-0.5 break-words text-xs text-red-700">{execution.resultMessage}</p>
        ) : reason ? (
          <p className="mt-0.5 text-xs text-red-700">{reason}</p>
        ) : null}
      </div>
      {execution.nodeId ? (
        <button
          type="button"
          onClick={() => onJump(execution.nodeId!)}
          className="shrink-0 rounded border border-red-300 bg-white px-2 py-1 text-[11px] font-medium text-red-700 transition-colors hover:bg-red-100"
        >
          Jump to error
        </button>
      ) : null}
    </div>
  );
}

/** Red callouts for the errored step(s); one banner per errored step. */
export function ErrorBanner({
  errorExecutions,
  workflowNodes,
  onJump,
  contentClass,
}: {
  errorExecutions: CanvasesCanvasNodeExecution[];
  workflowNodes: ComponentsNode[];
  onJump: (nodeId: string) => void;
  contentClass: string;
}) {
  if (errorExecutions.length === 0) return null;

  return (
    <div className={cn("flex flex-col gap-2 py-3", contentClass)}>
      {errorExecutions.map((execution, index) => (
        <ErrorBannerItem
          key={execution.id || execution.nodeId || index}
          execution={execution}
          workflowNodes={workflowNodes}
          onJump={onJump}
        />
      ))}
    </div>
  );
}

/** Sticky, state-based step filters (Errors / Running / Waiting). */
export function StepToolbar({
  summary,
  statusFilter,
  onToggleStatus,
  contentClass,
  stickyTop = 0,
  rootRef,
}: {
  summary: RunStepSummary;
  statusFilter: RunStepFilter[];
  onToggleStatus: (status: RunStepFilter) => void;
  contentClass: string;
  /** Offset from the top of the scroll container, so it stops below the sticky header. */
  stickyTop?: number;
  rootRef?: Ref<HTMLDivElement>;
}) {
  const countFor = (id: RunStepFilter) =>
    id === "error" ? summary.errors : id === "running" ? summary.running : summary.waiting;
  const available = RUN_STEP_FILTERS.filter((option) => countFor(option.id) > 0);

  return (
    <div
      ref={rootRef}
      style={{ top: stickyTop }}
      className={cn(
        "sticky z-10 flex flex-wrap items-center gap-1 border-b border-slate-950/10 bg-white/95 py-2 backdrop-blur",
        contentClass,
      )}
    >
      <span className="mr-1 text-[11px] font-semibold uppercase tracking-wide text-gray-400">Steps</span>
      {available.map((option) => {
        const active = statusFilter.includes(option.id);
        return (
          <button
            key={option.id}
            type="button"
            onClick={() => onToggleStatus(option.id)}
            className={cn(
              "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
              active
                ? "border-slate-300 bg-slate-100 text-slate-800"
                : "border-transparent text-gray-500 hover:bg-gray-100 hover:text-gray-700",
            )}
          >
            <span className={cn("h-1.5 w-1.5 rounded-full", option.dotClassName)} />
            {option.label}
            <span className="tabular-nums text-gray-400">{countFor(option.id)}</span>
          </button>
        );
      })}
    </div>
  );
}
