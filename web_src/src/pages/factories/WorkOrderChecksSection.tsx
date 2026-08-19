import { MoveDown, MoveUp } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

import {
  formatCheckScore,
  isBooleanCheck,
  LEVEL_LABEL,
  type ScoreCheckPresentation,
  type WorkOrderCheckLevel,
  type WorkOrderCheckPresentation,
} from "./lib/workOrderChecks";
import { getWorkOrderRunHref } from "./lib/workOrderExecutions";
import { useWorkOrderChecksPrototypeSlot } from "./workOrderChecksPrototypeSlot";
import { WorkOrderCheckAttribution } from "./WorkOrderCheckAttribution";
import { WorkOrderCheckDialog } from "./WorkOrderCheckDialog";

const LEVEL_METER_CLASSNAME: Record<WorkOrderCheckLevel, string> = {
  positive: "bg-emerald-500",
  neutral: "bg-slate-400 dark:bg-slate-500",
  caution: "bg-amber-500",
  critical: "bg-red-500",
};

/**
 * Scores reported by automations that reviewed the work order (risk review,
 * coverage, confidence, …). Each renders as a scorecard; clicking one opens
 * the full analysis in a dialog.
 */
export function WorkOrderChecksSection({
  checks,
  isLoading,
  error,
  organizationId,
  factoryKey,
  workOrderId,
  orderNumber,
  className,
}: {
  checks: WorkOrderCheckPresentation[];
  /** Initial fetch in flight — renders the section with a loading line. */
  isLoading?: boolean;
  /** Fetch failure — renders the section with an error line instead of hiding it. */
  error?: Error | null;
  organizationId: string;
  factoryKey: string;
  /** Storybook-only: keys the boolean check prototype slot. The live app has no provider, so this is unused there. */
  workOrderId?: string;
  orderNumber?: string;
  className?: string;
}) {
  // Storybook-only: a harnessed story can supply pass/fail checks (CI,
  // security scan, …) alongside the real scored ones. The live app has no
  // provider for this context, so `prototypeChecks` is always empty there.
  const prototypeChecksSlot = useWorkOrderChecksPrototypeSlot();
  const prototypeChecks = prototypeChecksSlot?.(workOrderId ?? "") ?? [];
  const allChecks = prototypeChecks.length ? [...checks, ...prototypeChecks] : checks;

  if (!isLoading && !error && allChecks.length === 0) {
    return null;
  }

  return (
    <section className={className} data-testid="work-order-checks">
      <h2 className="workspace-section-title">Checks</h2>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        Scores and pass/fail gates reported by automations that reviewed this work order.
      </p>
      {error ? (
        <p className="mt-4 text-[13px] text-destructive">Failed to load checks.</p>
      ) : isLoading ? (
        <p className="mt-4 text-[13px] text-muted-foreground">Loading checks…</p>
      ) : (
        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {allChecks.map((check) => (
            <WorkOrderCheckCard
              key={check.id}
              check={check}
              runHref={getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function WorkOrderCheckCard({ check, runHref }: { check: WorkOrderCheckPresentation; runHref: string | null }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="group rounded-lg border border-border bg-card px-4 py-3 text-left transition-colors hover:border-foreground/20 hover:bg-accent/40"
        data-testid={`work-order-check-${check.id}`}
      >
        <span className="block truncate text-[12px] font-medium text-muted-foreground">{check.name}</span>
        {isBooleanCheck(check) ? (
          <span className="mt-2 block">
            <Badge variant="outline" className={cn("border", LEVEL_LABEL[check.level].badgeClassName)}>
              {check.passed ? "Pass" : "Fail"}
            </Badge>
          </span>
        ) : (
          <ScoreCheckCardBody check={check} />
        )}
        <span className="mt-2 block truncate text-[11px] text-muted-foreground">
          <WorkOrderCheckAttribution check={check} />
        </span>
      </button>
      <WorkOrderCheckDialog open={open} onClose={() => setOpen(false)} check={check} runHref={runHref} />
    </>
  );
}

/** Numeric value, level label, and the progress meter — scored checks only. */
function ScoreCheckCardBody({ check }: { check: ScoreCheckPresentation }) {
  const { value, scale } = formatCheckScore(check);
  const ratio = check.maxScore > 0 ? Math.min(Math.max(check.score / check.maxScore, 0), 1) : 0;

  return (
    <>
      <span className="mt-1 flex items-baseline justify-between gap-2">
        <span className="flex items-baseline gap-0.5">
          <span className="text-xl font-semibold tabular-nums tracking-tight text-foreground">{value}</span>
          <span className="text-[12px] text-muted-foreground">{scale}</span>
          <CheckTrendDelta check={check} />
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
    </>
  );
}

/**
 * Direction of travel since the previous report ("↓ 17"). The arrow stays
 * neutral muted — whether a move is good or bad is the level's job, since
 * lower is better for risk but worse for coverage.
 */
function CheckTrendDelta({ check }: { check: Pick<ScoreCheckPresentation, "score" | "previousScore"> }) {
  if (check.previousScore === undefined || check.previousScore === check.score) {
    return null;
  }

  const rising = check.score > check.previousScore;
  const Arrow = rising ? MoveUp : MoveDown;
  // Trim float noise (7.5 - 6.3 = 1.2000000000000002) without forcing
  // decimals onto integer scores.
  const delta = Number(Math.abs(check.score - check.previousScore).toFixed(2));
  return (
    <span
      className="inline-flex items-baseline text-[11px] tabular-nums text-muted-foreground"
      title={`Previous run: ${check.previousScore}`}
    >
      <Arrow className="size-3 shrink-0 self-center" aria-hidden />
      {delta}
      <span className="sr-only">{rising ? "up" : "down"} from </span>
      <span className="sr-only">{check.previousScore}</span>
    </span>
  );
}
