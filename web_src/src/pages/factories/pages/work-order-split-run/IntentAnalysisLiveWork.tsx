import { useEffect, useMemo, useRef, useState } from "react";

import { cn } from "@/lib/utils";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { ChevronRight } from "lucide-react";

import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";
import { PLANNING_SESSION_AGENT_LINE_ID } from "../planningSessionActivity";
import {
  ANALYSIS_REPLY_THINKING_STATES,
  ANALYSIS_THINKING_INTERVAL_MS,
  ANALYSIS_THINKING_STATES,
  analysisLiveWorkKind,
  hasAgentReasoning,
  liveReasoningForTurn,
  reasoningLinesFromPlanningNotes,
  staleReasoningIds,
  thinkingStatusFor,
  type ReasoningItem,
} from "./analysisLiveWorkState";
import { notesForLiveStream } from "./streamNotesFromLiveLog";

export function AnalysisLiveWork({
  machineStatus,
  organizationId,
  canvasId,
  executionId,
  items,
  waitingForAgent = false,
  previousAgentText,
}: {
  machineStatus: CreateWithAgentMachineStatus;
  organizationId?: string;
  canvasId?: string;
  executionId?: string;
  items?: ReasoningItem[];
  waitingForAgent?: boolean;
  previousAgentText?: string;
}) {
  const liveItems = useAnalysisReasoningItems({
    organizationId,
    canvasId,
    executionId,
    active: machineStatus === "starting" || machineStatus === "running",
    skip: items !== undefined,
  });
  const rawItems = items ?? liveItems;
  const staleIds = useStaleReasoningIds(rawItems, Boolean(previousAgentText));
  const reasoningItems = liveReasoningForTurn(rawItems, previousAgentText, staleIds);
  const kind = analysisLiveWorkKind({ machineStatus, items: reasoningItems });

  if (kind === "idle") {
    return null;
  }
  const showThinking =
    waitingForAgent && !hasAgentReasoning(reasoningItems) && (kind === "thinking" || kind === "reasoning");
  return (
    <div data-testid="split-run-intent-live-work">
      {showThinking ? <ThinkingStatus reply={Boolean(previousAgentText)} /> : null}
      {kind === "reasoning" ? <ReasoningStream items={reasoningItems} /> : null}
    </div>
  );
}

function useStaleReasoningIds(items: ReasoningItem[], hidePrevious: boolean): ReadonlySet<string> {
  const staleIds = useRef<Set<string>>(new Set());
  const wasHiding = useRef<boolean | null>(null);
  if (wasHiding.current === null) {
    wasHiding.current = hidePrevious;
    return staleIds.current;
  }
  if (hidePrevious && !wasHiding.current) {
    staleIds.current = staleReasoningIds(items);
  }
  if (!hidePrevious) {
    staleIds.current = new Set();
  }
  wasHiding.current = hidePrevious;
  return staleIds.current;
}

function ThinkingStatus({ reply = false }: { reply?: boolean }) {
  const [elapsedMs, setElapsedMs] = useState(0);
  useEffect(() => {
    const timer = window.setInterval(() => {
      setElapsedMs((current) => current + ANALYSIS_THINKING_INTERVAL_MS);
    }, ANALYSIS_THINKING_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, []);
  const status = thinkingStatusFor(
    elapsedMs,
    ANALYSIS_THINKING_INTERVAL_MS,
    reply ? ANALYSIS_REPLY_THINKING_STATES : ANALYSIS_THINKING_STATES,
  );

  return (
    <div className="px-2 py-1.5">
      <p
        className="sp-ai-thinking text-[13px] leading-5 text-muted-foreground"
        data-testid="split-run-intent-thinking"
        data-text={status}
      >
        {status}
      </p>
    </div>
  );
}

function ReasoningStream({ items }: { items: ReasoningItem[] }) {
  const liveId = items.at(-1)?.id;

  return (
    <div className="sp-reasoning-stream px-2 py-1.5" data-testid="split-run-intent-reasoning">
      {items.map((item) => (
        <ReasoningLine key={item.id} item={item} live={item.id === liveId} />
      ))}
    </div>
  );
}

function ReasoningLine({ item, live }: { item: ReasoningItem; live: boolean }) {
  if (item.details && item.details.length > 0) {
    return <ReasoningTools item={item} live={live} />;
  }
  return (
    <p
      data-testid={`split-run-intent-reasoning-${item.id}`}
      data-text={live ? item.text : undefined}
      className={cn("sp-reasoning-stream-line", live && "sp-ai-thinking")}
    >
      {item.text}
    </p>
  );
}

function ReasoningTools({ item, live }: { item: ReasoningItem; live: boolean }) {
  const [open, setOpen] = useState(false);

  return (
    <div>
      <button
        type="button"
        data-testid={`split-run-intent-tools-${item.id}`}
        aria-expanded={open}
        aria-label={item.text}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-start gap-1 text-left"
      >
        <ChevronRight className={cn("mt-0.5 size-3 shrink-0 text-muted-foreground", open && "rotate-90")} aria-hidden />
        <span
          data-testid={`split-run-intent-reasoning-${item.id}`}
          data-text={live ? item.text : undefined}
          className={cn("sp-reasoning-stream-line", live && "sp-ai-thinking")}
        >
          {item.text}
        </span>
      </button>
      {open ? (
        <ul className="ml-4 space-y-0.5">
          {item.details?.map((detail, index) => (
            <li key={`${index}:${detail}`} className="text-[12px] leading-4 text-muted-foreground">
              {detail}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function useAnalysisReasoningItems(args: {
  organizationId?: string;
  canvasId?: string;
  executionId?: string;
  active: boolean;
  skip: boolean;
}): ReasoningItem[] {
  const canStream = !args.skip && Boolean(args.organizationId && args.canvasId && args.executionId && args.active);
  const { sections, orphanLines, error, isStreaming } = useLiveLogStream(
    canStream ? (args.executionId ?? "") : "",
    canStream && args.active,
    null,
    null,
    { organizationId: args.organizationId, canvasId: args.canvasId },
  );
  return useMemo(() => {
    if (!canStream) {
      return [];
    }
    const notes = notesForLiveStream({
      nodeId: PLANNING_SESSION_AGENT_LINE_ID,
      sections,
      orphanLines,
      error,
      isStreaming,
      nodeStatus: "running",
    });
    return reasoningLinesFromPlanningNotes(notes ?? []);
  }, [canStream, error, isStreaming, orphanLines, sections]);
}
