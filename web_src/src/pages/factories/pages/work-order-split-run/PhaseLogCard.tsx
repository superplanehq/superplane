import { formatClockDurationLabel } from "@/lib/duration";
import { cn, resolveIcon } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";

import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";

import type { PhaseGlyphKind } from "../../lib/linePhaseRuns";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { PhaseGlyph } from "../linePhaseGlyph";
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

function streamTone(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "text-[color:var(--status-success)]";
  if (status === "running") return "text-[color:var(--status-running)]";
  if (status === "waiting") return "text-[color:var(--status-waiting-fg)]";
  if (status === "failed") return "text-[color:var(--status-failed-fg)]";
  return "text-muted-foreground";
}

const STREAM_LINE_ROW =
  "flex h-[1.375rem] w-full min-w-0 max-w-full items-center justify-start overflow-hidden whitespace-nowrap text-left";

function StreamLineTitle({ children }: { children: string }) {
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
 * Collapsed automations show produced artifacts on the title line.
 * Expanded automations show those artifacts on the producing steps.
 * Node icons keep the phase glyph column. Agent steps indent under them.
 */
export function PhaseLogCard({
  phase,
  expanded,
  stream,
  selectedNodeId,
  onToggle,
  onSelectNode,
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
  collapsible?: boolean;
  organizationId?: string;
  canvasId?: string;
}) {
  const groups = groupSplitRunStream(stream ?? phase.stream);
  const producedArtifacts = artifactsProducedBySteps(groups, phase.artifacts);

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
    <div className="min-w-0" data-testid={`split-run-phase-${phase.id}`} aria-current={expanded ? "step" : undefined}>
      <div
        className={cn(
          "flex w-full min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap leading-tight",
          LOG_FACE,
        )}
      >
        {collapsible ? (
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={expanded}
            className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
          >
            <ChevronRight
              className={cn("size-3 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-90")}
              aria-hidden
            />
            <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3" />
            <span className="min-w-0 truncate text-foreground">{phase.name}</span>
          </button>
        ) : (
          <div className="flex min-w-0 flex-1 items-center gap-1.5">
            <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3" />
            <span className="min-w-0 truncate text-foreground">{phase.name}</span>
          </div>
        )}
        {phase.checks && phase.checks.length > 0 ? (
          <span className="shrink-0">
            <SplitRunCheckPills checks={phase.checks} testId={`split-run-phase-checks-${phase.id}`} />
          </span>
        ) : null}
        {!expanded && producedArtifacts.length > 0 ? (
          <span
            data-testid={`split-run-phase-artifacts-${phase.id}`}
            className="flex min-w-0 items-center justify-end gap-2 overflow-hidden whitespace-nowrap"
          >
            {producedArtifacts.map((artifact) => (
              <StreamArtifact key={artifact.id ?? `${artifact.type}`} artifact={artifact} />
            ))}
          </span>
        ) : null}
        <PhaseDuration phase={phase} />
      </div>

      {expanded ? (
        <ol
          className={cn("mt-0.5 mb-1 min-w-0 overflow-hidden leading-tight", LOG_FACE)}
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
  );
}

function PhaseDuration({ phase }: { phase: SplitRunPhase }) {
  const duration = formatClockDurationLabel(phase.duration);
  return (
    <span
      data-testid={`split-run-phase-duration-${phase.id}`}
      className="ml-auto min-w-[5ch] shrink-0 text-right tabular-nums [font-feature-settings:'zero'] text-muted-foreground"
    >
      {duration}
    </span>
  );
}

function StreamDuration({ line }: { line: SplitRunStreamLine }) {
  const duration = line.duration ? formatClockDurationLabel(line.duration) : "";
  if (!duration || duration === "—") {
    return null;
  }
  return (
    <span
      data-testid={`split-run-stream-duration-${line.id}`}
      className="ml-auto min-w-[5ch] shrink-0 text-right tabular-nums [font-feature-settings:'zero'] text-muted-foreground"
    >
      {duration}
    </span>
  );
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
  const action = streamActionOf(line);
  const [expanded, setExpanded] = useState(line.status === "running" || highlighted);
  const liveNotes = useRunnerNodeLiveNotes(line, expanded, organizationId, canvasId);
  const steps = groupClaudeSteps(liveNotes ?? notes);
  const hasChildren = steps.length > 0 || isRunnerComponent(line.component);

  useEffect(() => {
    if (highlighted && hasChildren) {
      setExpanded(true);
    }
  }, [highlighted, hasChildren]);

  return (
    <li className="min-w-0">
      <StreamNodeHeader
        line={line}
        expanded={expanded}
        hasChildren={hasChildren}
        highlighted={highlighted}
        action={action}
        artifact={artifact}
        onClick={() => toggleStreamNode(hasChildren, line.nodeId, setExpanded, onSelect)}
      />
      {expanded ? (
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
  expanded,
  hasChildren,
  highlighted,
  action,
  artifact,
  onClick,
}: {
  line: SplitRunStreamLine;
  expanded: boolean;
  hasChildren: boolean;
  highlighted: boolean;
  action: string;
  artifact?: FactoriesWorkOrderArtifact;
  onClick: () => void;
}) {
  return (
    <div
      data-testid={`split-run-stream-line-${line.id}`}
      data-highlighted={highlighted ? "true" : undefined}
      aria-current={highlighted ? "true" : undefined}
      className={cn(
        "flex h-[1.375rem] w-full min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap rounded-sm",
        highlighted && "bg-accent ring-1 ring-foreground/15",
      )}
    >
      <button
        type="button"
        data-testid={`split-run-node-toggle-${line.id}`}
        aria-expanded={hasChildren ? expanded : undefined}
        onClick={onClick}
        className={cn(
          "flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden text-left",
          (hasChildren || line.nodeId) && "cursor-pointer hover:text-foreground",
        )}
      >
        <span className="inline-flex w-3 shrink-0 items-center justify-center text-muted-foreground">
          {hasChildren ? (
            <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
          ) : null}
        </span>
        <StreamLineIcon iconSlug={line.iconSlug} iconSrc={line.iconSrc} />
        <span
          className={cn(
            "min-w-0 overflow-hidden truncate",
            line.status === "pending" ? "text-muted-foreground" : "text-foreground",
          )}
        >
          {line.componentName}
        </span>
        <span className="shrink-0 text-muted-foreground" aria-hidden>
          {">"}
        </span>
        <span className={cn("shrink-0", streamTone(line.status))}>{action}</span>
      </button>
      {artifact ? <StreamArtifact artifact={artifact} /> : null}
      <StreamDuration line={line} />
    </div>
  );
}

function StreamStep({ step }: { step: ClaudeStepGroup }) {
  const [expanded, setExpanded] = useState(step.line.status === "running");
  useEffect(() => {
    if (step.line.status === "running") {
      setExpanded(true);
    }
  }, [step.line.status]);
  const hasOutput = Boolean(step.line.detail);
  const hasBody = step.events.length > 0 || hasOutput;

  const header = (
    <>
      <StreamIndent ch={8} />
      <ExpandChevron expanded={expanded} visible={hasBody} />
      {step.line.componentType ? (
        <span className={cn("mr-2 shrink-0", stepTypeTone(step.line.componentType))}>{step.line.componentType}</span>
      ) : null}
      <StreamLineTitle>{step.line.componentName}</StreamLineTitle>
      <StepStatusMark status={step.line.status} />
    </>
  );

  return (
    <li className="min-w-0">
      {hasBody ? (
        <button
          type="button"
          data-testid={`split-run-stream-line-${step.line.id}`}
          aria-expanded={expanded}
          onClick={() => setExpanded((open) => !open)}
          className={cn(STREAM_LINE_ROW, "cursor-pointer hover:text-foreground")}
        >
          {header}
        </button>
      ) : (
        <div data-testid={`split-run-stream-line-${step.line.id}`} className={STREAM_LINE_ROW}>
          {header}
        </div>
      )}
      {expanded ? (
        <>
          {hasOutput ? <StreamOutput text={step.line.detail ?? ""} /> : null}
          {step.events.map((event) =>
            event.kind === "note" ? (
              <div
                key={event.line.id}
                data-testid={`split-run-stream-line-${event.line.id}`}
                className="flex w-full items-start"
              >
                <StreamIndent ch={12} />
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
  const streaming = tools.some((tool) => tool.status === "running");
  const [expanded, setExpanded] = useState(streaming);
  useEffect(() => {
    if (streaming) {
      setExpanded(true);
    }
  }, [streaming]);
  const summary = toolCallSummary(tools);

  return (
    <div className="min-w-0">
      <button
        type="button"
        data-testid={`split-run-tools-toggle-${stepId}`}
        aria-expanded={expanded}
        aria-label={summary}
        onClick={() => setExpanded((open) => !open)}
        className={cn(STREAM_LINE_ROW, "text-muted-foreground")}
      >
        <StreamIndent ch={12} />
        <ChevronRight className={cn("mr-1 size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
        <StreamLineTitle>{summary}</StreamLineTitle>
      </button>
      {expanded ? (
        <ol className="min-w-0 pl-2">
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
      <StreamIndent ch={12} />
      <ExpandChevron expanded={expanded} visible={hasOutput} />
      {tool.componentType ? (
        <span className={cn("mr-2 shrink-0", stepTypeTone(tool.componentType))}>{tool.componentType}</span>
      ) : null}
      <StreamLineTitle>{tool.componentName}</StreamLineTitle>
      <StepStatusMark status={tool.status} />
    </>
  );

  return (
    <li className="min-w-0">
      {hasOutput ? (
        <button
          type="button"
          data-testid={`split-run-stream-line-${tool.id}`}
          aria-expanded={expanded}
          onClick={() => setExpanded((open) => !open)}
          className={cn(STREAM_LINE_ROW, "cursor-pointer hover:text-foreground")}
        >
          {row}
        </button>
      ) : (
        <div data-testid={`split-run-stream-line-${tool.id}`} className={STREAM_LINE_ROW}>
          {row}
        </div>
      )}
      {expanded && hasOutput ? <StreamOutput indentCh={16} text={tool.detail ?? ""} /> : null}
    </li>
  );
}

function StreamOutput({ text, indentCh = 12 }: { text: string; indentCh?: number }) {
  return (
    <div className="flex w-full items-start" data-testid="split-run-stream-output">
      <StreamIndent ch={indentCh} testId="split-run-stream-output-indent" />
      <pre
        className={cn(
          "min-w-0 flex-1 whitespace-pre-wrap break-words py-0.5 leading-5 text-muted-foreground",
          LOG_FACE,
        )}
      >
        {text}
      </pre>
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

function StreamIndent({ ch, testId }: { ch: number; testId?: string }) {
  return (
    <span
      data-testid={testId}
      className="block shrink-0 overflow-hidden"
      style={{ width: `${ch}ch`, minWidth: `${ch}ch` }}
      aria-hidden
    />
  );
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
  expanded: boolean,
  organizationId?: string,
  canvasId?: string,
): SplitRunStreamLine[] | undefined {
  const canStream = Boolean(
    expanded && organizationId && canvasId && line.executionId && isRunnerComponent(line.component),
  );
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

function toggleStreamNode(
  hasChildren: boolean,
  nodeId: string | undefined,
  setExpanded: (update: (open: boolean) => boolean) => void,
  onSelect?: (nodeId: string) => void,
) {
  if (hasChildren) {
    setExpanded((open) => !open);
  }
  if (nodeId) {
    onSelect?.(nodeId);
  }
}

function streamActionOf(line: SplitRunStreamLine): string {
  if (line.action) {
    return line.action;
  }
  if (line.status === "passed") {
    return "passed";
  }
  if (line.status === "failed") {
    return "failed";
  }
  if (line.status === "running") {
    return "running";
  }
  if (line.status === "waiting") {
    return "waiting";
  }
  return "—";
}
