import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

import {
  WORKSPACE_NEXT_STEPS_COPY,
  workspaceNextStepsProgressCopy,
  type WorkspaceNextStep,
} from "./workspaceNextStepCatalog";

export function NextStepsPanel({
  steps,
  onSelect,
}: {
  steps: WorkspaceNextStep[];
  onSelect: (step: WorkspaceNextStep) => void;
}) {
  if (steps.length === 0) {
    return null;
  }

  const doneCount = steps.filter((step) => step.done).length;

  return (
    <section className="px-3 pb-3" data-testid="workspace-next-steps">
      <div role="status" className="rounded-lg border border-border bg-background px-3.5 py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-[18px] font-semibold tracking-[-0.02em] text-foreground">
              {WORKSPACE_NEXT_STEPS_COPY.title}
            </h2>
            <p className="mt-1.5 text-[13px] text-muted-foreground">{WORKSPACE_NEXT_STEPS_COPY.description}</p>
          </div>
          <p className="shrink-0 pt-1 text-[12px] font-medium text-muted-foreground">
            {workspaceNextStepsProgressCopy(doneCount, steps.length)}
          </p>
        </div>
        <ul className="mt-3 overflow-hidden rounded-md border border-border">
          {steps.map((step, index) => (
            <li key={step.id} className={index > 0 ? "border-t border-border" : undefined}>
              <NextStepRow step={step} onSelect={onSelect} />
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function NextStepRow({ step, onSelect }: { step: WorkspaceNextStep; onSelect: (step: WorkspaceNextStep) => void }) {
  const status = step.done ? WORKSPACE_NEXT_STEPS_COPY.done : WORKSPACE_NEXT_STEPS_COPY.configure;
  const body = (
    <>
      <NextStepStatusIcon done={step.done} />
      <span className="min-w-0 flex-1">
        <span
          className={cn(
            "block text-[13px] font-medium tracking-[-0.01em]",
            step.done ? "text-muted-foreground" : "text-foreground",
          )}
        >
          {step.title}
        </span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{step.description}</span>
      </span>
      <span
        className={cn(
          "shrink-0 text-[12px] font-medium",
          step.done ? "text-emerald-700 dark:text-emerald-400" : "text-foreground",
        )}
      >
        {status}
      </span>
    </>
  );

  if (step.done) {
    return (
      <div
        className="flex w-full items-center gap-3 px-3 py-2.5 text-left"
        data-testid={`workspace-next-step-${step.id}`}
        data-state="done"
        aria-label={`${step.title}, ${status}`}
      >
        {body}
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => onSelect(step)}
      className="flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors hover:bg-accent/30"
      data-testid={`workspace-next-step-${step.id}`}
      data-state="open"
      aria-label={`${step.title}, ${status}`}
    >
      {body}
    </button>
  );
}

function NextStepStatusIcon({ done }: { done: boolean }) {
  if (done) {
    return (
      <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-600 text-white">
        <Check className="size-3" strokeWidth={2.5} aria-hidden />
      </span>
    );
  }

  return <span className="size-5 shrink-0 rounded-full border-2 border-foreground bg-background" aria-hidden />;
}
