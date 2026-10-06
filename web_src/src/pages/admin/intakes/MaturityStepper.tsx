import { Archive, Check } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { IntakeStatus } from "@/lib/intakeCatalog";
import { cn } from "@/lib/utils";

import {
  INTAKE_STATUS_INFO,
  intakeStatusInfo,
  MATURITY_STEPS,
  NOT_IMPLEMENTED_REASON,
  type AdminIntakeEntry,
} from "./intakeCatalogModel";

interface MaturityStepperProps {
  entry: AdminIntakeEntry;
  pending: boolean;
  onChangeStatus: (status: IntakeStatus) => void;
}

export function MaturityStepper({ entry, pending, onChangeStatus }: MaturityStepperProps) {
  const [target, setTarget] = useState<IntakeStatus | null>(null);
  const current = intakeStatusInfo(entry.status);
  const deprecated = entry.status === "deprecated";

  const confirm = () => {
    if (target) onChangeStatus(target);
    setTarget(null);
  };

  return (
    <div className="flex flex-col gap-4">
      {deprecated ? <DeprecatedBanner onRestore={() => setTarget("beta")} disabled={pending} /> : null}

      <ol className="flex items-center" aria-label="Maturity">
        {MATURITY_STEPS.map((step, index) => (
          <StepItem
            key={step}
            step={step}
            index={index}
            entry={entry}
            disabled={pending}
            onSelect={() => setTarget(step)}
          />
        ))}
      </ol>

      <div className="rounded-md bg-slate-50 px-3 py-2.5 text-sm dark:bg-gray-800/60">
        <p className="text-slate-800 dark:text-gray-100">{current.summary}</p>
        {current.nextStep ? (
          <p className="mt-1 text-slate-500 dark:text-gray-400">
            <span className="font-medium text-slate-600 dark:text-gray-300">Next step: </span>
            {current.nextStep}
          </p>
        ) : null}
      </div>

      {!deprecated && entry.implemented && entry.status !== "planned" ? (
        <div>
          <Button variant="outline" size="sm" disabled={pending} onClick={() => setTarget("deprecated")}>
            <Archive size={14} />
            Mark as deprecated
          </Button>
        </div>
      ) : null}

      <StatusChangeDialog entry={entry} target={target} onCancel={() => setTarget(null)} onConfirm={confirm} />
    </div>
  );
}

interface StepItemProps {
  step: IntakeStatus;
  index: number;
  entry: AdminIntakeEntry;
  disabled: boolean;
  onSelect: () => void;
}

function StepItem({ step, index, entry, disabled, onSelect }: StepItemProps) {
  const info = INTAKE_STATUS_INFO[step];
  const currentIndex = MATURITY_STEPS.indexOf(entry.status as IntakeStatus);
  const active = entry.status === step;
  const done = currentIndex > index;
  const locked = !entry.implemented && step !== "planned";

  return (
    <li className="flex flex-1 items-center last:flex-none">
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            aria-current={active ? "step" : undefined}
            aria-disabled={locked || undefined}
            disabled={disabled || active}
            onClick={locked ? undefined : onSelect}
            className={stepClassName(info.pillClassName, { active, done, locked })}
          >
            <span className={stepDotClassName(info.dotClassName, active || done)}>
              {done ? <Check size={10} /> : index + 1}
            </span>
            {info.label}
          </button>
        </TooltipTrigger>
        <TooltipContent className="max-w-64">{locked ? NOT_IMPLEMENTED_REASON : info.summary}</TooltipContent>
      </Tooltip>
      {index < MATURITY_STEPS.length - 1 ? (
        <span className={cn("mx-2 h-px flex-1", done ? "bg-slate-400" : "bg-slate-200 dark:bg-gray-700")} aria-hidden />
      ) : null}
    </li>
  );
}

interface StepState {
  active: boolean;
  done: boolean;
  locked: boolean;
}

function stepClassName(pillClassName: string, { active, done, locked }: StepState): string {
  const base =
    "flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm font-medium whitespace-nowrap transition-colors";
  const hover = locked ? "cursor-not-allowed opacity-50" : "hover:border-slate-400 hover:text-slate-900";
  if (active) {
    return cn(base, pillClassName, locked && "opacity-50");
  }
  if (done) {
    return cn(base, "border-slate-300 bg-white text-slate-700 dark:border-gray-600 dark:bg-gray-900", hover);
  }
  return cn(
    base,
    "border-dashed border-slate-300 bg-white text-slate-500 dark:border-gray-600 dark:bg-gray-900",
    hover,
  );
}

function stepDotClassName(dotClassName: string, filled: boolean): string {
  return cn(
    "flex size-4 items-center justify-center rounded-full text-[10px]",
    filled ? cn(dotClassName, "text-white") : "bg-slate-200 text-slate-500 dark:bg-gray-700",
  );
}

function DeprecatedBanner({ onRestore, disabled }: { onRestore: () => void; disabled: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-200">
      <span>This intake is deprecated. Companies cannot create new intakes of this type.</span>
      <Button variant="outline" size="sm" disabled={disabled} onClick={onRestore}>
        Restore
      </Button>
    </div>
  );
}

interface StatusChangeDialogProps {
  entry: AdminIntakeEntry;
  target: IntakeStatus | null;
  onCancel: () => void;
  onConfirm: () => void;
}

function statusChangeTitle(name: string, from: string, to: IntakeStatus): string {
  if (to === "ga") return `Make ${name} generally available?`;
  if (to === "deprecated") return `Deprecate ${name}?`;
  if (from === "deprecated") return `Restore ${name} as ${INTAKE_STATUS_INFO[to].label}?`;
  return `Move ${name} to ${INTAKE_STATUS_INFO[to].label}?`;
}

function StatusChangeDialog({ entry, target, onCancel, onConfirm }: StatusChangeDialogProps) {
  const info = target ? INTAKE_STATUS_INFO[target] : null;

  return (
    <Dialog open={target !== null} onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{target ? statusChangeTitle(entry.name, entry.status, target) : ""}</DialogTitle>
          <DialogDescription>{info?.summary}</DialogDescription>
        </DialogHeader>
        {target === "deprecated" ? (
          <p className="text-sm text-slate-500 dark:text-gray-400">Existing intakes continue to run.</p>
        ) : null}
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant={target === "deprecated" ? "destructive" : "default"} onClick={onConfirm}>
            {target === "deprecated" ? "Deprecate" : "Change status"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
