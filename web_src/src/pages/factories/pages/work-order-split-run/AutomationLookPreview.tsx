import { formatClockDurationLabel } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { CircleX, RotateCw } from "lucide-react";
import { useState } from "react";

import { PhaseGlyph } from "../linePhaseGlyph";
import { SectionTitle } from "../work-order-popup-redesign/popupShared";
import type { SplitRunPhaseStatus } from "./splitRunMocks";

export type AutomationLook = "bar" | "card" | "rule";

type PreviewPhase = {
  id: string;
  name: string;
  status: SplitRunPhaseStatus;
  duration: string;
  nodes: { name: string; status: SplitRunPhaseStatus }[];
};

const START: PreviewPhase[] = [
  {
    id: "plan",
    name: "Plan",
    status: "passed",
    duration: "4m",
    nodes: [{ name: "Write PLAN.md", status: "passed" }],
  },
  {
    id: "implement",
    name: "Implement",
    status: "running",
    duration: "1m 12s so far",
    nodes: [
      { name: "Claude Code", status: "running" },
      { name: "Run tests", status: "pending" },
    ],
  },
  {
    id: "verify",
    name: "Verify",
    status: "failed",
    duration: "41s",
    nodes: [{ name: "Create Draft Pull Request", status: "failed" }],
  },
];

function statusWord(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "Passed";
  if (status === "running") return "Running";
  if (status === "failed") return "Failed";
  if (status === "waiting") return "Waiting";
  return "";
}

function statusTone(status: SplitRunPhaseStatus): string {
  if (status === "passed") {
    return "bg-[color:var(--status-completed-bg)] text-[color:var(--status-completed-fg)]";
  }
  if (status === "running") {
    return "bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]";
  }
  if (status === "failed") {
    return "bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]";
  }
  return "text-muted-foreground";
}

function accent(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "border-l-[#10b981]";
  if (status === "running") return "border-l-[#3b82f6]";
  if (status === "failed") return "border-l-[#ef4444]";
  return "border-l-border";
}

/**
 * Storybook-only looks for automation rows. Production log rows stay unchanged
 * until one look is chosen.
 */
export function AutomationLookPreview({ look }: { look: AutomationLook }) {
  const [phases, setPhases] = useState(START);
  const [openId, setOpenId] = useState("implement");

  const onStop = (id: string) => {
    setPhases((current) =>
      current.map((phase) => (phase.id === id ? { ...phase, status: "failed" as const, duration: "1m 12s" } : phase)),
    );
  };
  const onRerun = (id: string) => {
    setPhases((current) =>
      current.map((phase) =>
        phase.id === id ? { ...phase, status: "running" as const, duration: "0s so far" } : phase,
      ),
    );
    setOpenId(id);
  };

  return (
    <div className="flex h-[32rem] min-h-0 w-full max-w-xl flex-col border border-border bg-background">
      <div className="flex items-center justify-between gap-3 px-3 pt-3">
        <SectionTitle>Automations</SectionTitle>
        <span className="text-xs font-medium text-muted-foreground">Follow</span>
      </div>
      <ol className="min-h-0 flex-1 list-none space-y-1 overflow-y-auto px-3 py-3">
        {phases.map((phase) => (
          <li key={phase.id}>
            <AutomationRow
              look={look}
              phase={phase}
              expanded={openId === phase.id}
              onToggle={() => setOpenId((current) => (current === phase.id ? "" : phase.id))}
              onStop={() => onStop(phase.id)}
              onRerun={() => onRerun(phase.id)}
            />
          </li>
        ))}
      </ol>
    </div>
  );
}

function previewHeaderAction(status: SplitRunPhaseStatus, onStop: () => void, onRerun: () => void) {
  if (status === "running") {
    return (
      <button
        type="button"
        aria-label="Stop"
        className="inline-flex size-6 shrink-0 items-center justify-center text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
        onClick={onStop}
      >
        <CircleX className="size-3.5" aria-hidden />
      </button>
    );
  }
  if (status === "failed") {
    return (
      <button
        type="button"
        aria-label="Rerun"
        className="inline-flex size-6 shrink-0 items-center justify-center text-muted-foreground hover:text-foreground"
        onClick={onRerun}
      >
        <RotateCw className="size-3.5" aria-hidden />
      </button>
    );
  }
  return null;
}

function previewGlyphKind(status: SplitRunPhaseStatus) {
  if (status === "passed") {
    return "passed" as const;
  }
  if (status === "running") {
    return "running" as const;
  }
  return "failed" as const;
}

function AutomationRow({
  look,
  phase,
  expanded,
  onToggle,
  onStop,
  onRerun,
}: {
  look: AutomationLook;
  phase: PreviewPhase;
  expanded: boolean;
  onToggle: () => void;
  onStop: () => void;
  onRerun: () => void;
}) {
  const action = previewHeaderAction(phase.status, onStop, onRerun);

  const header = (
    <div className="flex min-w-0 items-center gap-2">
      <button
        type="button"
        aria-expanded={expanded}
        onClick={onToggle}
        className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
      >
        <PhaseGlyph kind={previewGlyphKind(phase.status)} className="size-3.5" />
        <span
          className={cn(
            "min-w-0 truncate",
            look === "rule" ? "workspace-section-title" : "text-[13px] font-medium tracking-[-0.01em]",
          )}
        >
          {phase.name}
        </span>
      </button>
      {action}
    </div>
  );

  const body = expanded ? (
    <ol className="mt-1 list-none font-mono text-[14px] leading-tight text-muted-foreground">
      {phase.nodes.map((node) => (
        <li key={node.name} className="flex h-[1.375rem] items-center gap-1.5">
          <span className="min-w-0 flex-1 truncate">{node.name}</span>
          {node.status !== "pending" ? (
            <span
              className={cn(
                "shrink-0 rounded-sm px-1.5 text-right text-[12px] leading-[1.125rem] tabular-nums",
                statusTone(node.status),
              )}
            >
              {statusWord(node.status)} {formatClockDurationLabel(phase.duration)}
            </span>
          ) : null}
        </li>
      ))}
    </ol>
  ) : null;

  if (look === "card") {
    return (
      <div className={cn("rounded-md border border-border bg-card px-2 py-1.5", "border-l-2", accent(phase.status))}>
        {header}
        {body}
      </div>
    );
  }
  if (look === "rule") {
    return (
      <div className="border-b border-border py-1.5 last:border-b-0">
        {header}
        {body}
      </div>
    );
  }
  return (
    <div>
      <div className="rounded-md bg-muted/60 px-2 py-1.5">{header}</div>
      {body ? <div className="pl-1">{body}</div> : null}
    </div>
  );
}
