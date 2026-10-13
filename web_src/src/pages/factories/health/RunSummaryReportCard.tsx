import { AlertTriangle, CheckCircle2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import type { RunSummaryReport, SeverityCounts } from "@/pages/factories/verification/types";

interface RunSummaryReportCardProps {
  report: RunSummaryReport;
  /** `work-order` renders the timeline card; `slack` renders a message preview. */
  variant?: "work-order" | "slack";
}

/**
 * Structured summary of one verification run: findings detected, fixed, and
 * remaining by severity, plus the gate result.
 */
export function RunSummaryReportCard({ report, variant = "work-order" }: RunSummaryReportCardProps) {
  if (variant === "slack") return <SlackPreview report={report} />;

  return (
    <section className={cn(factoryCardClassName, "flex flex-col gap-3 p-4")} aria-label="Verification summary">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-[13px] font-medium text-foreground">{report.runLabel}</span>
        <GateBadge report={report} />
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        <CountsBlock label="Detected" counts={report.detected} />
        <CountsBlock label="Fixed" counts={report.fixed} />
        <CountsBlock label="Remaining" counts={report.remaining} />
      </div>
    </section>
  );
}

function GateBadge({ report }: { report: RunSummaryReport }) {
  if (report.gatePassed) {
    return (
      <span className="inline-flex items-center gap-1 rounded bg-emerald-100 py-0.5 pl-1 pr-1.5 text-[12px] font-medium text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300">
        <CheckCircle2 className="size-3.5" aria-hidden />
        Gate passed
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 rounded bg-red-100 py-0.5 pl-1 pr-1.5 text-[12px] font-medium text-red-700 dark:bg-red-950/70 dark:text-red-300">
      <AlertTriangle className="size-3.5" aria-hidden />
      Gate failed · {report.blockingCount} blocking
    </span>
  );
}

function CountsBlock({ label, counts }: { label: string; counts: SeverityCounts }) {
  const total = counts.high + counts.medium + counts.low;
  return (
    <div className="rounded-md border border-border bg-background px-3 py-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="workspace-section-label text-muted-foreground">{label}</span>
        <span className="text-lg font-medium tabular-nums text-foreground">{total}</span>
      </div>
      <dl className="mt-1 flex flex-col gap-0.5 text-[12px] text-muted-foreground">
        <SeverityRow label="High" value={counts.high} emphasizeClass="text-red-600 dark:text-red-400" />
        <SeverityRow label="Medium" value={counts.medium} emphasizeClass="text-amber-700 dark:text-amber-400" />
        <SeverityRow label="Low" value={counts.low} />
      </dl>
    </div>
  );
}

function SeverityRow({ label, value, emphasizeClass }: { label: string; value: number; emphasizeClass?: string }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <dt>{label}</dt>
      <dd className={cn("tabular-nums", value > 0 && emphasizeClass ? `font-medium ${emphasizeClass}` : undefined)}>
        {value}
      </dd>
    </div>
  );
}

function SlackPreview({ report }: { report: RunSummaryReport }) {
  const line = (label: string, counts: SeverityCounts) =>
    `${label}: ${counts.high} high, ${counts.medium} medium, ${counts.low} low`;
  return (
    <div className={cn(factoryCardClassName, "flex max-w-md flex-col gap-2 p-4")} aria-label="Slack message preview">
      <p className="workspace-section-label text-muted-foreground">Slack preview</p>
      <div className="flex gap-2.5 rounded-md border border-border bg-background p-3">
        <span className="flex size-8 shrink-0 items-center justify-center rounded bg-slate-200 text-[11px] font-semibold text-slate-700 dark:bg-gray-800 dark:text-gray-300">
          SP
        </span>
        <div className="flex min-w-0 flex-col gap-1 text-[13px]">
          <p>
            <span className="font-semibold text-foreground">SuperPlane</span>{" "}
            <span className="text-[11px] text-muted-foreground">APP</span>
          </p>
          <p className="font-medium text-foreground">
            {report.gatePassed ? "Verification passed" : `Verification failed · ${report.blockingCount} blocking`}
          </p>
          <p className="text-muted-foreground">{report.runLabel}</p>
          <ul className="text-muted-foreground">
            <li>{line("Detected", report.detected)}</li>
            <li>{line("Fixed", report.fixed)}</li>
            <li>{line("Remaining", report.remaining)}</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
