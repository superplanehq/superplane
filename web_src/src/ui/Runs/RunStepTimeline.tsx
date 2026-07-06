import JsonView from "@uiw/react-json-view";
import { ArrowUp, Check, ChevronDown, ChevronRight, CircleHelp, Copy, Maximize2, Sparkles } from "lucide-react";
import { Fragment, useCallback, useMemo, useState, useSyncExternalStore, type ReactNode } from "react";
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
import { DetailBox, ErrorDetailBox, HeaderIconButton, PayloadMonaco } from "./RunStepAccordion";
import {
  buildExecutionChain,
  buildStepStatusTimeline,
  extractExecutionPayload,
  isErrorValue,
} from "./runNodeDetailModel";
import { formatEventTimestamp, formatStepElapsed } from "./runSummary";
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
            {steps.map((step, index) => (
              <Fragment key={step.nodeId}>
                {index > 0 ? (
                  <ArrowUp aria-hidden className="mx-auto my-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" />
                ) : null}
                <button
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
              </Fragment>
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
 * setting persisted to local storage, so toggling e.g. the runtime "config" open
 * stays open across every step the user inspects. Backed by a tiny module-level store
 * so all mounted timeline items stay in sync.
 */
type TimelineToggleKey = "config";
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

/** Rail-column marker for a big event card: the source node icon in a ringed circle. */
export function CardMarker({ children }: { children: ReactNode }) {
  return (
    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-slate-500 ring-1 ring-slate-200">
      {children}
    </span>
  );
}

/** One entry hanging off the shared vertical rail: a marker column plus the row/card content. */
export function EventRail({ marker, isLast, children }: { marker: ReactNode; isLast?: boolean; children: ReactNode }) {
  return (
    <div className="flex gap-3">
      <div className="flex flex-col items-center">
        {marker}
        {!isLast ? <div className="min-h-4 w-px flex-1 bg-slate-200" /> : null}
      </div>
      <div className="min-w-0 flex-1 pb-4">{children}</div>
    </div>
  );
}

/** Merged status pill (colored dot + label) shown at the start of a card header. */
export function EventStatusPill({
  dotClassName,
  label,
  tone = "default",
}: {
  dotClassName: string;
  label: string;
  tone?: "default" | "error";
}) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center gap-1.5 rounded-full bg-white px-2 py-0.5 ring-1",
        tone === "error" ? "ring-red-200" : "ring-slate-200",
      )}
    >
      <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClassName)} />
      <span className={cn("text-[11px] font-medium capitalize", tone === "error" ? "text-red-600" : "text-slate-700")}>
        {label}
      </span>
    </span>
  );
}

/**
 * A GitHub-comment-style card for a timeline event. The header leads with the lifecycle
 * status pill, then an optional label (kicker) and the source name, meta (elapsed /
 * timestamp) and actions. With a payload, the JSON body is expanded by default and
 * collapsible ("Expand" opens the full Monaco modal); without one it renders as a clean
 * header-only "title box" for lifecycle-only events like Queued.
 */
