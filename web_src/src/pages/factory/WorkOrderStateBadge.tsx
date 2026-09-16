import { CheckCircle2, CircleDot, Clock3, LoaderCircle, XCircle } from "lucide-react";

import { cn } from "@/lib/utils";

import type { WorkOrderState } from "./factoryTypes";

const statePresentation = {
  draft: {
    label: "Draft",
    icon: Clock3,
    className:
      "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-300",
  },
  ready: {
    label: "Ready",
    icon: CircleDot,
    className: "border-blue-200 bg-blue-50 text-blue-800 dark:border-blue-900 dark:bg-blue-950/50 dark:text-blue-300",
  },
  running: {
    label: "Running",
    icon: LoaderCircle,
    className:
      "border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-900 dark:bg-violet-950/50 dark:text-violet-300",
  },
  successful: {
    label: "Successful",
    icon: CheckCircle2,
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/50 dark:text-emerald-300",
  },
  unsuccessful: {
    label: "Unsuccessful",
    icon: XCircle,
    className: "border-red-200 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950/50 dark:text-red-300",
  },
} as const satisfies Record<WorkOrderState, { label: string; icon: typeof Clock3; className: string }>;

export function WorkOrderStateBadge({ state }: { state: WorkOrderState }) {
  const presentation = statePresentation[state];
  const StateIcon = presentation.icon;

  return (
    <span
      className={cn(
        "inline-flex h-6 shrink-0 items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium",
        presentation.className,
      )}
    >
      <StateIcon
        className={cn("size-3.5", state === "running" && "animate-spin motion-reduce:animate-none")}
        aria-hidden
      />
      {presentation.label}
    </span>
  );
}
