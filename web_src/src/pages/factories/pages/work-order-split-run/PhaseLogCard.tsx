import { formatClockDurationLabel } from "@/lib/duration";
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { cn, resolveIcon } from "@/lib/utils";
import { ChevronRight, CircleX, Loader2, Maximize2, Pencil, RotateCw } from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Link } from "@/components/Link/link";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import type { PhaseGlyphKind } from "../../lib/linePhaseRuns";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { WorkOrderPullRequestInline } from "../../WorkOrderPullRequestInline";
import { PhaseGlyph } from "../linePhaseGlyph";
import { logStatusTimeLabel, tickingRunningClock } from "./logStatusTime";
import { SplitRunCheckPills } from "./SplitRunReview";
import { type SplitRunPhase, type SplitRunPhaseStatus, type SplitRunStreamLine } from "./splitRunMocks";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { groupPlanningSessionLog, mergePlanningSessionNotes } from "../planningSessionLog";
import { isRunnerComponent, mergeLiveStreamNotes, notesForLiveStream } from "./streamNotesFromLiveLog";

/** One face and size for every log row, matched to the run log viewer. */
const LOG_FACE = "font-mono text-[14px]";
const PHASE_NAME_FACE = cn("flex min-w-0 items-center gap-1.5", LOG_FACE, "font-medium");

function statusGlyph(status: SplitRunPhaseStatus): PhaseGlyphKind {
  if (status === "running") return "running";
  if (status === "passed") return "passed";
  if (status === "waiting") return "waiting";
  if (status === "cancelled") return "cancelled";
  if (status === "failed") return "failed";
  return "pending";
}

function statusTimeTone(status: SplitRunPhaseStatus): string {
  if (status === "passed") {
    return "text-[color:var(--status-completed-fg)]";
  }
  if (status === "running") {
    return "text-[color:var(--status-running-fg)]";
  }
  if (status === "waiting") {
    return "text-[color:var(--status-waiting-fg)]";
  }
  if (status === "failed") {
    return "text-[color:var(--status-failed-fg)]";
  }
  if (status === "cancelled") {
    return "text-[color:var(--status-cancelled-fg)]";
  }
  return "text-muted-foreground";
}

const LOG_ROW_HOVER = "hover:bg-[color:var(--status-running-bg)]";
const LOG_ROW_H = "h-[1.375rem]";
const STREAM_SECTION = "bg-muted px-2";
const STICKY_PHASE = "sticky top-0 z-30 h-8 bg-muted";
const STICKY_NODE = cn("sticky top-8 z-20", STREAM_SECTION);
const STICKY_STEP = cn("sticky top-[3.375rem] z-10", STREAM_SECTION);

const LAST_RUNNING_LINE_PULSE = "data-[last-running-line]:animate-pulse";

const STREAM_LINE_ROW = cn(
  "flex w-full min-w-0 max-w-full items-center justify-start overflow-hidden whitespace-nowrap text-left",
  STREAM_SECTION,
  LOG_ROW_H,
  LAST_RUNNING_LINE_PULSE,
);

const STREAM_LINE_WRAP_ROW = cn(
  "flex w-full min-w-0 max-w-full items-start justify-start text-left",
  STREAM_SECTION,
  LAST_RUNNING_LINE_PULSE,
);

function StreamLineTitle({ children, wrap = false }: { children: string; wrap?: boolean }) {
  if (wrap) {
    return <span className="min-w-0 flex-1 whitespace-normal break-words text-muted-foreground">{children}</span>;
  }
  return (
    <span className="min-w-0 w-0 flex-1 overflow-hidden">
      <span className="block truncate text-muted-foreground">{children}</span>
    </span>
  );
}

type StreamNodeGroup = {
  line: SplitRunStreamLine;
  notes: SplitRunStreamLine[];
  artifact?: FactoriesWorkOrderArtifact;
  pullRequest?: FactoriesFactoryPullRequest;
};

export function toolCallSummary(tools: Array<{ type?: string; componentType?: string }>): string {
  const files = tools.filter((tool) => (tool.type ?? tool.componentType) === "read").length;
  const commands = tools.length - files;
  const parts: string[] = [];
  if (files > 0) {
    parts.push(files === 1 ? "Read 1 file" : `Read ${files} files`);
  }
  if (commands > 0) {
    const ran = commands === 1 ? "ran 1 command" : `ran ${commands} commands`;
    parts.push(parts.length === 0 ? ran.charAt(0).toUpperCase() + ran.slice(1) : ran);
  }
  return parts.join(", ");
}