export function PayloadEventCard({
  kicker,
  status,
  sourceName,
  sourceTrailing,
  meta,
  headerExtras,
  open: openProp,
  onToggleOpen,
  payload,
  modalNodeName,
  modalNodeIcon,
}: {
  /** Optional uppercase label (INPUT / OUTPUT / RUNTIME CONFIG); omitted for lifecycle-only cards. */
  kicker?: string;
  status: { dotClassName: string; label: string };
  sourceName: string;
  /** Rendered immediately to the right of the source name (e.g. the input-chain "+X more"). */
  sourceTrailing?: ReactNode;
  meta?: string | null;
  headerExtras?: ReactNode;
  /** When provided, the expand arrow is controlled by the parent (e.g. to co-toggle a sibling Summary box). */
  open?: boolean;
  onToggleOpen?: () => void;
  payload: unknown;
  modalNodeName: string;
  modalNodeIcon: ReactNode;
}) {
  const [openInternal, setOpenInternal] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [modalCopied, setModalCopied] = useState(false);
  const controlled = openProp !== undefined;
  const open = controlled ? openProp : openInternal;
  const toggleOpen = () => (onToggleOpen ? onToggleOpen() : setOpenInternal((value) => !value));
  const canShowPayload = hasData(payload);
  // When controlled, the parent has a sibling box (Summary) to co-reveal, so keep the arrow.
  const canExpand = canShowPayload || controlled;
  const payloadString = useMemo(() => JSON.stringify(payload ?? {}, null, 2), [payload]);

  const copyPayload = (mark: (value: boolean) => void) => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    mark(true);
    setTimeout(() => mark(false), 1500);
  };

  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white">
      <div className="flex items-center gap-1.5 border-b border-slate-200 bg-slate-50 px-3 py-1.5">
        <EventStatusPill dotClassName={status.dotClassName} label={status.label} />
        {kicker ? (
          <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400">{kicker}</span>
        ) : null}
        <span className="min-w-0 truncate text-[12px] font-medium text-slate-600">{sourceName}</span>
        {sourceTrailing}
        <div className="ml-auto flex shrink-0 items-center gap-0.5">
          {meta ? <span className="pr-1 text-[11px] tabular-nums text-slate-600">{meta}</span> : null}
          {headerExtras}
          {canShowPayload ? (
            <>
              <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
              <HeaderIconButton
                label={copied ? "Copied" : "Copy"}
                icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                onClick={() => copyPayload(setCopied)}
              />
              <HeaderIconButton
                label="Open fullscreen"
                icon={<Maximize2 className="h-3.5 w-3.5" />}
                onClick={() => setModalOpen(true)}
              />
            </>
          ) : null}
          {canExpand ? (
            <HeaderIconButton
              label={open ? "Collapse" : "Expand"}
              icon={open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
              active={open}
              onClick={toggleOpen}
            />
          ) : null}
        </div>
      </div>
      {open && canShowPayload ? (
        <div className="px-3 py-2.5">
          <JsonView value={payload as object} collapsed={2} displayDataTypes={false} style={{ fontSize: 12 }} />
        </div>
      ) : null}

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent
          size="large"
          className="flex h-[80vh] w-[60vw] max-w-[60vw] flex-col gap-0 overflow-hidden p-0"
          onClick={(event) => event.stopPropagation()}
        >
          <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10">
            <div className="flex min-w-0 items-center gap-1.5">
              {modalNodeIcon}
              <span className="truncate text-[12px] font-medium text-slate-700">{modalNodeName}</span>
              <DialogTitle className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                {kicker}
              </DialogTitle>
            </div>
            <div className="flex items-center gap-0.5">
              <HeaderIconButton label="Send to AI" icon={<Sparkles className="h-3.5 w-3.5" />} />
              <HeaderIconButton
                label={modalCopied ? "Copied" : "Copy"}
                icon={
                  modalCopied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />
                }
                onClick={() => copyPayload(setModalCopied)}
              />
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-hidden">
            <PayloadMonaco value={payloadString} />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/**
 * The final event in the feed: the output emitted (payload card) or, when the step
 * errored, a red error card - either way merged with the terminal status. The finished
 * step's Summary is shown before the output payload, both revealed by the card's
 * single expand arrow.
 */
function TerminalEventCard({
  isError,
  errorMessage,
  errorReason,
  errorMetadata,
  status,
  sourceName,
  meta,
  payload,
  summaryBody,
  nodeName,
  marker,
}: {
  isError: boolean;
  errorMessage?: string;
  errorReason?: string;
  errorMetadata?: Record<string, unknown>;
  status: { dotClassName: string; label: string };
  sourceName: string;
  meta?: string | null;
  payload: unknown;
  /** The finished step's Summary, shown above the output payload under the expand arrow. */
  summaryBody?: ReactNode;
  nodeName: string;
  marker: ReactNode;
}) {
  // Summary (Status box) and the output payload are two sibling boxes co-toggled by the
  // output card's single expand arrow.
  const [open, setOpen] = useState(true);
  return (
    <EventRail marker={<CardMarker>{marker}</CardMarker>} isLast>
      {isError ? (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-1.5">
            <EventStatusPill dotClassName="bg-red-500" label={status.label || "Errored"} tone="error" />
            <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400">Output</span>
            <span className="min-w-0 truncate text-[12px] font-medium text-slate-600">{sourceName}</span>
            <div className="ml-auto flex shrink-0 items-center gap-0.5">
              {meta ? <span className="pr-1 text-[11px] tabular-nums text-slate-600">{meta}</span> : null}
            </div>
          </div>
          {summaryBody}
          <ErrorDetailBox message={errorMessage} reason={errorReason} metadata={errorMetadata} />
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {open ? summaryBody : null}
          <PayloadEventCard
            kicker="Output"
            status={status}
            sourceName={sourceName}
            meta={meta}
            open={open}
            onToggleOpen={() => setOpen((value) => !value)}
            payload={payload}
            modalNodeName={nodeName}
            modalNodeIcon={marker}
          />
        </div>
      )}
    </EventRail>
  );
}

/** Joins timestamp/elapsed meta fragments into a single `A · B` string, dropping empties. */
function joinMeta(...parts: (string | null | undefined)[]): string | null {
  const present = parts.filter((part): part is string => !!part);
  return present.length ? present.join(" · ") : null;
}

/**
 * The expanded content of a run step, rendered as a GitHub-issue-style event feed on a
 * shared vertical rail where every event is a card: an input card (merging the
 * "Triggered" status), header-only cards for the middle lifecycle statuses (Queued) plus
 * a Running/Waiting card that hosts the runtime config, and a terminal output/error card
 * (merging the final outcome and the finished step's Summary/Status box). Triggers show
 * only the input card.
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

  const [showConfig, toggleConfig] = useTimelineToggle("config", true);

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

  // Feed shape: the "Triggered" status is folded into the input card and the terminal
  // outcome into the output/error card, so only the middle statuses render as their own cards.
  const finished = execution?.state === "STATE_FINISHED";
  const showTerminalCard = !isTrigger && !!execution && finished;
  const middleEntries = useMemo(
    () => (statusTimeline.length ? statusTimeline.slice(1, finished ? -1 : undefined) : []),
    [statusTimeline, finished],
  );
  const hasMiddle = middleEntries.length > 0;

  const inputSourceName = previousStep?.name ?? presentation.nodeName;
  const inputSourceIcon = previousStep?.icon ?? actionMarker;
  const terminalStatus = presentation.headerEventBadge
    ? { dotClassName: presentation.headerEventBadge.badgeColor, label: presentation.headerEventBadge.label }
    : { dotClassName: "bg-slate-400", label: "Done" };

  // Runtime config is hosted on the Running/Waiting card and stays there even after the
  // step finishes. The Summary (Status box) belongs to the finished step and is shown
  // before the output payload under the terminal card's arrow.
  const config = presentation.tabData?.configuration;
  const canShowConfig = hasData(config);
  // The Summary box always shows Status / Relative time, so render it for any finished
  // step that has a status badge (not only when there are extra detail fields).
  const hasSummary =
    Object.keys(summaryDetails).length > 0 || !!presentation.headerEventBadge || !!presentation.createdAt;
  const summaryBody = hasSummary ? (
    <DetailBox title="Summary">
      <RunNodeDetailDetailsView
        details={summaryDetails}
        statusBadge={presentation.headerEventBadge}
        relativeTime={presentation.createdAt}
      />
    </DetailBox>
  ) : null;

  return (
    <div className="bg-slate-50 px-3 py-3">
      <EventRail marker={<CardMarker>{inputSourceIcon}</CardMarker>} isLast={!hasMiddle && !showTerminalCard}>
        <PayloadEventCard
          kicker="Input"
          status={{ dotClassName: "bg-violet-400", label: "Triggered" }}
          sourceName={inputSourceName}
          sourceTrailing={
            inputMoreCount > 0 && inputChainSteps.length > 0 ? (
              <button
                type="button"
                onClick={() => setChainModalOpen(true)}
                title="Open input chain"
                className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700"
              >
                +{inputMoreCount} more
              </button>
            ) : null
          }
          meta={inputTimestamp}
          payload={previousStep?.payload ?? inputPayload}
          modalNodeName={inputSourceName}
          modalNodeIcon={inputSourceIcon}
        />
      </EventRail>
      {inputChainSteps.length > 0 ? (
        <InputChainModal open={chainModalOpen} onOpenChange={setChainModalOpen} steps={inputChainSteps} />
      ) : null}

      {middleEntries.map((entry, index) => {
        const isRunning = entry.key === "running" || entry.key === "waiting";
        const isLast = index === middleEntries.length - 1 && !showTerminalCard;
        return (
          <EventRail key={entry.key} marker={<CardMarker>{actionMarker}</CardMarker>} isLast={isLast}>
            <PayloadEventCard
              status={{ dotClassName: entry.dotClassName, label: entry.label }}
              kicker={isRunning ? "Runtime Config" : undefined}
              sourceName={presentation.nodeName}
              meta={
                isRunning
                  ? joinMeta(execution ? formatStepElapsed(execution) : null, formatEventTimestamp(entry.timestamp))
                  : formatEventTimestamp(entry.timestamp)
              }
              open={isRunning && canShowConfig ? showConfig : undefined}
              onToggleOpen={isRunning && canShowConfig ? toggleConfig : undefined}
              payload={isRunning ? config : undefined}
              modalNodeName={presentation.nodeName}
              modalNodeIcon={actionMarker}
            />
          </EventRail>
        );
      })}

      {showTerminalCard && execution ? (
        <TerminalEventCard
          isError={isError}
          errorMessage={isError ? (errorValue as { message?: string }).message : undefined}
          errorReason={execution.resultReason}
          errorMetadata={execution.metadata as Record<string, unknown> | undefined}
          status={terminalStatus}
          sourceName={presentation.nodeName}
          meta={joinMeta(
            formatStepElapsed(execution),
            formatEventTimestamp(execution.updatedAt ?? execution.createdAt),
          )}
          payload={presentation.tabData?.payload}
          summaryBody={summaryBody}
          nodeName={presentation.nodeName}
          marker={actionMarker}
        />
      ) : null}
    </div>
  );
}
