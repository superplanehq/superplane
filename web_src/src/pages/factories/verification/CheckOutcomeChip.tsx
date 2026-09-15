import { AlertTriangle, Bot, CheckCircle2, Clock, MinusCircle, Terminal, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

import type { CheckKind, CheckOutcome } from "./types";

const OUTCOME_META: Record<CheckOutcome, { label: string; className: string; icon: LucideIcon }> = {
  running: {
    label: "Running",
    className: "bg-blue-100 text-blue-700 dark:bg-blue-950/70 dark:text-blue-300",
    icon: Clock,
  },
  passed: {
    label: "Passed",
    className: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300",
    icon: CheckCircle2,
  },
  failed: {
    label: "Failed",
    className: "bg-red-100 text-red-700 dark:bg-red-950/70 dark:text-red-300",
    icon: AlertTriangle,
  },
  skipped: {
    label: "Skipped",
    className: "bg-slate-200 text-gray-700 dark:bg-slate-900 dark:text-gray-300",
    icon: MinusCircle,
  },
};

const KIND_META: Record<CheckKind, { label: string; icon: LucideIcon }> = {
  agent: { label: "Agent review", icon: Bot },
  command: { label: "Command", icon: Terminal },
};

/** Outcome chip for one check in a verification run. */
export function CheckOutcomeChip({ outcome }: { outcome: CheckOutcome }) {
  const meta = OUTCOME_META[outcome];
  const OutcomeIcon = meta.icon;
  return (
    <span
      aria-label={meta.label}
      title={meta.label}
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded py-0.5 pl-1 pr-1.5 text-[12px] font-medium leading-4",
        meta.className,
      )}
    >
      <OutcomeIcon className="size-3.5" aria-hidden />
      <span>{meta.label}</span>
    </span>
  );
}

/** Label that tells an agent review apart from a deterministic command check. */
export function CheckKindLabel({ kind }: { kind: CheckKind }) {
  const meta = KIND_META[kind];
  const KindIcon = meta.icon;
  return (
    <span className="inline-flex items-center gap-1 text-[12px] font-medium text-muted-foreground">
      <KindIcon className="size-3.5" aria-hidden />
      <span>{meta.label}</span>
    </span>
  );
}