export type ClaudeStepEvent =
  | { kind: "note"; line: SplitRunStreamLine }
  | { kind: "tools"; id: string; tools: SplitRunStreamLine[]; label?: string };

export type ClaudeStepGroup = {
  line: SplitRunStreamLine;
  events: ClaudeStepEvent[];
};

export function groupClaudeSteps(notes: SplitRunStreamLine[]): ClaudeStepGroup[] {
  const steps: ClaudeStepGroup[] = [];
  let pendingTools: SplitRunStreamLine[] = [];
  let toolGroup = 0;

  const flushTools = (parent: ClaudeStepGroup) => {
    if (pendingTools.length === 0) {
      return;
    }
    parent.events.push({
      kind: "tools",
      id: `${parent.line.id}-tools-${toolGroup}`,
      tools: pendingTools,
    });
    toolGroup += 1;
    pendingTools = [];
  };

  for (const line of notes) {
    if (!line.noteParentId) {
      const current = steps.at(-1);
      if (current) {
        flushTools(current);
      }
      toolGroup = 0;
      steps.push({ line, events: [] });
      continue;
    }
    const parent = steps.find((step) => step.line.id === line.noteParentId);
    if (!parent) {
      continue;
    }
    if (line.componentType === "note") {
      appendStepNote(parent, line, flushTools);
      continue;
    }
    pendingTools.push(line);
  }

  const last = steps.at(-1);
  if (last) {
    flushTools(last);
  }
  return steps;
}

function appendStepNote(
  parent: ClaudeStepGroup,
  line: SplitRunStreamLine,
  flushTools: (parent: ClaudeStepGroup) => void,
) {
  if (!line.componentName.trim()) {
    return;
  }
  flushTools(parent);
  parent.events.push({ kind: "note", line });
}

export function groupSplitRunStream(lines: SplitRunStreamLine[]): StreamNodeGroup[] {
  const notesByNode = new Map<string, SplitRunStreamLine[]>();
  for (const line of lines) {
    if (!line.note || !line.nodeId) {
      continue;
    }
    const notes = notesByNode.get(line.nodeId) ?? [];
    notes.push(line);
    notesByNode.set(line.nodeId, notes);
  }

  return lines
    .filter((line) => !line.note && line.action !== "did not run")
    .map((line) => ({
      line,
      notes: notesByNode.get(line.nodeId ?? "") ?? [],
      artifact: line.artifact,
      pullRequest: line.pullRequest,
    }));
}

function artifactsProducedBySteps(
  groups: StreamNodeGroup[],
  fallback: FactoriesWorkOrderArtifact[],
): FactoriesWorkOrderArtifact[] {
  const produced: FactoriesWorkOrderArtifact[] = [];
  const seen = new Set<string>();
  const add = (artifact?: FactoriesWorkOrderArtifact) => {
    if (!artifact) {
      return;
    }
    const key = artifact.id ?? `${artifact.type}`;
    if (seen.has(key)) {
      return;
    }
    seen.add(key);
    produced.push(artifact);
  };
  for (const group of groups) {
    add(group.artifact);
    for (const note of group.notes) {
      add(note.artifact);
    }
  }
  return produced.length > 0 ? produced : fallback;
}

function pullRequestsProducedBySteps(groups: StreamNodeGroup[]): FactoriesFactoryPullRequest[] {
  const produced: FactoriesFactoryPullRequest[] = [];
  const seen = new Set<string>();
  const add = (pullRequest?: FactoriesFactoryPullRequest) => {
    if (!pullRequest) {
      return;
    }
    const key = pullRequest.id ?? pullRequest.url ?? String(pullRequest.number ?? "");
    if (!key || seen.has(key)) {
      return;
    }
    seen.add(key);
    produced.push(pullRequest);
  };
  for (const group of groups) {
    add(group.pullRequest);
    for (const note of group.notes) {
      add(note.pullRequest);
    }
  }
  return produced;
}

/**
 * Terminal log. The automation is the root. Each node hangs under it.
 * Automations keep produced artifacts on the title line when collapsed or expanded.
 * Expanded automations also show those artifacts on the producing steps.
 * Node icons keep the phase glyph column. Agent steps stay flush under them.
 * The last visible line of a running automation pulses so collapsed tools still show activity.
 */
