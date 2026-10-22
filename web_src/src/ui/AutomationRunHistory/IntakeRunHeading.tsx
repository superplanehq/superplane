import { ExternalLink } from "lucide-react";
import { CopyButton } from "@/ui/CopyButton";
import { RunIdCopy } from "./RunIdentity";
import { IntakeOutcomeLabel } from "./IntakeOutcomeLabel";
import type { AutomationRun } from "./types";
export function IntakeRunHeading({ run }: { run: AutomationRun }) {
  const intake = run.intake;
  if (!intake) return null;
  return (
    <>
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {run.source?.title ?? run.task?.title ?? run.title ?? "Run details"}
        </h1>
        <IntakeOutcomeLabel outcome={intake.outcome} />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-4 text-sm text-muted-foreground">
        <div className="flex flex-wrap items-center gap-4">
          {run.source?.url ? (
            <a
              href={run.source.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 font-mono hover:text-foreground hover:underline"
            >
              {run.source.reference}
              <ExternalLink aria-hidden className="size-4 shrink-0" />
            </a>
          ) : (
            <span className="font-mono">{run.source?.reference ?? "Source unavailable"}</span>
          )}
          <span>{intake.event}</span>
          {run.task ? (
            <span className="inline-flex items-center gap-2">
              <span>
                Task <code className="font-mono">{run.task.id}</code>
              </span>
              <CopyButton
                text={run.task.id}
                ariaLabel={`Copy task ID ${run.task.id}`}
                copiedAriaLabel="Task ID copied"
              />
            </span>
          ) : null}
        </div>
        <RunIdCopy id={run.id} full />
      </div>
      <p className="text-sm text-muted-foreground">{intake.summary}</p>
    </>
  );
}
