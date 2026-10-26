import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ArrowDown, ArrowUp, MessageSquareText, Plus, Terminal, Trash2, Zap } from "lucide-react";
import type { ReactNode } from "react";

import type { CustomVerificationMode } from "../lib/customVerification";
import { mergeCheckPromptBody } from "./mergeConfidenceSteps";
import type { PlanningReviewStep, PlanningReviewStepKind } from "./planningReviewMockup";

export function AutomationFlow({
  mode,
  steps,
  selectedStep,
  onSelectTrigger,
  onSelectStep,
  onMoveStep,
  onRemoveStep,
  onAddStep,
}: {
  mode: CustomVerificationMode;
  steps: PlanningReviewStep[];
  selectedStep: number | null;
  onSelectTrigger: () => void;
  onSelectStep: (index: number) => void;
  onMoveStep: (index: number, offset: -1 | 1) => void;
  onRemoveStep: (index: number) => void;
  onAddStep: () => void;
}) {
  return (
    <aside className="relative min-h-0 w-[42%] shrink-0 overflow-y-auto border-r border-border bg-muted/20">
      <div className="absolute inset-0 bg-[radial-gradient(circle,var(--border)_1px,transparent_1px)] [background-size:18px_18px] opacity-50" />
      <div className="relative mx-auto flex w-full max-w-lg flex-col items-center px-8 py-10">
        <FlowCard selected={selectedStep === null} onClick={onSelectTrigger}>
          <FlowBadge kind="trigger" />
          <p className="mt-3 text-sm font-semibold text-foreground">
            {mode === "automatic" ? "Every pull request commit" : "Manual"}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {mode === "automatic" ? "Runs for every new pull request commit." : "Runs when you start it from a task."}
          </p>
        </FlowCard>
        {steps.map((step, index) => (
          <div className="flex w-full flex-col items-center" key={index}>
            <FlowConnector />
            <FlowCard
              selected={selectedStep === index}
              onClick={() => onSelectStep(index)}
              actions={
                <StepCardActions
                  position={index}
                  stepCount={steps.length}
                  onMove={(offset) => onMoveStep(index, offset)}
                  onRemove={() => onRemoveStep(index)}
                />
              }
            >
              <FlowBadge kind={step.type} />
              <p className="mt-3 truncate text-sm font-semibold text-foreground">{step.name || `Step ${index + 1}`}</p>
              <p className="mt-1 truncate text-xs text-muted-foreground">
                {step.type === "prompt" ? mergeCheckPromptBody(step.prompt) : step.command || "No commands yet."}
              </p>
            </FlowCard>
          </div>
        ))}
        <FlowConnector />
        <Button type="button" variant="outline" size="sm" className="gap-1.5 bg-background" onClick={onAddStep}>
          <Plus className="size-3.5" aria-hidden />
          Add step
        </Button>
      </div>
    </aside>
  );
}

export function FlowBadge({ kind }: { kind: PlanningReviewStepKind | "trigger" }) {
  const Icon = kind === "trigger" ? Zap : kind === "prompt" ? MessageSquareText : Terminal;
  const label = kind === "trigger" ? "Trigger" : kind === "prompt" ? "Prompt" : "Bash";
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium",
        kind === "trigger" && "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
        kind === "prompt" && "border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-400",
        kind === "bash" && "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
      )}
    >
      <Icon className="size-3" aria-hidden />
      {label}
    </span>
  );
}

function FlowCard({
  selected,
  onClick,
  actions,
  children,
}: {
  selected: boolean;
  onClick: () => void;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="group/flow-card relative w-full">
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "w-full rounded-xl border bg-card px-4 py-4 text-left shadow-sm transition-colors hover:border-foreground/30",
          selected ? "border-primary ring-2 ring-primary/15" : "border-border",
        )}
      >
        {children}
      </button>
      {actions ? (
        <div className="absolute top-3 right-3 flex items-center opacity-0 transition-opacity group-focus-within/flow-card:opacity-100 group-hover/flow-card:opacity-100">
          {actions}
        </div>
      ) : null}
    </div>
  );
}

function StepCardActions({
  position,
  stepCount,
  onMove,
  onRemove,
}: {
  position: number;
  stepCount: number;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
}) {
  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-muted hover:text-foreground"
        aria-label="Move step up"
        disabled={position === 0}
        onClick={() => onMove(-1)}
      >
        <ArrowUp className="size-3.5" aria-hidden />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-muted hover:text-foreground"
        aria-label="Move step down"
        disabled={position === stepCount - 1}
        onClick={() => onMove(1)}
      >
        <ArrowDown className="size-3.5" aria-hidden />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-6 text-muted-foreground/60 hover:bg-destructive/10 hover:text-destructive"
        aria-label="Remove step"
        onClick={onRemove}
      >
        <Trash2 className="size-3.5" aria-hidden />
      </Button>
    </>
  );
}

function FlowConnector() {
  return (
    <div className="flex h-8 w-6 items-center justify-center" aria-hidden>
      <span className="h-full w-0.5 bg-foreground/30" />
    </div>
  );
}
