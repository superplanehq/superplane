import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ChevronRight, ChevronsRight, Loader2, Square } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type PointerEvent } from "react";
import {
  canvasesCancelExecution,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
  type ComponentsEdge,
  type SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { formatDuration } from "@/lib/duration";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { Accordion, AccordionContent, AccordionItem } from "@/ui/accordion";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import { RunInspectorStepTimeline } from "./RunInspectorStepTimeline";
import { buildNodeMap, buildRunPresentation, getRunStatus, RUN_STATUS_META } from "./runPresentation";
import {
  buildRunInspectorNodeSections,
  calculateRunDuration,
  findRunInspectorErrorSummaries,
  type RunInspectorNodeSection,
} from "./runNodeDetailModel";

const INSPECTOR_WIDTH_STORAGE_KEY = "superplane.runInspector.width.v3";
const DEFAULT_INSPECTOR_WIDTH = 480;
const MIN_INSPECTOR_WIDTH = 360;
const MAX_INSPECTOR_WIDTH_RATIO = 0.48;
const CANVAS_MIN_WIDTH = 280;

export interface RunInspectorPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  workflowEdges?: ComponentsEdge[];
  componentIconMap?: Record<string, string>;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  onClearSelectedNode?: () => void;
  onClose: () => void;
}

export function RunInspectorPanel({
  canvasId,
  run,
  workflowNodes,
  workflowEdges,
  componentIconMap = {},
  selectedNodeId = null,
  onSelectNode,
  onClearSelectedNode,
  onClose,
}: RunInspectorPanelProps) {
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [nodeMap, run]);
  const sections = useMemo(
    () => buildRunInspectorNodeSections({ run, executions, workflowNodes, workflowEdges }),
    [executions, run, workflowEdges, workflowNodes],
  );
  const errorSummaries = useMemo(() => findRunInspectorErrorSummaries(sections), [sections]);
  const inspectorWidth = useResizableInspectorWidth();
  const queryClient = useQueryClient();
  const selectedValue = selectedNodeId ?? "";
  const [pendingErrorScrollNodeId, setPendingErrorScrollNodeId] = useState<string | null>(null);
  const runningExecutionIds = useMemo(
    () =>
      sections
        .map((section) => section.execution)
        .filter((execution) => execution?.id && execution.state === "STATE_STARTED")
        .map((execution) => execution!.id!),
    [sections],
  );

  const refreshRunQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  }, [queryClient]);

  const rerunMutation = useMutation({
    mutationFn: async () => {
      if (!run.rootEvent?.nodeId || !run.rootEvent?.id) {
        throw new Error("Run root event is missing");
      }

      await canvasesReemitTriggerEvent(
        withOrganizationHeader({
          path: {
            canvasId,
            nodeId: run.rootEvent.nodeId,
            eventId: run.rootEvent.id,
          },
        }),
      );
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run restarted");
    },
    onError: (error) => {
      console.error("Failed to restart run", error);
      showErrorToast("Failed to restart run");
    },
  });

  const stopMutation = useMutation({
    mutationFn: async () => {
      if (runningExecutionIds.length === 0) {
        throw new Error("No running steps to stop");
      }

      await Promise.all(
        runningExecutionIds.map((executionId) =>
          canvasesCancelExecution(
            withOrganizationHeader({
              path: {
                canvasId,
                executionId,
              },
            }),
          ),
        ),
      );
    },
    onSuccess: async () => {
      await refreshRunQueries();
      showSuccessToast("Run stopped");
    },
    onError: (error) => {
      console.error("Failed to stop run", error);
      showErrorToast("Failed to stop run");
    },
  });

  const handleValueChange = (value: string) => {
    if (value) {
      onSelectNode(value);
      return;
    }

    onClearSelectedNode?.();
  };

  const jumpToErrorOutput = (nodeId: string) => {
    setPendingErrorScrollNodeId(nodeId);
    onSelectNode(nodeId);
  };

  useEffect(() => {
    if (!pendingErrorScrollNodeId || selectedNodeId !== pendingErrorScrollNodeId) return;

    const frame = window.requestAnimationFrame(() => {
      const errorOutput = document.querySelector(`[data-run-error-output-node-id="${pendingErrorScrollNodeId}"]`);
      errorOutput?.scrollIntoView({ block: "center", behavior: "smooth" });
      setPendingErrorScrollNodeId(null);
    });

    return () => window.cancelAnimationFrame(frame);
  }, [pendingErrorScrollNodeId, selectedNodeId]);

  return (
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col border-l bg-white shadow-sm dark:bg-gray-950",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width: inspectorWidth.width }}
      data-testid="run-inspector-panel"
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={inspectorWidth.startResize} isResizing={inspectorWidth.isResizing} />
      <RunInspectorChrome onClose={onClose} />
      <RunInspectorHeader
        run={run}
        title={presentation.title}
        stepCount={sections.length || run.executions?.length || 0}
        onAction={() => (presentation.status === "running" ? stopMutation.mutate() : rerunMutation.mutate())}
        actionPending={presentation.status === "running" ? stopMutation.isPending : rerunMutation.isPending}
        actionDisabled={
          presentation.status === "running"
            ? runningExecutionIds.length === 0 || stopMutation.isPending
            : !run.rootEvent?.id
        }
      />

      <div className="min-h-0 flex-1 overflow-y-auto" data-testid="run-panel-step-list">
        {errorSummaries.length > 0 ? (
          <div className="space-y-2 px-4 py-3">
            {errorSummaries.map((summary) => (
              <ErrorSummaryCard
                key={summary.nodeId}
                nodeName={summary.nodeName}
                message={summary.message}
                onJump={() => jumpToErrorOutput(summary.nodeId)}
              />
            ))}
          </div>
        ) : null}

        <StepsHeader status={presentation.status} errorCount={errorSummaries.length} stepCount={sections.length} />

        {executionsQuery.isLoading ? (
          <div className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500 dark:text-gray-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading run steps...
          </div>
        ) : sections.length === 0 ? (
          <div className="px-4 py-8 text-sm text-slate-500 dark:text-gray-400">No executed nodes in this run.</div>
        ) : (
          <Accordion type="single" collapsible value={selectedValue} onValueChange={handleValueChange}>
            {sections.map((section) => (
              <RunInspectorNodeAccordion
                key={section.nodeId}
                section={section}
                componentIconMap={componentIconMap}
                isOpen={selectedValue === section.nodeId}
                onRerun={() => rerunMutation.mutate()}
                rerunPending={rerunMutation.isPending}
              />
            ))}
          </Accordion>
        )}
      </div>
    </aside>
  );
}

