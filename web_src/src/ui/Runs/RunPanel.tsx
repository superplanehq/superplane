import {
  ChevronDown,
  ChevronsRight,
  ChevronUp,
  Columns2,
  InspectionPanel,
  Link as LinkIcon,
  type LucideIcon,
  PanelRight,
  Rows3,
  Sparkles,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/ui/hoverCard";
import { AccordionNodeDetail, AccordionRow, type StepDetailMode } from "./RunStepAccordion";
import { buildExecutionChain } from "./runNodeDetailModel";
import { ErrorBanner, IdentityHeader, StepToolbar } from "./runDetailParts";
import { buildNodeMap, buildRunPresentation } from "./runPresentation";
import { findErrorExecutions, getRunStepSummary, getStepActivity, type RunStepFilter } from "./runSummary";

export type RunDetailContext = "live" | "inspection";
export type RunDisplayMode = "full" | "split" | "min";
export type { StepDetailMode };

const STEP_DETAIL_MODE_STORAGE_KEY = "superplane.run-panel.step-detail-mode";

/** Persisted step-content mode (run details vs read-only step configuration), shared across runs. */
function useStepDetailMode(): [StepDetailMode, (mode: StepDetailMode) => void] {
  const [mode, setMode] = useState<StepDetailMode>(() => {
    try {
      return window.localStorage.getItem(STEP_DETAIL_MODE_STORAGE_KEY) === "step-config"
        ? "step-config"
        : "run-details";
    } catch {
      return "run-details";
    }
  });
  const update = useCallback((next: StepDetailMode) => {
    setMode(next);
    try {
      window.localStorage.setItem(STEP_DETAIL_MODE_STORAGE_KEY, next);
    } catch {
      // Persisting is best-effort; ignore quota/availability errors.
    }
  }, []);
  return [mode, update];
}

/**
 * Tracks an element's rendered height via a callback ref + ResizeObserver, so
 * sticky offsets and scroll margins stay correct as content (and mounting) change.
 */
function useElementHeight<T extends HTMLElement>(): [(node: T | null) => void, number] {
  const [height, setHeight] = useState(0);
  const observerRef = useRef<ResizeObserver | null>(null);
  const setRef = useCallback((node: T | null) => {
    observerRef.current?.disconnect();
    if (!node) {
      setHeight(0);
      return;
    }
    setHeight(node.offsetHeight);
    const observer = new ResizeObserver(() => setHeight(node.offsetHeight));
    observer.observe(node);
    observerRef.current = observer;
  }, []);
  return [setRef, height];
}

const SIZE_OPTIONS: { mode: RunDisplayMode; label: string; icon: LucideIcon }[] = [
  { mode: "full", label: "Full page", icon: InspectionPanel },
  { mode: "split", label: "Split screen", icon: Columns2 },
  { mode: "min", label: "Minimized", icon: PanelRight },
];

export interface RunPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun | null;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  context?: RunDetailContext;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  /** Expand a specific step (used by "jump to failed"); falls back to onToggleNode. */
  onExpandNode?: (nodeId: string) => void;
  onClose: () => void;
  /** Tooltip for the close button; defaults from context. */
  closeLabel?: string;
  displayMode?: RunDisplayMode;
  onSetDisplayMode?: (mode: RunDisplayMode) => void;
  onPrevRun?: () => void;
  onNextRun?: () => void;
  hasPrevRun?: boolean;
  hasNextRun?: boolean;
  /** Opens this run in the dedicated run inspection view (as if picked from the runs sidebar). */
  onOpenInRunView?: () => void;
  onAskAgent?: () => void;
}

