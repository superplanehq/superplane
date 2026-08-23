import { cn, resolveIcon } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useEffect } from "react";

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
    .filter((line) => !line.note)
    .map((line) => ({
      line,
      notes: notesByNode.get(line.nodeId ?? "") ?? [],
      artifact: line.artifact,
    }));
}

/**
 * Terminal log. The automation is the root. Each node hangs under it.
 * Notes and artifacts hang under their node. Connectors use ├ and └.
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
      <div className="flex items-start gap-1.5">
        {collapsible ? (
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={expanded}
            className="flex min-w-0 flex-1 items-start gap-1.5 py-0.5 text-left font-mono text-[12px] leading-5"
          >
            <ChevronRight
              className={cn(
                "mt-0.5 size-3 shrink-0 text-muted-foreground transition-transform",
                expanded && "rotate-90",
              )}
              aria-hidden
            />
            <PhaseGlyph kind={statusGlyph(phase.status)} className="mt-0.5 size-3" />
            <span className="min-w-0 flex-1 whitespace-nowrap">
              <span className="text-foreground">{phase.name}</span>
              <span className="text-muted-foreground">{` > ${phase.componentName}`}</span>
            </span>
            <span className={cn("shrink-0", streamTone(phase.status))}>{splitRunStatusLabel(phase.status)}</span>
            <span className="w-16 shrink-0 text-right tabular-nums text-muted-foreground">{phase.duration}</span>
          </button>
        ) : (
          <div className="flex min-w-0 flex-1 items-start gap-1.5 py-0.5 font-mono text-[12px] leading-5">
            <PhaseGlyph kind={statusGlyph(phase.status)} className="mt-0.5 size-3" />
            <span className="min-w-0 flex-1 whitespace-nowrap">
              <span className="text-foreground">{phase.name}</span>
              <span className="text-muted-foreground">{` > ${phase.componentName}`}</span>
            </span>
            <span className={cn("shrink-0", streamTone(phase.status))}>{splitRunStatusLabel(phase.status)}</span>
            <span className="w-16 shrink-0 text-right tabular-nums text-muted-foreground">{phase.duration}</span>
          </div>
        )}
      </div>

      {!expanded && phase.artifacts.length > 0 ? (
        <ul className="mt-0.5 ml-8 flex flex-wrap gap-x-3 gap-y-0.5">
          {phase.artifacts.map((artifact) => (
            <li key={artifact.id ?? `${artifact.type}`}>
              <WorkOrderArtifactInline
                className="font-mono text-[12px] font-normal tracking-normal"
                artifact={{
                  id: artifact.id,
                  type: artifact.type ?? "TYPE_UNSPECIFIED",
                  data: toArtifactDataRecord(artifact.data),
                }}
              />
            </li>
          ))}
        </ul>
      ) : null}

      {expanded ? (
        <ol className="mt-0.5 mb-1 font-mono text-[12px] leading-5" data-testid={`split-run-stream-${phase.id}`}>
          {groups.map((group, index) => (
            <StreamNode
              key={group.line.id}
              group={group}
              isLast={index === groups.length - 1}
              highlighted={Boolean(group.line.nodeId && group.line.nodeId === selectedNodeId)}
              onSelect={onSelectNode}
            />
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function StreamNode({
  group,
  isLast,
  highlighted,
  onSelect,
}: {
  group: StreamNodeGroup;
  isLast: boolean;
  highlighted: boolean;
  onSelect?: (nodeId: string) => void;
}) {
  const { line, notes, artifact } = group;
  const action = streamActionOf(line);

  return (
    <li>
      <div
        data-testid={`split-run-stream-line-${line.id}`}
        data-highlighted={highlighted ? "true" : undefined}
        aria-current={highlighted ? "true" : undefined}
        className={cn(
          "flex w-full items-center whitespace-nowrap rounded-sm py-0.5",
          highlighted && "bg-accent ring-1 ring-foreground/15",
        )}
      >
        <TreePrefix isLast={isLast} />
        <button
          type="button"
          onClick={() => {
            if (line.nodeId) {
              onSelect?.(line.nodeId);
            }
          }}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-2 text-left",
            line.nodeId && "cursor-pointer hover:text-foreground",
          )}
        >
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
        {artifact ? (
          <WorkOrderArtifactInline
            className="font-mono text-[12px] font-normal tracking-normal"
            artifact={{
              id: artifact.id,
              type: artifact.type ?? "TYPE_UNSPECIFIED",
              data: toArtifactDataRecord(artifact.data),
            }}
          />
        ) : null}
      </div>
      {notes.length > 0 ? (
        <ol>
          {notes.map((note, noteIndex) => (
            <li
              key={note.id}
              data-testid={`split-run-stream-line-${note.id}`}
              className="flex w-full items-center whitespace-nowrap py-0.5"
            >
              <TreePrefix isLast={noteIndex === notes.length - 1} parentLast={isLast} depth={2} />
              <span className="min-w-0 truncate text-muted-foreground">{note.componentName}</span>
            </li>
          ))}
        </ol>
      ) : null}
    </li>
  );
}

function TreePrefix({ isLast, parentLast, depth = 1 }: { isLast: boolean; parentLast?: boolean; depth?: 1 | 2 }) {
  const elbow = isLast ? "└─ " : "├─ ";
  const text = depth === 1 ? elbow : `${parentLast ? "   " : "│  "}${elbow}`;
  return (
    <span className="shrink-0 text-muted-foreground" aria-hidden>
      {text}
    </span>
  );
}

function StreamLineIcon({ iconSlug, iconSrc }: { iconSlug?: string; iconSrc?: string }) {
  const Icon = resolveIcon(iconSlug ?? "box");
  return (
    <span className="inline-flex size-3.5 shrink-0 items-center justify-center text-muted-foreground" aria-hidden>
      {iconSrc ? <img src={iconSrc} alt="" className="size-3.5 object-contain" /> : <Icon className="size-3.5" />}
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