function useLastRunningLine(rootRef: { current: HTMLElement | null }, active: boolean) {
  useLayoutEffect(() => {
    const root = rootRef.current;
    if (!root) {
      return;
    }

    const clear = () => {
      for (const el of root.querySelectorAll("[data-last-running-line]")) {
        el.removeAttribute("data-last-running-line");
      }
    };

    if (!active) {
      clear();
      return;
    }

    const mark = () => {
      const last = lastActiveStreamLine(root);
      for (const el of root.querySelectorAll("[data-stream-line]")) {
        if (el === last) {
          el.setAttribute("data-last-running-line", "");
        } else {
          el.removeAttribute("data-last-running-line");
        }
      }
    };

    mark();
    const observer = new MutationObserver(mark);
    observer.observe(root, { childList: true, subtree: true });
    return () => {
      observer.disconnect();
      clear();
    };
  }, [active, rootRef]);
}

function lastActiveStreamLine(root: HTMLElement): Element | undefined {
  const lines = root.querySelectorAll("[data-stream-line]");
  for (let index = lines.length - 1; index >= 0; index -= 1) {
    const line = lines.item(index);
    if (line.getAttribute("data-stream-status") !== "pending") {
      return line;
    }
  }
  return undefined;
}

function streamLineAttrs(status?: SplitRunPhaseStatus) {
  if (!status) {
    return { "data-stream-line": "" };
  }
  return { "data-stream-line": "", "data-stream-status": status };
}

function isNestedHeaderControl(target: EventTarget | null): boolean {
  return target instanceof Element && Boolean(target.closest("a, button, [role='button']"));
}

export function PhaseLogCard({
  phase,
  expanded,
  stream,
  selectedNodeId,
  onToggle,
  onSelectNode,
  onStop,
  onRerun,
  runHref,
  editHref,
  actionBusy = false,
  collapsible = true,
  organizationId,
  canvasId,
  compactSessionLog = false,
}: {
  phase: SplitRunPhase;
  expanded: boolean;
  stream?: SplitRunStreamLine[];
  selectedNodeId?: string | null;
  onToggle?: () => void;
  onSelectNode?: (nodeId: string) => void;
  onStop?: () => void;
  onRerun?: () => void;
  actionBusy?: boolean;
  /** Full-screen run page with the log in the sidebar. */
  runHref?: string;
  /** Configure URL of the automation that owns the phase. */
  editHref?: string;
  collapsible?: boolean;
  organizationId?: string;
  canvasId?: string;
  /** Collapse setup noise and bash in the Create with an Agent session log. */
  compactSessionLog?: boolean;
}) {
  const groups = groupSplitRunStream(stream ?? phase.stream);
  const producedArtifacts = artifactsProducedBySteps(groups, phase.artifacts);
  const producedPullRequests = pullRequestsProducedBySteps(groups);
  const rootRef = useRef<HTMLDivElement>(null);
  useLastRunningLine(rootRef, phase.status === "running");

  useEffect(() => {
    if (!selectedNodeId) {
      return;
    }
    const row = document.querySelector(`[data-testid="split-run-stream-line-${selectedNodeId}"]`);
    if (row instanceof HTMLElement && typeof row.scrollIntoView === "function") {
      row.scrollIntoView({ block: "nearest" });
    }
  }, [selectedNodeId]);

  const canToggleFromHeader = collapsible && Boolean(onToggle);

  return (
    <div
      ref={rootRef}
      className="min-w-0"
      data-testid={`split-run-phase-${phase.id}`}
      aria-current={expanded ? "step" : undefined}
    >
      <div
        className={cn(
          "rounded-md bg-muted",
          !expanded && LOG_ROW_HOVER,
          expanded ? "pb-2" : "py-2",
          canToggleFromHeader && !expanded && "cursor-pointer",
        )}
        data-testid={canToggleFromHeader ? `split-run-phase-expand-${phase.id}` : undefined}
        onClick={
          canToggleFromHeader
            ? (event) => {
                if (isNestedHeaderControl(event.target)) {
                  return;
                }
                onToggle?.();
              }
            : undefined
        }
      >
        <AutomationHeader
          phase={phase}
          expanded={expanded}
          collapsible={collapsible}
          producedArtifacts={producedArtifacts}
          producedPullRequests={producedPullRequests}
          onToggle={onToggle}
          onStop={onStop}
          onRerun={onRerun}
          runHref={runHref}
          editHref={editHref}
          actionBusy={actionBusy}
        />

        {expanded ? (
          <ol
            className={cn("mt-1 min-w-0 list-none leading-tight", LOG_FACE)}
            data-testid={`split-run-stream-${phase.id}`}
            onClick={(event) => event.stopPropagation()}
          >
            {groups.map((group) => (
              <StreamNode
                key={group.line.id}
                group={group}
                highlighted={Boolean(group.line.nodeId && group.line.nodeId === selectedNodeId)}
                onSelect={onSelectNode}
                organizationId={organizationId}
                canvasId={canvasId ?? phase.appId}
                compactSessionLog={compactSessionLog}
              />
            ))}
          </ol>
        ) : null}
      </div>
    </div>
  );
}

