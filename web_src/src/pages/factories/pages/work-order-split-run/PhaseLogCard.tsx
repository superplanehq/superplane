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
 * Node lines indent under the automation. Note lines use ├ and └.
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
        {artifact ? <StreamArtifact artifact={artifact} /> : null}
      </div>
      {notes.length > 0 ? (
        <ol>
          {notes.map((note, noteIndex) => (
            <li
              key={note.id}
              data-testid={`split-run-stream-line-${note.id}`}
              className="flex h-4 w-full items-center whitespace-nowrap"
            >
              <NotePrefix isLast={noteIndex === notes.length - 1} />
              <span className="min-w-0 truncate text-muted-foreground">{note.componentName}</span>
            </li>
          ))}
        </ol>
      ) : null}
    </li>
  );
}

function NodeIndent() {
  return (
    <span data-testid="split-run-node-indent" className="inline-block w-[4ch] shrink-0 whitespace-pre" aria-hidden>
      {"    "}
    </span>
  );
}

function NotePrefix({ isLast }: { isLast: boolean }) {
  return (
    <span className="inline-block w-[8ch] shrink-0 whitespace-pre text-muted-foreground" aria-hidden>
      {isLast ? "    └── " : "    ├── "}
    </span>
  );
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
