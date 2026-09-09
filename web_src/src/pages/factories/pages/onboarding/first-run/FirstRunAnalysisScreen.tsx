import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunAnalysisProgress } from "./firstRunAnalysisProgress";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

function stageLabels(progress: FirstRunAnalysisProgress): [string, string, string] {
  const copy = FIRST_RUN_COPY.analysis;
  return [
    progress.total > 0 ? copy.stageImported(progress.total) : copy.stageImporting,
    copy.stageScoring(progress.scored, progress.total),
    copy.stageBoard,
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
  const stages = stageLabels(progress);

  return (
    <FirstRunShell testId="first-run-analysis" chrome={chrome} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <ol className="mt-8 space-y-3">
        {stages.map((stage, index) => {
          const done = index < progress.stageIndex;
          const current = index === progress.stageIndex;
          return (
            <li key={stage} className="flex items-center gap-3 text-[13px]">
              {done ? (
                <Check className="size-3.5 shrink-0 text-emerald-600" strokeWidth={2.5} aria-hidden />
              ) : (
                <Loader2
                  className={cn(
                    "size-3.5 shrink-0",
                    current ? "animate-spin text-foreground" : "text-muted-foreground",
                  )}
                  aria-hidden
                />
              )}
              <span className={current || done ? "text-foreground" : "text-muted-foreground"}>{stage}</span>
            </li>
          );
        })}
      </ol>

      {failed ? <p className="mt-4 text-[13px] text-muted-foreground">{copy.failure}</p> : null}

      <Button type="button" className="mt-8 min-w-44" onClick={onGoToBoard} data-testid="first-run-go-to-board">
        {copy.goToBoard}
      </Button>
      <p className="mt-3 text-[12px] text-muted-foreground">{copy.note}</p>
    </FirstRunShell>
  );
}
