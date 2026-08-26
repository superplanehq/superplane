import { cn, resolveIcon } from "@/lib/utils";
import { ChevronRight, Loader2, Pencil } from "lucide-react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";

import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import type { PhaseGlyphKind } from "../../lib/linePhaseRuns";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { PhaseGlyph } from "../linePhaseGlyph";
import { logStatusTimeLabel, runningSpinnerFrame, tickingRunningClock } from "./logStatusTime";
import { SplitRunCheckPills } from "./SplitRunReview";
import { type SplitRunPhase, type SplitRunPhaseStatus, type SplitRunStreamLine } from "./splitRunMocks";
import { isRunnerComponent, notesForLiveStream } from "./streamNotesFromLiveLog";

/** One face and size for every log row, matched to the run log viewer. */
const LOG_FACE = "font-mono text-[14px]";

function statusGlyph(status: SplitRunPhaseStatus): PhaseGlyphKind {
  if (status === "running") return "running";
  if (status === "passed") return "passed";
  if (status === "waiting") return "waiting";
  if (status === "failed") return "failed";
  return "pending";
}

function statusTimeTone(status: SplitRunPhaseStatus): string {
  if (status === "passed") {
    return "bg-[color:var(--status-completed-bg)] text-[color:var(--status-completed-fg)]";
  }
  if (status === "running") {
    return "bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]";
  }
  if (status === "waiting") {
    return "bg-[color:var(--status-waiting-bg)] text-[color:var(--status-waiting-fg)]";
  }
  if (status === "failed") {
    return "bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]";
  }
  return "text-muted-foreground";
}

const LOG_ROW_HOVER = "hover:bg-[color:var(--status-running-bg)]";
const LOG_ROW_H = "h-[1.375rem]";
const STREAM_SECTION = "bg-muted border-b border-border px-2";
const STICKY_PHASE = "sticky top-0 z-30 h-8 bg-muted";
const STICKY_NODE = cn("sticky top-8 z-20", STREAM_SECTION);
const STICKY_STEP = cn("sticky top-[3.375rem] z-10", STREAM_SECTION);

const LAST_RUNNING_LINE_PULSE = "data-[last-running-line]:animate-pulse";

const STREAM_LINE_ROW = cn(
  "flex w-full min-w-0 max-w-full items-center justify-start overflow-hidden whitespace-nowrap px-2 text-left",
  LOG_ROW_H,
  LOG_ROW_HOVER,
  LAST_RUNNING_LINE_PULSE,
);

