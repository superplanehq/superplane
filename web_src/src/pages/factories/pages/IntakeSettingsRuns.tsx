import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import {
  INTAKE_SETTINGS_COPY,
  intakePlacementActivity,
  intakePlacementLabel,
  intakeRelativeTime,
  type IntakeAutomationRun,
  type IntakeTicketPlacement,
} from "./intakeSourceSettingsModel";

const PLACEMENT_CHIP_CLASS: Record<IntakeTicketPlacement, string> = {
  progressed:
    "border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]",
  backlog:
    "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)] text-[color:var(--status-draft-fg)]",
  rejected:
    "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]",
  "below-threshold":
    "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)] text-[color:var(--status-cancelled-fg)]",
};

const PLACEMENT_DOT_CLASS: Record<IntakeTicketPlacement, string> = {
  progressed: "bg-[color:var(--status-running-dot)]",
  backlog: "bg-[color:var(--status-draft-dot)]",
  rejected: "bg-[color:var(--status-failed-dot)]",
  "below-threshold": "bg-[color:var(--status-cancelled-dot)]",
};

export function IntakeRunsList({
  runs,
  loading,
  error,
  onRetry,
  onOpenRun,
}: {
  runs: IntakeAutomationRun[];
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  onOpenRun?: (run: IntakeAutomationRun) => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="intake-source-runs">
        {INTAKE_SETTINGS_COPY.runsLoading}
      </p>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-3 px-6 py-6" data-testid="intake-source-runs">
        <p className="workspace-body-text text-destructive">{INTAKE_SETTINGS_COPY.runsError}</p>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {INTAKE_SETTINGS_COPY.retryRuns}
          </Button>
        ) : null}
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="intake-source-runs">
        {INTAKE_SETTINGS_COPY.runsEmpty}
      </p>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
      <ul
        className="mx-auto flex w-full max-w-2xl flex-col gap-2"
        data-testid="intake-source-runs"
        aria-label={INTAKE_SETTINGS_COPY.runsTab}
      >
        {runs.map((run) => (
          <li key={run.id}>
            <IntakeRunCard run={run} onOpenRun={onOpenRun} />
          </li>
        ))}
      </ul>
    </div>
  );
}

function IntakeRunCard({
  run,
  onOpenRun,
}: {
  run: IntakeAutomationRun;
  onOpenRun?: (run: IntakeAutomationRun) => void;
}) {
  const placementLabel = intakePlacementLabel(run);
  const activity = intakePlacementActivity(run);

  return (
    <article
      className="group relative w-full rounded-lg border border-border bg-card p-3.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`intake-source-run-${run.id}`}
    >
      {onOpenRun ? (
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-lg"
          aria-label={INTAKE_SETTINGS_COPY.viewRunFor(run.title)}
          onClick={() => onOpenRun(run)}
        />
      ) : null}
      <div className="relative z-10 pointer-events-none">
        <div className="flex items-start gap-3">
          <span className="relative mt-1.5 size-2 shrink-0" title={placementLabel} aria-label={placementLabel}>
            {run.placement === "progressed" ? (
              <span
                className={cn(
                  "absolute inset-0 animate-ping rounded-full opacity-60",
                  PLACEMENT_DOT_CLASS[run.placement],
                )}
                aria-hidden
              />
            ) : null}
            <span className={cn("relative block size-2 rounded-full", PLACEMENT_DOT_CLASS[run.placement])} />
          </span>
          <h3 className="min-w-0 flex-1 text-[13px] font-medium leading-snug tracking-[-0.01em] text-foreground">
            {run.title}
          </h3>
        </div>

        <dl className="mt-3 grid grid-cols-3 gap-3">
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.runWhen}</dt>
            <dd className="mt-0.5 text-[13px] tabular-nums text-foreground">{intakeRelativeTime(run.ranMinutesAgo)}</dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.analysisWhen}</dt>
            <dd className="mt-0.5 text-[13px] tabular-nums text-foreground">
              {intakeRelativeTime(run.analyzedMinutesAgo)}
            </dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.scoreWhen}</dt>
            <dd className="mt-0.5 text-[13px] font-medium tabular-nums text-foreground">{run.confidencePct}%</dd>
          </div>
        </dl>

        <div className="mt-3 flex items-start gap-2">
          <Badge variant="outline" className={cn("mt-0.5", PLACEMENT_CHIP_CLASS[run.placement])}>
            {placementLabel}
          </Badge>
          {activity ? <p className="min-w-0 text-[12px] leading-5 text-muted-foreground">{activity}</p> : null}
        </div>

        {onOpenRun ? (
          <p className="mt-3 text-[12px] font-medium text-foreground">{INTAKE_SETTINGS_COPY.viewRun}</p>
        ) : null}
      </div>
    </article>
  );
}
