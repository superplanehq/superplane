import type { ReactNode } from "react";

/** Label + value row used in dashboard confirm dialogs (Run, row actions, etc.). */
export function ConfirmFact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0 space-y-0.5">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <div className="min-w-0 text-slate-700">{children}</div>
    </div>
  );
}

export function ConfirmParametersPreview({ children, testId }: { children: ReactNode; testId: string }) {
  return (
    <pre
      className="mt-1 max-h-40 w-full max-w-full min-w-0 overflow-x-auto overflow-y-auto rounded-md border border-slate-200 bg-slate-50 p-2 font-mono text-[11px] leading-snug whitespace-pre text-slate-700"
      data-testid={testId}
    >
      {children}
    </pre>
  );
}
