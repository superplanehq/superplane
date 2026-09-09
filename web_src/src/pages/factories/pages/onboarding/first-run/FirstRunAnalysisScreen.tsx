import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

type StageState = "done" | "current" | "pending" | "empty";

function StageIcon({ state }: { state: StageState }) {
  if (state === "done") {
    return <Check className="size-3.5 shrink-0 text-emerald-600" strokeWidth={2.5} aria-hidden />;
  }
  if (state === "empty") {
    return (
      <span className="inline-flex w-[14px] shrink-0 items-center justify-center text-[14px] leading-none" aria-hidden>
        🤔
      </span>
    );
  }
  return (
    <Loader2
      className={cn(
        "size-3.5 shrink-0",
        state === "current" ? "animate-spin text-foreground" : "text-muted-foreground",
      )}
      aria-hidden
    />
  );
}

/**
 * The screen is a hand-off, not a wait: setup finished, the newest open
 * tickets are in, and scoring runs on its own. Both rows read as done —
 * the second confirms that scoring started and points to the board. The
 * counter beside it is the payoff, not a progress gate: it shows how many
 * tickets already scored high enough to run, and grows as scores land.
 */
function stageRows(
  progress: FirstRunAnalysisProgress,
  sourceName?: string,
): Array<{ label: string; state: StageState; count?: string }> {
  const copy = FIRST_RUN_COPY.analysis;
  if (progress.empty) {
    return [{ label: copy.emptyImport(sourceName), state: "empty" }];
  }
  if (progress.stageIndex === 0) {
    return [
      { label: copy.stageImporting, state: "current" },
      { label: copy.stageScoringPending, state: "pending" },
    ];
  }
  const count = progress.ready > 0 ? copy.readyCount(progress.ready) : undefined;
  return [
    { label: copy.stageImported(progress.total, sourceName), state: "done" },
    progress.stageIndex === 2
      ? { label: copy.stageScored(progress.total), state: "done", count }
      : { label: copy.stageScoring, state: "done", count },
  ];
}

export function FirstRunAnalysisScreen({
  progress,
  sourceName,
  failed = false,
  chrome,
  sphere,
  onGoToBoard,
}: {
  progress: FirstRunAnalysisProgress;
  /** Ticket source display name, e.g. "GitHub issues". */
  sourceName?: string;
  failed?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onGoToBoard: () => void;
}) {
  const copy = FIRST_RUN_COPY.analysis;
  const rows = stageRows(progress, sourceName);

  return (
    <FirstRunShell testId="first-run-analysis" chrome={chrome} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <ol className="mt-8 space-y-3">
        {rows.map(({ label, state, count }) => (
          <li key={label} className="flex items-center gap-3 text-[13px]">
            <StageIcon state={state} />
            <span className={state === "pending" ? "text-muted-foreground" : "text-foreground"}>{label}</span>
            {count ? (
              <span
                className="ml-auto shrink-0 rounded-full border border-emerald-600/25 bg-emerald-600/10 px-2 py-0.5 font-mono text-[11px] tabular-nums text-emerald-700 dark:text-emerald-400"
                data-testid="first-run-ready-count"
              >
                {count}
              </span>
            ) : null}
          </li>
        ))}
      </ol>

      {progress.empty ? (
        <p className="mt-4 text-[13px] text-muted-foreground" data-testid="first-run-analysis-empty">
          {copy.emptyNext}
        </p>
      ) : null}

      {failed ? <p className="mt-4 text-[13px] text-muted-foreground">{copy.failure}</p> : null}

      <Button type="button" className="mt-8 min-w-44" onClick={onGoToBoard} data-testid="first-run-go-to-board">
        {copy.goToBoard}
      </Button>
    </FirstRunShell>
  );
}