function ResizeHandle({
  isResizing,
  onPointerDown,
}: {
  isResizing: boolean;
  onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize run inspector"
      data-testid="run-inspector-resize-handle"
      onPointerDown={onPointerDown}
      className={cn(
        "absolute left-0 top-0 z-30 h-full w-1 -translate-x-1/2 cursor-ew-resize transition-colors hover:bg-blue-300/60",
        isResizing && "bg-blue-400/70",
      )}
    />
  );
}

function RunInspectorChrome({ onClose }: { onClose: () => void }) {
  return (
    <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-950/10 px-2 py-1.5 dark:border-gray-800">
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
            data-testid="run-panel-close"
          >
            <ChevronsRight className="h-4 w-4" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="bottom">Close</TooltipContent>
      </Tooltip>
    </div>
  );
}

function RunInspectorHeader({
  run,
  title,
  stepCount,
  actionPending,
  actionDisabled,
  onAction,
}: {
  run: CanvasesCanvasRun;
  title: string;
  stepCount: number;
  actionPending: boolean;
  actionDisabled: boolean;
  onAction: () => void;
}) {
  const status = getRunStatus(run);
  const meta = RUN_STATUS_META[status];
  const Icon = meta.icon;
  const duration = calculateRunDuration(run);
  const actionLabel = status === "running" ? "Stop" : "Rerun";
  const actionTooltip =
    status === "running"
      ? "Stop all running steps and cancel queued ones"
      : "Restart this whole run from trigger event";

  return (
    <div className="sticky top-0 z-20 border-b border-slate-950/10 bg-white px-4 py-4 dark:border-gray-800 dark:bg-gray-950">
      <div className="flex flex-col gap-1.5">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span
            className={cn(
              "inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset",
              meta.badgeClassName,
            )}
          >
            <Icon className="h-3.5 w-3.5" />
            {meta.label}
          </span>
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold leading-tight text-gray-900 dark:text-gray-100">
            {title}
          </h2>
        </div>
        <div className="flex items-center justify-between gap-2">
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-gray-600 dark:text-gray-400">
            {run.createdAt ? <Timestamp date={run.createdAt} display="relative" relativeStyle="abbreviated" /> : null}
            {duration !== null ? (
              <>
                <span className="text-gray-300" aria-hidden>
                  ·
                </span>
                <span>{formatDuration(duration)}</span>
              </>
            ) : null}
            <span className="text-gray-300" aria-hidden>
              ·
            </span>
            <span>
              {stepCount} {stepCount === 1 ? "step" : "steps"}
            </span>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="shrink-0">
                <button
                  type="button"
                  disabled={actionDisabled || actionPending}
                  onClick={onAction}
                  className={cn(
                    "inline-flex shrink-0 items-center rounded border px-2 py-0.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60",
                    status === "running"
                      ? "border-red-200 bg-white text-red-600 hover:bg-red-50 dark:border-red-900/70 dark:bg-gray-950 dark:text-red-300"
                      : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50 hover:text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200",
                  )}
                >
                  <span className="inline-flex items-center gap-2">
                    {actionPending ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : status === "running" ? (
                      <Square className="h-3.5 w-3.5" />
                    ) : null}
                    <span>{actionPending ? `${actionLabel}...` : actionLabel}</span>
                  </span>
                </button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">{actionTooltip}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}

function StepsHeader({
  status,
  errorCount,
  stepCount,
}: {
  status: keyof typeof RUN_STATUS_META;
  errorCount: number;
  stepCount: number;
}) {
  const statusMeta = RUN_STATUS_META[status];
  const label = errorCount > 0 ? `Errors ${errorCount}` : statusMeta.label;
  const dotClassName = errorCount > 0 ? "bg-red-500" : statusMeta.dotClassName;

  return (
    <div className="sticky top-0 z-10 flex items-center gap-2 border-b border-slate-950/10 bg-white/95 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500 backdrop-blur dark:border-gray-800 dark:bg-gray-950/95 dark:text-gray-400">
      <span>Steps</span>
      <span className="ml-2 inline-flex items-center gap-2 font-medium normal-case tracking-normal text-slate-500 dark:text-gray-400">
        <span className={cn("h-2 w-2 rounded-full", dotClassName)} />
        {label}
      </span>
      <span className="sr-only">{stepCount} total steps</span>
    </div>
  );
}

function ErrorSummaryCard({ nodeName, message, onJump }: { nodeName: string; message: string; onJump: () => void }) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2.5 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600 dark:text-red-300" />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-semibold text-red-800 dark:text-red-200">Errored at &quot;{nodeName}&quot;</p>
        <p className="mt-0.5 line-clamp-3 break-words text-xs text-red-700 dark:text-red-300">{message}</p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="shrink-0 rounded-sm border-red-300 bg-white text-red-700 hover:bg-red-100 dark:border-red-800 dark:bg-gray-950 dark:text-red-300 dark:hover:bg-red-950"
        onClick={onJump}
      >
        Jump to error
      </Button>
    </div>
  );
}

