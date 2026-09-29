import { cn } from "@/lib/utils";
import { useState } from "react";

import { formatCheckScore, workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderCheckDialog } from "../../../WorkOrderCheckDialog";
import { checkTickBar } from "./checkTickBar";

const TICK_FILL: Record<string, string> = {
  High: "bg-emerald-500",
  Healthy: "bg-emerald-500",
  Medium: "bg-orange-500",
  Caution: "bg-amber-500",
  Low: "bg-red-500",
  Critical: "bg-red-500",
  Neutral: "bg-slate-400",
};

/**
 * Checks in the summary panel: name, a full-width segmented gauge
 * (stats-13), then the score. A click opens the analysis dialog.
 */
export function ConsoleCheckRows({
  checks,
  testId = "redesign-console-checks",
}: {
  checks: WorkOrderCheckPresentation[];
  testId?: string;
}) {
  if (checks.length === 0) {
    return null;
  }
  return (
    <ul className="flex flex-col gap-2" data-testid={testId}>
      {checks.map((check) => (
        <li key={check.id}>
          <ConsoleCheckRow check={check} />
        </li>
      ))}
    </ul>
  );
}

function ConsoleCheckRow({ check }: { check: WorkOrderCheckPresentation }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);
  const score = `${value}${scale}`;
  const ticks = checkTickBar(check);
  const fill = TICK_FILL[status.label] ?? "bg-slate-400";

  return (
    <>
      <button
        type="button"
        onClick={() => setDialogOpen(true)}
        aria-label={`${check.name} ${score}. ${status.label}`}
        data-testid={`split-run-check-${check.id}`}
        className="flex w-full min-w-0 flex-col gap-1.5 text-left"
      >
        <span className="truncate text-[13px] font-medium text-foreground">{check.name}</span>
        {ticks.total > 0 ? (
          <span className="flex h-2 w-full gap-px" aria-hidden>
            {Array.from({ length: ticks.total }, (_, index) => (
              <span
                key={index}
                className={cn("min-w-0 flex-1 rounded-full", index < ticks.filled ? fill : "bg-muted")}
              />
            ))}
          </span>
        ) : null}
        <span className="flex items-center justify-between gap-2 text-[12px]">
          <span className="tabular-nums text-muted-foreground">{score}</span>
          <span className={cn("truncate font-medium", status.className)}>{status.label}</span>
        </span>
      </button>
      <WorkOrderCheckDialog open={dialogOpen} onClose={() => setDialogOpen(false)} check={check} />
    </>
  );
}
