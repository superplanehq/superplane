import {
  AlignLeft,
  Braces,
  Check,
  CircleHelp,
  Copy,
  GitCommitVertical,
  SlidersHorizontal,
  Sparkles,
  SquareArrowOutUpRight,
  SquareArrowRight,
  TriangleAlert,
} from "lucide-react";
import { useCallback, useMemo, useState, useSyncExternalStore, type ReactNode } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import { DetailBox, ErrorDetailBox, HeaderIconButton, JsonDetailBox, PayloadMonaco } from "./RunStepAccordion";
import {
  buildExecutionChain,
  buildStepStatusTimeline,
  extractExecutionPayload,
  isErrorValue,
  type StepStatusEntry,
} from "./runNodeDetailModel";
import { formatEventTimestamp, formatRelativeOffset, formatStepElapsed } from "./runSummary";
import { useRunNodeDetailPresentation } from "./useRunNodeDetailPresentation";

function hasData(value: unknown): boolean {
  return !!value && typeof value === "object" && Object.keys(value as object).length > 0;
}

interface InputChainStep {
  nodeId: string;
  name: string;
  icon: ReactNode;
  payload: unknown;
}

/**
 * Payload inspector for a step's input chain: left rail of vertical tabs listing
 * the preceding steps (most recent on top), with the selected step's payload shown
 * in a read-only Monaco editor.
 */
function InputChainModal({
  open,
  onOpenChange,
  steps,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  steps: InputChainStep[];
}) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const selected = steps.find((step) => step.nodeId === selectedNodeId) ?? steps[0];
  const payloadString = useMemo(() => JSON.stringify(selected?.payload ?? {}, null, 2), [selected?.payload]);

  const copyPayload = () => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-[80vh] w-[70vw] max-w-[70vw] flex-col gap-0 overflow-hidden p-0"
        onClick={(event) => event.stopPropagation()}
      >
        <DialogTitle className="sr-only">Input chain</DialogTitle>
        <div className="flex min-h-0 flex-1">
          <div className="flex w-56 shrink-0 flex-col gap-0.5 overflow-y-auto border-r border-slate-200 bg-slate-50 p-2">
            <div className="flex items-center gap-1 px-2 py-1">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Input chain</span>
              <HoverCard openDelay={100}>
                <HoverCardTrigger asChild>
                  <button
                    type="button"
                    aria-label="What is the input chain?"
                    className="text-slate-400 transition-colors hover:text-slate-600"
                  >
                    <CircleHelp className="h-3.5 w-3.5" />
                  </button>
                </HoverCardTrigger>
                <HoverCardContent side="right" className="w-64 text-[12px] leading-snug text-slate-600">
                  The list of outputs this component has access to from upstream nodes.
                </HoverCardContent>
              </HoverCard>
            </div>
            {steps.map((step) => (
              <button
                key={step.nodeId}
                type="button"
                onClick={() => setSelectedNodeId(step.nodeId)}
                className={cn(
                  "flex items-center gap-2 rounded px-2 py-1.5 text-left text-[12px] transition-colors",
                  selected?.nodeId === step.nodeId
                    ? "bg-white font-medium text-slate-900 shadow-sm ring-1 ring-slate-200"
                    : "text-slate-600 hover:bg-slate-100",
                )}
              >
                {step.icon}
                <span className="min-w-0 truncate">{step.name}</span>
              </button>
            ))}
          </div>
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10">
              <div className="flex min-w-0 items-center gap-1.5">
                {selected?.icon}
                <span className="truncate text-[12px] font-medium text-slate-700">{selected?.name}</span>
                <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                  Output
                </span>
              </div>
              <div className="flex items-center gap-0.5">
                <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
                <HeaderIconButton
                  label={copied ? "Copied" : "Copy"}
                  icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={copyPayload}
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-hidden">
              <PayloadMonaco value={payloadString} />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

/**
 * The expand/collapse preference for each timeline panel is a single global
 * setting persisted to local storage, so toggling e.g. "summary" open stays
 * open across every step the user inspects. Backed by a tiny module-level store
 * so all mounted timeline items stay in sync.
 */
