import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useEffect } from "react";

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

/**
 * Terminal log row. Completed phases stay collapsed. The open phase
 * lists one line per canvas node.
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
  const lines = stream ?? phase.stream;

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
            <span className="min-w-0 flex-1">
              <span className="text-foreground">{phase.name}</span>
              <span className="text-muted-foreground"> · {phase.componentName}</span>
            </span>
            <span className={cn("shrink-0", streamTone(phase.status))}>{splitRunStatusLabel(phase.status)}</span>
            <span className="w-16 shrink-0 text-right tabular-nums text-muted-foreground">{phase.duration}</span>
          </button>
        ) : (
          <div className="flex min-w-0 flex-1 items-start gap-1.5 py-0.5 font-mono text-[12px] leading-5">
            <PhaseGlyph kind={statusGlyph(phase.status)} className="mt-0.5 size-3" />
            <span className="min-w-0 flex-1">
              <span className="text-foreground">{phase.name}</span>
              <span className="text-muted-foreground"> · {phase.componentName}</span>
            </span>
            <span className={cn("shrink-0", streamTone(phase.status))}>{splitRunStatusLabel(phase.status)}</span>
            <span className="w-16 shrink-0 text-right tabular-nums text-muted-foreground">{phase.duration}</span>
          </div>
        )}
      </div>

      {phase.artifacts.length > 0 ? (
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
        <ol className="mt-1 mb-1 ml-8 space-y-0.5" data-testid={`split-run-stream-${phase.id}`}>
          {lines.map((line) => (
            <StreamLine
              key={line.id}
              line={line}
              highlighted={Boolean(line.nodeId && line.nodeId === selectedNodeId)}
              onSelect={onSelectNode}
            />
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function StreamLine({
  line,
  highlighted,
  onSelect,
}: {
  line: SplitRunStreamLine;
  highlighted: boolean;
  onSelect?: (nodeId: string) => void;
}) {
  const passed = !line.note && line.status === "passed";
  const failed = !line.note && line.status === "failed";

  return (
    <li>
      <div
        data-testid={`split-run-stream-line-${line.id}`}
        data-highlighted={highlighted ? "true" : undefined}
        aria-current={highlighted ? "true" : undefined}
        className={cn(
          "flex w-full items-baseline gap-2 rounded-sm px-1 py-0.5 font-mono text-[12px] leading-5",
          highlighted && "bg-accent ring-1 ring-foreground/15",
        )}
      >
        <button
          type="button"
          onClick={() => {
            if (line.nodeId) {
              onSelect?.(line.nodeId);
            }
          }}
          className="flex min-w-0 flex-1 items-baseline gap-2 text-left"
        >
          <span className="w-14 shrink-0 tabular-nums text-muted-foreground">{line.at}</span>
          <span
            className={cn(
              "min-w-0 flex-1 truncate",
              line.note || line.status === "pending" ? "text-muted-foreground" : "text-foreground",
            )}
          >
            {line.componentName}
          </span>
        </button>
        {line.artifact ? (
          <WorkOrderArtifactInline
            className="font-mono text-[12px] font-normal tracking-normal"
            artifact={{
              id: line.artifact.id,
              type: line.artifact.type ?? "TYPE_UNSPECIFIED",
              data: toArtifactDataRecord(line.artifact.data),
            }}
          />
        ) : null}
        {passed ? (
          <span className="w-4 shrink-0 text-right text-[color:var(--status-success)]" aria-label="Completed">
            ✓
          </span>
        ) : failed ? (
          <span className="w-4 shrink-0 text-right text-[color:var(--status-failed-fg)]" aria-label="Failed">
            ×
          </span>
        ) : (
          <span className="w-4 shrink-0" aria-hidden />
        )}
      </div>
    </li>
  );
}