function AutomationHeader({
  phase,
  expanded,
  collapsible,
  producedArtifacts,
  producedPullRequests,
  onToggle,
  onStop,
  onRerun,
  runHref,
  editHref,
  actionBusy,
}: {
  phase: SplitRunPhase;
  expanded: boolean;
  collapsible: boolean;
  producedArtifacts: FactoriesWorkOrderArtifact[];
  producedPullRequests: FactoriesFactoryPullRequest[];
  onToggle?: () => void;
  onStop?: () => void;
  onRerun?: () => void;
  runHref?: string;
  editHref?: string;
  actionBusy: boolean;
}) {
  return (
    <div
      data-testid={`split-run-automation-header-${phase.id}`}
      {...streamLineAttrs(phase.status)}
      className={cn(
        "flex w-full min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap px-2 leading-tight bg-muted",
        LAST_RUNNING_LINE_PULSE,
        expanded && STICKY_PHASE,
        collapsible && onToggle && "cursor-pointer",
      )}
    >
      {collapsible ? (
        <button type="button" onClick={onToggle} aria-expanded={expanded} className={cn(PHASE_NAME_FACE, "text-left")}>
          <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3.5" />
          <span className="min-w-0 truncate text-foreground">{phase.name}</span>
        </button>
      ) : (
        <div className={PHASE_NAME_FACE}>
          <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3.5" />
          <span className="min-w-0 truncate text-foreground">{phase.name}</span>
        </div>
      )}
      {expanded ? <PhaseActionPills phase={phase} runHref={runHref} editHref={editHref} /> : null}
      {phase.status === "running" && onStop ? (
        <PhaseStopButton phaseId={phase.id} busy={actionBusy} onStop={onStop} />
      ) : null}
      {phase.status === "failed" && onRerun ? (
        <PhaseRerunButton phaseId={phase.id} busy={actionBusy} onRerun={onRerun} />
      ) : null}
      <span className="ml-auto flex min-w-0 items-center justify-end gap-2 overflow-hidden">
        {phase.checks && phase.checks.length > 0 ? (
          <span className="shrink-0">
            <SplitRunCheckPills checks={phase.checks} testId={`split-run-phase-checks-${phase.id}`} />
          </span>
        ) : null}
        {producedArtifacts.length > 0 || producedPullRequests.length > 0 ? (
          <span
            data-testid={`split-run-phase-artifacts-${phase.id}`}
            className="flex min-w-0 items-center justify-end gap-2 overflow-hidden whitespace-nowrap"
          >
            {producedPullRequests.map((pullRequest) => (
              <StreamPullRequest key={pullRequest.id ?? pullRequest.url} pullRequest={pullRequest} />
            ))}
            {producedArtifacts.map((artifact) => (
              <StreamArtifact key={artifact.id ?? `${artifact.type}`} artifact={artifact} />
            ))}
          </span>
        ) : null}
        <PhaseMetrics phase={phase} />
      </span>
    </div>
  );
}

const PHASE_ACTION_LINK = cn(
  LOG_FACE,
  "inline-flex size-6 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground",
);
const VIEW_RUN_LABEL = "View automation run";
const EDIT_AUTOMATION_LABEL = "Edit automation";
const RERUN_LABEL = "Rerun";
const STOP_LABEL = "Stop";
const PHASE_STOP_ACTION = cn(PHASE_ACTION_LINK, "hover:bg-destructive/10 hover:text-destructive");

