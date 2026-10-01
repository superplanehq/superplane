import { cn } from "@/lib/utils";
import {
  ChevronDown,
  Circle,
  Gauge,
  GitCompareArrows,
  Shield,
  ShieldAlert,
  Undo2,
  type LucideIcon,
} from "lucide-react";
import { useState } from "react";

import { confidenceBandForScore, isScoreCheckName } from "../../../lib/confidenceScore";
import { MERGE_CONFIDENCE_SCORE_NAME } from "../../../lib/mergeConfidenceScore";
import {
  workOrderCheckDisplayName,
  workOrderCheckStatus,
  type WorkOrderCheckPresentation,
} from "../../../lib/workOrderChecks";
import { WorkOrderCheckDialog } from "../../../WorkOrderCheckDialog";

type ChecksTone = "passed" | "attention" | "failed";

/**
 * Checks in the summary panel. The header states the result. Each row is an
 * icon, the check name, and three vertical bars. A row click opens the analysis.
 */
export function ConsoleCheckRows({
  checks,
  testId = "redesign-console-checks",
}: {
  checks: WorkOrderCheckPresentation[];
  testId?: string;
}) {
  const [open, setOpen] = useState(true);
  if (checks.length === 0) {
    return null;
  }
  const summary = checksSummary(checks);
  return (
    <div data-testid={testId}>
      <button
        type="button"
        aria-expanded={open}
        aria-label={summary.title}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center gap-2.5 text-left"
      >
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[14px] font-semibold leading-5 text-foreground">
            {MERGE_CONFIDENCE_SCORE_NAME}
          </span>
          <span className="block truncate text-[12px] leading-4 text-muted-foreground">{summary.detail}</span>
        </span>
        <ResultBars tone={summary.tone} />
        <ChevronDown
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform", open ? "" : "-rotate-90")}
          aria-hidden
        />
      </button>
      {open ? (
        <ul className="mt-3 flex flex-col border-t border-border">
          {checks.map((check) => (
            <li key={check.id} className="border-b border-border last:border-b-0">
              <ConsoleCheckRow check={check} />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function checkTone(check: WorkOrderCheckPresentation): ChecksTone | "other" {
  if (isScoreCheckName(check.name)) {
    const band = confidenceBandForScore(check.score);
    if (band === "Low") return "failed";
    if (band === "Medium") return "attention";
    return "passed";
  }
  if (check.level === "critical") return "failed";
  if (check.level === "caution" || check.level === "neutral") return "attention";
  if (check.level === "positive") return "passed";
  return "other";
}

function checksSummary(checks: WorkOrderCheckPresentation[]): { title: string; detail: string; tone: ChecksTone } {
  const tones = checks.map(checkTone);
  const failed = tones.filter((tone) => tone === "failed").length;
  const attention = tones.filter((tone) => tone === "attention" || tone === "other").length;
  const total = checks.length;
  const title = MERGE_CONFIDENCE_SCORE_NAME;
  if (failed > 0) {
    return {
      title,
      detail: `${failed} of ${total} indicates high caution`,
      tone: "failed",
    };
  }
  if (attention > 0) {
    return {
      title,
      detail: `${attention} of ${total} indicates higher caution`,
      tone: "attention",
    };
  }
  return {
    title,
    detail: "All checks indicate high confidence",
    tone: "passed",
  };
}

const CHECK_ICONS: Record<string, LucideIcon> = {
  "risk-review": ShieldAlert,
  "drift-review": GitCompareArrows,
  "reversibility-review": Undo2,
  "performance-review": Gauge,
  "security-review": Shield,
  "Risk score": ShieldAlert,
  "Blast radius": ShieldAlert,
  Drift: GitCompareArrows,
  "Drift from Specification": GitCompareArrows,
  Reversibility: Undo2,
  Performance: Gauge,
  Security: Shield,
};

function CheckIcon({ check }: { check: WorkOrderCheckPresentation }) {
  const Icon = (check.key && CHECK_ICONS[check.key]) || CHECK_ICONS[check.name] || Circle;
  return <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />;
}

/** More filled bars means a better result. The color matches that result. */
function ResultBars({ tone }: { tone: ReturnType<typeof checkTone> }) {
  const filled = tone === "passed" ? 3 : tone === "failed" ? 1 : 2;
  const fill = tone === "failed" ? "bg-red-600" : tone === "passed" ? "bg-emerald-600" : "bg-amber-500";
  return (
    <span className="inline-flex shrink-0 items-center gap-0.5" aria-hidden data-filled={filled}>
      {[0, 1, 2].map((index) => (
        <span
          key={index}
          data-bar-filled={index < filled ? "true" : "false"}
          className={cn("h-3.5 w-1 rounded-[1px]", index < filled ? fill : "bg-muted-foreground/25")}
        />
      ))}
    </span>
  );
}

function ConsoleCheckRow({ check }: { check: WorkOrderCheckPresentation }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const status = workOrderCheckStatus(check);
  const name = workOrderCheckDisplayName(check);

  return (
    <>
      <button
        type="button"
        onClick={() => setDialogOpen(true)}
        aria-label={`${name}. ${status.label}`}
        data-testid={`split-run-check-${check.id}`}
        className="flex w-full min-w-0 items-center justify-between gap-3 py-2 text-left"
      >
        <span className="flex min-w-0 items-center gap-2">
          <CheckIcon check={check} />
          <span className="min-w-0 truncate text-[13px] font-medium leading-5 text-foreground">{name}</span>
        </span>
        <ResultBars tone={checkTone(check)} />
      </button>
      <WorkOrderCheckDialog open={dialogOpen} onClose={() => setDialogOpen(false)} check={check} />
    </>
  );
}
