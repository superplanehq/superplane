import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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

type ChecksTone = "passed" | "attention" | "failed";

/**
 * Checks in the summary panel. The header states the result. Each row is an
 * icon, the check name, and three vertical bars. Hover a row to see the
 * result and its message.
 */
export function ConsoleCheckRows({
  checks,
  title = MERGE_CONFIDENCE_SCORE_NAME,
  testId = "redesign-console-checks",
  defaultOpen = false,
}: {
  checks: WorkOrderCheckPresentation[];
  title?: string;
  testId?: string;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  if (checks.length === 0) {
    return null;
  }
  const summary = checksSummary(checks, title);
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
          <span className="block truncate text-[14px] font-semibold leading-5 text-foreground">{title}</span>
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
  if (check.level === "caution") return "attention";
  if (check.level === "positive") return "passed";
  return "other";
}

function checksSummary(
  checks: WorkOrderCheckPresentation[],
  title: string,
): { title: string; detail: string; tone: ChecksTone } {
  const tones = checks.map(checkTone);
  const failed = tones.filter((tone) => tone === "failed").length;
  const attention = tones.filter((tone) => tone === "attention").length;
  const total = checks.length;
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

/** Title color on the dark tooltip. Dark mode inverts the tooltip, so the shade flips with it. */
function titleToneClass(tone: ReturnType<typeof checkTone>): string {
  if (tone === "failed") return "text-red-400 dark:text-red-600";
  if (tone === "passed") return "text-emerald-400 dark:text-emerald-600";
  return "text-amber-400 dark:text-amber-500";
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
  const name = workOrderCheckDisplayName(check);
  const status = workOrderCheckStatus(check).label;
  const valueLabel = check.summary ? `${status}. ${check.summary}` : status;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={`${name}. ${valueLabel}`}
          data-testid={`split-run-check-${check.id}`}
          className="-mx-(--frame-panel-px) flex w-auto min-w-0 cursor-default items-center justify-between gap-3 px-(--frame-panel-px) py-2 text-left transition-colors hover:bg-muted focus-visible:bg-muted focus-visible:outline-none"
        >
          <span className="flex min-w-0 items-center gap-2">
            <CheckIcon check={check} />
            <span className="min-w-0 truncate text-[13px] font-medium leading-5 text-foreground">{name}</span>
          </span>
          <ResultBars tone={checkTone(check)} />
        </button>
      </TooltipTrigger>
      <TooltipContent side="right" collisionPadding={8} className="max-w-72 px-4 py-3 text-left">
        <span className={cn("block font-medium", titleToneClass(checkTone(check)))}>{status}</span>
        {check.summary ? <span className="mt-0.5 block font-normal">{check.summary}</span> : null}
      </TooltipContent>
    </Tooltip>
  );
}