function RunInspectorNodeAccordion({
  section,
  componentIconMap,
  isOpen,
  onRerun,
  rerunPending,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
  isOpen: boolean;
  onRerun: () => void;
  rerunPending: boolean;
}) {
  const iconSrc = getHeaderIconSrc(section.workflowNode?.component);
  const iconSlug = section.workflowNode?.component ? componentIconMap[section.workflowNode.component] : undefined;
  const itemRef = useRef<HTMLDivElement>(null);
  const wasOpenRef = useRef(false);

  useEffect(() => {
    if (!isOpen) {
      wasOpenRef.current = false;
      return;
    }

    if (wasOpenRef.current) {
      return;
    }

    wasOpenRef.current = true;
    const frame = window.requestAnimationFrame(() => {
      itemRef.current?.scrollIntoView?.({ block: "start", behavior: "smooth" });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [isOpen]);

  return (
    <AccordionItem
      ref={itemRef}
      value={section.nodeId}
      className="scroll-mt-8 border-slate-950/10 dark:border-gray-800"
    >
      <AccordionPrimitive.Header
        className={cn(
          "flex items-center bg-white transition-colors hover:bg-slate-50 dark:bg-gray-950 dark:hover:bg-gray-900",
          isOpen &&
            "sticky top-8 z-20 bg-[#e1f5ff] text-slate-950 shadow-[0_1px_0_rgba(15,23,42,0.08)] dark:bg-sky-950 dark:text-gray-100 dark:shadow-[0_1px_0_rgba(31,41,55,0.8)]",
        )}
      >
        <AccordionPrimitive.Trigger className="flex min-w-0 flex-1 items-center gap-3 px-4 py-3 text-left hover:no-underline">
          <ChevronRight
            className={cn(
              "h-4 w-4 shrink-0 text-slate-400 transition-transform duration-200",
              isOpen && "rotate-90 text-slate-600 dark:text-gray-300",
            )}
          />
          <RunNodeIcon
            iconSrc={iconSrc}
            iconSlug={iconSlug}
            alt={section.nodeName}
            size={RUN_NODE_ICON_SIZE}
            className="text-slate-500 dark:text-gray-400"
          />
          <span className="min-w-0 flex-1 truncate text-sm font-medium text-slate-900 dark:text-gray-100">
            {section.nodeName}
          </span>
        </AccordionPrimitive.Trigger>
        <NodeMetadata section={section} onRerun={onRerun} rerunPending={rerunPending} />
      </AccordionPrimitive.Header>
      <AccordionContent className="bg-slate-50 px-3 pb-3 pt-3 dark:bg-gray-950">
        <RunInspectorStepTimeline section={section} componentIconMap={componentIconMap} />
      </AccordionContent>
    </AccordionItem>
  );
}

function NodeMetadata({
  section,
  onRerun,
  rerunPending,
}: {
  section: RunInspectorNodeSection;
  onRerun: () => void;
  rerunPending: boolean;
}) {
  return (
    <div className="ml-auto flex shrink-0 items-center gap-3 px-4 text-xs text-slate-500 dark:text-gray-400">
      {section.isTrigger ? (
        <button
          type="button"
          disabled={rerunPending}
          className="inline-flex h-6 items-center rounded-sm border border-slate-200 bg-white px-2 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-800 dark:hover:text-gray-100"
          onClick={(event) => {
            event.stopPropagation();
            onRerun();
          }}
        >
          {rerunPending ? "Rerun..." : "Rerun"}
        </button>
      ) : null}
      {section.isTrigger && section.createdAt ? (
        <span>{formatEventTimestamp(section.createdAt)}</span>
      ) : section.durationMs !== undefined ? (
        <span>{formatStepDuration(section.durationMs)}</span>
      ) : null}
      {section.badge ? <EventStatusBadge badgeColor={section.badge.badgeColor} label={section.badge.label} /> : null}
    </div>
  );
}

function formatStepDuration(durationMs: number): string {
  if (durationMs > 0 && durationMs < 1000) return "<1s";
  return formatDuration(durationMs);
}

function formatEventTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "";

  const pad = (value: number) => String(value).padStart(2, "0");
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  return `${pad(date.getHours())}:${pad(date.getMinutes())} - ${date.getDate()}.${months[date.getMonth()]}`;
}

function EventStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
        withEventStatusBadgeClasses(badgeColor),
      )}
    >
      {label}
    </span>
  );
}

