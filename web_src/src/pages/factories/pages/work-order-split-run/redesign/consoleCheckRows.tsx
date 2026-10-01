import { cn } from "@/lib/utils";
import { Info } from "lucide-react";
import { useState } from "react";

import { consoleCheckList, MERGE_CONFIDENCE_SCORE_NAME } from "../../../lib/mergeConfidenceScore";
import { formatCheckScore, workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderCheckDialog } from "../../../WorkOrderCheckDialog";

/** Darker than the shared status colors so the words stay clear on the tinted panel. */
const PIP_FILL: Record<string, string> = {
  High: "bg-emerald-500",
  Healthy: "bg-emerald-500",
  Medium: "bg-orange-500",
  Caution: "bg-amber-500",
  Low: "bg-red-500",
  Critical: "bg-red-500",
  Neutral: "bg-slate-400",
};

const STATUS_TEXT: Record<string, string> = {
  High: "text-emerald-950 dark:text-emerald-200",
  Healthy: "text-emerald-950 dark:text-emerald-200",
  Medium: "text-orange-950 dark:text-orange-200",
  Caution: "text-orange-950 dark:text-orange-200",
  Low: "text-red-950 dark:text-red-200",
  Critical: "text-red-950 dark:text-red-200",
  Neutral: "text-foreground",
};

/**
 * Merge confidence in the summary panel. The verdict word sits with the
 * score. A metric click opens the analysis.
 */
export function ConsoleCheckRows({
  checks,
  testId = "redesign-console-checks",
  heading = true,
}: {
  checks: WorkOrderCheckPresentation[];
  testId?: string;
  /** False when a pull request title already names the section. */
  heading?: boolean;
}) {
  const group = consoleCheckList(checks);
  if (!group) {
    return null;
  }
  return (
    <ul className="flex flex-col gap-2" data-testid={testId}>
      <MergeConfidenceCheckRow check={group.check} metrics={group.metrics} heading={heading} />
    </ul>
  );
}

function MergeConfidenceCheckRow({
  check,
  metrics,
  heading,
}: {
  check: WorkOrderCheckPresentation;
  metrics: WorkOrderCheckPresentation[];
  heading: boolean;
}) {
  const title = heading ? (
    <h3 className="truncate text-[14px] font-semibold leading-5 text-foreground">{MERGE_CONFIDENCE_SCORE_NAME}</h3>
  ) : (
    <span className="truncate text-[13px] font-medium text-foreground">{MERGE_CONFIDENCE_SCORE_NAME}</span>
  );
  return (
    <li data-testid="split-run-check-merge-confidence">
      <div className="flex min-w-0 items-center justify-between gap-3 text-left">
        {title}
        <ScoreReadout check={check} testId="split-run-merge-confidence-meter" />
      </div>
      <ul className="mt-1.5 flex flex-col">
        {metrics.map((metric) => (
          <li key={metric.id}>
            <ConsoleCheckRow check={metric} />
          </li>
        ))}
      </ul>
    </li>
  );
}

function ConsoleCheckRow({ check }: { check: WorkOrderCheckPresentation }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);
  const score = `${value}${scale}`;

  return (
    <>
      <button
        type="button"
        onClick={() => setDialogOpen(true)}
        aria-label={`${check.name}, ${status.label}, ${score}. Read the reason.`}
        data-testid={`split-run-check-${check.id}`}
        className="-mx-1.5 flex w-[calc(100%+0.75rem)] cursor-pointer items-center justify-between gap-3 rounded-md px-1.5 py-0.5 text-left hover:bg-black/5 dark:hover:bg-white/10"
      >
        <span className="flex min-w-0 items-center gap-1">
          <CheckName check={check} />
          <Info className="size-3 shrink-0 text-foreground/55" aria-hidden />
        </span>
        <ScoreReadout check={check} />
      </button>
      <WorkOrderCheckDialog open={dialogOpen} onClose={() => setDialogOpen(false)} check={check} />
    </>
  );
}

function CheckName({ check }: { check: WorkOrderCheckPresentation }) {
  return <span className="truncate text-[12px] font-medium text-foreground">{check.name}</span>;
}

function ScoreReadout({ check, testId }: { check: WorkOrderCheckPresentation; testId?: string }) {
  const status = workOrderCheckStatus(check);
  return (
    <span className="inline-flex shrink-0 items-center gap-2">
      <span className={cn("text-[12px] font-medium", STATUS_TEXT[status.label] ?? "text-foreground")}>
        {status.label}
      </span>
      <ScorePips check={check} testId={testId} />
    </span>
  );
}

const PIP_COUNT = 5;

function ScorePips({ check, testId }: { check: WorkOrderCheckPresentation; testId?: string }) {
  const status = workOrderCheckStatus(check);
  const value = Math.max(0, Math.round(check.score));
  const filled =
    check.maxScore > 0 ? Math.min(PIP_COUNT, Math.max(0, Math.round((check.score / check.maxScore) * PIP_COUNT))) : 0;
  const fill = PIP_FILL[status.label] ?? "bg-slate-400";

  return (
    <span className="inline-flex shrink-0 items-center gap-1.5" data-testid={testId}>
      <span className="inline-flex items-center gap-0.5" aria-hidden>
        {Array.from({ length: PIP_COUNT }, (_, index) => (
          <span
            key={index}
            data-filled={index < filled ? "true" : "false"}
            className={cn("h-2 w-1.5 rounded-[1px]", index < filled ? fill : "bg-muted-foreground/25")}
          />
        ))}
      </span>
      <span className={cn("text-[13px] font-semibold tabular-nums", STATUS_TEXT[status.label] ?? "text-foreground")}>
        {value}
      </span>
    </span>
  );
}