type TimelineToggleKey = "input" | "summary" | "config" | "output" | "statusTimeline";
type ToggleState = Partial<Record<TimelineToggleKey, boolean>>;
const TOGGLE_STORAGE_KEY = "superplane.runStepTimeline.panelToggles";

function readToggleState(): ToggleState {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.localStorage.getItem(TOGGLE_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as ToggleState) : {};
  } catch {
    return {};
  }
}

let toggleState: ToggleState = readToggleState();
const toggleListeners = new Set<() => void>();

function writeToggleState(next: ToggleState) {
  toggleState = next;
  try {
    window.localStorage.setItem(TOGGLE_STORAGE_KEY, JSON.stringify(next));
  } catch {
    // Persisting is best-effort; ignore quota/availability errors.
  }
  toggleListeners.forEach((listener) => listener());
}

function subscribeToggles(listener: () => void) {
  toggleListeners.add(listener);
  return () => {
    toggleListeners.delete(listener);
  };
}

/**
 * Reads a panel's open state (falling back to `defaultOpen` until the user has
 * an explicit stored preference) and returns a toggler that persists globally.
 */
function useTimelineToggle(key: TimelineToggleKey, defaultOpen: boolean): [boolean, () => void] {
  const state = useSyncExternalStore(
    subscribeToggles,
    () => toggleState,
    () => toggleState,
  );
  const open = state[key] ?? defaultOpen;
  const toggle = useCallback(() => {
    writeToggleState({ ...toggleState, [key]: !(toggleState[key] ?? defaultOpen) });
  }, [key, defaultOpen]);
  return [open, toggle];
}

/** A single node on the step timeline: circular marker, connector line, a header row with a right-aligned timestamp, and optional expanded content. */
function TimelineItem({
  marker,
  timestamp,
  header,
  isLast,
  children,
}: {
  marker: ReactNode;
  timestamp?: string | null;
  header: ReactNode;
  isLast?: boolean;
  children?: ReactNode;
}) {
  return (
    <div className="flex gap-3">
      <div className="flex flex-col items-center">
        <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-slate-500 ring-1 ring-slate-200">
          {marker}
        </div>
        {!isLast ? <div className="min-h-4 w-px flex-1 bg-slate-200" /> : null}
      </div>
      <div className="min-w-0 flex-1 pb-4">
        <div className="flex min-h-6 items-center gap-1.5">
          {header}
          {timestamp ? (
            <span className="ml-auto shrink-0 pl-2 text-[11px] tabular-nums text-slate-400">{timestamp}</span>
          ) : null}
        </div>
        {children ? <div className="mt-2">{children}</div> : null}
      </div>
    </div>
  );
}

/** First timeline item: where the step's input came from (previous step chain / trigger event). */
function InputItem({
  moreCount,
  payload,
  timestamp,
  nodeName,
  nodeIcon,
  onOpenChain,
}: {
  moreCount: number;
  payload: unknown;
  timestamp: string | null;
  nodeName: string;
  nodeIcon: ReactNode;
  onOpenChain?: () => void;
}) {
  const [open, toggleOpen] = useTimelineToggle("input", false);
  const canShowPayload = hasData(payload);

  return (
    <TimelineItem
      marker={<SquareArrowRight className="h-3.5 w-3.5" />}
      timestamp={timestamp}
      header={
        <>
          <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Input</span>
          {canShowPayload ? (
            <HeaderIconButton
              label="Show input payload"
              icon={<Braces className="h-3.5 w-3.5" />}
              active={open}
              onClick={toggleOpen}
            />
          ) : null}
          {moreCount > 0 && onOpenChain ? (
            <button
              type="button"
              onClick={onOpenChain}
              title="Open input chain"
              className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700"
            >
              +{moreCount} more
            </button>
          ) : null}
        </>
      }
    >
      {open && canShowPayload ? (
        <JsonDetailBox title="Input" value={payload} nodeName={nodeName} nodeIcon={nodeIcon} />
      ) : null}
    </TimelineItem>
  );
}