const STREAM_LINE_WRAP_ROW = cn(
  "flex w-full min-w-0 max-w-full items-start justify-start px-2 text-left",
  LOG_ROW_HOVER,
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
  | { kind: "tools"; id: string; tools: SplitRunStreamLine[] };

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

export function PhaseLogCard({
  phase,
  expanded,
  stream,
  selectedNodeId,
  onToggle,
  onSelectNode,
  onStop,
  onRerun,
  onEdit,
  actionBusy = false,
  collapsible = true,
  organizationId,
  canvasId,
}: {
  phase: SplitRunPhase;
  expanded: boolean;
  stream?: SplitRunStreamLine[];
  selectedNodeId?: string | null;
  onToggle?: () => void;
  onSelectNode?: (nodeId: string) => void;
  onStop?: () => void;
  onRerun?: () => void;
  onEdit?: () => void;
  actionBusy?: boolean;
  collapsible?: boolean;
  organizationId?: string;
  canvasId?: string;
}) {
  const groups = groupSplitRunStream(stream ?? phase.stream);
  const producedArtifacts = artifactsProducedBySteps(groups, phase.artifacts);
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

  return (
    <div
      ref={rootRef}
      className="min-w-0"
      data-testid={`split-run-phase-${phase.id}`}
      aria-current={expanded ? "step" : undefined}
    >
      <div
        className={cn(
          "rounded-md border border-border border-l-2 bg-card",
          expanded ? "pb-1.5" : "py-1.5",
          automationAccent(phase.status),
        )}
      >
        <AutomationHeader
          phase={phase}
          expanded={expanded}
          collapsible={collapsible}
          producedArtifacts={producedArtifacts}
          onToggle={onToggle}
          onStop={onStop}
          onRerun={onRerun}
          onEdit={onEdit}
          actionBusy={actionBusy}
        />

        {expanded ? (
          <ol
            className={cn("mt-1 min-w-0 list-none leading-tight", LOG_FACE)}
            data-testid={`split-run-stream-${phase.id}`}
          >
            {groups.map((group) => (
              <StreamNode
                key={group.line.id}
                group={group}
                highlighted={Boolean(group.line.nodeId && group.line.nodeId === selectedNodeId)}
                onSelect={onSelectNode}
                organizationId={organizationId}
                canvasId={canvasId ?? phase.appId}
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
  onToggle,
  onStop,
  onRerun,
  onEdit,
  actionBusy,
}: {
  phase: SplitRunPhase;
  expanded: boolean;
  collapsible: boolean;
  producedArtifacts: FactoriesWorkOrderArtifact[];
  onToggle?: () => void;
  onStop?: () => void;
  onRerun?: () => void;
  onEdit?: () => void;
  actionBusy: boolean;
}) {
  return (
    <div
      data-testid={`split-run-automation-header-${phase.id}`}
      {...streamLineAttrs(phase.status)}
      className={cn(
        "flex w-full min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap px-2 leading-tight",
        LAST_RUNNING_LINE_PULSE,
        expanded && STICKY_PHASE,
      )}
    >
      {collapsible ? (
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          className="flex min-w-0 flex-1 items-center gap-1.5 text-left text-[13px] font-medium tracking-[-0.01em]"
        >
          <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3.5" />
          <span className="min-w-0 truncate text-foreground">{phase.name}</span>
        </button>
      ) : (
        <div className="flex min-w-0 flex-1 items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em]">
          <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3.5" />
          <span className="min-w-0 truncate text-foreground">{phase.name}</span>
        </div>
      )}
      {onEdit && expanded ? <PhaseEditButton phase={phase} onEdit={onEdit} /> : null}
      {producedArtifacts.length > 0 ? (
        <span
          data-testid={`split-run-phase-artifacts-${phase.id}`}
          className="flex min-w-0 items-center justify-end gap-2 overflow-hidden whitespace-nowrap"
        >
          {producedArtifacts.map((artifact) => (
            <StreamArtifact key={artifact.id ?? `${artifact.type}`} artifact={artifact} />
          ))}
        </span>
      ) : null}
      {phase.checks && phase.checks.length > 0 ? (
        <span className="shrink-0">
          <SplitRunCheckPills checks={phase.checks} testId={`split-run-phase-checks-${phase.id}`} />
        </span>
      ) : null}
      {phase.status === "running" && onStop ? (
        <Button
          type="button"
          size="xs"
          variant="outline"
          className="text-destructive"
          disabled={actionBusy}
          onClick={onStop}
        >
          {actionBusy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
          Stop
        </Button>
      ) : null}
      {phase.status === "failed" && onRerun ? (
        <Button type="button" size="xs" variant="ghost" disabled={actionBusy} onClick={onRerun}>
          {actionBusy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
          Rerun
        </Button>
      ) : null}
    </div>
  );
}

function automationAccent(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "border-l-[#10b981]";
  if (status === "running") return "border-l-[#3b82f6]";
  if (status === "failed") return "border-l-[#ef4444]";
  if (status === "waiting") return "border-l-[#f59e0b]";
  return "border-l-border";
}

/** Opens the automation editor for an expanded phase. */
function PhaseEditButton({ phase, onEdit }: { phase: SplitRunPhase; onEdit: () => void }) {
  return (
    <button
      type="button"
      data-testid={`split-run-phase-edit-${phase.id}`}
      aria-label={`Edit ${phase.name} automation`}
      onClick={onEdit}
      className="inline-flex shrink-0 items-center gap-1 rounded-full border border-border bg-card px-1.5 py-0.5 font-sans text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
    >
      <Pencil className="size-2.5" aria-hidden />
      Edit
    </button>
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
  const label = running
    ? `Running ${tickingRunningClock(duration, sampledAt, now)}`
    : logStatusTimeLabel(status, duration);
  if (!label) {
    return null;
  }
  return (
    <span
      data-testid={testId}
      aria-label={running ? "Running" : undefined}
      className={cn(
        "ml-auto shrink-0 rounded-sm px-1.5 text-right text-[12px] leading-[1.125rem] tabular-nums [font-feature-settings:'zero']",
        statusTimeTone(status),
      )}
    >
      {running ? (
        <span data-testid="split-run-running-spinner" className="mr-1 inline-block w-[1ch]" aria-hidden>
          {runningSpinnerFrame(now - sampledAt)}
        </span>
      ) : null}
      {label}
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
}: {
  group: StreamNodeGroup;
  highlighted: boolean;
  onSelect?: (nodeId: string) => void;
  organizationId?: string;
  canvasId?: string;
}) {
  const { line, notes, artifact } = group;
  const liveNotes = useRunnerNodeLiveNotes(line, organizationId, canvasId);
  const steps = groupClaudeSteps(liveNotes ?? notes);
  const hasChildren = steps.length > 0 || isRunnerComponent(line.component);

  return (
    <li className="min-w-0">
      <StreamNodeHeader
        line={line}
        hasChildren={hasChildren}
        highlighted={highlighted}
        artifact={artifact}
        onSelect={onSelect}
      />
      {hasChildren ? (
        <ol className="min-w-0">
          {steps.map((step) => (
            <StreamStep key={step.line.id} step={step} />
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
  onSelect,
}: {
  line: SplitRunStreamLine;
  hasChildren: boolean;
  highlighted: boolean;
  artifact?: FactoriesWorkOrderArtifact;
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
        LOG_ROW_HOVER,
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
      {artifact ? <StreamArtifact artifact={artifact} /> : null}
      <StreamDuration line={line} />
    </div>
  );
}

function StreamStep({ step }: { step: ClaudeStepGroup }) {
  const hasOutput = Boolean(step.line.detail);
  const hasBody = step.events.length > 0 || hasOutput;

  return (
    <li className="min-w-0">
      <div
        data-testid={`split-run-stream-line-${step.line.id}`}
        {...streamLineAttrs(step.line.status)}
        className={cn(STREAM_LINE_WRAP_ROW, STREAM_SECTION, hasBody && STICKY_STEP)}
      >
        {step.line.componentType ? (
          <span className={cn("mr-2 shrink-0", stepTypeTone(step.line.componentType))}>{step.line.componentType}</span>
        ) : null}
        <StreamLineTitle wrap>{step.line.componentName}</StreamLineTitle>
        <StepStatusMark status={step.line.status} />
      </div>
      {hasBody ? (
        <>
          {hasOutput ? <StreamOutput text={step.line.detail ?? ""} /> : null}
          {step.events.map((event) =>
            event.kind === "note" ? (
              <div
                key={event.line.id}
                data-testid={`split-run-stream-line-${event.line.id}`}
                {...streamLineAttrs(event.line.status)}
                className={cn("flex w-full items-start px-2", LOG_ROW_HOVER, LAST_RUNNING_LINE_PULSE)}
              >
                <span className="inline-flex w-4 shrink-0" aria-hidden />
                <span className="min-w-0 flex-1 whitespace-normal break-words py-0.5 leading-5 text-foreground">
                  {event.line.componentName}
                </span>
              </div>
            ) : (
              <StreamToolGroup key={event.id} stepId={event.id} tools={event.tools} />
            ),
          )}
        </>
      ) : null}
    </li>
  );
}

function StreamToolGroup({ stepId, tools }: { stepId: string; tools: SplitRunStreamLine[] }) {
  const [expanded, setExpanded] = useState(false);
  const summary = toolCallSummary(tools);

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
          className={cn("flex min-w-0 w-full items-start px-2", LOG_FACE, LOG_ROW_HOVER, LAST_RUNNING_LINE_PULSE)}
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
  const { sections, error, isStreaming } = useLiveLogStream(
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
    error,
    isStreaming,
    nodeStatus: line.status,
  });
}
