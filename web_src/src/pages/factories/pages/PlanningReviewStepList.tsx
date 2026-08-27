import { useRef } from "react";
import { Check, ChevronDown, GripVertical, MessageSquareText, Plus, Terminal, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { useListFieldDragReorder } from "@/ui/configurationFieldRenderer/useListFieldDragReorder";

import type { PlanningReviewStep, PlanningReviewStepKind } from "./planningReviewMockup";

const KIND_LABEL: Record<PlanningReviewStepKind, string> = { bash: "Bash", prompt: "Prompt" };

const BODY_PLACEHOLDER: Record<PlanningReviewStepKind, string> = {
  bash: "Shell commands to run on the runner.",
  prompt: "Tell the agent what to do in this step.",
};

/**
 * Light step rows for the phase editor: a kind chip, the step name, and the
 * body. Type changes and removal stay in the row menu.
 */
export function PlanningReviewStepList({
  steps,
  onChange,
}: {
  steps: PlanningReviewStep[];
  onChange: (steps: PlanningReviewStep[]) => void;
}) {
  const rowRefs = useRef<Array<HTMLDivElement | null>>([]);
  const { renderedItems, startDrag } = useListFieldDragReorder({
    items: steps,
    allowReorder: true,
    useAccordion: false,
    onChange: (next) => onChange(next as PlanningReviewStep[]),
    setOpenItem: () => undefined,
    rowRefs,
  });

  const update = (index: number, next: PlanningReviewStep) =>
    onChange(steps.map((step, position) => (position === index ? next : step)));

  return (
    <section className="flex flex-col gap-2" aria-label="Steps">
      <h4 className="text-sm font-medium text-foreground">Steps</h4>
      <div className="rounded-lg border border-border bg-muted p-2">
        <ol className="flex flex-col gap-2">
          {(renderedItems as PlanningReviewStep[]).map((step, index) => (
            <li key={index}>
              <StepRow
                step={step}
                position={index}
                canReorder={steps.length > 1}
                rowRef={(element) => {
                  rowRefs.current[index] = element;
                }}
                onDragStart={(event) => startDrag(event, index, "")}
                onChange={(next) => update(index, next)}
                onRemove={() => onChange(steps.filter((_, position) => position !== index))}
              />
            </li>
          ))}
        </ol>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mt-1 w-fit text-muted-foreground"
          onClick={() => onChange([...steps, { name: "", type: "bash", command: "" }])}
          data-testid="planning-review-add-step"
        >
          <Plus aria-hidden />
          Add step
        </Button>
      </div>
    </section>
  );
}

function StepRow({
  step,
  position,
  canReorder,
  rowRef,
  onDragStart,
  onChange,
  onRemove,
}: {
  step: PlanningReviewStep;
  position: number;
  canReorder: boolean;
  rowRef: (element: HTMLDivElement | null) => void;
  onDragStart: (event: React.MouseEvent) => void;
  onChange: (step: PlanningReviewStep) => void;
  onRemove: () => void;
}) {
  const kind = step.type;
  const label = step.name || `Step ${position + 1}`;

  return (
    <div ref={rowRef} className="flex items-start gap-1" data-testid={`planning-review-step-${position}`}>
      {canReorder ? (
        <button
          type="button"
          aria-label={`Reorder ${label}`}
          className="mt-2 flex size-6 shrink-0 cursor-grab items-center justify-center rounded-sm text-muted-foreground/60 hover:bg-accent hover:text-foreground"
          onMouseDown={onDragStart}
        >
          <GripVertical className="size-3.5" aria-hidden />
        </button>
      ) : (
        <span className="size-6 shrink-0" aria-hidden />
      )}
      <div className="min-w-0 flex-1 overflow-hidden rounded-lg border border-border bg-card shadow-xs">
        <div className="flex items-center gap-1.5 px-2.5 py-1.5">
          <StepKindMenu step={step} label={label} position={position} onChange={onChange} />
          <Input
            value={step.name}
            onChange={(event) => onChange({ ...step, name: event.target.value })}
            placeholder="Step name"
            aria-label={`Name for ${label}`}
            data-testid={`planning-review-step-name-${position}`}
            className="h-7 flex-1 border-0 bg-transparent px-1 text-[13px] font-medium shadow-none focus-visible:ring-0"
          />
          <button
            type="button"
            aria-label={`Remove ${label}`}
            className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
            data-testid={`planning-review-step-remove-${position}`}
            onClick={onRemove}
          >
            <Trash2 className="size-3.5" aria-hidden />
          </button>
        </div>
        <Textarea
          value={(kind === "prompt" ? step.prompt : step.command) ?? ""}
          onChange={(event) =>
            onChange(
              kind === "prompt" ? { ...step, prompt: event.target.value } : { ...step, command: event.target.value },
            )
          }
          placeholder={BODY_PLACEHOLDER[kind]}
          aria-label={`${KIND_LABEL[kind]} for ${label}`}
          data-testid={`planning-review-step-body-${position}`}
          className={cn(
            "min-h-14 resize-y rounded-none border-0 border-t border-border bg-transparent px-2.5 py-2 text-[13px] shadow-none focus-visible:ring-0",
            kind === "bash" && "bg-muted/60 font-mono text-[12px]",
          )}
        />
      </div>
    </div>
  );
}

const KIND_ICON: Record<PlanningReviewStepKind, typeof Terminal> = {
  bash: Terminal,
  prompt: MessageSquareText,
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
  const CurrentIcon = KIND_ICON[step.type];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Kind for ${label}`}
          className="inline-flex shrink-0 items-center gap-1 rounded-md bg-muted px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={`planning-review-step-kind-${position}`}
        >
          <CurrentIcon className="size-3" aria-hidden />
          {KIND_LABEL[step.type]}
          <ChevronDown className="size-3" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-28">
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