function useResizableInspectorWidth() {
  const [width, setWidth] = useState(readInspectorWidth);
  const [isResizing, setIsResizing] = useState(false);
  const activePointerIdRef = useRef<number | null>(null);

  const resizeToClientX = useCallback((clientX: number) => {
    if (!Number.isFinite(clientX)) return;

    const nextWidth = clampInspectorWidth(window.innerWidth - clientX);
    setWidth(nextWidth);
    localStorage.setItem(INSPECTOR_WIDTH_STORAGE_KEY, String(nextWidth));
  }, []);

  const startResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      activePointerIdRef.current = event.pointerId;
      resizeToClientX(event.clientX);
      setIsResizing(true);
    },
    [resizeToClientX],
  );

  useEffect(() => {
    if (!isResizing) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (activePointerIdRef.current !== null && event.pointerId !== activePointerIdRef.current) return;
      resizeToClientX(event.clientX);
    };

    const finishResize = (event: globalThis.PointerEvent) => {
      if (activePointerIdRef.current !== null && event.pointerId !== activePointerIdRef.current) return;
      activePointerIdRef.current = null;
      setIsResizing(false);
    };

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, resizeToClientX]);

  return { width, isResizing, startResize } as const;
}

function readInspectorWidth(): number {
  if (typeof window === "undefined") return MIN_INSPECTOR_WIDTH;

  const storedWidth = Number.parseInt(localStorage.getItem(INSPECTOR_WIDTH_STORAGE_KEY) || "", 10);
  if (Number.isFinite(storedWidth)) return clampInspectorWidth(storedWidth);

  return clampInspectorWidth(DEFAULT_INSPECTOR_WIDTH);
}

function clampInspectorWidth(width: number): number {
  if (typeof window === "undefined") return Math.max(MIN_INSPECTOR_WIDTH, width);

  const maxByViewport = Math.max(MIN_INSPECTOR_WIDTH, window.innerWidth - CANVAS_MIN_WIDTH);
  const maxByRatio = Math.round(window.innerWidth * MAX_INSPECTOR_WIDTH_RATIO);
  const maxWidth = Math.min(maxByViewport, maxByRatio);

  return Math.max(MIN_INSPECTOR_WIDTH, Math.min(maxWidth, Math.round(width)));
}