/** Ordered status changes the step went through while executing, timestamped as offsets from the first (Triggered) entry. */
function StatusChangeSubTimeline({ entries }: { entries: StepStatusEntry[] }) {
  if (entries.length === 0) return null;
  const startMs = new Date(entries[0].timestamp).getTime();
  return (
    <ol className="flex flex-col gap-1.5 rounded border border-slate-200 bg-white px-3 py-2">
      {entries.map((entry) => (
        <li key={entry.key} className="flex items-center gap-2 text-[12px]">
          <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", entry.dotClassName)} />
          <span className="capitalize text-slate-700">{entry.label}</span>
          <span className="ml-auto shrink-0 text-[11px] tabular-nums text-slate-400">
            {formatRelativeOffset(startMs, new Date(entry.timestamp).getTime())}
          </span>
        </li>
      ))}
    </ol>
  );
}

/** Middle timeline item (omitted for triggers): the component action, its live status/elapsed, and the status sub-timeline. */
function ActionItem({
  marker,
  badge,
  elapsed,
  summaryDetails,
  statusBadge,
  relativeTime,
  config,
  statusTimeline,
  timestamp,
  nodeName,
}: {
  marker: ReactNode;
  badge: { badgeColor: string; label: string };
  elapsed: string | null;
  summaryDetails: Record<string, unknown>;
  statusBadge: { badgeColor: string; label: string } | null;
  relativeTime?: string;
  config: unknown;
  statusTimeline: StepStatusEntry[];
  timestamp: string | null;
  nodeName: string;
}) {
  const [showSummary, toggleSummary] = useTimelineToggle("summary", true);
  const [showConfig, toggleConfig] = useTimelineToggle("config", false);
  const [showStatusTimeline, toggleStatusTimeline] = useTimelineToggle("statusTimeline", true);
  const hasSummary = Object.keys(summaryDetails).length > 0;
  const canShowConfig = hasData(config);
  const hasStatusTimeline = statusTimeline.length > 0;

  return (
    <TimelineItem
      marker={marker}
      timestamp={timestamp}
      header={
        <>
          <span className="flex items-center gap-1.5 rounded-full bg-slate-100 px-2 py-0.5">
            <span className={cn("h-2 w-2 shrink-0 rounded-full", badge.badgeColor)} />
            <span className="text-[11px] font-medium capitalize text-slate-700">{badge.label || "Action"}</span>
            {elapsed ? <span className="text-[11px] tabular-nums text-slate-400">{elapsed}</span> : null}
          </span>
          {hasStatusTimeline ? (
            <HeaderIconButton
              label="Show status timeline"
              icon={<GitCommitVertical className="h-3.5 w-3.5" />}
              active={showStatusTimeline}
              onClick={toggleStatusTimeline}
            />
          ) : null}
          {hasSummary ? (
            <HeaderIconButton
              label="Show summary"
              icon={<AlignLeft className="h-3.5 w-3.5" />}
              active={showSummary}
              onClick={toggleSummary}
            />
          ) : null}
          {canShowConfig ? (
            <HeaderIconButton
              label="Show runtime config"
              icon={<SlidersHorizontal className="h-3.5 w-3.5" />}
              active={showConfig}
              onClick={toggleConfig}
            />
          ) : null}
        </>
      }
    >
      <div className="flex flex-col gap-2">
        {showStatusTimeline && hasStatusTimeline ? <StatusChangeSubTimeline entries={statusTimeline} /> : null}
        {showSummary && hasSummary ? (
          <DetailBox title="Summary">
            <RunNodeDetailDetailsView details={summaryDetails} statusBadge={statusBadge} relativeTime={relativeTime} />
          </DetailBox>
        ) : null}
        {showConfig && canShowConfig ? (
          <JsonDetailBox title="Runtime Config" value={config} nodeName={nodeName} nodeIcon={marker} />
        ) : null}
      </div>
    </TimelineItem>
  );
}