function PhaseActionPills({ phase, runHref, editHref }: { phase: SplitRunPhase; runHref?: string; editHref?: string }) {
  if (!runHref && !editHref) {
    return null;
  }
  return (
    <>
      {runHref ? (
        <PhaseActionLink href={runHref} testId={`split-run-phase-run-${phase.id}`} label={VIEW_RUN_LABEL}>
          <Maximize2 className="size-3.5" aria-hidden />
        </PhaseActionLink>
      ) : null}
      {editHref ? (
        <PhaseActionLink href={editHref} testId={`split-run-phase-edit-${phase.id}`} label={EDIT_AUTOMATION_LABEL}>
          <Pencil className="size-3.5" aria-hidden />
        </PhaseActionLink>
      ) : null}
    </>
  );
}

function PhaseActionLink({
  href,
  testId,
  label,
  children,
}: {
  href: string;
  testId: string;
  label: string;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Link href={href} data-testid={testId} aria-label={label} className={PHASE_ACTION_LINK}>
          {children}
        </Link>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

function PhaseIconButton({
  testId,
  label,
  className = PHASE_ACTION_LINK,
  disabled,
  onClick,
  children,
}: {
  testId: string;
  label: string;
  className?: string;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          data-testid={testId}
          aria-label={label}
          className={className}
          disabled={disabled}
          onClick={onClick}
        >
          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

function PhaseRerunButton({ phaseId, busy, onRerun }: { phaseId: string; busy: boolean; onRerun: () => void }) {
  return (
    <PhaseIconButton testId={`split-run-phase-rerun-${phaseId}`} label={RERUN_LABEL} disabled={busy} onClick={onRerun}>
      {busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : <RotateCw className="size-3.5" aria-hidden />}
    </PhaseIconButton>
  );
}

function PhaseStopButton({ phaseId, busy, onStop }: { phaseId: string; busy: boolean; onStop: () => void }) {
  return (
    <PhaseIconButton
      testId={`split-run-phase-stop-${phaseId}`}
      label={STOP_LABEL}
      className={PHASE_STOP_ACTION}
      disabled={busy}
      onClick={onStop}
    >
      {busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : <CircleX className="size-3.5" aria-hidden />}
    </PhaseIconButton>
  );
}

function StreamDuration({ line }: { line: SplitRunStreamLine }) {
  return (
    <LogStatusTime status={line.status} duration={line.duration} testId={`split-run-stream-duration-${line.id}`} />
  );
}

function LogStatusTime({
  status,
  duration,
  testId,
}: {
  status: SplitRunPhaseStatus;
  duration?: string;
  testId: string;
}) {
  const running = status === "running";
  const { now, sampledAt } = useRunningLogClock(running, duration);
  const clock = running ? tickingRunningClock(duration, sampledAt, now) : logStatusTimeLabel(duration);
  const mark = statusTimeMark(status);
  if (!mark && !clock) {
    return null;
  }
  return (
    <span
      data-testid={testId}
      aria-label={statusTimeName(status)}
      className={cn(
        "inline-flex shrink-0 items-center gap-1 text-right tabular-nums [font-feature-settings:'zero']",
        LOG_FACE,
        statusTimeTone(status),
      )}
    >
      {mark}
      {clock ? <span>{clock}</span> : null}
    </span>
  );
}

function statusTimeName(status: SplitRunPhaseStatus): string | undefined {
  if (status === "passed") {
    return "Passed";
  }
  if (status === "running") {
    return "Running";
  }
  if (status === "failed") {
    return "Failed";
  }
  if (status === "waiting") {
    return "Waiting";
  }
  if (status === "cancelled") {
    return "Canceled";
  }
  return undefined;
}

function statusTimeMark(status: SplitRunPhaseStatus): ReactNode {
  if (status === "passed") {
    return <span aria-hidden>✓</span>;
  }
  if (status === "failed") {
    return <span aria-hidden>✗</span>;
  }
  return null;
}

function PhaseMetrics({ phase }: { phase: SplitRunPhase }) {
  const running = phase.status === "running";
  const { now, sampledAt } = useRunningLogClock(running, phase.duration);
  const clock = running
    ? tickingRunningClock(phase.duration, sampledAt, now)
    : formatClockDurationLabel(phase.duration);
  const tokens = parseWorkOrderMetric(phase.totalTokens);
  const cents = parseWorkOrderMetric(phase.costCents);
  const parts: string[] = [];
  if (cents > 0) {
    parts.push(formatUsdCents(cents));
  }
  if (tokens > 0) {
    parts.push(formatCompactTokenValue(tokens));
  }
  if (clock && clock !== "—") {
    parts.push(clock);
  }
  if (parts.length === 0) {
    return null;
  }
  return (
    <span
      data-testid={`split-run-phase-duration-${phase.id}`}
      className={cn(LOG_FACE, "tabular-nums text-muted-foreground")}
    >
      {parts.join(" · ")}
    </span>
  );
}

function useRunningLogClock(active: boolean, duration?: string) {
  const [now, setNow] = useState(() => Date.now());
  const sampleRef = useRef({ duration, at: Date.now() });
  if (sampleRef.current.duration !== duration) {
    sampleRef.current = { duration, at: Date.now() };
  }
  useEffect(() => {
    if (!active) {
      return;
    }
    const id = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(id);
  }, [active]);
  return { now, sampledAt: sampleRef.current.at };
}

function StreamNode({
  group,
  highlighted,
  onSelect,
  organizationId,
  canvasId,
  compactSessionLog,
}: {
  group: StreamNodeGroup;
  highlighted: boolean;
  onSelect?: (nodeId: string) => void;
  organizationId?: string;
  canvasId?: string;
  compactSessionLog: boolean;
}) {
  const { line, notes, artifact, pullRequest } = group;
  const liveNotes = useRunnerNodeLiveNotes(line, organizationId, canvasId);
  const merged = compactSessionLog
    ? mergePlanningSessionNotes(liveNotes, notes)
    : mergeLiveStreamNotes(liveNotes, notes);
  const steps = compactSessionLog ? groupPlanningSessionLog(merged) : groupClaudeSteps(merged);
  const hasChildren = steps.length > 0 || isRunnerComponent(line.component);

  return (
    <li className="min-w-0">
      <StreamNodeHeader
        line={line}
        hasChildren={hasChildren}
        highlighted={highlighted}
        artifact={artifact}
        pullRequest={pullRequest}
        onSelect={onSelect}
      />
      {hasChildren ? (
        <ol className="min-w-0">
          {steps.map((step) => (
            <StreamStep key={step.line.id} step={step} highlightUserTalk={compactSessionLog} />
          ))}
        </ol>
      ) : null}
    </li>
  );
}

function StreamNodeHeader({
  line,
  hasChildren,
  highlighted,
  artifact,
  pullRequest,
  onSelect,
}: {
  line: SplitRunStreamLine;
  hasChildren: boolean;
  highlighted: boolean;
  artifact?: FactoriesWorkOrderArtifact;
  pullRequest?: FactoriesFactoryPullRequest;
  onSelect?: (nodeId: string) => void;
}) {
  const name = (
    <>
      <StreamLineIcon iconSlug={line.iconSlug} iconSrc={line.iconSrc} />
      <span
        className={cn(
          "min-w-0 overflow-hidden truncate",
          line.status === "pending" ? "text-muted-foreground" : "text-foreground",
        )}
      >
        {line.componentName}
      </span>
    </>
  );

  return (
    <div
      data-testid={`split-run-stream-line-${line.id}`}
      {...streamLineAttrs(line.status)}
      data-highlighted={highlighted ? "true" : undefined}
      aria-current={highlighted ? "true" : undefined}
      className={cn(
        "flex w-full min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap",
        LOG_ROW_H,
        STREAM_SECTION,
        LAST_RUNNING_LINE_PULSE,
        hasChildren && STICKY_NODE,
        highlighted && "ring-1 ring-foreground/15",
      )}
    >
      {line.nodeId ? (
        <button
          type="button"
          data-testid={`split-run-node-toggle-${line.id}`}
          onClick={() => onSelect?.(line.nodeId ?? line.id)}
          className="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden text-left cursor-pointer hover:text-foreground"
        >
          {name}
        </button>
      ) : (
        <div
          data-testid={`split-run-node-toggle-${line.id}`}
          className="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden"
        >
          {name}
        </div>
      )}
      <span className="ml-auto flex min-w-0 shrink-0 items-center justify-end gap-2 overflow-hidden whitespace-nowrap">
        {pullRequest ? <StreamPullRequest pullRequest={pullRequest} /> : null}
        {artifact ? <StreamArtifact artifact={artifact} /> : null}
        <StreamDuration line={line} />
      </span>
    </div>
  );
}

function StreamStep({ step, highlightUserTalk = false }: { step: ClaudeStepGroup; highlightUserTalk?: boolean }) {
  const hasOutput = Boolean(step.line.detail);
  const hasBody = step.events.length > 0 || hasOutput;
  const showHeader = Boolean(step.line.componentName.trim() || step.line.componentType);

  return (
    <li className="min-w-0">
      {showHeader ? (
        <div
          data-testid={`split-run-stream-line-${step.line.id}`}
          {...streamLineAttrs(step.line.status)}
          className={cn(STREAM_LINE_WRAP_ROW, hasBody && STICKY_STEP)}
        >
          {step.line.componentType ? (
            <span className={cn("mr-2 shrink-0", stepTypeTone(step.line.componentType))}>
              {step.line.componentType}
            </span>
          ) : null}
          <StreamLineTitle wrap>{step.line.componentName}</StreamLineTitle>
          <StepStatusMark status={step.line.status} />
        </div>
      ) : null}
      {hasBody ? (
        <>
          {hasOutput ? <StreamOutput text={step.line.detail ?? ""} /> : null}
          {step.events.map((event) =>
            event.kind === "note" ? (
              <StreamTalkNote key={event.line.id} line={event.line} highlightUserTalk={highlightUserTalk} />
            ) : (
              <StreamToolGroup key={event.id} stepId={event.id} tools={event.tools} label={event.label} />
            ),
          )}
        </>
      ) : null}
    </li>
  );
}

function StreamTalkNote({ line, highlightUserTalk }: { line: SplitRunStreamLine; highlightUserTalk: boolean }) {
  const isUserTalk = highlightUserTalk && (line.componentType === "prompt" || Boolean(line.userTalk));
  const youLabel = line.userTalk === "survey" ? CREATE_WITH_AGENT_COPY.youSurvey : CREATE_WITH_AGENT_COPY.you;
  return (
    <div
      data-testid={isUserTalk ? "split-run-user-note" : `split-run-stream-line-${line.id}`}
      {...streamLineAttrs(line.status)}
      className={cn("flex w-full items-start", STREAM_SECTION, LAST_RUNNING_LINE_PULSE)}
    >
      <span className="inline-flex w-4 shrink-0" aria-hidden />
      {isUserTalk ? (
        <div className="min-w-0 flex-1 rounded-md border-l-2 border-primary/50 bg-primary/10 px-2 py-1">
          <span className="mb-0.5 block text-[11px] font-medium leading-none text-primary">{youLabel}</span>
          <span className="whitespace-normal break-words leading-5 text-foreground">{line.componentName}</span>
        </div>
      ) : (
        <span className="min-w-0 flex-1 whitespace-normal break-words py-0.5 leading-5 text-foreground">
          {line.componentName}
        </span>
      )}
    </div>
  );
}

function StreamToolGroup({ stepId, tools, label }: { stepId: string; tools: SplitRunStreamLine[]; label?: string }) {
  const [expanded, setExpanded] = useState(false);
  const summary = label ?? toolCallSummary(tools);

  return (
    <div className="min-w-0">
      <button
        type="button"
        data-testid={`split-run-tools-toggle-${stepId}`}
        {...streamLineAttrs(tools.some((tool) => tool.status === "running") ? "running" : tools.at(-1)?.status)}
        aria-expanded={expanded}
        aria-label={summary}
        onClick={() => setExpanded((open) => !open)}
        className={cn(STREAM_LINE_ROW, "text-muted-foreground")}
      >
        <span className="inline-flex w-4 shrink-0 items-center justify-start">
          <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
        </span>
        <StreamLineTitle>{summary}</StreamLineTitle>
      </button>
      {expanded ? (
        <ol className="min-w-0">
          {tools.map((tool) => (
            <StreamTool key={tool.id} tool={tool} />
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function StreamTool({ tool }: { tool: SplitRunStreamLine }) {
  const [expanded, setExpanded] = useState(tool.status === "running");
  useEffect(() => {
    if (tool.status === "running") {
      setExpanded(true);
    }
  }, [tool.status]);
  const hasOutput = Boolean(tool.detail);
  const row = (
    <>
      <ExpandChevron expanded={expanded} visible={hasOutput} />
      {tool.componentType ? (
        <span className={cn("mr-2 shrink-0", stepTypeTone(tool.componentType))}>{tool.componentType}</span>
      ) : null}
      <StreamLineTitle wrap>{tool.componentName}</StreamLineTitle>
      <StepStatusMark status={tool.status} />
    </>
  );

  return (
    <li className="min-w-0">
      {hasOutput ? (
        <button
          type="button"
          data-testid={`split-run-stream-line-${tool.id}`}
          {...streamLineAttrs(tool.status)}
          aria-expanded={expanded}
          onClick={() => setExpanded((open) => !open)}
          className={cn(STREAM_LINE_WRAP_ROW, "cursor-pointer hover:text-foreground")}
        >
          {row}
        </button>
      ) : (
        <div
          data-testid={`split-run-stream-line-${tool.id}`}
          {...streamLineAttrs(tool.status)}
          className={STREAM_LINE_WRAP_ROW}
        >
          {row}
        </div>
      )}
      {expanded && hasOutput ? <StreamOutput text={tool.detail ?? ""} /> : null}
    </li>
  );
}

function StreamOutput({ text }: { text: string }) {
  return (
    <div data-testid="split-run-stream-output" className="min-w-0">
      {text.split("\n").map((line, index) => (
        <div
          key={index}
          data-stream-line=""
          className={cn("flex min-w-0 w-full items-start", STREAM_SECTION, LOG_FACE, LAST_RUNNING_LINE_PULSE)}
        >
          <span className="inline-flex w-4 shrink-0" aria-hidden />
          <pre className="min-w-0 flex-1 whitespace-pre-wrap break-words py-0.5 leading-5 text-muted-foreground">
            {line.length > 0 ? line : " "}
          </pre>
        </div>
      ))}
    </div>
  );
}

function StepStatusMark({ status }: { status: SplitRunPhaseStatus }) {
  if (status === "failed") {
    return (
      <span className="ml-auto shrink-0 pl-2 text-[color:var(--status-failed-fg)]" aria-label="failed">
        ✗
      </span>
    );
  }
  if (status === "passed") {
    return (
      <span className="ml-auto shrink-0 pl-2 text-[color:var(--status-success)]" aria-label="passed">
        ✓
      </span>
    );
  }
  return null;
}

function ExpandChevron({ expanded, visible }: { expanded: boolean; visible: boolean }) {
  return (
    <span className="inline-flex w-4 shrink-0 items-center justify-start text-muted-foreground">
      {visible ? (
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
      ) : null}
    </span>
  );
}

function stepTypeTone(type: string): string {
  if (type === "prompt") {
    return "text-[#8658d6]";
  }
  if (type === "bash") {
    return "text-[#fd7e14]";
  }
  return "text-muted-foreground";
}

function StreamArtifact({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  return (
    <WorkOrderArtifactInline
      className={cn(LOG_FACE, "font-bold tracking-normal")}
      artifact={{
        id: artifact.id,
        type: artifact.type ?? "TYPE_UNSPECIFIED",
        data: toArtifactDataRecord(artifact.data),
      }}
    />
  );
}

function StreamPullRequest({ pullRequest }: { pullRequest: FactoriesFactoryPullRequest }) {
  return <WorkOrderPullRequestInline className={cn(LOG_FACE, "font-bold tracking-normal")} pullRequest={pullRequest} />;
}

function StreamLineIcon({ iconSlug, iconSrc }: { iconSlug?: string; iconSrc?: string }) {
  const Icon = resolveIcon(iconSlug ?? "box");
  return (
    <span className="inline-flex size-3 shrink-0 items-center justify-center text-muted-foreground" aria-hidden>
      {iconSrc ? <img src={iconSrc} alt="" className="size-3 object-contain" /> : <Icon className="size-3" />}
    </span>
  );
}

function useRunnerNodeLiveNotes(
  line: SplitRunStreamLine,
  organizationId?: string,
  canvasId?: string,
): SplitRunStreamLine[] | undefined {
  const canStream = Boolean(organizationId && canvasId && line.executionId && isRunnerComponent(line.component));
  const { sections, orphanLines, error, isStreaming } = useLiveLogStream(
    canStream ? (line.executionId ?? "") : "",
    line.status === "running",
    line.status === "failed" ? "failed" : line.status === "passed" ? "passed" : null,
    null,
    { organizationId, canvasId },
  );
  if (!canStream) {
    return undefined;
  }
  return notesForLiveStream({
    nodeId: line.nodeId ?? line.id,
    sections,
    orphanLines,
    error,
    isStreaming,
    nodeStatus: line.status,
  });
}
