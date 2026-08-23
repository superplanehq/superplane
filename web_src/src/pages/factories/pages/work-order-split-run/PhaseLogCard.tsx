import { cn, resolveIcon } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";

import type { FactoriesWorkOrderArtifact } from "@/api-client";

import type { PhaseGlyphKind } from "../../lib/linePhaseRuns";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../../WorkOrderArtifactInline";
import { PhaseGlyph } from "../linePhaseGlyph";
import {
  splitRunStatusLabel,
  type SplitRunPhase,
  type SplitRunPhaseStatus,
  type SplitRunStreamLine,
} from "./splitRunMocks";

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
      flushTools(parent);
      parent.events.push({ kind: "note", line });
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
  for (const group of groups) {
    const artifact = group.artifact;
    if (!artifact) {
      continue;
    }
    const key = artifact.id ?? `${artifact.type}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    produced.push(artifact);
  }
  return produced.length > 0 ? produced : fallback;
}

/**
 * Terminal log. The automation is the root. Each node hangs under it.
 * Collapsed automations show produced artifacts on the title line.
 * Expanded automations show those artifacts on the producing steps.
 * Node lines indent under the automation.
 */
export function PhaseLogCard({
  phase,
  expanded,
  stream,
  selectedNodeId,
  onToggle,
  onSelectNode,
  collapsible = true,
}: {
  phase: SplitRunPhase;
  expanded: boolean;
  stream?: SplitRunStreamLine[];
  selectedNodeId?: string | null;
  onToggle?: () => void;
  onSelectNode?: (nodeId: string) => void;
  collapsible?: boolean;
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
    <div data-testid={`split-run-phase-${phase.id}`} aria-current={expanded ? "step" : undefined}>
      <div className="flex min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap">
        {collapsible ? (
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={expanded}
            className="flex min-w-0 items-center gap-1.5 text-left font-mono text-[12px] leading-none"
          >
            <ChevronRight
              className={cn("size-3 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-90")}
              aria-hidden
            />
            <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3" />
            <PhaseTitle phase={phase} />
          </button>
        ) : (
          <div className="flex min-w-0 items-center gap-1.5 font-mono text-[12px] leading-none">
            <PhaseGlyph kind={statusGlyph(phase.status)} className="size-3" />
            <PhaseTitle phase={phase} />
          </div>
        )}
        {!expanded && producedArtifacts.length > 0 ? (
          <span
            data-testid={`split-run-phase-artifacts-${phase.id}`}
            className="flex min-w-0 items-center gap-2 overflow-hidden whitespace-nowrap"
          >
            {producedArtifacts.map((artifact) => (
              <StreamArtifact key={artifact.id ?? `${artifact.type}`} artifact={artifact} />
            ))}
          </span>
        ) : null}
      </div>

      {expanded ? (
        <ol className="mt-0.5 mb-1 font-mono text-[12px] leading-none" data-testid={`split-run-stream-${phase.id}`}>
          {groups.map((group) => (
            <StreamNode
              key={group.line.id}
              group={group}
              highlighted={Boolean(group.line.nodeId && group.line.nodeId === selectedNodeId)}
              onSelect={onSelectNode}
            />
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function PhaseTitle({ phase }: { phase: SplitRunPhase }) {
  const duration = phase.duration.replace(/\s+so far$/i, "").trim() || "—";
  return (
    <span className="min-w-0 flex-1 whitespace-nowrap">
      <span className="text-foreground">{phase.name}</span>
      <span className="text-muted-foreground">{` > ${phase.componentName}`}</span>
      <span className={cn("ml-2", streamTone(phase.status))}>{splitRunStatusLabel(phase.status)}</span>
      <span className="ml-2 tabular-nums text-muted-foreground">{duration}</span>
    </span>
  );
}

function StreamNode({
  group,
  highlighted,
  onSelect,
}: {
  group: StreamNodeGroup;
  highlighted: boolean;
  onSelect?: (nodeId: string) => void;
}) {
  const { line, notes, artifact } = group;
  const action = streamActionOf(line);
  const steps = groupClaudeSteps(notes);
  const [expanded, setExpanded] = useState(line.status === "running");
  const hasChildren = steps.length > 0;

  return (
    <li>
      <div
        data-testid={`split-run-stream-line-${line.id}`}
        data-highlighted={highlighted ? "true" : undefined}
        aria-current={highlighted ? "true" : undefined}
        className={cn(
          "flex h-4 w-full items-center whitespace-nowrap rounded-sm",
          highlighted && "bg-accent ring-1 ring-foreground/15",
        )}
      >
        <NodeIndent />
        <button
          type="button"
          data-testid={`split-run-node-toggle-${line.id}`}
          aria-expanded={hasChildren ? expanded : undefined}
          onClick={() => {
            if (hasChildren) {
              setExpanded((open) => !open);
            }
            if (line.nodeId) {
              onSelect?.(line.nodeId);
            }
          }}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-2 text-left",
            (hasChildren || line.nodeId) && "cursor-pointer hover:text-foreground",
          )}
        >
          <span className="inline-flex w-3 shrink-0 items-center justify-center text-muted-foreground">
            {hasChildren ? (
              <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
            ) : null}
          </span>
          <span className="w-14 shrink-0 tabular-nums text-muted-foreground">{line.at}</span>
          <StreamLineIcon iconSlug={line.iconSlug} iconSrc={line.iconSrc} />
          {line.componentType ? <span className="shrink-0 text-muted-foreground">{line.componentType}</span> : null}
          <span
            className={cn("min-w-0 truncate", line.status === "pending" ? "text-muted-foreground" : "text-foreground")}
          >
            {line.componentName}
          </span>
          <span className="shrink-0 text-muted-foreground" aria-hidden>
            {">"}
          </span>
          <span className={cn("shrink-0", streamTone(line.status))}>{action}</span>
        </button>
        {artifact ? <StreamArtifact artifact={artifact} /> : null}
      </div>
      {expanded ? (
        <ol>
          {steps.map((step) => (
            <StreamStep key={step.line.id} step={step} />
          ))}
        </ol>
      ) : null}
    </li>
  );
}

function StreamStep({ step }: { step: ClaudeStepGroup }) {
  const [expanded, setExpanded] = useState(false);
  const hasBody = step.events.length > 0;

  const header = (
    <>
      <StreamIndent ch={8} />
      {hasBody ? (
        <ChevronRight
          className={cn("mr-1 size-3 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-90")}
          aria-hidden
        />
      ) : null}
      {step.line.componentType ? (
        <span className={cn("mr-2 shrink-0", stepTypeTone(step.line.componentType))}>{step.line.componentType}</span>
      ) : null}
      <span className="min-w-0 truncate text-muted-foreground">{step.line.componentName}</span>
    </>
  );

  return (
    <li>
      {hasBody ? (
        <button
          type="button"
          data-testid={`split-run-stream-line-${step.line.id}`}
          aria-expanded={expanded}
          onClick={() => setExpanded((open) => !open)}
          className="flex h-4 w-full cursor-pointer items-center whitespace-nowrap text-left hover:text-foreground"
        >
          {header}
        </button>
      ) : (
        <div
          data-testid={`split-run-stream-line-${step.line.id}`}
          className="flex h-4 w-full items-center whitespace-nowrap"
        >
          {header}
        </div>
      )}
      {expanded
        ? step.events.map((event) =>
            event.kind === "note" ? (
              <div
                key={event.line.id}
                data-testid={`split-run-stream-line-${event.line.id}`}
                className="flex w-full items-start"
              >
                <StreamIndent ch={12} />
                <span className="min-w-0 flex-1 whitespace-normal break-words py-0.5 leading-4 text-foreground">
                  {event.line.componentName}
                </span>
              </div>
            ) : (
              <StreamToolGroup key={event.id} stepId={event.id} tools={event.tools} />
            ),
          )
        : null}
    </li>
  );
}

function StreamToolGroup({ stepId, tools }: { stepId: string; tools: SplitRunStreamLine[] }) {
  const [expanded, setExpanded] = useState(false);
  const summary = toolCallSummary(tools);

  return (
    <div>
      <button
        type="button"
        data-testid={`split-run-tools-toggle-${stepId}`}
        aria-expanded={expanded}
        aria-label={summary}
        onClick={() => setExpanded((open) => !open)}
        className="flex h-4 w-full items-center whitespace-nowrap text-muted-foreground"
      >
        <StreamIndent ch={12} />
        <ChevronRight className={cn("mr-1 size-3 transition-transform", expanded && "rotate-90")} aria-hidden />
        <span className="min-w-0 truncate">{summary}</span>
      </button>
      {expanded ? (
        <ol>
          {tools.map((tool) => (
            <li
              key={tool.id}
              data-testid={`split-run-stream-line-${tool.id}`}
              className="flex h-4 w-full items-center whitespace-nowrap"
            >
              <StreamIndent ch={12} />
              {tool.componentType ? (
                <span className={cn("mr-2 shrink-0", stepTypeTone(tool.componentType))}>{tool.componentType}</span>
              ) : null}
              <span className="min-w-0 truncate text-muted-foreground">{tool.componentName}</span>
            </li>
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function NodeIndent() {
  return <StreamIndent ch={4} testId="split-run-node-indent" />;
}

function StreamIndent({ ch, testId }: { ch: number; testId?: string }) {
  return (
    <span
      data-testid={testId}
      className="inline-block shrink-0 whitespace-pre"
      style={{ width: `${ch}ch` }}
      aria-hidden
    >
      {" ".repeat(ch)}
    </span>
  );
}

function stepTypeTone(type: string): string {
  if (type === "prompt") {
    return "text-[color:var(--status-running)]";
  }
  if (type === "bash") {
    return "text-[color:var(--status-success)]";
  }
  return "text-muted-foreground";
}

function StreamArtifact({ artifact }: { artifact: FactoriesWorkOrderArtifact }) {
  return (
    <WorkOrderArtifactInline
      className="font-mono text-[12px] font-normal tracking-normal"
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