function ChromeIconButton({
  label,
  icon,
  onClick,
  disabled,
  testId,
}: {
  label: string;
  icon: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  testId?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          data-testid={testId}
          disabled={disabled}
          onClick={onClick}
          className={cn(
            "flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors",
            "hover:bg-gray-100 hover:text-gray-800",
            "disabled:pointer-events-none disabled:opacity-40",
          )}
        >
          {icon}
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}

/** Hover menu for switching the run panel between full / split / minimized widths. */
function DisplaySizeMenu({
  displayMode,
  onSetDisplayMode,
}: {
  displayMode: RunDisplayMode;
  onSetDisplayMode: (mode: RunDisplayMode) => void;
}) {
  const active = SIZE_OPTIONS.find((option) => option.mode === displayMode) ?? SIZE_OPTIONS[1];
  const ActiveIcon = active.icon;

  return (
    <HoverCard openDelay={80} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          aria-label="Change panel size"
          className={cn(
            "flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors",
            "hover:bg-gray-100 hover:text-gray-800 data-[state=open]:bg-gray-100 data-[state=open]:text-gray-800",
          )}
        >
          <ActiveIcon className="h-4 w-4" />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" side="bottom" sideOffset={4} className="w-44 p-1">
        <div className="flex flex-col">
          {SIZE_OPTIONS.map((option) => {
            const Icon = option.icon;
            const isActive = option.mode === displayMode;
            return (
              <button
                key={option.mode}
                type="button"
                onClick={() => onSetDisplayMode(option.mode)}
                className={cn(
                  "flex items-center gap-2 rounded px-2 py-1.5 text-left text-[13px] transition-colors",
                  isActive
                    ? "bg-gray-100 font-medium text-gray-900"
                    : "text-gray-600 hover:bg-gray-50 hover:text-gray-900",
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                {option.label}
              </button>
            );
          })}
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function RunActionsMenu({
  run,
  onOpenInRunView,
  onAskAgent,
}: {
  run: CanvasesCanvasRun;
  onOpenInRunView?: () => void;
  onAskAgent?: () => void;
}) {
  const copyRunLink = useCallback(async () => {
    const url = new URL(window.location.href);
    url.searchParams.delete("view");
    url.searchParams.set("run", run.id || "");
    try {
      await navigator.clipboard.writeText(url.toString());
      toast.success("Run link copied");
    } catch {
      toast.error("Failed to copy run link");
    }
  }, [run.id]);

  const sendToAgent = useCallback(() => {
    if (onAskAgent) {
      onAskAgent();
      return;
    }
    toast.info("Send to agent", { description: "Ask the agent about this run" });
  }, [onAskAgent]);

  return (
    <>
      {onOpenInRunView ? (
        <ChromeIconButton label="Open in run view" icon={<Rows3 className="h-4 w-4" />} onClick={onOpenInRunView} />
      ) : null}
      <ChromeIconButton label="Copy link" icon={<LinkIcon className="h-4 w-4" />} onClick={copyRunLink} />
      <ChromeIconButton label="Send to agent" icon={<Sparkles className="h-4 w-4" />} onClick={sendToAgent} />
    </>
  );
}

function ChromeRow({
  run,
  context,
  displayMode,
  onSetDisplayMode,
  onPrevRun,
  onNextRun,
  hasPrevRun,
  hasNextRun,
  onOpenInRunView,
  onAskAgent,
  onClose,
  closeLabel,
}: {
  run: CanvasesCanvasRun | null;
  context: RunDetailContext;
  displayMode: RunDisplayMode;
  onSetDisplayMode?: (mode: RunDisplayMode) => void;
  onPrevRun?: () => void;
  onNextRun?: () => void;
  hasPrevRun?: boolean;
  hasNextRun?: boolean;
  onOpenInRunView?: () => void;
  onAskAgent?: () => void;
  onClose: () => void;
  closeLabel: string;
}) {
  const showNav = context === "inspection" && (Boolean(onPrevRun) || Boolean(onNextRun));

  return (
    <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-950/10 px-2 py-1.5">
      <div className="flex items-center gap-0.5">
        <ChromeIconButton
          label={closeLabel}
          icon={<ChevronsRight className="h-4 w-4" />}
          onClick={onClose}
          testId="run-panel-close"
        />
        {onSetDisplayMode ? <DisplaySizeMenu displayMode={displayMode} onSetDisplayMode={onSetDisplayMode} /> : null}
        {showNav ? (
          <>
            <ChromeIconButton
              label="Previous run"
              icon={<ChevronUp className="h-4 w-4" />}
              onClick={onPrevRun}
              disabled={!hasPrevRun}
            />
            <ChromeIconButton
              label="Next run"
              icon={<ChevronDown className="h-4 w-4" />}
              onClick={onNextRun}
              disabled={!hasNextRun}
            />
          </>
        ) : null}
      </div>

      <div className="flex items-center gap-0.5">
        {run ? <RunActionsMenu run={run} onOpenInRunView={onOpenInRunView} onAskAgent={onAskAgent} /> : null}
      </div>
    </div>
  );
}

function StepList({
  run,
  executions,
  workflowNodes,
  componentIconMap,
  statusFilter,
  expandedNodeId,
  onToggleNode,
  stickyOffset,
  isLoading,
  stepDetailMode,
}: {
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  statusFilter: RunStepFilter[];
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  stickyOffset: number;
  isLoading: boolean;
  stepDetailMode: StepDetailMode;
}) {
  const triggerNodeId = run.rootEvent?.nodeId;
  const chain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);

  const visibleNodeIds = useMemo(() => {
    if (statusFilter.length === 0) return chain;
    return chain.filter((nodeId) => {
      if (nodeId === triggerNodeId) return false;
      const execution = executions.find((item) => item.nodeId === nodeId);
      if (!execution) return false;
      const activity = getStepActivity(
        workflowNodes.find((node) => node.id === nodeId),
        execution,
      );
      return activity !== "done" && statusFilter.includes(activity);
    });
  }, [chain, statusFilter, executions, triggerNodeId, workflowNodes]);

  if (isLoading && chain.length === 0) {
    return <p className="px-4 py-4 text-xs text-gray-400">Loading steps...</p>;
  }

  if (chain.length === 0) {
    return <p className="px-4 py-4 text-xs text-gray-400">No executed nodes in this run.</p>;
  }

  if (visibleNodeIds.length === 0) {
    return <p className="px-4 py-6 text-center text-xs text-gray-400">No steps match the selected filters.</p>;
  }

  return (
    <div className="divide-y divide-slate-950/10">
      {visibleNodeIds.map((nodeId) => {
        const isTrigger = nodeId === triggerNodeId;
        const workflowNode = workflowNodes.find((node) => node.id === nodeId);
        const execution = executions.find((item) => item.nodeId === nodeId);

        return (
          <div key={nodeId} data-step-id={nodeId} style={{ scrollMarginTop: stickyOffset }}>
            <AccordionRow
              nodeId={nodeId}
              workflowNode={workflowNode}
              componentIconMap={componentIconMap}
              execution={execution}
              isTrigger={isTrigger}
              triggerTimestamp={run.rootEvent?.createdAt ?? run.createdAt}
              isExpanded={expandedNodeId === nodeId}
              onToggle={onToggleNode}
            />
            {expandedNodeId === nodeId ? (
              <AccordionNodeDetail
                run={run}
                nodeId={nodeId}
                workflowNodes={workflowNodes}
                componentIconMap={componentIconMap}
                executions={executions}
                stepDetailMode={stepDetailMode}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

/** The scrollable run-detail body below the chrome row: header, summary, banner, steps. */
function RunPanelBody({
  run,
  presentation,
  executions,
  workflowNodes,
  componentIconMap,
  expandedNodeId,
  onToggleNode,
  onExpandNode,
  isFull,
  isLoading,
}: {
  run: CanvasesCanvasRun;
  presentation: ReturnType<typeof buildRunPresentation>;
  executions: CanvasesCanvasNodeExecution[];
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  onExpandNode?: (nodeId: string) => void;
  isFull: boolean;
  isLoading: boolean;
}) {
  const [statusFilter, setStatusFilter] = useState<RunStepFilter[]>([]);
  const [stepDetailMode, setStepDetailMode] = useStepDetailMode();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [headerRef, headerHeight] = useElementHeight<HTMLDivElement>();
  const [toolbarRef, toolbarHeight] = useElementHeight<HTMLDivElement>();

  const summary = useMemo(() => getRunStepSummary(executions, workflowNodes), [executions, workflowNodes]);
  const errorExecutions = useMemo(() => findErrorExecutions(executions), [executions]);
  const stepCount = summary.total;
  const contentClass = isFull ? "px-8" : "px-4";
  // Steps must clear the sticky header and the always-present filter/mode toolbar
  // when scrolled to the top.
  const stickyOffset = headerHeight + toolbarHeight;

  const lastScrolledNodeId = useRef<string | null>(null);

  const scrollStepToTop = useCallback((nodeId: string) => {
    requestAnimationFrame(() => {
      scrollRef.current
        ?.querySelector<HTMLElement>(`[data-step-id="${nodeId}"]`)
        ?.scrollIntoView({ block: "start", behavior: "smooth" });
    });
  }, []);

  // Scroll the selected step to the top whenever the selection changes — whether
  // it was selected from the step list, a canvas node click (which drives
  // `expandedNodeId` from outside), or a jump. Guarded by the last-scrolled id so
  // that background updates (e.g. a running run polling its executions) don't
  // re-yank the view; the `executions` dependency lets it retry once the step
  // renders if the run's steps weren't loaded yet when it was selected.
  useEffect(() => {
    if (!expandedNodeId) {
      lastScrolledNodeId.current = null;
      return;
    }
    if (lastScrolledNodeId.current === expandedNodeId) return;
    if (!scrollRef.current?.querySelector(`[data-step-id="${expandedNodeId}"]`)) return;
    lastScrolledNodeId.current = expandedNodeId;
    scrollStepToTop(expandedNodeId);
  }, [expandedNodeId, executions, scrollStepToTop]);

  const jumpToStep = useCallback(
    (nodeId: string) => {
      setStatusFilter([]);
      // Re-arm so a jump re-scrolls even to the already-selected step.
      lastScrolledNodeId.current = null;
      if (onExpandNode) {
        onExpandNode(nodeId);
      } else if (expandedNodeId !== nodeId) {
        onToggleNode(nodeId);
      }
      scrollStepToTop(nodeId);
    },
    [onExpandNode, onToggleNode, expandedNodeId, scrollStepToTop],
  );

  const toggleStatus = useCallback((value: RunStepFilter) => {
    setStatusFilter((current) =>
      current.includes(value) ? current.filter((item) => item !== value) : [...current, value],
    );
  }, []);

  return (
    <div ref={scrollRef} className="min-h-0 flex-1 overflow-y-auto" data-testid="run-panel-step-list">
      <div className={cn("mx-auto flex w-full flex-col", isFull ? "max-w-3xl" : "")}>
        <div ref={headerRef} className="sticky top-0 z-20 bg-white">
          <IdentityHeader
            run={run}
            title={presentation.title}
            status={presentation.status}
            stepCount={stepCount}
            errorCount={summary.errors}
            contentClass={contentClass}
            isFull={isFull}
          />
        </div>
        {errorExecutions.length > 0 ? (
          <ErrorBanner
            errorExecutions={errorExecutions}
            workflowNodes={workflowNodes}
            onJump={jumpToStep}
            contentClass={contentClass}
          />
        ) : null}
        <StepToolbar
          summary={summary}
          statusFilter={statusFilter}
          onToggleStatus={toggleStatus}
          contentClass={contentClass}
          stickyTop={headerHeight}
          rootRef={toolbarRef}
          stepDetailMode={stepDetailMode}
          onSetStepDetailMode={setStepDetailMode}
        />
        <StepList
          run={run}
          executions={executions}
          workflowNodes={workflowNodes}
          componentIconMap={componentIconMap}
          statusFilter={statusFilter}
          expandedNodeId={expandedNodeId}
          onToggleNode={onToggleNode}
          stickyOffset={stickyOffset}
          isLoading={isLoading}
          stepDetailMode={stepDetailMode}
        />
      </div>
    </div>
  );
}

/**
 * Dedicated run panel that reads like a run-detail page: a peek-style chrome row
 * (display-mode toggle, run navigation, overflow actions, close), a run-focused
 * identity header, an at-a-glance summary strip, a conditional failure banner,
 * and a filterable run-step accordion. Used for both run inspection and the live
 * node inspector; the outer width / border wrapper is provided by the caller.
 */
export function RunPanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap = {},
  context = "inspection",
  expandedNodeId,
  onToggleNode,
  onExpandNode,
  onClose,
  closeLabel,
  displayMode = "split",
  onSetDisplayMode,
  onPrevRun,
  onNextRun,
  hasPrevRun,
  hasNextRun,
  onOpenInRunView,
  onAskAgent,
}: RunPanelProps) {
  const executionsQuery = useEventExecutions(canvasId, run?.rootEvent?.id || null);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);

  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => (run ? buildRunPresentation(run, nodeMap) : null), [run, nodeMap]);

  const resolvedCloseLabel = closeLabel ?? (context === "inspection" ? "Back to live canvas" : "Close");

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
      <ChromeRow
        run={run}
        context={context}
        displayMode={displayMode}
        onSetDisplayMode={onSetDisplayMode}
        onPrevRun={onPrevRun}
        onNextRun={onNextRun}
        hasPrevRun={hasPrevRun}
        hasNextRun={hasNextRun}
        onOpenInRunView={onOpenInRunView}
        onAskAgent={onAskAgent}
        onClose={onClose}
        closeLabel={resolvedCloseLabel}
      />
      {run && presentation ? (
        <RunPanelBody
          run={run}
          presentation={presentation}
          executions={executions}
          workflowNodes={workflowNodes}
          componentIconMap={componentIconMap}
          expandedNodeId={expandedNodeId}
          onToggleNode={onToggleNode}
          onExpandNode={onExpandNode}
          isFull={displayMode === "full"}
          isLoading={executionsQuery.isLoading}
        />
      ) : (
        <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-16 text-center">
          <p className="text-xs text-gray-400">This node has not run yet.</p>
        </div>
      )}
    </div>
  );
}
