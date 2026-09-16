import { cn } from "@/lib/utils";

import type { FindingSeverity, RuleEnforcement } from "./types";

const SEVERITY_META: Record<FindingSeverity, { label: string; className: string }> = {
  high: { label: "High", className: "bg-red-100 text-red-700 dark:bg-red-950/70 dark:text-red-300" },
  medium: { label: "Medium", className: "bg-amber-100 text-amber-800 dark:bg-amber-950/70 dark:text-amber-300" },
  low: { label: "Low", className: "bg-slate-200 text-gray-700 dark:bg-slate-900 dark:text-gray-300" },
};

/** Severity chip for a rule or a finding. */
export function SeverityBadge({ severity }: { severity: FindingSeverity }) {
  const meta = SEVERITY_META[severity];
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded px-1.5 py-0.5 text-[12px] font-medium leading-4",
        meta.className,
      )}
    >
      {meta.label}
    </span>
  );
}

/** Enforcement chip: blocking findings fail the verification step. */
export function EnforcementBadge({ enforcement }: { enforcement: RuleEnforcement }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center rounded border px-1.5 py-0.5 text-[11px] font-medium uppercase leading-4 tracking-wide",
        enforcement === "blocking"
          ? "border-red-200 text-red-700 dark:border-red-900 dark:text-red-300"
          : "border-border text-muted-foreground",
      )}
    >
      {enforcement === "blocking" ? "Blocking" : "Advisory"}
    </span>
  );
}
