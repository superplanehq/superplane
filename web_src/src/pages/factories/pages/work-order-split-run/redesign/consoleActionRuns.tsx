import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import type { StepActionRun } from "./consoleStepActions";

export function ActionsPage({
  runs,
  failed,
  onRetry,
}: {
  runs: StepActionRun[];
  failed?: boolean;
  onRetry?: () => void;
}) {
  return (
    <div className="space-y-3">
      {failed ? (
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-[12px] leading-5 text-muted-foreground">The actions did not load.</p>
          {onRetry ? (
            <Button size="sm" variant="outline" onClick={onRetry}>
              Try again
            </Button>
          ) : null}
        </div>
      ) : null}
      <div className="flex flex-col divide-y divide-border">
        {runs.map((run, index) => (
          <ActionRun key={run.runId} run={run} defaultOpen={index === runs.length - 1} />
        ))}
      </div>
    </div>
  );
}

function ActionRun({ run, defaultOpen }: { run: StepActionRun; defaultOpen: boolean }) {
  const started = run.startedAt ? new Date(run.startedAt) : undefined;
  const dateTime = started?.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
  return (
    <Collapsible defaultOpen={defaultOpen} className="group py-1.5 first:pt-0 last:pb-0">
      <CollapsibleTrigger className="flex w-full cursor-pointer items-center gap-1.5 rounded-sm py-1 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
        <ChevronRight
          className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
          aria-hidden
        />
        <span className="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-2 gap-y-0.5">
          <span className="break-words text-[12px] font-medium text-foreground">{run.title}</span>
          <span className="text-[12px] text-muted-foreground">Run {run.ordinal}</span>
          {dateTime ? (
            <time dateTime={run.startedAt} className="text-[12px] tabular-nums text-muted-foreground">
              {dateTime}
            </time>
          ) : null}
        </span>
        <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">
          {run.rows.length} {run.rows.length === 1 ? "action" : "actions"}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="py-2 pl-5">
        <ul className="flex flex-col divide-y divide-border">
          {run.rows.map((row) => (
            <li key={row.id} className="min-w-0 break-words py-2.5 first:pt-0 last:pb-0">
              <p className="text-[12px] font-medium text-foreground">{row.title}</p>
              {row.detail ? (
                <p className="mt-1.5 whitespace-pre-wrap text-[12px] leading-5 text-muted-foreground">{row.detail}</p>
              ) : null}
            </li>
          ))}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}
