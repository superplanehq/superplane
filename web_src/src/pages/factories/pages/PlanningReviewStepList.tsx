import { useRef, useState } from "react";
import { Check, ChevronDown, GripVertical, MessageSquareText, Plus, Terminal, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useListFieldDragReorder } from "@/ui/configurationFieldRenderer/useListFieldDragReorder";

import type { PlanningReviewStep, PlanningReviewStepKind } from "./planningReviewMockup";
import { PlanningReviewStepBody } from "./PlanningReviewStepBody";

const KIND_LABEL: Record<PlanningReviewStepKind, string> = { bash: "Bash", prompt: "Prompt" };

/**
 * Rows stay closed until you open one, so a long script cannot bury the
 * settings below the list. The list reads top to bottom in run order.
 */
export function PlanningReviewStepList({
  steps,
  onChange,
}: {
  steps: PlanningReviewStep[];
  onChange: (steps: PlanningReviewStep[]) => void;
}) {
  const rowRefs = useRef<Array<HTMLDivElement | null>>([]);
  const [openStep, setOpenStep] = useState("");
  const { renderedItems, startDrag } = useListFieldDragReorder({
    items: steps,
    allowReorder: true,
    useAccordion: true,
    onChange: (next) => onChange(next as PlanningReviewStep[]),
    setOpenItem: setOpenStep,
    rowRefs,
  });

  const update = (index: number, next: PlanningReviewStep) =>
    onChange(steps.map((step, position) => (position === index ? next : step)));

  const addStep = () => {
    onChange([...steps, { name: "", type: "bash", command: "" }]);
    setOpenStep(String(steps.length));
  };

  const removeStep = (index: number) => {
    onChange(steps.filter((_, position) => position !== index));
    setOpenStep("");
  };

  return (
    <section className="overflow-hidden rounded-xl border border-border bg-card shadow-sm" aria-label="Steps">
      <header className="flex items-center gap-2.5 border-b border-border px-5 py-3.5">
        <h3 className="text-sm font-semibold text-foreground">Steps</h3>
        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
          {steps.length}
        </span>
        <span className="flex-1" />
        <Button type="button" variant="outline" size="sm" onClick={addStep} data-testid="planning-review-add-step">
          <Plus aria-hidden />
          Add step
        </Button>
      </header>
      {steps.length === 0 ? (
        <p className="px-5 py-10 text-center text-sm text-muted-foreground" data-testid="planning-review-steps-empty">
          This agent has no steps yet. Add a step to tell the runner what to do.
        </p>
      ) : (
        <ol className="divide-y divide-border">
          {(renderedItems as PlanningReviewStep[]).map((step, index) => (
            <li key={index}>
              <StepRow
                step={step}
                position={index}
                canReorder={steps.length > 1}
                isOpen={openStep === String(index)}
                onToggle={() => setOpenStep((current) => (current === String(index) ? "" : String(index)))}
                rowRef={(element) => {
                  rowRefs.current[index] = element;
                }}
                onDragStart={(event) => startDrag(event, index, openStep)}
                onChange={(next) => update(index, next)}
                onRemove={() => removeStep(index)}
              />
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function StepRow({
  step,
  position,
  canReorder,
  isOpen,
  onToggle,
  rowRef,
  onDragStart,
  onChange,
  onRemove,
}: {
  step: PlanningReviewStep;
  position: number;
  canReorder: boolean;
  isOpen: boolean;
  onToggle: () => void;
  rowRef: (element: HTMLDivElement | null) => void;
  onDragStart: (event: React.MouseEvent) => void;
  onChange: (step: PlanningReviewStep) => void;
  onRemove: () => void;
}) {
  const kind = step.type;
  const label = step.name || `Step ${position + 1}`;
  const bodyId = `planning-review-step-body-wrapper-${position}`;

  return (
    <div
      ref={rowRef}
      className={cn("group/step bg-card transition-colors", isOpen ? "bg-muted/30" : "hover:bg-muted/40")}
      data-testid={`planning-review-step-${position}`}
    >
      <div className="flex items-center gap-2 px-3 py-2.5">
        <button
          type="button"
          aria-expanded={isOpen}
          aria-controls={bodyId}
          onClick={onToggle}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={`planning-review-step-toggle-${position}`}
        >
          <ChevronDown className={cn("size-4 transition-transform", isOpen && "rotate-180")} aria-hidden />
        </button>
        {isOpen ? (
          <Input
            value={step.name}
            onChange={(event) => onChange({ ...step, name: event.target.value })}
            placeholder="Step name"
            aria-label={`Name for ${label}`}
            data-testid={`planning-review-step-name-${position}`}
            className="h-8 min-w-0 flex-1 border-transparent bg-transparent px-2 text-sm font-medium hover:border-border hover:bg-background focus:bg-background"
          />
        ) : (
          <button
            type="button"
            onClick={onToggle}
            className="min-w-0 flex-1 truncate px-2 text-left text-sm font-medium text-foreground"
            data-testid={`planning-review-step-summary-${position}`}
          >
            {step.name || <span className="text-muted-foreground">Untitled step</span>}
          </button>
        )}
        <StepKindMenu step={step} label={label} position={position} onChange={onChange} />
        <button
          type="button"
          aria-label={`Remove ${label}`}
          className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground opacity-0 transition-colors group-focus-within/step:opacity-100 group-hover/step:opacity-100 hover:bg-destructive/10 hover:text-destructive focus-visible:opacity-100"
          data-testid={`planning-review-step-remove-${position}`}
          onClick={onRemove}
        >
          <Trash2 className="size-4" aria-hidden />
        </button>
        <StepDragHandle label={label} canReorder={canReorder} onDragStart={onDragStart} />
      </div>
      {isOpen ? (
        <div id={bodyId} className="px-3 pb-3">
          <PlanningReviewStepBody
            kind={kind}
            value={(kind === "prompt" ? step.prompt : step.command) ?? ""}
            onChange={(next) => onChange(kind === "prompt" ? { ...step, prompt: next } : { ...step, command: next })}
            label={`${KIND_LABEL[kind]} for ${label}`}
            testId={`planning-review-step-body-${position}`}
          />
        </div>
      ) : null}
    </div>
  );
}

/** Quiet until the row is hovered, so the list reads as names, not controls. */
function StepDragHandle({
  label,
  canReorder,
  onDragStart,
}: {
  label: string;
  canReorder: boolean;
  onDragStart: (event: React.MouseEvent) => void;
}) {
  if (!canReorder) {
    return <span className="size-5 shrink-0" aria-hidden />;
  }

  return (
    <button
      type="button"
      aria-label={`Reorder ${label}`}
      className="flex size-5 shrink-0 cursor-grab items-center justify-center rounded-sm text-muted-foreground/70 opacity-0 transition-opacity group-focus-within/step:opacity-100 group-hover/step:opacity-100 hover:text-foreground focus-visible:opacity-100"
      onMouseDown={onDragStart}
    >
      <GripVertical className="size-4" aria-hidden />
    </button>
  );
}

const KIND_ICON: Record<PlanningReviewStepKind, typeof Terminal> = {
  bash: Terminal,
  prompt: MessageSquareText,
};

/** Colour carries the step kind, so the row needs no icon and no chevron. */
const KIND_BADGE: Record<PlanningReviewStepKind, string> = {
  bash: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  prompt: "border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-400",
};

const KINDS: PlanningReviewStepKind[] = ["bash", "prompt"];

function StepKindMenu({
  step,
  label,
  position,
  onChange,
}: {
  step: PlanningReviewStep;
  label: string;
  position: number;
  onChange: (step: PlanningReviewStep) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Kind for ${label}`}
          className={cn(
            "inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[11px] font-medium transition-opacity hover:opacity-80",
            KIND_BADGE[step.type],
          )}
          data-testid={`planning-review-step-kind-${position}`}
        >
          {KIND_LABEL[step.type]}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-32">
        {KINDS.map((kind) => {
          const KindIcon = KIND_ICON[kind];
          return (
            <DropdownMenuItem
              key={kind}
              onSelect={() => onChange({ ...step, type: kind })}
              data-testid={`planning-review-step-kind-${position}-${kind}`}
            >
              <KindIcon className="size-3.5" aria-hidden />
              {KIND_LABEL[kind]}
              {step.type === kind ? <Check className="ml-auto size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
