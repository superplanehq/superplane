import type { ReactNode } from "react";

/** Label + value row used in dashboard confirm dialogs (Run, row actions, etc.). */
export function ConfirmFact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-0.5">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <div className="text-slate-700">{children}</div>
    </div>
  );
}

/** Pretty-print trigger parameters for read-only JSON preview blocks. */
export function formatParameters(parameters: Record<string, unknown> | undefined): string {
  if (!parameters || Object.keys(parameters).length === 0) return "(empty)";
  try {
    return JSON.stringify(parameters, null, 2);
  } catch {
    return String(parameters);
  }
}
