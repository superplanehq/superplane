import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

type StageState = "done" | "current" | "pending";

/**
 * Every row maps to observed intake state; the screen never shows a stage
 * the system is not actually in. When scoring finishes, the last row turns
 * into the result instead of another wait.
 */
function stageRows(progress: FirstRunAnalysisProgress): Array<{ label: string; state: StageState }> {
  const copy = FIRST_RUN_COPY.analysis;
  if (progress.stageIndex === 0) {
    return [
      { label: copy.stageImporting, state: "current" },
      { label: copy.stageScoringPending, state: "pending" },
    ];
  }
  const scoringDone = progress.stageIndex === 2;
  return [
    { label: copy.stageImported(progress.total), state: "done" },
    scoringDone
      ? { label: copy.stageScored(progress.total, progress.ready), state: "done" }
      : { label: copy.stageScoring(progress.scored, progress.total), state: "current" },
  ];
}

export function FirstRunAnalysisScreen({
  progress,
  failed = false,
  chrome,
  sphere,
  onGoToBoard,
}: {
  progress: FirstRunAnalysisProgress;
  failed?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onGoToBoard: () => void;
}) {
  const copy = FIRST_RUN_COPY.analysis;
  const rows = stageRows(progress);

  return (
    <FirstRunShell testId="first-run-analysis" chrome={chrome} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <ol className="mt-8 space-y-3">
        {rows.map(({ label, state }) => (
          <li key={label} className="flex items-center gap-3 text-[13px]">
            {state === "done" ? (
              <Check className="size-3.5 shrink-0 text-emerald-600" strokeWidth={2.5} aria-hidden />
            ) : (
              <Loader2
                className={cn(
                  "size-3.5 shrink-0",
                  state === "current" ? "animate-spin text-foreground" : "text-muted-foreground",
                )}
                aria-hidden
              />
            )}
            <span className={state === "pending" ? "text-muted-foreground" : "text-foreground"}>{label}</span>
          </li>
        ))}
      </ol>

      {failed ? <p className="mt-4 text-[13px] text-muted-foreground">{copy.failure}</p> : null}

      <Button type="button" className="mt-8 min-w-44" onClick={onGoToBoard} data-testid="first-run-go-to-board">
        {copy.goToBoard}
      </Button>
      <p className="mt-3 text-[12px] text-muted-foreground">{progress.stageIndex === 2 ? copy.noteDone : copy.note}</p>
    </FirstRunShell>
  );
}