/** Final timeline item: the step's output, or its error (expanded by default when errored). */
function OutputItem({
  isError,
  errorMessage,
  errorReason,
  errorMetadata,
  payload,
  timestamp,
  nodeName,
  nodeIcon,
}: {
  isError: boolean;
  errorMessage?: string;
  errorReason?: string;
  errorMetadata?: Record<string, unknown>;
  payload: unknown;
  timestamp: string | null;
  nodeName: string;
  nodeIcon: ReactNode;
}) {
  // Errored output is always expanded by default and can be dismissed per-step, but that
  // dismissal is intentionally *not* persisted (unlike the payload toggle) so a fresh error
  // always surfaces. Non-error payloads use the shared, persisted preference.
  const [payloadOpen, togglePayload] = useTimelineToggle("output", false);
  const [errorOpen, setErrorOpen] = useState(true);
  const open = isError ? errorOpen : payloadOpen;
  const toggleOpen = isError ? () => setErrorOpen((value) => !value) : togglePayload;
  const canShowPayload = hasData(payload);
  const canToggle = isError || canShowPayload;

  return (
    <TimelineItem
      marker={<SquareArrowOutUpRight className="h-3.5 w-3.5" />}
      timestamp={timestamp}
      isLast
      header={
        <>
          <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Output</span>
          {canToggle ? (
            <HeaderIconButton
              label={isError ? "Show error" : "Show output payload"}
              icon={
                isError ? <TriangleAlert className="h-3.5 w-3.5 text-red-500" /> : <Braces className="h-3.5 w-3.5" />
              }
              active={open}
              onClick={toggleOpen}
            />
          ) : null}
        </>
      }
    >
      {open ? (
        isError ? (
          <ErrorDetailBox message={errorMessage} reason={errorReason} metadata={errorMetadata} />
        ) : canShowPayload ? (
          <JsonDetailBox title="Output" value={payload} nodeName={nodeName} nodeIcon={nodeIcon} />
        ) : null
      ) : null}
    </TimelineItem>
  );
}

/**
 * The expanded content of a run step, rendered as a top-to-bottom timeline:
 * Input (where the step's data came from), Action (the component run + its
 * status sub-timeline; omitted for triggers), and Output (payload or error).
 */
