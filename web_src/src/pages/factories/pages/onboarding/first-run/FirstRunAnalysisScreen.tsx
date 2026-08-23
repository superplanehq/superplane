import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";

import { FIRST_RUN_COPY, FIRST_RUN_STAGES } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunAnalysisStatus, FirstRunChrome } from "./firstRunTypes";

export function FirstRunAnalysisScreen({
  status,
  currentStageIndex,
  chrome,
  onRetry,
}: {
  status: FirstRunAnalysisStatus;
  /** 0–2 while running. Stages at or below this index are active or done. */
  currentStageIndex: number;
  chrome?: FirstRunChrome;
  onRetry: () => void;
}) {
  const copy = FIRST_RUN_COPY.analysis;

  return (
    <FirstRunShell testId="first-run-analysis" chrome={chrome}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.body}</p>
        <p className="text-[13px] text-muted-foreground">{copy.reassurance}</p>
      </FirstRunHeading>

      {status === "failed" ? (
        <div className="mt-8 space-y-4">
          <p className="text-[13px] text-destructive">{copy.failure}</p>
          <Button type="button" onClick={onRetry} data-testid="first-run-run-again">
            {copy.retry}
          </Button>
        </div>
      ) : (
        <FirstRunPanel>
          <ol className="space-y-3">
            {FIRST_RUN_STAGES.map((stage, index) => {
              const done = index < currentStageIndex;
              const current = index === currentStageIndex && status !== "failed";
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
        </FirstRunPanel>
      )}

      {status === "overrun" ? <p className="mt-6 text-[13px] text-muted-foreground">{copy.overrun}</p> : null}
      {status !== "failed" ? <p className="mt-6 text-[12px] text-muted-foreground">{copy.leaveHint}</p> : null}
    </FirstRunShell>
  );
}
