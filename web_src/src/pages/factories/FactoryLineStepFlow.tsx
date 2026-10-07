import { cn } from "@/lib/utils";
import { ChevronDown, Plus } from "lucide-react";
import type { ReactNode } from "react";

export function LineStepArrow({ className }: { className?: string }) {
  return (
    <div className={cn("flex flex-col items-center py-0.5", className)} aria-hidden>
      <div className="h-2 w-px bg-slate-300 dark:bg-gray-600" />
      <ChevronDown className="h-3.5 w-3.5 shrink-0 text-slate-400 dark:text-gray-500" strokeWidth={2.5} />
      <div className="h-2 w-px bg-slate-300 dark:bg-gray-600" />
    </div>
  );
}

interface LineStepDisplayNodeProps {
  appName: string;
  entrypoint?: string;
  className?: string;
}

export function LineStepDisplayNode({ appName, entrypoint, className }: LineStepDisplayNodeProps) {
  return (
    <div
      className={cn(
        "w-full rounded-lg border border-slate-200 bg-white px-3 py-2.5 shadow-sm dark:border-gray-700 dark:bg-gray-900",
        className,
      )}
    >
      <p className="text-sm font-medium text-slate-900 dark:text-gray-100">{appName || "Unnamed automation"}</p>
      {entrypoint ? <p className="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{entrypoint}</p> : null}
    </div>
  );
}

interface LineStepFlowProps {
  children: ReactNode;
  className?: string;
  variant?: "compact" | "editor";
}

export function LineStepFlow({ children, className, variant = "compact" }: LineStepFlowProps) {
  return (
    <div className={cn("flex flex-col", variant === "editor" ? "w-full items-stretch" : "items-center", className)}>
      {children}
    </div>
  );
}

interface LineStepAddButtonProps {
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

export function LineStepAddButton({ onClick, disabled = false, className }: LineStepAddButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn("group flex flex-col items-center disabled:cursor-not-allowed disabled:opacity-50", className)}
      aria-label="Add step"
      data-testid="factory-line-add-step-button"
    >
      <LineStepArrow />
      <span
        className={cn(
          "flex h-8 w-8 items-center justify-center rounded-full border-2 border-dashed transition-colors",
          "border-slate-300 bg-slate-50 text-slate-500 group-hover:border-slate-400 group-hover:bg-slate-100 group-hover:text-slate-700",
          "dark:border-gray-600 dark:bg-gray-800/50 dark:text-gray-400 dark:group-hover:border-gray-500 dark:group-hover:bg-gray-800 dark:group-hover:text-gray-200",
        )}
      >
        <Plus className="h-4 w-4" aria-hidden />
      </span>
    </button>
  );
}

interface LineStepEditorShellProps {
  children: ReactNode;
  className?: string;
}

export function LineStepEditorShell({ children, className }: LineStepEditorShellProps) {
  return (
    <div
      className={cn(
        "w-full rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900",
        className,
      )}
    >
      {children}
    </div>
  );
}