export function RunStepTimeline({
  run,
  nodeId,
  workflowNodes,
  componentIconMap = {},
  executions,
}: {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
}) {
  const presentation = useRunNodeDetailPresentation({ run, nodeId, workflowNodes, executions });
  const execution = useMemo(() => executions.find((item) => item.nodeId === nodeId), [executions, nodeId]);
  const isTrigger = presentation.isTriggerNode;
  const triggerNodeId = run.rootEvent?.nodeId;

  const chain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);
  const currentIndex = chain.indexOf(nodeId);

  const inputMoreCount = isTrigger ? 0 : Math.max(0, currentIndex - 1);
  const inputPayload = isTrigger ? run.rootEvent?.data : execution?.rootEvent?.data;
  const inputTimestamp = formatEventTimestamp(
    isTrigger ? run.rootEvent?.createdAt : (execution?.rootEvent?.createdAt ?? execution?.createdAt),
  );

  const errorValue = presentation.tabData?.details?.Error;
  const isError = isErrorValue(errorValue);
  const summaryDetails = useMemo(() => {
    const details = presentation.tabData?.details ?? {};
    if (!isError) return details;
    return Object.fromEntries(Object.entries(details).filter(([key]) => key !== "Error"));
  }, [presentation.tabData?.details, isError]);

  const actionMarker = (
    <RunNodeIcon
      iconSrc={getHeaderIconSrc(presentation.workflowNode?.component)}
      iconSlug={
        presentation.workflowNode?.component ? componentIconMap[presentation.workflowNode.component] : undefined
      }
      alt={presentation.nodeName}
      size={RUN_NODE_ICON_SIZE}
      className="h-3.5 w-3.5"
    />
  );
  const statusTimeline = useMemo(
    () => (execution ? buildStepStatusTimeline(execution, presentation.workflowNode) : []),
    [execution, presentation.workflowNode],
  );

  const [chainModalOpen, setChainModalOpen] = useState(false);
  const previousStep = useMemo<InputChainStep | null>(() => {
    if (isTrigger || currentIndex <= 0) return null;
    const previousNodeId = chain[currentIndex - 1];
    const node = workflowNodes.find((item) => item.id === previousNodeId);
    const stepExecution = executions.find((item) => item.nodeId === previousNodeId);
    const payload =
      previousNodeId === triggerNodeId
        ? run.rootEvent?.data
        : stepExecution
          ? (extractExecutionPayload(stepExecution) ?? stepExecution.rootEvent?.data)
          : undefined;
    return {
      nodeId: previousNodeId,
      name: node?.name || previousNodeId,
      icon: (
        <RunNodeIcon
          iconSrc={getHeaderIconSrc(node?.component)}
          iconSlug={node?.component ? componentIconMap[node.component] : undefined}
          alt={node?.name || previousNodeId}
          size={RUN_NODE_ICON_SIZE}
          className="h-3.5 w-3.5"
        />
      ),
      payload,
    };
  }, [isTrigger, chain, currentIndex, workflowNodes, executions, triggerNodeId, run.rootEvent?.data, componentIconMap]);
  const inputChainSteps = useMemo<InputChainStep[]>(() => {
    if (currentIndex <= 0) return [];
    return chain
      .slice(0, currentIndex)
      .map((id) => {
        const node = workflowNodes.find((item) => item.id === id);
        const stepExecution = executions.find((item) => item.nodeId === id);
        const payload =
          id === triggerNodeId
            ? run.rootEvent?.data
            : stepExecution
              ? (extractExecutionPayload(stepExecution) ?? stepExecution.rootEvent?.data)
              : undefined;
        return {
          nodeId: id,
          name: node?.name || id,
          icon: (
            <RunNodeIcon
              iconSrc={getHeaderIconSrc(node?.component)}
              iconSlug={node?.component ? componentIconMap[node.component] : undefined}
              alt={node?.name || id}
              size={RUN_NODE_ICON_SIZE}
              className="h-3.5 w-3.5 shrink-0"
            />
          ),
          payload,
        };
      })
      .reverse();
  }, [chain, currentIndex, workflowNodes, executions, triggerNodeId, run.rootEvent?.data, componentIconMap]);

  return (
    <div className="bg-slate-50 px-3 py-3">
      <InputItem
        moreCount={inputMoreCount}
        payload={previousStep?.payload ?? inputPayload}
        timestamp={inputTimestamp}
        nodeName={previousStep?.name ?? presentation.nodeName}
        nodeIcon={previousStep?.icon ?? actionMarker}
        onOpenChain={inputChainSteps.length > 0 ? () => setChainModalOpen(true) : undefined}
      />
      {inputChainSteps.length > 0 ? (
        <InputChainModal open={chainModalOpen} onOpenChange={setChainModalOpen} steps={inputChainSteps} />
      ) : null}
      {!isTrigger && execution ? (
        <ActionItem
          marker={actionMarker}
          badge={presentation.headerEventBadge ?? { badgeColor: "bg-slate-400", label: "" }}
          elapsed={formatStepElapsed(execution)}
          summaryDetails={summaryDetails}
          statusBadge={presentation.headerEventBadge}
          relativeTime={presentation.createdAt}
          config={presentation.tabData?.configuration}
          statusTimeline={statusTimeline}
          timestamp={formatEventTimestamp(execution.createdAt)}
          nodeName={presentation.nodeName}
        />
      ) : null}
      <OutputItem
        isError={isError}
        errorMessage={isError ? (errorValue as { message?: string }).message : undefined}
        errorReason={execution?.resultReason}
        errorMetadata={execution?.metadata as Record<string, unknown> | undefined}
        payload={presentation.tabData?.payload}
        timestamp={formatEventTimestamp(execution?.updatedAt ?? execution?.createdAt ?? run.rootEvent?.createdAt)}
        nodeName={presentation.nodeName}
        nodeIcon={actionMarker}
      />
    </div>
  );
}
