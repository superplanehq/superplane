import { AlertTriangle, CheckCircle2, Clock, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

import type { VerificationRunStatus } from "./types";

const BASE_CLASSES =
  "inline-flex shrink-0 items-center gap-1 rounded py-0.5 pl-1 pr-1.5 text-[12px] font-medium leading-4";

const STATUS_META: Record<VerificationRunStatus, { label: string; className: string; icon: LucideIcon }> = {
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
};

/** Status badge for a verification run, matching the run status badge style. */
export function VerificationStatusBadge({ status }: { status: VerificationRunStatus }) {
  const meta = STATUS_META[status];
  const StatusIcon = meta.icon;
  return (
    <span aria-label={meta.label} title={meta.label} className={cn(BASE_CLASSES, meta.className)}>
      <StatusIcon className="size-3.5" aria-hidden />
      <span>{meta.label}</span>
    </span>
  );
}
