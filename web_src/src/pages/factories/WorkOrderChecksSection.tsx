import { useState } from "react";

import { formatRelative } from "@/lib/datetime";
import { cn } from "@/lib/utils";

import { getWorkOrderRunHref } from "./lib/workOrderExecutions";
import { WorkOrderCheckDialog } from "./WorkOrderCheckDialog";

/** How strongly the reported score should alarm (or reassure) the reader.
 * The emitting automation decides — the UI cannot know whether a high
 * number is good (coverage) or bad (risk). */
export type WorkOrderCheckLevel = "positive" | "neutral" | "caution" | "critical";

export interface WorkOrderCheckPresentation {
  id: string;
  /** Short human name, e.g. "Risk review" or "Code coverage". */
  name: string;
  score: number;
  maxScore: number;
  /** "percent" renders `82%`; "fraction" (default) renders `65/100`. */
  format?: "fraction" | "percent";
  level: WorkOrderCheckLevel;
  /** One-line result, shown in the expanded dialog under the score. */
  summary?: string;
  /** Full markdown analysis behind the score. */
  analysis?: string;
  /** Automation that produced the check, e.g. "PR Risk Review". */
  sourceName?: string;
  /** App + run that reported the score — powers the "View run" link. */
  appId?: string;
  runId?: string;
  updatedAt?: string;
}

const LEVEL_METER_CLASSNAME: Record<WorkOrderCheckLevel, string> = {
  positive: "bg-emerald-500",
  neutral: "bg-slate-400 dark:bg-slate-500",
  caution: "bg-amber-500",
  critical: "bg-red-500",
};

/** Textual verdict next to the score — color alone must not carry the meaning
 * (see Vercel Speed Insights / Shopify fraud analysis). */
export const LEVEL_LABEL: Record<WorkOrderCheckLevel, { label: string; className: string; badgeClassName: string }> = {
  positive: {
    label: "Healthy",
    className: "text-emerald-700 dark:text-emerald-400",
    badgeClassName: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  },
  neutral: {
    label: "Neutral",
    className: "text-slate-600 dark:text-slate-400",
    badgeClassName: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
  },
  caution: {
    label: "Needs attention",
    className: "text-amber-700 dark:text-amber-400",
    badgeClassName: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  },
  critical: {
    label: "Critical",
    className: "text-red-700 dark:text-red-400",
    badgeClassName: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  },
};

export function formatCheckScore(check: Pick<WorkOrderCheckPresentation, "score" | "maxScore" | "format">): {
  value: string;
  scale: string;
} {
  if (check.format === "percent") {
    return { value: String(check.score), scale: "%" };
  }
  return { value: String(check.score), scale: `/${check.maxScore}` };
}

/**
 * Scores reported by automations that reviewed the work order (risk review,
 * coverage, confidence, …). Each renders as a scorecard; clicking one opens
 * the full analysis in a dialog.
 */
export function WorkOrderChecksSection({
  checks,
  organizationId,
  factoryKey,
  orderNumber,
  className,
}: {
  checks: WorkOrderCheckPresentation[];
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  className?: string;
}) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <section className={className} data-testid="work-order-checks">
      <h2 className="workspace-section-title">Checks</h2>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        Scores reported by automations that reviewed this work order.
      </p>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {checks.map((check) => (
          <WorkOrderCheckCard
            key={check.id}
            check={check}
            runHref={getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })}
          />
        ))}
      </div>
    </section>
  );
}

function WorkOrderCheckCard({ check, runHref }: { check: WorkOrderCheckPresentation; runHref: string | null }) {
  const [open, setOpen] = useState(false);
  const { value, scale } = formatCheckScore(check);
  const ratio = check.maxScore > 0 ? Math.min(Math.max(check.score / check.maxScore, 0), 1) : 0;

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="group rounded-lg border border-border bg-card px-4 py-3 text-left transition-colors hover:border-foreground/20 hover:bg-accent/40"
        data-testid={`work-order-check-${check.id}`}
      >
        <span className="block truncate text-[12px] font-medium text-muted-foreground">{check.name}</span>
        <span className="mt-1 flex items-baseline justify-between gap-2">
          <span className="flex items-baseline gap-0.5">
            <span className="text-xl font-semibold tabular-nums tracking-tight text-foreground">{value}</span>
            <span className="text-[12px] text-muted-foreground">{scale}</span>
          </span>
          <span className={cn("truncate text-[11px] font-medium", LEVEL_LABEL[check.level].className)}>
            {LEVEL_LABEL[check.level].label}
          </span>
        </span>
        <span aria-hidden className="mt-2 block h-1 overflow-hidden rounded-full bg-muted">
          <span
            className={cn("block h-full rounded-full", LEVEL_METER_CLASSNAME[check.level])}
            style={{ width: `${ratio * 100}%` }}
          />
        </span>
        <span className="mt-2 block truncate text-[11px] text-muted-foreground">
          <CheckAttribution check={check} />
        </span>
      </button>
      <WorkOrderCheckDialog open={open} onClose={() => setOpen(false)} check={check} runHref={runHref} />
    </>
  );
}

export function CheckAttribution({ check }: { check: WorkOrderCheckPresentation }) {
  const parts = [check.sourceName, check.updatedAt ? formatRelative(check.updatedAt) : undefined].filter(Boolean);
  return <>{parts.join(" · ")}</>;
}
